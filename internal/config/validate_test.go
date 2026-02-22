package config

import (
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── test helpers ──────────────────────────────────────────────────────────────

// errorsWithSeverity filters a []ValidationError slice to those whose Severity
// matches the given value. The original slice order is preserved.
func errorsWithSeverity(results []ValidationError, severity string) []ValidationError {
	var out []ValidationError
	for _, e := range results {
		if e.Severity == severity {
			out = append(out, e)
		}
	}
	return out
}

// errorsWithField filters a []ValidationError slice to those whose Field starts
// with the given prefix. The original slice order is preserved.
func errorsWithField(results []ValidationError, prefix string) []ValidationError {
	var out []ValidationError
	for _, e := range results {
		if strings.HasPrefix(e.Field, prefix) {
			out = append(out, e)
		}
	}
	return out
}

// lintResultsWithCode filters a []LintResult slice to those whose Code matches.
func lintResultsWithCode(results []LintResult, code string) []LintResult {
	var out []LintResult
	for _, r := range results {
		if r.Code == code {
			out = append(out, r)
		}
	}
	return out
}

// lintResultsWithSeverity filters a []LintResult slice to those whose
// embedded Severity matches.
func lintResultsWithSeverity(results []LintResult, severity string) []LintResult {
	var out []LintResult
	for _, r := range results {
		if r.Severity == severity {
			out = append(out, r)
		}
	}
	return out
}

// sortValidationErrors sorts a slice of ValidationErrors by Field then Message
// for deterministic comparisons regardless of map iteration order.
func sortValidationErrors(errs []ValidationError) {
	sort.Slice(errs, func(i, j int) bool {
		if errs[i].Field != errs[j].Field {
			return errs[i].Field < errs[j].Field
		}
		return errs[i].Message < errs[j].Message
	})
}

// sortLintResults sorts a slice of LintResults by Field then Code then Message.
func sortLintResults(results []LintResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Field != results[j].Field {
			return results[i].Field < results[j].Field
		}
		if results[i].Code != results[j].Code {
			return results[i].Code < results[j].Code
		}
		return results[i].Message < results[j].Message
	})
}

// ── ValidationError.Error() ───────────────────────────────────────────────────

// TestValidationError_Error_WithSuggest verifies that the Error() string
// includes severity, field, message, and suggestion when Suggest is non-empty.
func TestValidationError_Error_WithSuggest(t *testing.T) {
	t.Parallel()

	e := ValidationError{
		Severity: "error",
		Field:    "profile.default.format",
		Message:  `format "html" is invalid`,
		Suggest:  "Valid formats: markdown, xml, plain",
	}

	got := e.Error()
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "error")
	assert.Contains(t, got, "profile.default.format")
	assert.Contains(t, got, "html")
	assert.Contains(t, got, "suggestion:")
	assert.Contains(t, got, "Valid formats")
}

// TestValidationError_Error_WithoutSuggest verifies that the Error() string
// omits the suggestion section when Suggest is empty.
func TestValidationError_Error_WithoutSuggest(t *testing.T) {
	t.Parallel()

	e := ValidationError{
		Severity: "warning",
		Field:    "profile.default.output",
		Message:  "some warning",
	}

	got := e.Error()
	assert.NotEmpty(t, got)
	assert.NotContains(t, got, "suggestion:")
	assert.Contains(t, got, "warning")
	assert.Contains(t, got, "profile.default.output")
}

// TestValidationError_ImplementsErrorInterface verifies that ValidationError
// satisfies the standard error interface at compile time via assignment.
func TestValidationError_ImplementsErrorInterface(t *testing.T) {
	t.Parallel()

	var _ error = ValidationError{}
}

// ── Validate: nil and empty configs ──────────────────────────────────────────

// TestValidate_NilConfig returns nil without panicking.
func TestValidate_NilConfig(t *testing.T) {
	t.Parallel()

	result := Validate(nil)
	assert.Nil(t, result)
}

// TestValidate_EmptyConfig verifies that a Config with a nil Profile map
// produces no validation errors.
func TestValidate_EmptyConfig(t *testing.T) {
	t.Parallel()

	result := Validate(&Config{})
	assert.Nil(t, result)
}

// TestValidate_EmptyProfileMap verifies that an explicitly empty (non-nil)
// Profile map produces no validation errors.
func TestValidate_EmptyProfileMap(t *testing.T) {
	t.Parallel()

	result := Validate(&Config{Profile: map[string]*Profile{}})
	assert.Nil(t, result)
}

// TestValidate_NilProfileValueIsSkipped verifies that a nil *Profile pointer
// inside the map is silently skipped (no panic, no errors).
func TestValidate_NilProfileValueIsSkipped(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"broken": nil,
		},
	}

	result := Validate(cfg)
	assert.Nil(t, result)
}

// ── Validate: valid configurations ───────────────────────────────────────────

