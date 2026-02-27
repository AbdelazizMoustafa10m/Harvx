package diff

import (
	"fmt"
	"sort"
)

// DiffResult holds the outcome of comparing two StateSnapshot instances. Files
// are classified into added (new in current), modified (hash changed), and
// deleted (missing from current). Unchanged files are counted but not listed.
type DiffResult struct {
	// Added contains relative paths of files present in current but absent
	// from previous. Sorted alphabetically.
	Added []string

	// Modified contains relative paths of files present in both snapshots
	// whose ContentHash differs. Sorted alphabetically.
	Modified []string

	// Deleted contains relative paths of files present in previous but absent
	// from current. Sorted alphabetically.
	Deleted []string

	// Unchanged is the count of files present in both snapshots with matching
	// ContentHash values. These files are not listed individually.
	Unchanged int
}

// HasChanges reports whether any files were added, modified, or deleted.
func (r *DiffResult) HasChanges() bool {
	return len(r.Added) > 0 || len(r.Modified) > 0 || len(r.Deleted) > 0
}

// TotalChanged returns the total number of files that were added, modified, or
// deleted.
func (r *DiffResult) TotalChanged() int {
	return len(r.Added) + len(r.Modified) + len(r.Deleted)
}

// Summary returns a human-readable summary of the diff result, for example:
// "3 added, 5 modified, 1 deleted (42 unchanged)".
func (r *DiffResult) Summary() string {
	return fmt.Sprintf("%d added, %d modified, %d deleted (%d unchanged)",
		len(r.Added), len(r.Modified), len(r.Deleted), r.Unchanged)
}

// CompareStates performs an O(n) comparison of two state snapshots and returns
// a DiffResult classifying every file as added, modified, deleted, or unchanged.
//
// The algorithm uses two passes over the file maps:
//   - Pass 1: iterate current.Files to detect added and modified files.
//   - Pass 2: iterate previous.Files to detect deleted files.
//
// A nil previous snapshot is treated as empty (all current files are "added").
// A nil current snapshot is treated as empty (all previous files are "deleted").
func CompareStates(previous, current *StateSnapshot) *DiffResult {
	result := &DiffResult{}

	// Treat nil snapshots as having no files.
	var prevFiles map[string]FileState
	var currFiles map[string]FileState

	if previous != nil {
		prevFiles = previous.Files
	}
	if current != nil {
		currFiles = current.Files
	}

	// Pass 1: iterate current files -- detect added and modified.
	for path, currFile := range currFiles {
		prevFile, exists := prevFiles[path]
		if !exists {
			result.Added = append(result.Added, path)
			continue
		}

		// TODO: optimize with mod-time check
		if currFile.ContentHash != prevFile.ContentHash {
			result.Modified = append(result.Modified, path)
		} else {
			result.Unchanged++
		}
	}

	// Pass 2: iterate previous files -- detect deleted.
	for path := range prevFiles {
		if _, exists := currFiles[path]; !exists {
			result.Deleted = append(result.Deleted, path)
		}
	}

	// Sort for deterministic output.
	sort.Strings(result.Added)
	sort.Strings(result.Modified)
	sort.Strings(result.Deleted)

	return result
}
