package pipeline

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAssertInclude(t *testing.T) {
	t.Parallel()

	files := []FileDescriptor{
		{Path: "src/middleware.ts"},
		{Path: "src/components/Button.tsx"},
		{Path: "prisma/schema.prisma"},
		{Path: "lib/services/auth.go"},
		{Path: "lib/services/db.go"},
		{Path: "README.md"},
		{Path: "docs/api.md"},
	}

	tests := []struct {
		name       string
		patterns   []string
		files      []FileDescriptor
		wantErr    bool
		wantCount  int // expected number of failures
		checkError func(t *testing.T, err error)
	}{
		{
			name:     "empty patterns is a no-op",
			patterns: nil,
			files:    files,
			wantErr:  false,
		},
		{
			name:     "empty slice patterns is a no-op",
			patterns: []string{},
			files:    files,
			wantErr:  false,
		},
		{
			name:     "single pattern matching one file passes",
			patterns: []string{"src/middleware.ts"},
			files:    files,
			wantErr:  false,
		},
		{
			name:      "single pattern matching zero files fails",
			patterns:  []string{"nonexistent.go"},
			files:     files,
			wantErr:   true,
			wantCount: 1,
		},
		{
			name:     "multiple patterns all pass",
			patterns: []string{"src/middleware.ts", "prisma/**", "lib/services/**"},
			files:    files,
			wantErr:  false,
		},
		{
			name:      "multiple patterns some fail lists all failures",
			patterns:  []string{"src/middleware.ts", "missing-dir/**", "also-missing.*"},
			files:     files,
			wantErr:   true,
			wantCount: 2,
		},
		{
			name:     "star wildcard matches files",
			patterns: []string{"*.md"},
			files:    files,
			wantErr:  false,
		},
		{
			name:     "doublestar wildcard matches nested files",
			patterns: []string{"lib/**"},
			files:    files,
			wantErr:  false,
		},
		{
			name:     "doublestar with extension",
			patterns: []string{"**/*.go"},
			files:    files,
			wantErr:  false,
		},
		{
			name:     "star wildcard in directory",
			patterns: []string{"src/*"},
			files:    files,
			wantErr:  false,
		},
		{
			name:      "pattern with no files in repo",
			patterns:  []string{"**/*.rs"},
			files:     files,
			wantErr:   true,
			wantCount: 1,
		},
		{
			name:      "all patterns fail",
			patterns:  []string{"nonexistent/**", "missing.*", "**/*.rs"},
			files:     files,
			wantErr:   true,
			wantCount: 3,
		},
		{
			name:      "empty file list with patterns fails",
			patterns:  []string{"**/*.go"},
			files:     []FileDescriptor{},
			wantErr:   true,
			wantCount: 1,
			checkError: func(t *testing.T, err error) {
				t.Helper()
				var ae *AssertionError
				require.ErrorAs(t, err, &ae)
				assert.Equal(t, 0, ae.Failures[0].TotalFiles)
			},
		},
		{
			name:     "exact file path match",
			patterns: []string{"prisma/schema.prisma"},
			files:    files,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckAssertInclude(tt.patterns, tt.files)

			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)

			var ae *AssertionError
			require.ErrorAs(t, err, &ae, "error should be *AssertionError")
			assert.Len(t, ae.Failures, tt.wantCount)

			// Verify each failure has the correct total file count.
			for _, f := range ae.Failures {
				assert.Equal(t, len(tt.files), f.TotalFiles)
			}

			if tt.checkError != nil {
				tt.checkError(t, err)
			}
		})
	}
}

func TestAssertionError_Message(t *testing.T) {
	t.Parallel()

	t.Run("single failure message contains pattern and suggestion", func(t *testing.T) {
		t.Parallel()
		err := &AssertionError{
			Failures: []AssertionFailure{
				{Pattern: "middleware.*", TotalFiles: 10},
			},
		}
		msg := err.Error()
		assert.Contains(t, msg, "assert-include failed")
		assert.Contains(t, msg, `"middleware.*"`)
		assert.Contains(t, msg, "0 of 10 files")
		assert.Contains(t, msg, "check profile ignore rules")
	})

	t.Run("multiple failures separated by semicolons", func(t *testing.T) {
		t.Parallel()
		err := &AssertionError{
			Failures: []AssertionFailure{
				{Pattern: "first.*", TotalFiles: 5},
				{Pattern: "second.**", TotalFiles: 5},
			},
		}
		msg := err.Error()
		assert.Contains(t, msg, `"first.*"`)
		assert.Contains(t, msg, `"second.**"`)
		assert.Contains(t, msg, "; ")
	})
}

func TestAssertionError_Is(t *testing.T) {
	t.Parallel()

	err := CheckAssertInclude([]string{"nonexistent"}, []FileDescriptor{{Path: "file.go"}})
	require.Error(t, err)

	var ae *AssertionError
	assert.True(t, errors.As(err, &ae))
}

func TestCheckAssertInclude_ProfileAndCLIMerge(t *testing.T) {
	t.Parallel()

	files := []FileDescriptor{
		{Path: "src/main.go"},
		{Path: "config/app.toml"},
	}

	// Simulate merging profile patterns with CLI patterns.
	profilePatterns := []string{"src/**"}
	cliPatterns := []string{"config/**"}
	merged := append(profilePatterns, cliPatterns...)

	err := CheckAssertInclude(merged, files)
	assert.NoError(t, err, "merged profile + CLI patterns should all pass")
}
