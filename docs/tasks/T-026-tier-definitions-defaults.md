# T-026: Tier Definitions and Default Tier Assignments

**Priority:** Must Have
**Effort:** Medium (6-8hrs)
**Dependencies:** None (data types only; integrates with file discovery from Phase 1)
**Phase:** 2 - Intelligence (Relevance & Tokens)

---

## Description

Define the 6-tier relevance system (tier 0 = highest priority through tier 5 = lowest) as Go types and implement the default tier assignment rules. This task establishes the core data structures for the relevance system and provides built-in heuristics that classify files into tiers based on filename patterns, directory paths, and file extensions -- without requiring any profile configuration.

## User Story

As a developer running Harvx with no profile configured, I want my configuration files and core source code to automatically rank higher than tests and CI config so that the LLM sees the most important files first.

## Acceptance Criteria

- [ ] `Tier` type defined as an integer constant (0-5) with named constants (Tier0Critical through Tier5Low)
- [ ] `TierDefinition` struct holds a tier level and a list of glob patterns
- [ ] Default tier assignments match PRD Section 5.3:
  - Tier 0: Config files (package.json, tsconfig.json, Cargo.toml, go.mod, Makefile, Dockerfile, *.config.*)
  - Tier 1: Primary source directories (src/**, lib/**, app/**, cmd/**, internal/**, pkg/**)
  - Tier 2: Secondary source, components, utilities (default for unmatched files)
  - Tier 3: Test files (*_test.go, *.test.ts, *.spec.js, __tests__/**)
  - Tier 4: Documentation (*.md, docs/**, README*)
  - Tier 5: CI/CD (.github/**, .gitlab-ci.yml), lock files (*.lock, package-lock.json)
- [ ] Unmatched files default to Tier 2
- [ ] Default patterns are defined as a Go variable that can be overridden by profile-defined relevance tiers
- [ ] A function `DefaultTierDefinitions() []TierDefinition` returns the built-in defaults
- [ ] Unit tests achieve 95%+ coverage for the tier definitions module
- [ ] All tier constants, types, and defaults are exported for use by other packages

## Technical Notes

- Create in `internal/relevance/tiers.go`
- Tier patterns use glob syntax compatible with `bmatcuk/doublestar/v4` (validated in T-027)
- Keep patterns as plain strings at this stage; actual matching is implemented in T-027
- The `TierDefinition` should be serializable to/from TOML for profile integration (Phase 2 profile tasks)
- Unmatched default of Tier 2 is important: it avoids excluding unexpected but potentially important files
- Profile-defined relevance tiers override defaults **entirely** (not merged), per PRD

## Files to Create/Modify

- `internal/relevance/tiers.go` - Tier type, constants, TierDefinition struct, DefaultTierDefinitions()
- `internal/relevance/tiers_test.go` - Unit tests for all tier definitions and defaults

## Testing Requirements

- Unit tests verifying all 6 tiers have correct constant values (0-5)
- Unit tests verifying DefaultTierDefinitions() returns exactly 6 tier groups
- Unit tests verifying specific patterns exist in expected tiers (e.g., "go.mod" in tier 0, "src/**" in tier 1)
- Unit tests verifying the default tier for unmatched files is 2
- Table-driven tests for tier constant validity and ordering
- Test that TierDefinition struct is properly constructible with custom patterns (for profile override path)