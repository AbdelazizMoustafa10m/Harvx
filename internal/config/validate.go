package config

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// validFormats lists the only accepted values for Profile.Format.
// An empty string is valid for profiles that inherit the value from a parent.
var validFormats = map[string]bool{
	"markdown": true,
	"xml":      true,
	"plain":    true,
	"":         true,
}

// validTokenizers lists the only accepted values for Profile.Tokenizer.
// An empty string is valid for profiles that inherit the value from a parent.
var validTokenizers = map[string]bool{
	"cl100k_base": true,
	"o200k_base":  true,
	"none":        true,
	"":            true,
}

// validTargets lists the only accepted values for Profile.Target.
// An empty string is also valid (no LLM-specific optimizations).
var validTargets = map[string]bool{
	"claude":  true,
	"chatgpt": true,
	"generic": true,
	"":        true,
}

// validConfidenceThresholds lists the only accepted values for
// RedactionConfig.ConfidenceThreshold. An empty string is also valid
// (uses the built-in default).
var validConfidenceThresholds = map[string]bool{
	"high":   true,
	"medium": true,
	"low":    true,
	"":       true,
}

// maxTokensHardCap is the absolute upper limit for Profile.MaxTokens.
// Values above this are almost certainly a configuration mistake.
const maxTokensHardCap = 2_000_000

// maxTokensSoftCap triggers a warning when Profile.MaxTokens exceeds it,
// because unusually large budgets are a common misconfiguration.
const maxTokensSoftCap = 500_000

// maxInheritanceWarningDepth is the chain length above which validation emits
// a warning about deep inheritance (mirrors the resolver constant).
const maxInheritanceWarningDepth = 3

// globMetaChars are the characters that distinguish a glob pattern from a
// literal path. Used to detect priority_files entries that look like patterns.
const globMetaChars = "*?[{"

// Validate inspects every profile in cfg and returns a slice of
// ValidationErrors describing hard errors and warnings found in the
// configuration. It does not stop at the first error; all profiles are
// checked and all findings are accumulated before returning.
//
// The returned slice is nil when no issues are found. Each element carries
// a Severity field of either "error" or "warning".
//
// Validate does not modify cfg.
func Validate(cfg *Config) []ValidationError {
	if cfg == nil {
		return nil
	}

	var results []ValidationError

	for name, profile := range cfg.Profile {
		if profile == nil {
			continue
		}
		errs := validateProfile(name, profile, cfg.Profile)
		results = append(results, errs...)
	}

	if len(results) > 0 {
		slog.Debug("config validation complete",
			"total_issues", len(results),
		)
	}

	return results
}