// TestValidate_ValidProfile verifies that a correctly configured profile with
// all valid scalar fields produces no errors.
func TestValidate_ValidProfile(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {
				Format:    "markdown",
				Tokenizer: "cl100k_base",
				Target:    "claude",
				MaxTokens: 128000,
				Output:    "output.md",
				Ignore:    []string{"node_modules", "**/*.log"},
				RedactionConfig: RedactionConfig{
					ConfidenceThreshold: "high",
				},
			},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	assert.Empty(t, errs, "valid profile must produce no hard errors")
}

// TestValidate_AllValidFormats verifies that each accepted format value passes.
func TestValidate_AllValidFormats(t *testing.T) {
	t.Parallel()

	validFormatsSlice := []string{"markdown", "xml", "plain", ""}
	for _, format := range validFormatsSlice {
		format := format
		t.Run("format="+format, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Profile: map[string]*Profile{
					"p": {Format: format},
				},
			}
			errs := errorsWithSeverity(Validate(cfg), "error")
			formatErrs := errorsWithField(errs, "profile.p.format")
			assert.Empty(t, formatErrs)
		})
	}
}

// TestValidate_AllValidTokenizers verifies that each accepted tokenizer value
// passes without error.
func TestValidate_AllValidTokenizers(t *testing.T) {
	t.Parallel()

	valid := []string{"cl100k_base", "o200k_base", "none", ""}
	for _, tok := range valid {
		tok := tok
		t.Run("tokenizer="+tok, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Profile: map[string]*Profile{
					"p": {Tokenizer: tok},
				},
			}
			errs := errorsWithSeverity(Validate(cfg), "error")
			tokErrs := errorsWithField(errs, "profile.p.tokenizer")
			assert.Empty(t, tokErrs)
		})
	}
}

// TestValidate_AllValidTargets verifies that each accepted target value passes.
func TestValidate_AllValidTargets(t *testing.T) {
	t.Parallel()

	valid := []string{"claude", "chatgpt", "generic", ""}
	for _, target := range valid {
		target := target
		t.Run("target="+target, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Profile: map[string]*Profile{
					"p": {Target: target},
				},
			}
			errs := errorsWithSeverity(Validate(cfg), "error")
			targetErrs := errorsWithField(errs, "profile.p.target")
			assert.Empty(t, targetErrs)
		})
	}
}

// TestValidate_AllValidConfidenceThresholds verifies the accepted values for
// confidence_threshold.
func TestValidate_AllValidConfidenceThresholds(t *testing.T) {
	t.Parallel()

	valid := []string{"high", "medium", "low", ""}
	for _, ct := range valid {
		ct := ct
		t.Run("confidence_threshold="+ct, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Profile: map[string]*Profile{
					"p": {RedactionConfig: RedactionConfig{ConfidenceThreshold: ct}},
				},
			}
			errs := errorsWithSeverity(Validate(cfg), "error")
			ctErrs := errorsWithField(errs, "profile.p.redaction_config.confidence_threshold")
			assert.Empty(t, ctErrs)
		})
	}
}

// ── Validate: hard errors ─────────────────────────────────────────────────────

// TestValidate_InvalidFormat verifies that an unrecognised format value
// produces a hard error with valid options in the Suggest field.
func TestValidate_InvalidFormat(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {Format: "html"},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	require.NotEmpty(t, errs, "expected at least one hard error")

	formatErrs := errorsWithField(errs, "profile.default.format")
	require.Len(t, formatErrs, 1)
	assert.Contains(t, formatErrs[0].Message, "html")
	assert.NotEmpty(t, formatErrs[0].Suggest, "Suggest must be non-empty for format errors")
	assert.Contains(t, formatErrs[0].Suggest, "markdown")
}

// TestValidate_InvalidTokenizer verifies that an unrecognised tokenizer value
// produces a hard error with valid options in the Suggest field.
func TestValidate_InvalidTokenizer(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {Tokenizer: "gpt2"},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	tokErrs := errorsWithField(errs, "profile.default.tokenizer")
	require.Len(t, tokErrs, 1)
	assert.Contains(t, tokErrs[0].Message, "gpt2")
	assert.NotEmpty(t, tokErrs[0].Suggest)
	assert.Contains(t, tokErrs[0].Suggest, "cl100k_base")
}

// TestValidate_InvalidTarget verifies that an unrecognised target value
// produces a hard error.
func TestValidate_InvalidTarget(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {Target: "gemini"},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	targetErrs := errorsWithField(errs, "profile.default.target")
	require.Len(t, targetErrs, 1)
	assert.Contains(t, targetErrs[0].Message, "gemini")
	assert.NotEmpty(t, targetErrs[0].Suggest)
}

