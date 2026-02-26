package config

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
)

// FuzzParseConfig feeds arbitrary strings to LoadFromString to verify that the
// parser never panics regardless of input. It checks the invariant that the
// function always returns either a non-nil *Config with a nil error, or a nil
// *Config with a non-nil error -- never both nil and never both non-nil error
// with a non-nil config.
func FuzzParseConfig(f *testing.F) {
	// Seed corpus: empty input, valid minimal configs, boundary cases.
	f.Add("")
	f.Add("[profile.default]\nformat = \"markdown\"\n")
	f.Add("[profile.default]\nmax_tokens = 128000\ntokenizer = \"cl100k_base\"\n")
	f.Add("[[[invalid")
	f.Add("[profile.default]\n[profile.default]\n")
	f.Add("[")
	f.Add(strings.Repeat(" ", 4096))
	f.Add("format\x00= \"markdown\"\x00max_tokens = 100")
	// Valid TOML with various field types.
	f.Add("[profile.default]\nformat = \"xml\"\nmax_tokens = 200000\ntokenizer = \"o200k_base\"\n")
	f.Add("[profile.base]\nformat = \"markdown\"\n\n[profile.child]\nextends = \"base\"\n")
	f.Add("[profile.default]\nignore = [\"node_modules\", \".git\"]\npriority_files = [\"README.md\"]\n")
	// Redaction config sub-table.
	f.Add("[profile.default.redaction_config]\nenabled = true\nconfidence_threshold = \"high\"\n")
	// Deeply nested relevance tiers.
	f.Add("[profile.default.relevance]\ntier_0 = [\"go.mod\"]\ntier_1 = [\"src/**\"]\n")
	// Malformed TOML variants.
	f.Add("[profile.")
	f.Add("[[profile]]")
	f.Add("[profile.default]\nmax_tokens = 99999999999999999999999999\n")
	f.Add(strings.Repeat("[profile.x]\nformat = \"markdown\"\n", 50))

	f.Fuzz(func(t *testing.T, input string) {
		cfg, err := LoadFromString(input, "fuzz")

		// Invariant: never both nil.
		if err == nil && cfg == nil {
			t.Fatal("LoadFromString returned nil config with nil error")
		}

		// When parsing succeeds, Validate and Lint must not panic.
		if cfg != nil {
			if err != nil {
				t.Fatal("LoadFromString returned non-nil config with non-nil error")
			}
			_ = Validate(cfg)
			_ = Lint(cfg)
		}
	})
}

// FuzzProfileInheritance builds a three-level profile inheritance chain from
// fuzzed name strings and verifies that ResolveProfile never panics and always
// returns either a valid ProfileResolution or a non-nil error.
func FuzzProfileInheritance(f *testing.F) {
	// Seed corpus: normal names, empty names, self-referential names.
	f.Add("child", "parent", "default")
	f.Add("a", "b", "c")
	f.Add("", "", "")
	f.Add("self", "self", "self")
	// Names that collide with the built-in default.
	f.Add("default", "default", "default")
	f.Add("child", "default", "default")
	// Non-ASCII names.
	f.Add("profil\xc3\xa9", "base", "root")
	// Very long name.
	f.Add(strings.Repeat("x", 1024), "parent", "grandparent")

	f.Fuzz(func(t *testing.T, childName, parentName, grandparentName string) {
		// Skip inputs that are not valid UTF-8 since TOML keys must be valid
		// UTF-8 and using invalid strings would test TOML encoding rather than
		// profile resolution logic.
		if !utf8.ValidString(childName) || !utf8.ValidString(parentName) || !utf8.ValidString(grandparentName) {
			t.Skip("skipping invalid UTF-8 input")
		}

		// Build profiles forming an extends chain:
		//   childName -> parentName -> grandparentName
		// Handle edge cases where names collide by deduplicating the map entries.
		profiles := make(map[string]*Profile)

		// Start from the grandparent (no extends, serves as the chain root).
		profiles[grandparentName] = &Profile{Format: "markdown"}

		// Parent extends grandparent, but only if the names differ.
		// When parentName == grandparentName, the parent entry overwrites the
		// grandparent with an extends pointing to itself (self-referential),
		// which ResolveProfile must handle gracefully.
		parentExtends := grandparentName
		profiles[parentName] = &Profile{Extends: &parentExtends}

		// Child extends parent.
		childExtends := parentName
		profiles[childName] = &Profile{Extends: &childExtends}

		// Must not panic.
		resolution, err := ResolveProfile(childName, profiles)

		// Invariant: either a valid resolution or an error, never both nil.
		if err == nil && resolution == nil {
			t.Fatal("ResolveProfile returned nil resolution with nil error")
		}

		// When resolution succeeds, verify structural invariants.
		if resolution != nil {
			if resolution.Profile == nil {
				t.Fatal("ProfileResolution.Profile is nil on successful resolve")
			}
			if len(resolution.Chain) == 0 {
				t.Fatal("ProfileResolution.Chain is empty on successful resolve")
			}
			// The Extends field must always be cleared after resolution.
			if resolution.Profile.Extends != nil {
				t.Fatal("ProfileResolution.Profile.Extends is not nil after resolution")
			}
		}
	})
}

// FuzzGlobPattern feeds arbitrary strings to doublestar.Match to verify that
// the glob matching library never panics on untrusted pattern input. The
// function returns (bool, error) and both outcomes are acceptable; only panics
// are considered failures.
func FuzzGlobPattern(f *testing.F) {
	// Seed corpus: valid patterns, edge cases, and invalid syntax.
	f.Add("**/*.go")
	f.Add("*.txt")
	f.Add("[")
	f.Add("**")
	f.Add("")
	f.Add("a]b[c")
	f.Add("{foo,bar}")
	f.Add("\\")
	f.Add("/path/to/file.go")
	// Additional edge cases.
	f.Add("[!a-z]")
	f.Add("**/[")
	f.Add("{")
	f.Add("{{nested}}")
	f.Add(strings.Repeat("*", 1000))
	f.Add("test/[a-")

	f.Fuzz(func(t *testing.T, pattern string) {
		// Must not panic regardless of pattern content.
		// Both true/false and error/nil are valid outcomes.
		_, _ = doublestar.Match(pattern, "test/file.go")

		// Also test with a varied set of target paths to exercise more
		// matching code paths.
		_, _ = doublestar.Match(pattern, "")
		_, _ = doublestar.Match(pattern, "deeply/nested/path/to/some/file.txt")
		_, _ = doublestar.Match(pattern, pattern) // pattern as both sides
	})
}