// validateProfile checks a single named profile and returns all validation
// errors and warnings for that profile.
func validateProfile(name string, p *Profile, allProfiles map[string]*Profile) []ValidationError {
	var results []ValidationError

	field := func(f string) string {
		return fmt.Sprintf("profile.%s.%s", name, f)
	}

	// ── Hard errors ────────────────────────────────────────────────────────

	// format
	if !validFormats[p.Format] {
		results = append(results, ValidationError{
			Severity: "error",
			Field:    field("format"),
			Message:  fmt.Sprintf("format %q is invalid", p.Format),
			Suggest:  "Valid formats: markdown, xml, plain",
		})
	}

	// tokenizer
	if !validTokenizers[p.Tokenizer] {
		results = append(results, ValidationError{
			Severity: "error",
			Field:    field("tokenizer"),
			Message:  fmt.Sprintf("tokenizer %q is invalid", p.Tokenizer),
			Suggest:  "Valid tokenizers: cl100k_base, o200k_base, none",
		})
	}

	// target
	if !validTargets[p.Target] {
		results = append(results, ValidationError{
			Severity: "error",
			Field:    field("target"),
			Message:  fmt.Sprintf("target %q is invalid", p.Target),
			Suggest:  "Valid targets: claude, chatgpt, generic (or leave empty)",
		})
	}

	// confidence_threshold
	if !validConfidenceThresholds[p.RedactionConfig.ConfidenceThreshold] {
		results = append(results, ValidationError{
			Severity: "error",
			Field:    field("redaction_config.confidence_threshold"),
			Message:  fmt.Sprintf("confidence_threshold %q is invalid", p.RedactionConfig.ConfidenceThreshold),
			Suggest:  "Valid values: high, medium, low",
		})
	}

	// max_tokens: negative
	if p.MaxTokens < 0 {
		results = append(results, ValidationError{
			Severity: "error",
			Field:    field("max_tokens"),
			Message:  fmt.Sprintf("max_tokens %d is negative", p.MaxTokens),
			Suggest:  "Set max_tokens to a positive integer or remove it to use the default",
		})
	}

	// max_tokens: sanity cap (hard)
	if p.MaxTokens > maxTokensHardCap {
		results = append(results, ValidationError{
			Severity: "error",
			Field:    field("max_tokens"),
			Message:  fmt.Sprintf("max_tokens %d exceeds the maximum allowed value of %d", p.MaxTokens, maxTokensHardCap),
			Suggest:  fmt.Sprintf("Reduce max_tokens to at most %d", maxTokensHardCap),
		})
	}

	// glob pattern validity
	results = append(results, validateGlobPatterns(name, p)...)

	// circular inheritance
	if p.Extends != nil && *p.Extends != "" {
		if _, err := ResolveProfile(name, allProfiles); err != nil {
			// Report circular or missing parent.
			if strings.Contains(err.Error(), "circular") {
				results = append(results, ValidationError{
					Severity: "error",
					Field:    field("extends"),
					Message:  err.Error(),
					Suggest:  "Remove or restructure the extends chain to eliminate the cycle",
				})
			} else {
				results = append(results, ValidationError{
					Severity: "error",
					Field:    field("extends"),
					Message:  fmt.Sprintf("extends %q: %s", *p.Extends, err.Error()),
					Suggest:  fmt.Sprintf("Define a profile named %q or update the extends value", *p.Extends),
				})
			}
		}
	}

	// ── Warnings ───────────────────────────────────────────────────────────

	// Overlapping tier patterns (same exact pattern string in multiple tiers).
	results = append(results, warnOverlappingTiers(name, p)...)

	// Empty relevance tiers.
	results = append(results, warnEmptyTiers(name, p)...)

	// priority_files entries that also appear in ignore (contradictory).
	results = append(results, warnPriorityFilesInIgnore(name, p)...)

	// priority_files with glob metacharacters (should be exact paths).
	results = append(results, warnPriorityFilesWithGlobs(name, p)...)

	// redaction_config.exclude_paths overlapping with ignore (redundant).
	results = append(results, warnRedactionExcludeOverlap(name, p)...)

	// Inheritance depth > 3.
	results = append(results, warnDeepInheritance(name, p, allProfiles)...)

	// max_tokens soft cap.
	if p.MaxTokens > maxTokensSoftCap && p.MaxTokens <= maxTokensHardCap {
		results = append(results, ValidationError{
			Severity: "warning",
			Field:    field("max_tokens"),
			Message:  fmt.Sprintf("max_tokens %d is unusually large", p.MaxTokens),
			Suggest:  fmt.Sprintf("Values above %d may cause memory pressure; verify this is intentional", maxTokensSoftCap),
		})
	}

	// Output path outside the current directory tree.
	if p.Output != "" {
		if strings.HasPrefix(p.Output, "../") || filepath.IsAbs(p.Output) {
			results = append(results, ValidationError{
				Severity: "warning",
				Field:    field("output"),
				Message:  fmt.Sprintf("output path %q is outside the project directory", p.Output),
				Suggest:  "Use a relative path within the project directory, e.g. \".harvx/output.md\"",
			})
		}
	}

	return results
}

