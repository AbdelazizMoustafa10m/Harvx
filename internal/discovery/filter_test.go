package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternFilter_Matches_NoFilters(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "go file passes", path: "main.go", want: true},
		{name: "ts file passes", path: "src/app.ts", want: true},
		{name: "deeply nested passes", path: "a/b/c/d/file.txt", want: true},
		{name: "empty path fails", path: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_IncludePatterns(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Includes: []string{"**/*.ts"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "ts file matches", path: "src/app.ts", want: true},
		{name: "deeply nested ts matches", path: "src/deep/nested/file.ts", want: true},
		{name: "root ts matches", path: "index.ts", want: true},
		{name: "go file does not match", path: "main.go", want: false},
		{name: "tsx does not match", path: "App.tsx", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_ExcludePatterns(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Excludes: []string{"**/*.test.ts"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "test file excluded", path: "src/app.test.ts", want: false},
		{name: "deeply nested test excluded", path: "src/deep/util.test.ts", want: false},
		{name: "regular ts passes", path: "src/app.ts", want: true},
		{name: "go file passes", path: "main.go", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_ExtensionFilters(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Extensions: []string{"ts", "go"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "ts file matches", path: "src/app.ts", want: true},
		{name: "go file matches", path: "cmd/main.go", want: true},
		{name: "py file does not match", path: "script.py", want: false},
		{name: "js file does not match", path: "index.js", want: false},
		{name: "no extension does not match", path: "Makefile", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_ExtensionCaseInsensitive(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Extensions: []string{"ts"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "lowercase ts", path: "app.ts", want: true},
		{name: "uppercase TS", path: "app.TS", want: true},
		{name: "mixed case Ts", path: "app.Ts", want: true},
		{name: "mixed case tS", path: "app.tS", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_ExtensionWithDotNormalized(t *testing.T) {
	t.Parallel()

	// Users might pass ".ts" instead of "ts"; the filter should handle both.
	f := NewPatternFilter(PatternFilterOptions{
		Extensions: []string{".ts", "..go"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "ts matches despite dot prefix", path: "app.ts", want: true},
		{name: "go matches despite double dot prefix", path: "main.go", want: true},
		{name: "py does not match", path: "script.py", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_ExcludeTakesPrecedence(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Includes: []string{"src/**/*.ts"},
		Excludes: []string{"**/*.test.ts"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "include matches but exclude wins", path: "src/app.test.ts", want: false},
		{name: "include matches no exclude", path: "src/app.ts", want: true},
		{name: "no include match", path: "main.go", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_IncludeAndExtensionORLogic(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Includes:   []string{"docs/**"},
		Extensions: []string{"go"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "go file matches via extension", path: "cmd/main.go", want: true},
		{name: "doc file matches via include", path: "docs/README.md", want: true},
		{name: "deeply nested doc matches", path: "docs/api/auth.md", want: true},
		{name: "py file outside docs fails", path: "script.py", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_DoublestarPatterns(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Includes: []string{"**/*.ts"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "root level", path: "index.ts", want: true},
		{name: "one level deep", path: "src/app.ts", want: true},
		{name: "two levels deep", path: "src/components/Button.ts", want: true},
		{name: "deeply nested", path: "src/deep/nested/very/deep/file.ts", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_MultipleExcludePatterns(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Excludes: []string{"**/*.test.ts", "**/*.spec.ts", "**/__tests__/**"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "test file excluded", path: "src/app.test.ts", want: false},
		{name: "spec file excluded", path: "src/app.spec.ts", want: false},
		{name: "tests dir excluded", path: "src/__tests__/app.ts", want: false},
		{name: "regular file passes", path: "src/app.ts", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_Matches_PathNormalization(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Includes: []string{"**/*.go"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "forward slash path", path: "cmd/main.go", want: true},
		{name: "leading dot-slash stripped", path: "./cmd/main.go", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestPatternFilter_HasFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts PatternFilterOptions
		want bool
	}{
		{
			name: "no filters",
			opts: PatternFilterOptions{},
			want: false,
		},
		{
			name: "includes only",
			opts: PatternFilterOptions{Includes: []string{"**/*.ts"}},
			want: true,
		},
		{
			name: "excludes only",
			opts: PatternFilterOptions{Excludes: []string{"**/*.test.ts"}},
			want: true,
		},
		{
			name: "extensions only",
			opts: PatternFilterOptions{Extensions: []string{"go"}},
			want: true,
		},
		{
			name: "all three",
			opts: PatternFilterOptions{
				Includes:   []string{"src/**"},
				Excludes:   []string{"**/*.test.ts"},
				Extensions: []string{"go"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := NewPatternFilter(tt.opts)
			assert.Equal(t, tt.want, f.HasFilters())
		})
	}
}

func TestPatternFilter_Matches_DirectoryPattern(t *testing.T) {
	t.Parallel()

	f := NewPatternFilter(PatternFilterOptions{
		Includes: []string{"src/**"},
	})

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "file under src", path: "src/main.go", want: true},
		{name: "file under src subdir", path: "src/pkg/util.go", want: true},
		{name: "file outside src", path: "cmd/main.go", want: false},
		{name: "file at root", path: "README.md", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, f.Matches(tt.path))
		})
	}
}

func TestNewPatternFilter_CopiesSlices(t *testing.T) {
	t.Parallel()

	includes := []string{"**/*.ts"}
	excludes := []string{"**/*.test.ts"}
	extensions := []string{"go"}

	f := NewPatternFilter(PatternFilterOptions{
		Includes:   includes,
		Excludes:   excludes,
		Extensions: extensions,
	})

	// Mutate the original slices.
	includes[0] = "MUTATED"
	excludes[0] = "MUTATED"
	extensions[0] = "MUTATED"

	// The filter should not be affected by external mutation.
	assert.True(t, f.Matches("src/app.ts"), "filter should still match ts files after external mutation")
	assert.False(t, f.Matches("src/app.test.ts"), "filter should still exclude test.ts files after external mutation")
}

func BenchmarkPatternFilter_Matches(b *testing.B) {
	f := NewPatternFilter(PatternFilterOptions{
		Includes:   []string{"src/**/*.ts", "lib/**/*.ts"},
		Excludes:   []string{"**/*.test.ts", "**/*.spec.ts", "**/__tests__/**"},
		Extensions: []string{"go", "rs"},
	})

	paths := []string{
		"src/components/Button.ts",
		"src/app.test.ts",
		"lib/utils/format.ts",
		"cmd/main.go",
		"README.md",
		"src/deep/nested/very/deep/file.ts",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			f.Matches(p)
		}
	}
}
