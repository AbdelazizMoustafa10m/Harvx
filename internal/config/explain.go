package config

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// TraceStep records one evaluation step during file rule tracing.
type TraceStep struct {
	// StepNum is the 1-based step number in the evaluation sequence.
	StepNum int

	// Rule describes the rule being evaluated, e.g. "Default ignore patterns".
	Rule string

	// Matched indicates whether the rule matched the file path.
	Matched bool

	// Outcome describes the result of this step, e.g. "continue", "EXCLUDED",
	// "INCLUDED", or "tier N (pattern)".
	Outcome string
}

// ExplainResult holds the full explanation for a single file path showing
// how a profile would process the file during context generation.
type ExplainResult struct {
	// FilePath is the file path being explained.
	FilePath string

	// ProfileName is the name of the profile being used for display.
	ProfileName string

	// Extends is the parent profile name, or empty if there is no parent.
	Extends string

	// Included indicates whether the file is included (true) or excluded (false).
	Included bool

	// ExcludedBy names the rule that caused exclusion when Included is false.
	ExcludedBy string

	// Tier is the relevance tier assigned to the file (-1 if excluded or untiered).
	Tier int

	// TierPattern is the glob pattern that matched the tier assignment.
	TierPattern string

	// IsPriority indicates whether the file is in priority_files.
	IsPriority bool

	// RedactionOn indicates whether redaction is enabled for this file.
	RedactionOn bool

	// Compression is the language name if compression applies, otherwise "".
	Compression string

	// Trace is the ordered list of evaluation steps.
	Trace []TraceStep
}