// validateGlobPatterns validates all glob pattern lists in the profile and
// returns errors for any invalid patterns.
func validateGlobPatterns(profileName string, p *Profile) []ValidationError {
	var results []ValidationError

	field := func(f string) string {
		return fmt.Sprintf("profile.%s.%s", profileName, f)
	}

	type patternList struct {
		fieldPath string
		patterns  []string
	}

	lists := []patternList{
		{field("ignore"), p.Ignore},
		{field("include"), p.Include},
		{field("priority_files"), p.PriorityFiles},
		{field("redaction_config.exclude_paths"), p.RedactionConfig.ExcludePaths},
		{field("relevance.tier_0"), p.Relevance.Tier0},
		{field("relevance.tier_1"), p.Relevance.Tier1},
		{field("relevance.tier_2"), p.Relevance.Tier2},
		{field("relevance.tier_3"), p.Relevance.Tier3},
		{field("relevance.tier_4"), p.Relevance.Tier4},
		{field("relevance.tier_5"), p.Relevance.Tier5},
	}

	for _, list := range lists {
		for i, pattern := range list.patterns {
			if err := validateGlobPattern(pattern); err != nil {
				results = append(results, ValidationError{
					Severity: "error",
					Field:    fmt.Sprintf("%s[%d]", list.fieldPath, i),
					Message:  fmt.Sprintf("invalid glob pattern %q: %s", pattern, err.Error()),
					Suggest:  "Use doublestar glob syntax, e.g. \"**/*.go\" or \"src/**\"",
				})
			}
		}
	}

	return results
}

// validateGlobPattern checks whether pattern is syntactically valid according
// to the doublestar library. It uses doublestar.ValidatePattern which returns
// false for malformed patterns (e.g. unclosed character classes or alternations).
func validateGlobPattern(pattern string) error {
	if !doublestar.ValidatePattern(pattern) {
		return fmt.Errorf("syntax error in pattern %q", pattern)
	}
	return nil
}

// warnOverlappingTiers returns warnings for glob patterns that appear
// identically in more than one relevance tier.
func warnOverlappingTiers(profileName string, p *Profile) []ValidationError {
	tiers := []struct {
		name     string
		patterns []string
	}{
		{"tier_0", p.Relevance.Tier0},
		{"tier_1", p.Relevance.Tier1},
		{"tier_2", p.Relevance.Tier2},
		{"tier_3", p.Relevance.Tier3},
		{"tier_4", p.Relevance.Tier4},
		{"tier_5", p.Relevance.Tier5},
	}

	// Map each pattern to the first tier it appears in.
	seen := make(map[string]string) // pattern -> tier name
	var results []ValidationError

	for _, tier := range tiers {
		for _, pattern := range tier.patterns {
			if firstTier, exists := seen[pattern]; exists {
				results = append(results, ValidationError{
					Severity: "warning",
					Field:    fmt.Sprintf("profile.%s.relevance.%s", profileName, tier.name),
					Message: fmt.Sprintf(
						"pattern %q also appears in %s; duplicate tier patterns are redundant",
						pattern, firstTier,
					),
					Suggest: fmt.Sprintf("Remove the duplicate pattern from %s", tier.name),
				})
			} else {
				seen[pattern] = tier.name
			}
		}
	}

	return results
}

// warnEmptyTiers returns warnings for relevance tiers that are explicitly
// defined but contain no patterns.
func warnEmptyTiers(profileName string, p *Profile) []ValidationError {
	tiers := []struct {
		name     string
		patterns []string
	}{
		{"tier_0", p.Relevance.Tier0},
		{"tier_1", p.Relevance.Tier1},
		{"tier_2", p.Relevance.Tier2},
		{"tier_3", p.Relevance.Tier3},
		{"tier_4", p.Relevance.Tier4},
		{"tier_5", p.Relevance.Tier5},
	}

	var results []ValidationError
	for _, tier := range tiers {
		// Only warn if the field is a non-nil empty slice (explicitly set to empty).
		// A nil slice means the field was never set, which is fine.
		if tier.patterns != nil && len(tier.patterns) == 0 {
			results = append(results, ValidationError{
				Severity: "warning",
				Field:    fmt.Sprintf("profile.%s.relevance.%s", profileName, tier.name),
				Message:  fmt.Sprintf("relevance.%s has no patterns and could be removed", tier.name),
				Suggest:  fmt.Sprintf("Add glob patterns to %s or remove the empty tier definition", tier.name),
			})
		}
	}

	return results
}

