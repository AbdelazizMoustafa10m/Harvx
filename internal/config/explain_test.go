package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── ExplainFile ───────────────────────────────────────────────────────────────

// TestExplainFile_FileInTier1 verifies that a file matching a tier_1 pattern
// is reported as included at Tier 1.
func TestExplainFile_FileInTier1(t *testing.T) {
	t.Parallel()

	p := &Profile{
		Relevance: RelevanceConfig{
			Tier1: []string{"lib/services/**"},
		},
	}

	result := ExplainFile("lib/services/auth.go", "myprofile", p)

	assert.True(t, result.Included, "file matching tier_1 pattern must be included")
	assert.Equal(t, 1, result.Tier, "file must be assigned to tier 1")
	assert.Equal(t, "lib/services/**", result.TierPattern)
	assert.Equal(t, "myprofile", result.ProfileName)
}

// TestExplainFile_FileInIgnoreList verifies that a path matching a default
// ignore pattern is excluded. The default profile includes "node_modules" which
// matches the literal path segment "node_modules". We also test a profile with
// "node_modules/**" to cover nested paths.
func TestExplainFile_FileInIgnoreList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		profile  *Profile
	}{
		{
			name:     "exact directory name match",
			filePath: "node_modules",
			profile:  &Profile{},
		},
		{
			name:     "nested path via profile pattern",
			filePath: "node_modules/lodash/index.js",
			profile:  &Profile{Ignore: []string{"node_modules/**"}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExplainFile(tt.filePath, "default", tt.profile)
			assert.False(t, result.Included, "matched ignore path must be excluded")
			assert.Contains(t, result.ExcludedBy, "node_modules",
				"ExcludedBy must name the matched ignore pattern")
		})
	}
}

// TestExplainFile_PriorityFile verifies that a file listed in PriorityFiles is
// reported as a priority file with Tier=0 and TierPattern="priority_files".
func TestExplainFile_PriorityFile(t *testing.T) {
	t.Parallel()

	p := &Profile{
		PriorityFiles: []string{"go.mod", "CLAUDE.md"},
	}

	result := ExplainFile("CLAUDE.md", "myprofile", p)

	assert.True(t, result.Included, "priority file must be included")
	assert.True(t, result.IsPriority, "file in PriorityFiles must have IsPriority=true")
	assert.Equal(t, 0, result.Tier, "priority file must be assigned tier 0")
	assert.Equal(t, "priority_files", result.TierPattern)
}

// TestExplainFile_NoTierMatch verifies that a file that passes all filters but
// matches no tier is included with Tier=-1 (untiered).
func TestExplainFile_NoTierMatch(t *testing.T) {
	t.Parallel()

	p := &Profile{
		// Tiers contain patterns that will never match the test file.
		Relevance: RelevanceConfig{
			Tier0: []string{"go.mod"},
		},
	}

	result := ExplainFile("random/unknown.xyz", "default", p)

	assert.True(t, result.Included, "file not matching any tier must still be included")
	assert.Equal(t, -1, result.Tier, "untiered file must have Tier=-1")
	assert.Empty(t, result.TierPattern, "untiered file must have empty TierPattern")
}

// TestExplainFile_RedactionExcludePath verifies that a file matching the
// redaction ExcludePaths has RedactionOn=false even when Redaction is enabled
// globally.
func TestExplainFile_RedactionExcludePath(t *testing.T) {
	t.Parallel()

	p := &Profile{
		Redaction: true,
		RedactionConfig: RedactionConfig{
			ExcludePaths: []string{"testdata/**"},
		},
	}

	result := ExplainFile("testdata/secrets/mock.env", "secure", p)

	assert.True(t, result.Included, "file must be included (not in ignore)")
	assert.False(t, result.RedactionOn,
		"redaction must be off for files in ExcludePaths")
}

// TestExplainFile_CompressionSupportedLanguage verifies that a .go file is
// assigned the "Go" compression language.
func TestExplainFile_CompressionSupportedLanguage(t *testing.T) {
	t.Parallel()

	p := &Profile{
		Relevance: RelevanceConfig{
			Tier1: []string{"internal/**"},
		},
	}

	result := ExplainFile("internal/config/explain.go", "default", p)

	assert.True(t, result.Included)
	assert.Equal(t, "Go", result.Compression,
		".go file must receive Compression=\"Go\"")
}

