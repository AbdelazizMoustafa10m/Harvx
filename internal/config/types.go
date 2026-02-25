package config

// Config is the top-level configuration type parsed from a harvx.toml file.
// It holds a map of named profiles keyed by profile name. Profile names are
// case-sensitive. The special name "default" is the built-in fallback profile.
type Config struct {
	// Profile maps profile names to their configuration. Access via
	// cfg.Profile["default"] or cfg.Profile["finvault"].
	Profile map[string]*Profile `toml:"profile"`
}

// Profile defines all settings for a single named profile. Fields with zero
// values are considered unset and will be filled in by the merge/inheritance
// pipeline (T-017, T-019). The Extends field enables profile inheritance.
type Profile struct {
	// Extends is the name of a parent profile to inherit from. When set,
	// all unset fields in this profile are filled from the named parent.
	// A nil pointer means no inheritance.
	Extends *string `toml:"extends"`

	// Output is the file path for the generated context document.
	// Example: "harvx-output.md" or ".harvx/finvault-context.md"
	Output string `toml:"output"`

	// Format controls the output format. Valid values: "markdown", "xml", "plain".
	Format string `toml:"format"`

	// MaxTokens is the token budget cap for the generated output.
	// Files are pruned from the output if the total exceeds this limit.
	MaxTokens int `toml:"max_tokens"`

	// BriefMaxTokens is the token budget for the Repo Brief artifact.
	// Controls the maximum size of output from `harvx brief`. Default: 4000.
	BriefMaxTokens int `toml:"brief_max_tokens"`

	// Tokenizer selects the token counting model. Valid values: "cl100k_base", "o200k_base".
	Tokenizer string `toml:"tokenizer"`

	// Compression enables Tree-sitter WASM compression for source files.
	Compression bool `toml:"compression"`

	// Redaction enables secret redaction before writing output.
	Redaction bool `toml:"redaction"`

	// Target selects LLM-specific output optimizations.
	// Valid values: "claude", "chatgpt", "generic", or empty string.
	Target string `toml:"target"`

	// Ignore is the list of glob patterns for files and directories to
	// skip during discovery. Patterns are evaluated with doublestar.
	Ignore []string `toml:"ignore"`

	// PriorityFiles is the ordered list of files that must be included in
	// the output before any tier-based sorting is applied.
	PriorityFiles []string `toml:"priority_files"`

	// Include is the list of glob patterns for files to explicitly include
	// even if they would otherwise be ignored.
	Include []string `toml:"include"`

	// AssertInclude is the list of glob patterns that must each match at least
	// one included file. If any pattern matches zero files, the pipeline fails
	// with exit code 1. Used for CI coverage checks.
	AssertInclude []string `toml:"assert_include"`

	// Relevance holds tier-based file sorting configuration. Each tier is
	// a list of glob patterns that match files assigned to that tier.
	Relevance RelevanceConfig `toml:"relevance"`

	// RedactionConfig holds fine-grained redaction settings.
	RedactionConfig RedactionConfig `toml:"redaction_config"`
}

// RelevanceConfig defines glob patterns for each relevance tier. Files are
// assigned to the lowest-numbered matching tier (Tier 0 is highest priority).
// All fields are slices of doublestar glob patterns.
type RelevanceConfig struct {
	// Tier0 contains the highest-priority files (configuration, schema files).
	Tier0 []string `toml:"tier_0"`

	// Tier1 contains primary source code directories.
	Tier1 []string `toml:"tier_1"`

	// Tier2 contains secondary source files, components, and utilities.
	Tier2 []string `toml:"tier_2"`

	// Tier3 contains test files.
	Tier3 []string `toml:"tier_3"`

	// Tier4 contains documentation.
	Tier4 []string `toml:"tier_4"`

	// Tier5 contains CI/CD configs, lock files, and lowest-priority files.
	Tier5 []string `toml:"tier_5"`
}

// RedactionConfig controls secret detection and redaction behavior.
type RedactionConfig struct {
	// Enabled turns secret redaction on or off for this profile.
	Enabled bool `toml:"enabled"`

	// ExcludePaths is the list of glob patterns for paths to skip during
	// redaction scanning (e.g., test fixtures and documentation).
	ExcludePaths []string `toml:"exclude_paths"`

	// ConfidenceThreshold controls which detected secrets are redacted.
	// Valid values: "low", "medium", "high". Defaults to "high".
	ConfidenceThreshold string `toml:"confidence_threshold"`

	// OverrideSensitiveDefaults suppresses the warning that is emitted when
	// a sensitive file (e.g., .env, *.pem) is explicitly included by the
	// profile. Set to true only when you understand the risk (e.g., including
	// .env.example with placeholder values). Default: false.
	OverrideSensitiveDefaults bool `toml:"override_sensitive_defaults"`

	// SensitivePatterns is an optional list of additional glob patterns to
	// treat as sensitive, beyond the built-in defaults. Profiles can extend
	// the default list but cannot silently remove from it.
	SensitivePatterns []string `toml:"sensitive_patterns"`

	// CustomPatterns is an optional list of project-specific redaction rules
	// defined in the TOML config. These are compiled and added alongside built-in
	// patterns when the redactor is instantiated.
	CustomPatterns []CustomPatternDefinition `toml:"custom_patterns"`
}

// CustomPatternDefinition describes a user-defined redaction rule in TOML config.
// It uses "regex" as the TOML key to match the spec, distinct from the
// security.CustomPatternConfig which uses "pattern" internally.
type CustomPatternDefinition struct {
	// ID is a unique identifier for this custom rule.
	ID string `toml:"id"`
	// Description is a short human-readable explanation.
	Description string `toml:"description"`
	// Regex is a Go RE2-compatible regular expression. Validated at config load time.
	Regex string `toml:"regex"`
	// SecretType is the category label for the redaction marker.
	SecretType string `toml:"secret_type"`
	// Confidence is the certainty level: "high", "medium", or "low".
	Confidence string `toml:"confidence"`
	// Keywords is an optional list of strings that must appear in the same
	// line for the pattern to trigger (case-insensitive pre-filter).
	Keywords []string `toml:"keywords"`
}