// ExplainFile evaluates how profile p would process filePath and returns a
// full ExplainResult describing the evaluation. profileName is used for
// display only; it does not affect the evaluation logic.
//
// The function simulates the discovery pipeline steps in order:
//  1. Default ignore patterns
//  2. Profile ignore patterns
//  3. .gitignore rules (not simulated -- requires disk access)
//  4. Include filter
//  5. Priority files check
//  6-11. Relevance tiers 0-5
func ExplainFile(filePath, profileName string, p *Profile) ExplainResult {
	result := ExplainResult{
		FilePath:    filePath,
		ProfileName: profileName,
		Tier:        -1,
	}

	// Set Extends if the profile inherits from a parent.
	if p.Extends != nil && *p.Extends != "" {
		result.Extends = *p.Extends
	}

	stepNum := 0
	nextStep := func() int {
		stepNum++
		return stepNum
	}

	// ── Step 1: Default ignore patterns ────────────────────────────────────
	defaults := DefaultProfile()
	{
		step := TraceStep{
			StepNum: nextStep(),
			Rule:    "Default ignore patterns",
		}
		matchedPattern := ""
		for _, pattern := range defaults.Ignore {
			if matchesGlob(pattern, filePath) {
				matchedPattern = pattern
				break
			}
		}
		if matchedPattern != "" {
			step.Matched = true
			step.Outcome = "EXCLUDED"
			result.Trace = append(result.Trace, step)
			result.Included = false
			result.ExcludedBy = fmt.Sprintf("default ignore pattern %q", matchedPattern)
			return result
		}
		step.Matched = false
		step.Outcome = "no match -> continue"
		result.Trace = append(result.Trace, step)
	}

	// ── Step 2: Profile ignore patterns ────────────────────────────────────
	{
		step := TraceStep{
			StepNum: nextStep(),
			Rule:    "Profile ignore patterns",
		}
		matchedPattern := ""
		for _, pattern := range p.Ignore {
			if matchesGlob(pattern, filePath) {
				matchedPattern = pattern
				break
			}
		}
		if matchedPattern != "" {
			step.Matched = true
			step.Outcome = "EXCLUDED"
			result.Trace = append(result.Trace, step)
			result.Included = false
			result.ExcludedBy = fmt.Sprintf("profile ignore pattern %q", matchedPattern)
			return result
		}
		step.Matched = false
		step.Outcome = "no match -> continue"
		result.Trace = append(result.Trace, step)
	}

	// ── Step 3: .gitignore rules ────────────────────────────────────────────
	{
		result.Trace = append(result.Trace, TraceStep{
			StepNum: nextStep(),
			Rule:    ".gitignore rules",
			Matched: false,
			Outcome: "not simulated -> continue",
		})
	}

	// ── Step 4: Include filter ──────────────────────────────────────────────
	{
		step := TraceStep{
			StepNum: nextStep(),
			Rule:    "Include filter",
		}
		if len(p.Include) > 0 {
			if !matchesAny(filePath, p.Include) {
				step.Matched = true
				step.Outcome = "EXCLUDED"
				result.Trace = append(result.Trace, step)
				result.Included = false
				result.ExcludedBy = "include filter (not in include list)"
				return result
			}
			step.Matched = false
			step.Outcome = "include match -> continue"
		} else {
			step.Matched = false
			step.Outcome = "not active -> continue"
		}
		result.Trace = append(result.Trace, step)
	}

	// ── Step 5: Priority files check ────────────────────────────────────────
	{
		step := TraceStep{
			StepNum: nextStep(),
			Rule:    "Priority files",
		}
		for _, pf := range p.PriorityFiles {
			if filePath == pf {
				result.IsPriority = true
				result.Tier = 0
				result.TierPattern = "priority_files"
				step.Matched = true
				step.Outcome = "priority file -> tier 0"
				break
			}
		}
		if !step.Matched {
			step.Outcome = "no match -> continue"
		}
		result.Trace = append(result.Trace, step)
	}

	// ── Steps 6–11: Relevance tiers 0–5 ────────────────────────────────────
	tiers := []struct {
		num      int
		name     string
		patterns []string
	}{
		{0, "tier_0", p.Relevance.Tier0},
		{1, "tier_1", p.Relevance.Tier1},
		{2, "tier_2", p.Relevance.Tier2},
		{3, "tier_3", p.Relevance.Tier3},
		{4, "tier_4", p.Relevance.Tier4},
		{5, "tier_5", p.Relevance.Tier5},
	}

	for _, tier := range tiers {
		step := TraceStep{
			StepNum: nextStep(),
			Rule:    fmt.Sprintf("Relevance %s", tier.name),
		}

		if result.IsPriority {
			// Priority files skip tier matching but still add steps.
			step.Matched = false
			step.Outcome = "skipped (priority file)"
			result.Trace = append(result.Trace, step)
			continue
		}

		matchedPattern := ""
		for _, pattern := range tier.patterns {
			if matchesGlob(pattern, filePath) {
				matchedPattern = pattern
				break
			}
		}

		if matchedPattern != "" {
			// First match wins -- record the step and stop tier evaluation.
			result.Tier = tier.num
			result.TierPattern = matchedPattern
			step.Matched = true
			step.Outcome = fmt.Sprintf("MATCH %q -> assigned tier %d", matchedPattern, tier.num)
			result.Trace = append(result.Trace, step)
			break
		}
		step.Matched = false
		step.Outcome = "no match"
		result.Trace = append(result.Trace, step)
	}

	// All steps passed -- file is included.
	result.Included = true

	// ── Redaction check ─────────────────────────────────────────────────────
	result.RedactionOn = p.Redaction && !matchesAny(filePath, p.RedactionConfig.ExcludePaths)

	// ── Compression check ───────────────────────────────────────────────────
	result.Compression = compressionLanguage(filePath)

	return result
}

// compressionLanguage returns the language name for Tree-sitter compression
// support based on the file extension. Returns "" if the extension is not
// supported.
func compressionLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	languages := map[string]string{
		".go":    "Go",
		".ts":    "TypeScript",
		".tsx":   "TypeScript (TSX)",
		".js":    "JavaScript",
		".jsx":   "JavaScript (JSX)",
		".py":    "Python",
		".rs":    "Rust",
		".c":     "C",
		".cpp":   "C++",
		".h":     "C/C++ header",
		".java":  "Java",
		".rb":    "Ruby",
		".php":   "PHP",
		".swift": "Swift",
		".kt":    "Kotlin",
		".cs":    "C#",
	}
	return languages[ext]
}

// matchesAny reports whether path matches any of the given glob patterns.
// Pattern matching errors are silently ignored.
func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchesGlob(pattern, path) {
			return true
		}
	}
	return false
}

// matchesGlob reports whether filePath matches the given doublestar glob
// pattern. Match errors are silently ignored and treated as non-matches.
func matchesGlob(pattern, filePath string) bool {
	matched, err := doublestar.Match(pattern, filePath)
	if err != nil {
		return false
	}
	return matched
}