// TestExplainFile_CompressionUnsupportedLanguage verifies that a .txt file
// has an empty Compression field.
func TestExplainFile_CompressionUnsupportedLanguage(t *testing.T) {
	t.Parallel()

	p := &Profile{}

	result := ExplainFile("README.txt", "default", p)

	assert.True(t, result.Included)
	assert.Empty(t, result.Compression,
		".txt file must receive empty Compression (not supported)")
}

// TestExplainFile_RuleTraceOrder verifies that excluded files contain trace
// steps with correct sequential step numbers.
func TestExplainFile_RuleTraceOrder(t *testing.T) {
	t.Parallel()

	// The default ignore contains "node_modules" which matches the literal path
	// "node_modules" at step 1 (default ignore patterns).
	p := &Profile{}
	result := ExplainFile("node_modules", "default", p)

	require.NotEmpty(t, result.Trace, "excluded file must have at least one trace step")

	// Step numbers must start at 1 and be sequential.
	for i, step := range result.Trace {
		assert.Equal(t, i+1, step.StepNum,
			"step %d must have StepNum=%d, got %d", i, i+1, step.StepNum)
	}

	// Exclusion happens at the first step -- default ignore.
	assert.Equal(t, 1, result.Trace[0].StepNum)
	assert.True(t, result.Trace[0].Matched,
		"step 1 (default ignore) must be matched for node_modules path")
	assert.Equal(t, "EXCLUDED", result.Trace[0].Outcome)
}

// TestExplainFile_IncludeFilterExclusion verifies that when Include is active
// and a file doesn't match any Include pattern, the file is excluded by the
// include filter step.
func TestExplainFile_IncludeFilterExclusion(t *testing.T) {
	t.Parallel()

	p := &Profile{
		// Only .go files are included.
		Include: []string{"**/*.go"},
	}

	result := ExplainFile("src/styles/main.css", "default", p)

	assert.False(t, result.Included, "CSS file must be excluded when Include only allows .go")
	assert.Contains(t, result.ExcludedBy, "include filter",
		"ExcludedBy must mention the include filter")

	// Verify the include-filter step is present and marked EXCLUDED.
	var foundIncludeStep bool
	for _, step := range result.Trace {
		if step.Rule == "Include filter" && step.Outcome == "EXCLUDED" {
			foundIncludeStep = true
			break
		}
	}
	assert.True(t, foundIncludeStep, "trace must contain an EXCLUDED Include filter step")
}

// TestExplainFile_ExtendsField verifies that the ExplainResult.Extends field
// is populated from the profile's Extends pointer.
func TestExplainFile_ExtendsField(t *testing.T) {
	t.Parallel()

	parent := "default"
	p := &Profile{
		Extends: &parent,
	}

	result := ExplainFile("internal/main.go", "child", p)

	assert.Equal(t, "child", result.ProfileName)
	assert.Equal(t, "default", result.Extends,
		"ExplainResult.Extends must reflect the profile's Extends field")
}

// TestExplainFile_ExtendsNil verifies that a profile without Extends leaves
// the Extends field empty in the result.
func TestExplainFile_ExtendsNil(t *testing.T) {
	t.Parallel()

	p := &Profile{Extends: nil}

	result := ExplainFile("src/main.go", "default", p)

	assert.Empty(t, result.Extends,
		"ExplainResult.Extends must be empty when profile has no Extends")
}

// TestExplainFile_ProfileIgnoreExcludes verifies that a profile's own ignore
// patterns (step 2) can exclude files that pass the default ignore patterns.
func TestExplainFile_ProfileIgnoreExcludes(t *testing.T) {
	t.Parallel()

	p := &Profile{
		Ignore: []string{"build/**"},
	}

	result := ExplainFile("build/output/app.bin", "custom", p)

	assert.False(t, result.Included, "file matching profile ignore must be excluded")
	assert.Contains(t, result.ExcludedBy, "profile ignore pattern",
		"ExcludedBy must identify the profile ignore step")

	// The trace must have at least 2 steps: default ignore (no match) and
	// profile ignore (match -> EXCLUDED).
	require.GreaterOrEqual(t, len(result.Trace), 2)
	assert.Equal(t, "EXCLUDED", result.Trace[1].Outcome)
}

