package golden

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		dir     string
		want    string
	}{
		{
			name:  "replaces ISO 8601 timestamp with Z",
			input: "Generated | 2025-01-15T10:30:00Z |",
			want:  "Generated | [TIMESTAMP] |",
		},
		{
			name:  "replaces ISO 8601 timestamp with offset",
			input: "time: 2026-02-26T14:30:45+05:30",
			want:  "time: [TIMESTAMP]",
		},
		{
			name:  "replaces hex hash after Hash pipe",
			input: "| Content Hash | abc123def45678 |",
			want:  "| Content Hash | [HASH] |",
		},
		{
			name:  "replaces hex hash in XML content_hash element",
			input: "<content_hash>abc123def45678</content_hash>",
			want:  "<content_hash>[HASH]</content_hash>",
		},
		{
			name:  "replaces timing values",
			input: "took 123ms to process, 1.5s total",
			want:  "took [TIME] to process, [TIME] total",
		},
		{
			name:  "replaces microsecond timing",
			input: "duration: 200µs",
			want:  "duration: [TIME]",
		},
		{
			name:  "replaces absolute paths",
			input: "path: /home/user/project/src/main.go",
			dir:   "/home/user/project",
			want:  "path: src/main.go",
		},
		{
			name:  "handles no replacements needed",
			input: "nothing to replace here",
			want:  "nothing to replace here",
		},
		{
			name:  "handles empty input",
			input: "",
			want:  "",
		},
		{
			name:  "replaces multiple timestamps",
			input: "start: 2025-01-01T00:00:00Z end: 2025-12-31T23:59:59Z",
			want:  "start: [TIMESTAMP] end: [TIMESTAMP]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize([]byte(tt.input), tt.dir)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestDiff(t *testing.T) {
	t.Run("different strings produce unified diff", func(t *testing.T) {
		d := Diff("line1\nline2\n", "line1\nchanged\n")
		assert.Contains(t, d, "expected")
		assert.Contains(t, d, "actual")
		assert.Contains(t, d, "-line2")
		assert.Contains(t, d, "+changed")
	})

	t.Run("single line difference", func(t *testing.T) {
		d := Diff("hello world\n", "hello earth\n")
		assert.Contains(t, d, "-hello world")
		assert.Contains(t, d, "+hello earth")
	})

	t.Run("identical strings return fallback message", func(t *testing.T) {
		d := Diff("same\n", "same\n")
		// Identical strings produce empty unified diff; our function
		// returns a descriptive fallback.
		assert.Contains(t, d, "outputs differ")
	})
}