// warnPriorityFilesInIgnore returns warnings for priority_files entries that
// also appear in the ignore list (contradictory configuration).
func warnPriorityFilesInIgnore(profileName string, p *Profile) []ValidationError {
	if len(p.PriorityFiles) == 0 || len(p.Ignore) == 0 {
		return nil
	}

	ignoreSet := make(map[string]bool, len(p.Ignore))
	for _, ig := range p.Ignore {
		ignoreSet[ig] = true
	}

	var results []ValidationError
	for i, pf := range p.PriorityFiles {
		if ignoreSet[pf] {
			results = append(results, ValidationError{
				Severity: "warning",
				Field:    fmt.Sprintf("profile.%s.priority_files[%d]", profileName, i),
				Message:  fmt.Sprintf("priority file %q is also listed in ignore; it will be excluded", pf),
				Suggest:  fmt.Sprintf("Remove %q from ignore or from priority_files", pf),
			})
		}
	}

	return results
}

// warnPriorityFilesWithGlobs returns warnings for priority_files entries that
// contain glob metacharacters. Priority files should be exact paths.
func warnPriorityFilesWithGlobs(profileName string, p *Profile) []ValidationError {
	var results []ValidationError
	for i, pf := range p.PriorityFiles {
		if strings.ContainsAny(pf, globMetaChars) {
			results = append(results, ValidationError{
				Severity: "warning",
				Field:    fmt.Sprintf("profile.%s.priority_files[%d]", profileName, i),
				Message:  fmt.Sprintf("priority_files entry %q looks like a glob pattern", pf),
				Suggest:  "priority_files should contain exact file paths; use relevance tiers for glob patterns",
			})
		}
	}
	return results
}

// warnRedactionExcludeOverlap returns warnings for redaction_config.exclude_paths
// entries that also appear exactly in the ignore list (redundant configuration).
func warnRedactionExcludeOverlap(profileName string, p *Profile) []ValidationError {
	if len(p.RedactionConfig.ExcludePaths) == 0 || len(p.Ignore) == 0 {
		return nil
	}

	ignoreSet := make(map[string]bool, len(p.Ignore))
	for _, ig := range p.Ignore {
		ignoreSet[ig] = true
	}

	var results []ValidationError
	for i, ep := range p.RedactionConfig.ExcludePaths {
		if ignoreSet[ep] {
			results = append(results, ValidationError{
				Severity: "warning",
				Field:    fmt.Sprintf("profile.%s.redaction_config.exclude_paths[%d]", profileName, i),
				Message:  fmt.Sprintf("redaction exclude path %q is also in ignore; it will never be redaction-scanned", ep),
				Suggest:  fmt.Sprintf("Remove %q from redaction_config.exclude_paths as it is already excluded by ignore", ep),
			})
		}
	}

	return results
}

// warnDeepInheritance returns a warning when the inheritance chain for the
// profile exceeds maxInheritanceWarningDepth levels.
func warnDeepInheritance(profileName string, p *Profile, allProfiles map[string]*Profile) []ValidationError {
	if p.Extends == nil || *p.Extends == "" {
		return nil
	}

	resolution, err := ResolveProfile(profileName, allProfiles)
	if err != nil {
		// Errors are already reported elsewhere (e.g. circular inheritance).
		return nil
	}

	depth := len(resolution.Chain)
	if depth <= maxInheritanceWarningDepth {
		return nil
	}

	return []ValidationError{
		{
			Severity: "warning",
			Field:    fmt.Sprintf("profile.%s.extends", profileName),
			Message: fmt.Sprintf(
				"inheritance chain is %d levels deep (%s)",
				depth,
				strings.Join(resolution.Chain, " -> "),
			),
			Suggest: "Flatten the inheritance chain to 3 levels or fewer for maintainability",
		},
	}
}