// TestExplainFile_FullTraceIncludedFile verifies that a file with no tier
// matches has all 11 trace steps (all pipeline stages executed without early
// exit).
func TestExplainFile_FullTraceIncludedFile(t *testing.T) {
	t.Parallel()

	// Empty profile -- no tier patterns -- so no early break in tier loop.
	p := &Profile{}

	result := ExplainFile("src/app.go", "default", p)

	require.True(t, result.Included)
	// Steps: 1 default ignore, 2 profile ignore, 3 gitignore, 4 include filter,
	// 5 priority files, 6-11 relevance tiers 0-5 (all no-match).
	assert.Equal(t, 11, len(result.Trace),
		"file with no tier match must have all 11 trace steps")
}

// TestExplainFile_TierFirstMatchWins verifies that the first matching tier
// wins when patterns from multiple tiers could match.
func TestExplainFile_TierFirstMatchWins(t *testing.T) {
	t.Parallel()

	p := &Profile{
		Relevance: RelevanceConfig{
			Tier0: []string{"internal/**"},
			Tier1: []string{"internal/**"}, // same pattern -- should not win
		},
	}

	result := ExplainFile("internal/config/main.go", "default", p)

	assert.True(t, result.Included)
	assert.Equal(t, 0, result.Tier, "first matching tier (tier_0) must win")
}

// TestExplainFile_RedactionOnWhenEnabledAndNotExcluded verifies that
// RedactionOn is true when Redaction is enabled and the file is not in
// ExcludePaths.
func TestExplainFile_RedactionOnWhenEnabledAndNotExcluded(t *testing.T) {
	t.Parallel()

	p := &Profile{
		Redaction: true,
		RedactionConfig: RedactionConfig{
			ExcludePaths: []string{"vendor/**"},
		},
	}

	result := ExplainFile("internal/config/main.go", "default", p)

	assert.True(t, result.Included)
	assert.True(t, result.RedactionOn,
		"RedactionOn must be true for file not in ExcludePaths when Redaction=true")
}

// TestExplainFile_RedactionOffWhenDisabled verifies that RedactionOn is false
// when the profile has Redaction=false, regardless of ExcludePaths.
func TestExplainFile_RedactionOffWhenDisabled(t *testing.T) {
	t.Parallel()

	p := &Profile{
		Redaction: false,
	}

	result := ExplainFile("src/main.go", "default", p)

	assert.True(t, result.Included)
	assert.False(t, result.RedactionOn,
		"RedactionOn must be false when profile Redaction is disabled")
}

// TestExplainFile_AllSupportedExtensions verifies the compressionLanguage
// helper returns the expected language name for all supported extensions.
func TestExplainFile_AllSupportedExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		wantLang string
	}{
		{name: "Go", filePath: "main.go", wantLang: "Go"},
		{name: "TypeScript", filePath: "app.ts", wantLang: "TypeScript"},
		{name: "TypeScript TSX", filePath: "app.tsx", wantLang: "TypeScript (TSX)"},
		{name: "JavaScript", filePath: "app.js", wantLang: "JavaScript"},
		{name: "JavaScript JSX", filePath: "app.jsx", wantLang: "JavaScript (JSX)"},
		{name: "Python", filePath: "script.py", wantLang: "Python"},
		{name: "Rust", filePath: "main.rs", wantLang: "Rust"},
		{name: "C", filePath: "prog.c", wantLang: "C"},
		{name: "C++", filePath: "prog.cpp", wantLang: "C++"},
		{name: "C/C++ header", filePath: "header.h", wantLang: "C/C++ header"},
		{name: "Java", filePath: "Main.java", wantLang: "Java"},
		{name: "Ruby", filePath: "app.rb", wantLang: "Ruby"},
		{name: "PHP", filePath: "index.php", wantLang: "PHP"},
		{name: "Swift", filePath: "App.swift", wantLang: "Swift"},
		{name: "Kotlin", filePath: "Main.kt", wantLang: "Kotlin"},
		{name: "C#", filePath: "Program.cs", wantLang: "C#"},
		{name: "unsupported txt", filePath: "notes.txt", wantLang: ""},
		{name: "unsupported yml", filePath: "config.yml", wantLang: ""},
		{name: "unsupported no ext", filePath: "Makefile", wantLang: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := compressionLanguage(tt.filePath)
			assert.Equal(t, tt.wantLang, got,
				"compressionLanguage(%q) must return %q", tt.filePath, tt.wantLang)
		})
	}
}