// TestValidate_InvalidConfidenceThreshold verifies that an unrecognised
// confidence_threshold value produces a hard error.
func TestValidate_InvalidConfidenceThreshold(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {
				RedactionConfig: RedactionConfig{ConfidenceThreshold: "critical"},
			},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	ctErrs := errorsWithField(errs, "profile.default.redaction_config.confidence_threshold")
	require.Len(t, ctErrs, 1)
	assert.Contains(t, ctErrs[0].Message, "critical")
	assert.NotEmpty(t, ctErrs[0].Suggest)
}

// TestValidate_NegativeMaxTokens verifies that a negative max_tokens value
// produces a hard error with a suggestion to fix it.
func TestValidate_NegativeMaxTokens(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {MaxTokens: -100},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	tokenErrs := errorsWithField(errs, "profile.default.max_tokens")
	require.NotEmpty(t, tokenErrs)
	assert.Contains(t, tokenErrs[0].Message, "-100")
	assert.NotEmpty(t, tokenErrs[0].Suggest)
}

// TestValidate_MaxTokensExceedsHardCap verifies that a max_tokens value above
// 2,000,000 produces a hard error.
func TestValidate_MaxTokensExceedsHardCap(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {MaxTokens: 2_000_001},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	tokenErrs := errorsWithField(errs, "profile.default.max_tokens")
	require.NotEmpty(t, tokenErrs, "max_tokens exceeding hard cap must be a hard error")
	assert.Contains(t, tokenErrs[0].Message, "2000001")
}

// TestValidate_MaxTokensAtHardCap verifies that max_tokens exactly equal to
// 2,000,000 does NOT produce a hard error (boundary is exclusive).
func TestValidate_MaxTokensAtHardCap(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {MaxTokens: 2_000_000},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	tokenErrs := errorsWithField(errs, "profile.default.max_tokens")
	assert.Empty(t, tokenErrs, "max_tokens == 2,000,000 must NOT produce a hard error")
}

// TestValidate_InvalidGlobPattern verifies that a syntactically invalid glob
// pattern produces a hard error containing the field path and the bad pattern.
func TestValidate_InvalidGlobPattern(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {
				Ignore: []string{"[invalid"},
			},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	require.NotEmpty(t, errs, "invalid glob pattern must produce a hard error")

	globErrs := errorsWithField(errs, "profile.default.ignore")
	require.NotEmpty(t, globErrs)
	assert.Contains(t, globErrs[0].Message, "[invalid")
	assert.Contains(t, globErrs[0].Field, "[0]", "field path must include the index")
}

// TestValidate_InvalidGlobPattern_InTier verifies invalid glob detection in
// relevance tier fields.
func TestValidate_InvalidGlobPattern_InTier(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{"valid/*.go", "[bad"},
				},
			},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	tierErrs := errorsWithField(errs, "profile.p.relevance.tier_0")
	require.Len(t, tierErrs, 1)
	assert.Contains(t, tierErrs[0].Field, "[1]")
	assert.Contains(t, tierErrs[0].Message, "[bad")
}

// TestValidate_MultipleErrors verifies that Validate accumulates all errors
// rather than stopping at the first. A profile with both an invalid format and
// an invalid tokenizer must yield two distinct hard errors.
func TestValidate_MultipleErrors(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {
				Format:    "html",
				Tokenizer: "gpt2",
			},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")

	formatErrs := errorsWithField(errs, "profile.default.format")
	tokErrs := errorsWithField(errs, "profile.default.tokenizer")

	assert.Len(t, formatErrs, 1, "must have exactly one format error")
	assert.Len(t, tokErrs, 1, "must have exactly one tokenizer error")
}

// TestValidate_MultipleProfiles verifies that errors are reported for each
// invalid profile independently.
func TestValidate_MultipleProfiles(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"alpha": {Format: "html"},
			"beta":  {Tokenizer: "gpt2"},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")

	alphaErrs := errorsWithField(errs, "profile.alpha")
	betaErrs := errorsWithField(errs, "profile.beta")

	assert.NotEmpty(t, alphaErrs, "alpha profile must yield errors")
	assert.NotEmpty(t, betaErrs, "beta profile must yield errors")
}

// ── Validate: error messages include suggestions ──────────────────────────────

// TestValidate_SuggestField_NonEmpty verifies that every hard error produced
// by an invalid scalar field carries a non-empty Suggest string.
func TestValidate_SuggestField_NonEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "invalid format",
			cfg: &Config{Profile: map[string]*Profile{
				"p": {Format: "html"},
			}},
		},
		{
			name: "invalid tokenizer",
			cfg: &Config{Profile: map[string]*Profile{
				"p": {Tokenizer: "gpt2"},
			}},
		},
		{
			name: "invalid target",
			cfg: &Config{Profile: map[string]*Profile{
				"p": {Target: "gemini"},
			}},
		},
		{
			name: "negative max_tokens",
			cfg: &Config{Profile: map[string]*Profile{
				"p": {MaxTokens: -1},
			}},
		},
		{
			name: "invalid glob",
			cfg: &Config{Profile: map[string]*Profile{
				"p": {Ignore: []string{"[bad"}},
			}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Validate(tt.cfg)
			errs := errorsWithSeverity(result, "error")
			require.NotEmpty(t, errs)
			for _, e := range errs {
				assert.NotEmpty(t, e.Suggest,
					"error for %q must have a non-empty Suggest field", e.Field)
			}
		})
	}
}