// Lint runs all Validate checks and additionally performs deeper static
// analysis of the configuration. It returns a slice of LintResult values that
// embed ValidationError for unified severity/field/message access.
//
// Lint-only checks include:
//   - Unreachable tiers: a tier whose patterns are a subset of a higher-priority
//     tier (detected by exact string overlap heuristic).
//   - No-extension patterns: tier patterns that have no file-extension suffix,
//     meaning they match any file name regardless of type.
//   - Complexity score: profiles with many non-default fields set are flagged
//     to encourage splitting into focused sub-profiles.
//
// The returned slice is nil when no issues are found.
func Lint(cfg *Config) []LintResult {
	if cfg == nil {
		return nil
	}

	var results []LintResult

	// Include all Validate results as LintResults (Code left empty for these).
	for _, ve := range Validate(cfg) {
		results = append(results, LintResult{ValidationError: ve})
	}

	// Perform deeper lint-only analysis per profile.
	for name, profile := range cfg.Profile {
		if profile == nil {
			continue
		}
		results = append(results, lintProfile(name, profile)...)
	}

	return results
}

// lintProfile performs the deeper lint-only analysis for a single profile.
func lintProfile(profileName string, p *Profile) []LintResult {
	var results []LintResult

	results = append(results, lintUnreachableTiers(profileName, p)...)
	results = append(results, lintNoExtPatterns(profileName, p)...)
	results = append(results, lintComplexity(profileName, p)...)

	return results
}

// lintUnreachableTiers detects tiers whose patterns are all exact-string
// duplicates of patterns in a higher-priority (lower-numbered) tier.
// When every pattern in tier N already appears in tier N-1, tier N will
// never be reached by the relevance engine.
func lintUnreachableTiers(profileName string, p *Profile) []LintResult {
	tiers := []struct {
		name     string
		patterns []string
	}{
		{"tier_0", p.Relevance.Tier0},
		{"tier_1", p.Relevance.Tier1},
		{"tier_2", p.Relevance.Tier2},
		{"tier_3", p.Relevance.Tier3},
		{"tier_4", p.Relevance.Tier4},
		{"tier_5", p.Relevance.Tier5},
	}

	var results []LintResult

	// Build a cumulative set of patterns from all higher-priority tiers.
	higherPatterns := make(map[string]bool)

	for _, tier := range tiers {
		if len(tier.patterns) == 0 {
			continue
		}

		// Count how many of this tier's patterns already appear in higher tiers.
		covered := 0
		for _, pattern := range tier.patterns {
			if higherPatterns[pattern] {
				covered++
			}
		}

		// If all patterns are covered by higher tiers, this tier is unreachable.
		if covered == len(tier.patterns) {
			results = append(results, LintResult{
				ValidationError: ValidationError{
					Severity: "warning",
					Field:    fmt.Sprintf("profile.%s.relevance.%s", profileName, tier.name),
					Message: fmt.Sprintf(
						"all %d patterns in %s are already present in higher-priority tiers; this tier is unreachable",
						len(tier.patterns), tier.name,
					),
					Suggest: fmt.Sprintf("Remove duplicate patterns from %s or consolidate into a higher-priority tier", tier.name),
				},
				Code: "unreachable-tier",
			})
		}

		// Add this tier's patterns to the cumulative set.
		for _, pattern := range tier.patterns {
			higherPatterns[pattern] = true
		}
	}

	return results
}