// TestExplainFile_GitignoreStepAlwaysContinues verifies that the .gitignore
// step (step 3) always has Matched=false and Outcome containing "not simulated".
func TestExplainFile_GitignoreStepAlwaysContinues(t *testing.T) {
	t.Parallel()

	p := &Profile{}
	result := ExplainFile("src/main.go", "default", p)

	require.GreaterOrEqual(t, len(result.Trace), 3)
	gitignoreStep := result.Trace[2]
	assert.Equal(t, 3, gitignoreStep.StepNum)
	assert.Equal(t, ".gitignore rules", gitignoreStep.Rule)
	assert.False(t, gitignoreStep.Matched)
	assert.Contains(t, gitignoreStep.Outcome, "not simulated")
}

// TestExplainFile_PriorityFileSkipsTierMatching verifies that when a file is
// in PriorityFiles, all relevance tier steps in the trace are marked "skipped".
func TestExplainFile_PriorityFileSkipsTierMatching(t *testing.T) {
	t.Parallel()

	p := &Profile{
		PriorityFiles: []string{"go.mod"},
		Relevance: RelevanceConfig{
			Tier0: []string{"go.mod"}, // would match, but priority takes precedence
		},
	}

	result := ExplainFile("go.mod", "default", p)

	require.True(t, result.IsPriority)
	assert.Equal(t, 0, result.Tier)
	assert.Equal(t, "priority_files", result.TierPattern)

	// All tier steps (steps 6-11) must be "skipped".
	for _, step := range result.Trace {
		if strings.HasPrefix(step.Rule, "Relevance ") {
			assert.Contains(t, step.Outcome, "skipped",
				"relevance step %q must be skipped for priority file", step.Rule)
		}
	}
}

// TestExplainFile_EmptyProfile verifies that ExplainFile handles a zero-value
// profile without panicking, and includes the file at tier -1.
func TestExplainFile_EmptyProfile(t *testing.T) {
	t.Parallel()

	p := &Profile{}
	result := ExplainFile("src/app.go", "empty", p)

	assert.True(t, result.Included)
	assert.Equal(t, -1, result.Tier)
	assert.False(t, result.IsPriority)
	assert.False(t, result.RedactionOn)
}

// TestMatchesAny verifies that matchesAny correctly reports matches.
func TestMatchesAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "matches first pattern",
			path:     "vendor/pkg/file.go",
			patterns: []string{"vendor/**", "dist/**"},
			want:     true,
		},
		{
			name:     "matches second pattern",
			path:     "dist/bundle.js",
			patterns: []string{"vendor/**", "dist/**"},
			want:     true,
		},
		{
			name:     "no match",
			path:     "internal/config/main.go",
			patterns: []string{"vendor/**", "dist/**"},
			want:     false,
		},
		{
			name:     "empty patterns",
			path:     "anything",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "nil patterns",
			path:     "anything",
			patterns: nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := matchesAny(tt.path, tt.patterns)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestMatchesGlob verifies that matchesGlob handles valid and invalid patterns
// without panicking, and returns false for bad patterns.
func TestMatchesGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{name: "exact match", pattern: "go.mod", path: "go.mod", want: true},
		{name: "doublestar match", pattern: "internal/**", path: "internal/config/main.go", want: true},
		{name: "no match", pattern: "src/**", path: "internal/config/main.go", want: false},
		{name: "invalid pattern silenced", pattern: "[invalid", path: "anything", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := matchesGlob(tt.pattern, tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}