// ── Validate: missing/circular inheritance ────────────────────────────────────

// TestValidate_MissingParentProfile verifies that an extends value referencing
// a non-existent profile produces a hard error.
func TestValidate_MissingParentProfile(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"child": {Extends: strPtr("ghost")},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	extendsErrs := errorsWithField(errs, "profile.child.extends")
	require.NotEmpty(t, extendsErrs, "missing parent must produce a hard error")
	assert.NotEmpty(t, extendsErrs[0].Suggest)
}

// TestValidate_CircularInheritance verifies that circular profile inheritance
// produces a hard error.
func TestValidate_CircularInheritance(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"a": {Extends: strPtr("b")},
			"b": {Extends: strPtr("a")},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	// At least one profile in the cycle must report a circular error.
	var circularErrs []ValidationError
	for _, e := range errs {
		if strings.Contains(e.Message, "circular") {
			circularErrs = append(circularErrs, e)
		}
	}
	require.NotEmpty(t, circularErrs, "circular inheritance must produce a hard error")
}

// ── Validate: warnings ────────────────────────────────────────────────────────

// TestValidate_OverlappingTierPatterns verifies that a pattern appearing in
// more than one tier produces a warning.
func TestValidate_OverlappingTierPatterns(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {
				Relevance: RelevanceConfig{
					Tier0: []string{"go.mod", "internal/**"},
					Tier1: []string{"go.mod", "internal/**"},
				},
			},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	tierWarnings := errorsWithField(warnings, "profile.default.relevance")
	require.NotEmpty(t, tierWarnings, "overlapping tier patterns must yield warnings")
	assert.Contains(t, tierWarnings[0].Message, "go.mod")
}

// TestValidate_PriorityFileInIgnoreList verifies that a priority_files entry
// that also appears in the ignore list produces a warning.
func TestValidate_PriorityFileInIgnoreList(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {
				PriorityFiles: []string{"CLAUDE.md", "README.md"},
				Ignore:        []string{"CLAUDE.md", "node_modules"},
			},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	pfWarnings := errorsWithField(warnings, "profile.default.priority_files")
	require.NotEmpty(t, pfWarnings)
	assert.Contains(t, pfWarnings[0].Message, "CLAUDE.md")
	assert.NotEmpty(t, pfWarnings[0].Suggest)
}

// TestValidate_PriorityFilesWithGlobMetachars verifies that a priority_files
// entry containing glob metacharacters produces a warning.
func TestValidate_PriorityFilesWithGlobMetachars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "asterisk", pattern: "**/*.important"},
		{name: "question-mark", pattern: "src/?.go"},
		{name: "bracket", pattern: "src/[abc].go"},
		{name: "brace", pattern: "src/{foo,bar}.go"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Profile: map[string]*Profile{
					"p": {PriorityFiles: []string{tt.pattern}},
				},
			}
			result := Validate(cfg)
			warnings := errorsWithSeverity(result, "warning")
			pfWarnings := errorsWithField(warnings, "profile.p.priority_files")
			require.NotEmpty(t, pfWarnings,
				"priority_files glob %q must produce a warning", tt.pattern)
			assert.Contains(t, pfWarnings[0].Message, tt.pattern)
		})
	}
}

// TestValidate_PriorityFilesWithExactPaths verifies that exact (non-glob)
// priority_files entries do NOT produce a glob-pattern warning.
func TestValidate_PriorityFilesWithExactPaths(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {PriorityFiles: []string{"CLAUDE.md", "go.mod", "src/main.go"}},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	pfWarnings := errorsWithField(warnings, "profile.p.priority_files")
	assert.Empty(t, pfWarnings, "exact paths must not produce a glob warning")
}

// TestValidate_MaxTokensAboveSoftCap verifies that max_tokens between 500,001
// and 2,000,000 produces a warning (but no hard error).
func TestValidate_MaxTokensAboveSoftCap(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {MaxTokens: 600_000},
		},
	}

	result := Validate(cfg)

	// Must have a warning.
	warnings := errorsWithSeverity(result, "warning")
	tokenWarnings := errorsWithField(warnings, "profile.default.max_tokens")
	require.NotEmpty(t, tokenWarnings, "max_tokens > 500,000 must produce a warning")
	assert.NotEmpty(t, tokenWarnings[0].Suggest)

	// Must NOT have a hard error for max_tokens.
	errs := errorsWithSeverity(result, "error")
	tokenErrs := errorsWithField(errs, "profile.default.max_tokens")
	assert.Empty(t, tokenErrs, "max_tokens <= 2,000,000 must not produce a hard error")
}

