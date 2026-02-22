package config

import (
	"strings"
	"testing"
)

// FuzzConfigParse feeds arbitrary byte sequences to LoadFromString to verify
// that the parser never panics regardless of input. On valid-looking TOML
// input, it additionally checks that either an error or a non-nil Config is
// returned (never both nil with no error).
func FuzzConfigParse(f *testing.F) {
	// Seed corpus: valid TOMLs covering different schema areas.
	f.Add([]byte(``))
	f.Add([]byte(`[profile.default]`))
	f.Add([]byte(`
[profile.default]
format = "markdown"
max_tokens = 128000
tokenizer = "cl100k_base"
compression = false
redaction = true
output = "harvx-output.md"
`))
	f.Add([]byte(`
[profile.default]
format = "xml"
max_tokens = 200000
tokenizer = "o200k_base"
target = "claude"
compression = true
`))
	f.Add([]byte(`
[profile.base]
format = "markdown"
max_tokens = 80000

[profile.child]
extends = "base"
format = "xml"
`))
	f.Add([]byte(`
[profile.default]
ignore = ["node_modules", "dist", ".git"]
priority_files = ["README.md", "go.mod"]
include = ["**/*.go"]

[profile.default.relevance]
tier_0 = ["package.json", "go.mod"]
tier_1 = ["src/**", "internal/**"]
`))
	f.Add([]byte(`
[profile.default.redaction_config]
enabled = true
exclude_paths = ["testdata/**"]
confidence_threshold = "high"
`))
	// Edge cases: truncated, binary-ish, duplicate keys.
	f.Add([]byte(`[profile`))
	f.Add([]byte(`[profile.`))
	f.Add([]byte(`[[profile]]`))
	f.Add([]byte("format = \"markdown\"\x00max_tokens = 100"))
	f.Add([]byte(`
[profile.default]
max_tokens = 99999999999999999999999999
`))
	f.Add([]byte(strings.Repeat("[profile.x]\nformat = \"markdown\"\n", 50)))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic under any input.
		cfg, err := LoadFromString(string(data), "fuzz")

		// Invariant: if err == nil then cfg must be non-nil.
		if err == nil && cfg == nil {
			t.Fatal("LoadFromString returned nil config with nil error")
		}
		// If cfg is non-nil, calling Validate must not panic.
		if cfg != nil {
			_ = Validate(cfg)
		}
	})
}

// FuzzValidate feeds random Config structs (parsed from arbitrary TOML) into
// the Validate function to verify it never panics.
func FuzzValidate(f *testing.F) {
	// Seed corpus: configs with various validation edge cases.
	f.Add([]byte(`
[profile.default]
format = "markdown"
max_tokens = 128000
tokenizer = "cl100k_base"
`))
	f.Add([]byte(`
[profile.bad]
format = "notaformat"
max_tokens = -1
tokenizer = "badtokenizer"
target = "badtarget"
`))
	f.Add([]byte(`
[profile.hardcap]
max_tokens = 9999999
`))
	f.Add([]byte(`
[profile.a]
extends = "b"

[profile.b]
extends = "a"
`))
	f.Add([]byte(`
[profile.default]
priority_files = ["**/*.go"]
ignore = ["**/*.go"]
`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		cfg, err := LoadFromString(string(data), "fuzz-validate")
		if err != nil || cfg == nil {
			return
		}
		// Must not panic.
		_ = Validate(cfg)
		// Lint also must not panic.
		_ = Lint(cfg)
	})
}
