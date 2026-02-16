package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubIgnorer is a test double that always returns a fixed result.
type stubIgnorer struct {
	ignored bool
}

func (s *stubIgnorer) IsIgnored(_ string, _ bool) bool {
	return s.ignored
}

// recordingIgnorer records which paths were checked.
type recordingIgnorer struct {
	ignored bool
	calls   []string
}

func (r *recordingIgnorer) IsIgnored(path string, _ bool) bool {
	r.calls = append(r.calls, path)
	return r.ignored
}

func TestNewCompositeIgnorer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ignorers  []Ignorer
		wantCount int
	}{
		{
			name:      "no ignorers",
			ignorers:  nil,
			wantCount: 0,
		},
		{
			name:      "single ignorer",
			ignorers:  []Ignorer{&stubIgnorer{ignored: false}},
			wantCount: 1,
		},
		{
			name: "multiple ignorers",
			ignorers: []Ignorer{
				&stubIgnorer{ignored: false},
				&stubIgnorer{ignored: true},
				&stubIgnorer{ignored: false},
			},
			wantCount: 3,
		},
		{
			name: "nil ignorers are skipped",
			ignorers: []Ignorer{
				nil,
				&stubIgnorer{ignored: false},
				nil,
				&stubIgnorer{ignored: true},
				nil,
			},
			wantCount: 2,
		},
		{
			name:      "all nil ignorers",
			ignorers:  []Ignorer{nil, nil, nil},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewCompositeIgnorer(tt.ignorers...)
			require.NotNil(t, c)
			assert.Equal(t, tt.wantCount, c.IgnorerCount())
		})
	}
}

func TestCompositeIgnorer_IsIgnored_AnyMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ignorers []Ignorer
		expect   bool
	}{
		{
			name:     "no ignorers returns false",
			ignorers: nil,
			expect:   false,
		},
		{
			name: "all false returns false",
			ignorers: []Ignorer{
				&stubIgnorer{ignored: false},
				&stubIgnorer{ignored: false},
				&stubIgnorer{ignored: false},
			},
			expect: false,
		},
		{
			name: "first true returns true",
			ignorers: []Ignorer{
				&stubIgnorer{ignored: true},
				&stubIgnorer{ignored: false},
				&stubIgnorer{ignored: false},
			},
			expect: true,
		},
		{
			name: "last true returns true",
			ignorers: []Ignorer{
				&stubIgnorer{ignored: false},
				&stubIgnorer{ignored: false},
				&stubIgnorer{ignored: true},
			},
			expect: true,
		},
		{
			name: "middle true returns true",
			ignorers: []Ignorer{
				&stubIgnorer{ignored: false},
				&stubIgnorer{ignored: true},
				&stubIgnorer{ignored: false},
			},
			expect: true,
		},
		{
			name: "all true returns true",
			ignorers: []Ignorer{
				&stubIgnorer{ignored: true},
				&stubIgnorer{ignored: true},
				&stubIgnorer{ignored: true},
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewCompositeIgnorer(tt.ignorers...)
			got := c.IsIgnored("some/file.go", false)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestCompositeIgnorer_ShortCircuits(t *testing.T) {
	t.Parallel()

	// The first ignorer returns true, so the second should not be called.
	first := &recordingIgnorer{ignored: true}
	second := &recordingIgnorer{ignored: false}

	c := NewCompositeIgnorer(first, second)
	result := c.IsIgnored("test.go", false)

	assert.True(t, result)
	assert.Len(t, first.calls, 1)
	assert.Empty(t, second.calls, "second ignorer should not be called when first matches")
}

func TestCompositeIgnorer_AllCheckedWhenNoMatch(t *testing.T) {
	t.Parallel()

	first := &recordingIgnorer{ignored: false}
	second := &recordingIgnorer{ignored: false}
	third := &recordingIgnorer{ignored: false}

	c := NewCompositeIgnorer(first, second, third)
	result := c.IsIgnored("test.go", false)

	assert.False(t, result)
	assert.Len(t, first.calls, 1)
	assert.Len(t, second.calls, 1)
	assert.Len(t, third.calls, 1)
}

func TestCompositeIgnorer_WithRealMatchers(t *testing.T) {
	t.Parallel()

	// Create a default ignore matcher (has node_modules/, .env, etc.).
	defaults := NewDefaultIgnoreMatcher()

	// Create a composite with only defaults.
	c := NewCompositeIgnorer(defaults)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "node_modules ignored", path: "node_modules", isDir: true, expect: true},
		{name: ".env ignored", path: ".env", isDir: false, expect: true},
		{name: "go.sum ignored", path: "go.sum", isDir: false, expect: true},
		{name: "main.go not ignored", path: "main.go", isDir: false, expect: false},
		{name: "README not ignored", path: "README.md", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestCompositeIgnorer_ChainsDefaultsAndGitignore(t *testing.T) {
	t.Parallel()

	// Create a temp directory with a .gitignore.
	dir := t.TempDir()
	writeGitignore(t, dir, "*.log\ncustom-dir/\n")

	gitMatcher, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	defaults := NewDefaultIgnoreMatcher()
	c := NewCompositeIgnorer(defaults, gitMatcher)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		// Default patterns.
		{name: "node_modules from defaults", path: "node_modules", isDir: true, expect: true},
		{name: ".env from defaults", path: ".env", isDir: false, expect: true},
		// Gitignore patterns.
		{name: "log from gitignore", path: "error.log", isDir: false, expect: true},
		{name: "custom-dir from gitignore", path: "custom-dir", isDir: true, expect: true},
		// Neither.
		{name: "normal file not ignored", path: "main.go", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestCompositeIgnorer_ImplementsIgnorer(t *testing.T) {
	t.Parallel()

	// Verify CompositeIgnorer can be used as an Ignorer (compile-time check
	// is also in ignore.go, but this is an explicit runtime verification).
	var ig Ignorer = NewCompositeIgnorer()
	assert.NotNil(t, ig)
	assert.False(t, ig.IsIgnored("test.go", false))
}