// lintNoExtPatterns detects tier patterns that do not contain any
// file-extension-like suffix (no dot after the last path separator or
// wildcard). Such patterns match files of any type, which may be unintentional.
func lintNoExtPatterns(profileName string, p *Profile) []LintResult {
	tiers := []struct {
		name     string
		patterns []string
	}{
		{"tier_0", p.Relevance.Tier0},
		{"tier_1", p.Relevance.Tier1},
		{"tier_2", p.Relevance.Tier2},
		{"tier_3", p.Relevance.Tier3},
		{"tier_4", p.Relevance.Tier4},
		{"tier_5", p.Relevance.Tier5},
	}

	var results []LintResult

	for _, tier := range tiers {
		for i, pattern := range tier.patterns {
			if !patternHasExtension(pattern) {
				results = append(results, LintResult{
					ValidationError: ValidationError{
						Severity: "warning",
						Field:    fmt.Sprintf("profile.%s.relevance.%s[%d]", profileName, tier.name, i),
						Message:  fmt.Sprintf("pattern %q has no file extension; it will match files of any type", pattern),
						Suggest:  "Add an extension suffix (e.g. \"**/*.go\") unless matching all file types is intentional",
					},
					Code: "no-ext-match",
				})
			}
		}
	}

	return results
}

// patternHasExtension reports whether pattern contains a dot after the last
// path separator or wildcard segment, indicating it matches a specific file
// extension. This is a heuristic, not a precise check.
func patternHasExtension(pattern string) bool {
	// Find the last component after the final '/' or '**'.
	last := pattern
	if idx := strings.LastIndex(pattern, "/"); idx >= 0 {
		last = pattern[idx+1:]
	}
	// The last segment should contain a dot for it to have an extension.
	// Ignore patterns where the dot is only at the start (hidden files like ".git").
	dotIdx := strings.LastIndex(last, ".")
	if dotIdx < 0 {
		return false
	}
	// A leading dot alone (e.g. ".git") does not constitute a file extension.
	if dotIdx == 0 && !strings.Contains(last[1:], ".") {
		return false
	}
	return true
}

// complexityThreshold is the number of non-default fields above which a
// profile is considered overly complex.
const complexityThreshold = 8

// lintComplexity computes the number of non-zero/non-empty fields in a profile
// and emits a warning when the count exceeds complexityThreshold.
func lintComplexity(profileName string, p *Profile) []LintResult {
	score := profileComplexityScore(p)
	if score <= complexityThreshold {
		return nil
	}

	return []LintResult{
		{
			ValidationError: ValidationError{
				Severity: "warning",
				Field:    fmt.Sprintf("profile.%s", profileName),
				Message:  fmt.Sprintf("profile has a complexity score of %d (threshold: %d)", score, complexityThreshold),
				Suggest:  "Consider splitting into multiple profiles connected via extends to improve maintainability",
			},
			Code: "complexity",
		},
	}
}

// profileComplexityScore counts the number of non-empty / non-zero fields in
// the profile. Scalar fields each count as 1; each non-empty slice counts as 1.
func profileComplexityScore(p *Profile) int {
	score := 0

	if p.Output != "" {
		score++
	}
	if p.Format != "" {
		score++
	}
	if p.MaxTokens != 0 {
		score++
	}
	if p.Tokenizer != "" {
		score++
	}
	if p.Compression {
		score++
	}
	if p.Redaction {
		score++
	}
	if p.Target != "" {
		score++
	}
	if len(p.Ignore) > 0 {
		score++
	}
	if len(p.PriorityFiles) > 0 {
		score++
	}
	if len(p.Include) > 0 {
		score++
	}
	if len(p.Relevance.Tier0) > 0 {
		score++
	}
	if len(p.Relevance.Tier1) > 0 {
		score++
	}
	if len(p.Relevance.Tier2) > 0 {
		score++
	}
	if len(p.Relevance.Tier3) > 0 {
		score++
	}
	if len(p.Relevance.Tier4) > 0 {
		score++
	}
	if len(p.Relevance.Tier5) > 0 {
		score++
	}
	if p.RedactionConfig.Enabled {
		score++
	}
	if len(p.RedactionConfig.ExcludePaths) > 0 {
		score++
	}
	if p.RedactionConfig.ConfidenceThreshold != "" {
		score++
	}

	return score
}