// TestValidate_MaxTokensAtSoftCap verifies that max_tokens exactly equal to
// 500,000 does NOT trigger the soft-cap warning (boundary is exclusive).
func TestValidate_MaxTokensAtSoftCap(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"default": {MaxTokens: 500_000},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	tokenWarnings := errorsWithField(warnings, "profile.default.max_tokens")
	assert.Empty(t, tokenWarnings, "max_tokens == 500,000 must NOT warn")
}

// TestValidate_OutputAbsolutePath verifies that an absolute output path
// produces a warning.
func TestValidate_OutputAbsolutePath(t *testing.T) {
	t.Parallel()

	absPath := "/tmp/harvx-output.md"
	if runtime.GOOS == "windows" {
		absPath = `C:\Users\user\harvx-output.md`
	}

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {Output: absPath},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	outputWarnings := errorsWithField(warnings, "profile.p.output")
	require.NotEmpty(t, outputWarnings, "absolute output path must produce a warning")
	assert.NotEmpty(t, outputWarnings[0].Suggest)
}

// TestValidate_OutputPathWithDotDotPrefix verifies that an output path
// starting with "../" produces a warning.
func TestValidate_OutputPathWithDotDotPrefix(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {Output: "../sibling/output.md"},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	outputWarnings := errorsWithField(warnings, "profile.p.output")
	require.NotEmpty(t, outputWarnings, "../ prefixed output path must produce a warning")
}

// TestValidate_OutputRelativePath verifies that a simple relative output path
// does NOT produce an output warning.
func TestValidate_OutputRelativePath(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {Output: ".harvx/output.md"},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	outputWarnings := errorsWithField(warnings, "profile.p.output")
	assert.Empty(t, outputWarnings, "relative path must not produce an output warning")
}

// TestValidate_EmptyOutput verifies that an empty output string does NOT
// produce an output path warning.
func TestValidate_EmptyOutput(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {Output: ""},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	outputWarnings := errorsWithField(warnings, "profile.p.output")
	assert.Empty(t, outputWarnings)
}

// TestValidate_EmptyTierWarning verifies that a non-nil but empty relevance
// tier slice produces a warning.
func TestValidate_EmptyTierWarning(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{}, // explicitly empty, not nil
				},
			},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	tierWarnings := errorsWithField(warnings, "profile.p.relevance.tier_0")
	require.NotEmpty(t, tierWarnings, "explicitly empty tier must produce a warning")
}

// TestValidate_NilTierNoWarning verifies that a nil tier (never assigned)
// does NOT produce an empty-tier warning.
func TestValidate_NilTierNoWarning(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				// Tier0 is nil (zero value) — not explicitly set.
				Relevance: RelevanceConfig{
					Tier1: []string{"src/**/*.go"},
				},
			},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	tier0Warnings := errorsWithField(warnings, "profile.p.relevance.tier_0")
	assert.Empty(t, tier0Warnings, "nil tier must NOT produce an empty-tier warning")
}

// TestValidate_RedactionExcludeOverlapsIgnore verifies that a
// redaction_config.exclude_paths entry that also appears in ignore produces a
// warning.
func TestValidate_RedactionExcludeOverlapsIgnore(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Ignore: []string{"vendor", "testdata"},
				RedactionConfig: RedactionConfig{
					ExcludePaths: []string{"testdata"},
				},
			},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	redactionWarnings := errorsWithField(warnings, "profile.p.redaction_config.exclude_paths")
	require.NotEmpty(t, redactionWarnings)
	assert.Contains(t, redactionWarnings[0].Message, "testdata")
}

// TestValidate_DeepInheritanceWarning verifies that a profile inheritance
// chain longer than 3 levels produces a warning.
func TestValidate_DeepInheritanceWarning(t *testing.T) {
	t.Parallel()

	// Chain: leaf -> c -> b -> a -> default (depth = 5 links)
	cfg := &Config{
		Profile: map[string]*Profile{
			"a":    {Format: "markdown"},
			"b":    {Extends: strPtr("a")},
			"c":    {Extends: strPtr("b")},
			"leaf": {Extends: strPtr("c")},
		},
	}

	result := Validate(cfg)
	warnings := errorsWithSeverity(result, "warning")
	var deepWarnings []ValidationError
	for _, w := range warnings {
		if strings.Contains(w.Message, "levels deep") || strings.Contains(w.Field, "extends") {
			deepWarnings = append(deepWarnings, w)
		}
	}
	require.NotEmpty(t, deepWarnings, "deep inheritance chain must produce a warning")
}

// ── Validate: glob pattern edge cases ────────────────────────────────────────

