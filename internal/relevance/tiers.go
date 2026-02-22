// Package relevance implements tier-based file sorting and token budget management.
// Tier definitions originate from PRD Section 5.3 (Relevance Tiers).
//
// Design note: profile-defined relevance tiers override the defaults entirely
// (no merging). When a profile specifies its own tier_definitions, the defaults
// returned by DefaultTierDefinitions are discarded and the profile list is used
// as-is. This allows profiles to define a completely different priority model
// without inheriting any default tier patterns.
package relevance

import "fmt"

// Tier represents the relevance priority of a file.
// Lower numbers indicate higher priority (Tier0Critical is the most important).
type Tier int

const (
	// Tier0Critical contains configuration and project root files that define
	// the project's shape (e.g. go.mod, Dockerfile, package.json).
	Tier0Critical Tier = 0

	// Tier1Primary contains primary source code directories (src/, cmd/, internal/).
	Tier1Primary Tier = 1

	// Tier2Secondary is the default tier for files that do not match any other
	// pattern. It contains secondary source directories (components/, utils/).
	Tier2Secondary Tier = 2

	// Tier3Tests contains test files and test directories.
	Tier3Tests Tier = 3

	// Tier4Docs contains documentation files and directories.
	Tier4Docs Tier = 4

	// Tier5Low contains low-priority CI, lock, and generated files.
	Tier5Low Tier = 5
)

// DefaultUnmatchedTier is the tier assigned to files that do not match any
// pattern in the active TierDefinition list.
const DefaultUnmatchedTier = Tier2Secondary

// String returns a human-readable label for the tier.
func (t Tier) String() string {
	switch t {
	case Tier0Critical:
		return "critical"
	case Tier1Primary:
		return "primary"
	case Tier2Secondary:
		return "secondary"
	case Tier3Tests:
		return "tests"
	case Tier4Docs:
		return "docs"
	case Tier5Low:
		return "low"
	default:
		return fmt.Sprintf("tier%d", int(t))
	}
}

// TierDefinition maps a Tier to the glob patterns that place a file into it.
// Patterns use doublestar (bmatcuk/doublestar/v4) glob syntax; validation is
// performed by the classifier in T-027.
type TierDefinition struct {
	Tier     Tier     `toml:"tier"`
	Patterns []string `toml:"patterns"`
}

// DefaultTierDefinitions returns the built-in tier definitions as specified in
// PRD Section 5.3. These are used when no profile overrides tier_definitions.
//
// Note: go.sum appears only in Tier5Low (lock files). It is intentionally
// absent from Tier0Critical to avoid double-counting the file.
func DefaultTierDefinitions() []TierDefinition {
	return []TierDefinition{
		{
			// Tier 0 — Critical/Config: project root files that define the
			// project's shape, build system, and runtime environment.
			Tier: Tier0Critical,
			Patterns: []string{
				"package.json",
				"tsconfig.json",
				"Cargo.toml",
				"go.mod",
				"Makefile",
				"Dockerfile",
				"*.config.*",
				"*.config.js",
				"*.config.ts",
				"pyproject.toml",
				"setup.py",
				"requirements.txt",
				".env.example",
				"docker-compose.yml",
				"docker-compose.yaml",
			},
		},
		{
			// Tier 1 — Primary source: top-level source directories that
			// contain the main application code.
			Tier: Tier1Primary,
			Patterns: []string{
				"src/**",
				"lib/**",
				"app/**",
				"cmd/**",
				"internal/**",
				"pkg/**",
			},
		},
		{
			// Tier 2 — Secondary source (default): auxiliary source
			// directories. Files that match no pattern also land here via
			// DefaultUnmatchedTier.
			Tier: Tier2Secondary,
			Patterns: []string{
				"components/**",
				"utils/**",
				"helpers/**",
				"services/**",
				"api/**",
				"handlers/**",
				"controllers/**",
				"models/**",
			},
		},
		{
			// Tier 3 — Tests: test files and dedicated test directories.
			Tier: Tier3Tests,
			Patterns: []string{
				"*_test.go",
				"*.test.ts",
				"*.test.js",
				"*.spec.ts",
				"*.spec.js",
				"__tests__/**",
				"test/**",
				"tests/**",
				"spec/**",
			},
		},
		{
			// Tier 4 — Docs: documentation, changelogs, licences, and plain
			// text files.
			Tier: Tier4Docs,
			Patterns: []string{
				"*.md",
				"docs/**",
				"README*",
				"CHANGELOG*",
				"LICENSE*",
				"*.txt",
				"*.rst",
			},
		},
		{
			// Tier 5 — Low/CI: CI configuration, lock files, and other
			// generated or machine-managed files.
			Tier: Tier5Low,
			Patterns: []string{
				".github/**",
				".gitlab-ci.yml",
				".circleci/**",
				"*.lock",
				"package-lock.json",
				"yarn.lock",
				"pnpm-lock.yaml",
				"go.sum",
				"Pipfile.lock",
				"poetry.lock",
			},
		},
	}
}
