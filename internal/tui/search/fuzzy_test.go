package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch_SubstringMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		query   string
		wantOK  bool
		wantLen int // expected number of matched indices
	}{
		{
			name:    "exact substring",
			path:    "src/middleware/auth.ts",
			query:   "middleware",
			wantOK:  true,
			wantLen: 10,
		},
		{
			name:    "case insensitive substring",
			path:    "src/Main.go",
			query:   "main",
			wantOK:  true,
			wantLen: 4,
		},
		{
			name:   "no match",
			path:   "src/util.go",
			query:  "xyz",
			wantOK: false,
		},
		{
			name:    "empty query matches everything",
			path:    "anything.go",
			query:   "",
			wantOK:  true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok, indices := Match(tt.path, tt.query)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Len(t, indices, tt.wantLen)
			}
		})
	}
}

func TestMatch_FuzzySubsequence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		query  string
		wantOK bool
	}{
		{
			name:   "fuzzy match midl -> middleware",
			path:   "middleware.ts",
			query:  "midl",
			wantOK: true,
		},
		{
			name:   "fuzzy match across path",
			path:   "src/middleware/auth.ts",
			query:  "midl",
			wantOK: true,
		},
		{
			name:   "fuzzy case insensitive",
			path:   "src/MyFile.go",
			query:  "mfg",
			wantOK: true,
		},
		{
			name:   "fuzzy no match",
			path:   "src/util.go",
			query:  "zxy",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok, indices := Match(tt.path, tt.query)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.NotEmpty(t, indices)
			}
		})
	}
}

func TestMatch_IndicesAreCorrect(t *testing.T) {
	t.Parallel()

	ok, indices := Match("abcdef", "bdf")
	assert.True(t, ok)
	// b=1, d=3, f=5
	assert.Equal(t, []int{1, 3, 5}, indices)
}

func TestMatch_SubstringIndicesConsecutive(t *testing.T) {
	t.Parallel()

	ok, indices := Match("hello world", "world")
	assert.True(t, ok)
	// "world" starts at index 6.
	assert.Equal(t, []int{6, 7, 8, 9, 10}, indices)
}

func TestMatchSubstring(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		query  string
		wantOK bool
	}{
		{"simple match", "src/main.go", "main", true},
		{"case insensitive", "src/Main.go", "main", true},
		{"no match", "src/util.go", "xyz", false},
		{"empty query", "src/main.go", "", true},
		{"empty path", "", "main", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := MatchSubstring(tt.path, tt.query)
			assert.Equal(t, tt.wantOK, result)
		})
	}
}