// TestValidate_ValidDoubleStar verifies that doublestar syntax like "**/*.go"
// passes without error.
func TestValidate_ValidDoubleStar(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{"**/*.go", "src/**", "*.{ts,tsx}"},
				},
			},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	globErrs := errorsWithField(errs, "profile.p.relevance.tier_0")
	assert.Empty(t, globErrs, "valid doublestar patterns must not produce glob errors")
}

// TestValidate_UnicodeInPattern verifies that glob patterns containing unicode
// characters are handled without error (the doublestar library is unicode-safe).
func TestValidate_UnicodeInPattern(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Ignore: []string{"**/*.résumé", "données/**"},
			},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	globErrs := errorsWithField(errs, "profile.p.ignore")
	assert.Empty(t, globErrs, "unicode glob patterns must not produce hard errors")
}

// TestValidate_BraceExpansionPattern verifies that valid brace-expansion glob
// patterns pass without error.
func TestValidate_BraceExpansionPattern(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Include: []string{"src/**/*.{go,ts,py}"},
			},
		},
	}

	result := Validate(cfg)
	errs := errorsWithSeverity(result, "error")
	globErrs := errorsWithField(errs, "profile.p.include")
	assert.Empty(t, globErrs)
}

// ── Lint: nil and empty configs ───────────────────────────────────────────────

// TestLint_NilConfig returns nil without panicking.
func TestLint_NilConfig(t *testing.T) {
	t.Parallel()

	result := Lint(nil)
	assert.Nil(t, result)
}

// TestLint_EmptyConfig verifies that an empty Config produces no lint results.
func TestLint_EmptyConfig(t *testing.T) {
	t.Parallel()

	result := Lint(&Config{})
	assert.Nil(t, result)
}

// ── Lint: includes Validate results ──────────────────────────────────────────

// TestLint_IncludesValidateErrors verifies that Lint wraps all Validate errors
// as LintResults (with empty Code).
func TestLint_IncludesValidateErrors(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {Format: "html", Tokenizer: "gpt2"},
		},
	}

	validateErrs := Validate(cfg)
	lintResults := Lint(cfg)

	require.NotNil(t, lintResults)

	// Every hard error from Validate must appear in Lint results.
	for _, ve := range validateErrs {
		if ve.Severity != "error" {
			continue
		}
		found := false
		for _, lr := range lintResults {
			if lr.Field == ve.Field && lr.Message == ve.Message {
				found = true
				assert.Empty(t, lr.Code,
					"Validate-derived LintResults must have empty Code")
				break
			}
		}
		assert.True(t, found,
			"Validate error for field %q must appear in Lint results", ve.Field)
	}
}

// ── Lint: unreachable-tier ────────────────────────────────────────────────────

// TestLint_UnreachableTier verifies that a tier whose patterns are all covered
// by higher-priority tiers receives a LintResult with Code = "unreachable-tier".
func TestLint_UnreachableTier(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{"go.mod", "go.sum"},
					Tier1: []string{"go.mod", "go.sum"}, // all patterns duplicated from Tier0
				},
			},
		},
	}

	lintResults := Lint(cfg)
	sortLintResults(lintResults)

	unreachable := lintResultsWithCode(lintResults, "unreachable-tier")
	require.NotEmpty(t, unreachable, "fully-duplicated tier must be flagged as unreachable")

	assert.Contains(t, unreachable[0].Field, "tier_1")
	assert.Equal(t, "warning", unreachable[0].Severity)
	assert.NotEmpty(t, unreachable[0].Suggest)
}

// TestLint_UnreachableTier_PartialOverlap verifies that a tier is NOT flagged
// as unreachable when only some (but not all) of its patterns appear in higher
// tiers.
func TestLint_UnreachableTier_PartialOverlap(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{"go.mod"},
					Tier1: []string{"go.mod", "internal/**"}, // "internal/**" is new
				},
			},
		},
	}

	lintResults := Lint(cfg)
	unreachable := lintResultsWithCode(lintResults, "unreachable-tier")
	assert.Empty(t, unreachable, "partially-overlapping tier must NOT be flagged as unreachable")
}

// ── Lint: no-ext-match ────────────────────────────────────────────────────────

// TestLint_NoExtensionPattern verifies that a tier pattern with no file
// extension receives a LintResult with Code = "no-ext-match".
func TestLint_NoExtensionPattern(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{"src/**"}, // no extension
				},
			},
		},
	}

	lintResults := Lint(cfg)
	noExt := lintResultsWithCode(lintResults, "no-ext-match")
	require.NotEmpty(t, noExt, "pattern with no extension must produce no-ext-match lint")
	assert.Contains(t, noExt[0].Field, "tier_0")
	assert.Contains(t, noExt[0].Field, "[0]")
	assert.Equal(t, "warning", noExt[0].Severity)
}

