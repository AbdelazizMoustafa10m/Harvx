package pipeline

import (
	"fmt"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// AssertionFailure records a single assert-include pattern that matched
// zero included files.
type AssertionFailure struct {
	// Pattern is the glob pattern that failed to match any file.
	Pattern string
	// TotalFiles is the total number of files that were checked.
	TotalFiles int
}

// AssertionError is returned when one or more --assert-include patterns
// fail to match any included files. It wraps all failures so the user
// can fix everything in one pass.
type AssertionError struct {
	Failures []AssertionFailure
}

// Error formats all assertion failures into a single error message.
func (e *AssertionError) Error() string {
	var b strings.Builder
	b.WriteString("assert-include failed: ")
	for i, f := range e.Failures {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "pattern %q matched 0 of %d files", f.Pattern, f.TotalFiles)
	}
	b.WriteString(" -- check profile ignore rules and relevance tier configuration")
	return b.String()
}

// CheckAssertInclude verifies that each pattern in patterns matches at least
// one file in files. Patterns use doublestar glob syntax (same as relevance
// tier patterns). An empty patterns slice is a no-op and returns nil.
//
// When multiple patterns fail, all failures are reported in the returned
// *AssertionError so the user can fix everything in one pass.
func CheckAssertInclude(patterns []string, files []FileDescriptor) error {
	if len(patterns) == 0 {
		return nil
	}

	totalFiles := len(files)
	var failures []AssertionFailure

	for _, pattern := range patterns {
		matched := false
		for _, fd := range files {
			ok, err := doublestar.Match(pattern, fd.Path)
			if err != nil {
				// Invalid pattern: treat as failure with descriptive message.
				failures = append(failures, AssertionFailure{
					Pattern:    pattern + " (invalid glob: " + err.Error() + ")",
					TotalFiles: totalFiles,
				})
				matched = true // Don't double-report.
				break
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			failures = append(failures, AssertionFailure{
				Pattern:    pattern,
				TotalFiles: totalFiles,
			})
		}
	}

	if len(failures) > 0 {
		return &AssertionError{Failures: failures}
	}
	return nil
}