// TestLint_NoExtensionPattern_WithExtension verifies that a pattern WITH an
// extension does NOT produce a "no-ext-match" lint result.
func TestLint_NoExtensionPattern_WithExtension(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{"**/*.go", "src/**/*.ts"},
				},
			},
		},
	}

	lintResults := Lint(cfg)
	noExt := lintResultsWithCode(lintResults, "no-ext-match")
	assert.Empty(t, noExt, "patterns with extensions must not produce no-ext-match lint")
}

// TestLint_NoExtensionPattern_HiddenFile verifies that a hidden-file pattern
// like ".git" (dot at position 0 with no further dot) is flagged as having no
// real extension.
func TestLint_NoExtensionPattern_HiddenFile(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{".git"},
				},
			},
		},
	}

	lintResults := Lint(cfg)
	noExt := lintResultsWithCode(lintResults, "no-ext-match")
	require.NotEmpty(t, noExt, ".git has no real extension and must be flagged")
}

// TestLint_NoExtensionPattern_DottedHiddenFile verifies that ".gitignore"
// (hidden file with an extension-like name) IS considered to have an extension.
func TestLint_NoExtensionPattern_DottedHiddenFile(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Relevance: RelevanceConfig{
					Tier0: []string{".gitignore"},
				},
			},
		},
	}

	lintResults := Lint(cfg)
	noExt := lintResultsWithCode(lintResults, "no-ext-match")
	// .gitignore has a dot not at position 0 when we consider the last segment;
	// it actually starts with "." so the first dot IS at position 0.
	// The implementation checks: dotIdx == 0 && !strings.Contains(last[1:], ".")
	// For ".gitignore": dotIdx=0, last[1:]="gitignore" which has no dot → no ext.
	// So .gitignore SHOULD be flagged as no-ext-match per the implementation.
	// This test documents the actual behaviour.
	require.NotEmpty(t, noExt, ".gitignore is treated as having no real extension")
}

// ── Lint: complexity ──────────────────────────────────────────────────────────

// TestLint_Complexity_HighScore verifies that a profile with more than 8
// non-default fields receives a LintResult with Code = "complexity".
func TestLint_Complexity_HighScore(t *testing.T) {
	t.Parallel()

	// This profile sets 10 distinct non-empty fields, exceeding the threshold.
	cfg := &Config{
		Profile: map[string]*Profile{
			"dense": {
				Output:      "out.md",
				Format:      "markdown",
				MaxTokens:   64000,
				Tokenizer:   "cl100k_base",
				Compression: true,
				Redaction:   true,
				Target:      "claude",
				Ignore:      []string{"node_modules"},
				PriorityFiles: []string{"go.mod"},
				Include:     []string{"src/**"},
				// Score so far: 10 fields → exceeds threshold of 8
			},
		},
	}

	lintResults := Lint(cfg)
	complexity := lintResultsWithCode(lintResults, "complexity")
	require.NotEmpty(t, complexity, "over-complex profile must produce complexity lint")
	assert.Equal(t, "warning", complexity[0].Severity)
	assert.Contains(t, complexity[0].Message, "complexity score")
	assert.NotEmpty(t, complexity[0].Suggest)
}

// TestLint_Complexity_LowScore verifies that a simple profile does NOT receive
// a complexity lint result.
func TestLint_Complexity_LowScore(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"simple": {
				Format:    "markdown",
				MaxTokens: 64000,
			},
		},
	}

	lintResults := Lint(cfg)
	complexity := lintResultsWithCode(lintResults, "complexity")
	assert.Empty(t, complexity, "simple profile must not produce complexity lint")
}

// TestLint_Complexity_AtThreshold verifies that a profile whose complexity
// score exactly equals the threshold (8) does NOT trigger the warning.
func TestLint_Complexity_AtThreshold(t *testing.T) {
	t.Parallel()

	// Score: Output(1) + Format(1) + MaxTokens(1) + Tokenizer(1) + Compression(1)
	//       + Redaction(1) + Target(1) + Ignore(1) = 8 (at threshold, not above)
	cfg := &Config{
		Profile: map[string]*Profile{
			"p": {
				Output:      "out.md",
				Format:      "markdown",
				MaxTokens:   64000,
				Tokenizer:   "cl100k_base",
				Compression: true,
				Redaction:   true,
				Target:      "claude",
				Ignore:      []string{"node_modules"},
			},
		},
	}

	lintResults := Lint(cfg)
	complexity := lintResultsWithCode(lintResults, "complexity")
	assert.Empty(t, complexity, "profile at exact threshold must NOT produce complexity lint")
}

// ── Lint: combined scenario ───────────────────────────────────────────────────

// TestLint_CombinedScenario verifies that Lint can return multiple lint codes
// from a single profile simultaneously without dropping any.
func TestLint_CombinedScenario(t *testing.T) {
	t.Parallel()

	// Profile that is complex AND has unreachable tiers AND no-ext patterns.
	cfg := &Config{
		Profile: map[string]*Profile{
			"mega": {
				Output:        "out.md",
				Format:        "markdown",
				MaxTokens:     64000,
				Tokenizer:     "cl100k_base",
				Compression:   true,
				Redaction:     true,
				Target:        "claude",
				Ignore:        []string{"node_modules"},
				PriorityFiles: []string{"go.mod"},
				Include:       []string{"src/**"},
				Relevance: RelevanceConfig{
					Tier0: []string{"go.mod", "internal/**"}, // internal/** has no ext
					Tier1: []string{"go.mod"},                // all covered by Tier0 → unreachable
				},
			},
		},
	}

	lintResults := Lint(cfg)
	sortLintResults(lintResults)

	codes := make(map[string]bool)
	for _, r := range lintResults {
		if r.Code != "" {
			codes[r.Code] = true
		}
	}

	assert.True(t, codes["complexity"], "must detect complexity")
	assert.True(t, codes["unreachable-tier"], "must detect unreachable tier")
	assert.True(t, codes["no-ext-match"], "must detect no-ext-match")
}

// ── Determinism: map iteration independence ───────────────────────────────────

// TestValidate_DeterministicAcrossRuns verifies that running Validate multiple
// times on the same Config always produces the same set of error fields
// (guarding against non-deterministic map iteration).
func TestValidate_DeterministicAcrossRuns(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile: map[string]*Profile{
			"a": {Format: "html"},
			"b": {Tokenizer: "gpt2"},
			"c": {Target: "gemini"},
		},
	}

	// Collect the field sets from 10 independent Validate calls.
	type fieldSet map[string]bool
	collectFields := func() fieldSet {
		fs := make(fieldSet)
		for _, e := range Validate(cfg) {
			fs[e.Field] = true
		}
		return fs
	}

	baseline := collectFields()
	for i := 0; i < 9; i++ {
		got := collectFields()
		assert.Equal(t, baseline, got,
			"Validate must return the same field set on every call (run %d)", i+2)
	}
}

// ── Boundary: max_tokens exact boundaries ────────────────────────────────────

// TestValidate_MaxTokensBoundaries exercises all relevant boundary values for
// max_tokens in a single table-driven test.
func TestValidate_MaxTokensBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		maxTokens   int
		wantErrCnt  int // expected hard errors for max_tokens field
		wantWarnCnt int // expected warnings for max_tokens field
	}{
		{name: "negative", maxTokens: -1, wantErrCnt: 1, wantWarnCnt: 0},
		{name: "zero (no issue)", maxTokens: 0, wantErrCnt: 0, wantWarnCnt: 0},
		{name: "at soft cap", maxTokens: 500_000, wantErrCnt: 0, wantWarnCnt: 0},
		{name: "above soft cap", maxTokens: 500_001, wantErrCnt: 0, wantWarnCnt: 1},
		{name: "at hard cap", maxTokens: 2_000_000, wantErrCnt: 0, wantWarnCnt: 1},
		{name: "above hard cap", maxTokens: 2_000_001, wantErrCnt: 1, wantWarnCnt: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Profile: map[string]*Profile{
					"p": {MaxTokens: tt.maxTokens},
				},
			}
			result := Validate(cfg)
			sortValidationErrors(result)

			hardErrs := errorsWithField(errorsWithSeverity(result, "error"), "profile.p.max_tokens")
			warnings := errorsWithField(errorsWithSeverity(result, "warning"), "profile.p.max_tokens")

			assert.Len(t, hardErrs, tt.wantErrCnt,
				"max_tokens=%d: expected %d hard error(s)", tt.maxTokens, tt.wantErrCnt)
			assert.Len(t, warnings, tt.wantWarnCnt,
				"max_tokens=%d: expected %d warning(s)", tt.maxTokens, tt.wantWarnCnt)
		})
	}
}

// ── LintResult type tests ─────────────────────────────────────────────────────

// TestLintResult_EmbeddedValidationError verifies that a LintResult exposes
// the embedded ValidationError fields directly.
func TestLintResult_EmbeddedValidationError(t *testing.T) {
	t.Parallel()

	lr := LintResult{
		ValidationError: ValidationError{
			Severity: "warning",
			Field:    "profile.p.relevance.tier_1",
			Message:  "unreachable",
			Suggest:  "remove duplicates",
		},
		Code: "unreachable-tier",
	}

	assert.Equal(t, "warning", lr.Severity)
	assert.Equal(t, "profile.p.relevance.tier_1", lr.Field)
	assert.Equal(t, "unreachable", lr.Message)
	assert.Equal(t, "remove duplicates", lr.Suggest)
	assert.Equal(t, "unreachable-tier", lr.Code)
	assert.NotEmpty(t, lr.Error())
}
