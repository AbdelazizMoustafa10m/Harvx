# T-023: Profile CLI Subcommands -- lint and explain

**Priority:** Should Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-020 (Config Validation), T-022 (profiles init/list/show), T-005 (File Discovery -- Phase 1)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Implement two diagnostic profile subcommands: `harvx profiles lint` for comprehensive configuration validation with actionable fix suggestions, and `harvx profiles explain <filepath>` which shows which profile rules apply to a specific file and why it would be included, excluded, or assigned to a particular tier. These commands are essential for debugging complex profile configurations.

## User Story

As a developer with a complex profile configuration, I want to lint my config for issues and explain why a specific file is being included or excluded, so that I can debug unexpected context generation behavior without guessing.

## Acceptance Criteria

### `harvx profiles lint`
- [ ] Runs all validation checks from T-020 `Validate()` plus extended `Lint()` analysis
- [ ] Groups output by severity: errors first, then warnings, then info
- [ ] Shows total counts: `N errors, M warnings, K info`
- [ ] Each issue shows:
  - Severity icon (X for error, ! for warning, i for info)
  - Field path or context
  - Description of the issue
  - Suggested fix
- [ ] Returns exit code 1 if any errors found, exit code 0 otherwise (warnings are OK)
- [ ] With `--profile <name>`: lint only the specified profile
- [ ] Without `--profile`: lint all profiles in the config
- [ ] Example output:
  ```
  Linting harvx.toml...

  Errors:
    X [profile.work.format] "html" is not a valid format
      Fix: Use one of: markdown, xml, plain

  Warnings:
    ! [profile.finvault.relevance] Pattern "*.config.*" in tier_0 also matches tier_2 pattern "**/*"
      Fix: Remove the broader pattern from tier_2 or make it more specific
    ! [profile.finvault] priority_files entry "CLAUDE.md" also in tier_0 relevance (redundant)
      Fix: Priority files are automatically tier 0; remove from relevance.tier_0

  Info:
    i [profile.session] max_tokens=8000 is very small; output may omit most files
    i [profile.finvault] 4 relevance tiers defined (2 empty: tier_3, tier_4)

  Result: 1 error, 2 warnings, 2 info
  ```

### `harvx profiles explain <filepath>`
- [ ] Takes a file path (relative to repo root) and shows how the active profile processes it
- [ ] Shows:
  - Whether the file is included or excluded
  - Which rule caused inclusion/exclusion (ignore pattern, include pattern, extension filter, etc.)
  - Which relevance tier the file is assigned to and which pattern matched
  - The priority level of the tier
  - Whether redaction would scan the file (or skip via exclude_paths)
  - Whether compression would apply (based on language detection)
- [ ] With `--profile <name>`: explain against specified profile
- [ ] Works with glob patterns too: `harvx profiles explain "src/**/*.ts"` shows matching files
- [ ] Example output:
  ```
  Explaining: lib/services/transaction.ts
  Profile: finvault (extends: default)

    Status:     INCLUDED
    Tier:       1 (high priority)
    Matched by: tier_1 pattern "lib/services/**"
    Redaction:  enabled (not in exclude_paths)
    Compress:   yes (TypeScript supported)
    Priority:   not in priority_files

  Rule trace:
    1. Default ignore patterns: no match -> continue
    2. Profile ignore patterns: no match -> continue
    3. .gitignore rules: no match -> continue
    4. Extension filter: not active -> continue
    5. Relevance tier_0: no match
    6. Relevance tier_1: MATCH "lib/services/**" -> assigned tier 1
  ```
- [ ] If the file would be excluded, shows which rule excluded it:
  ```
  Explaining: node_modules/lodash/index.js
  Profile: finvault (extends: default)

    Status:     EXCLUDED
    Excluded by: default ignore pattern "node_modules"

  Rule trace:
    1. Default ignore patterns: MATCH "node_modules" -> EXCLUDED
  ```

### General
- [ ] Both subcommands registered under `harvx profiles` parent command
- [ ] `explain` has `ValidArgsFunction` for filepath completion
- [ ] Both commands work without a loaded repo config (using defaults)

## Technical Notes

- `profiles lint` primarily calls `Lint()` from T-020 and formats the output
- `profiles explain` needs to simulate the file processing pipeline:
  1. Check default ignores (from `defaults.go`)
  2. Check profile-specific ignores
  3. Check .gitignore patterns (if a git repo -- depends on Phase 1 discovery module)
  4. Check include/exclude filters
  5. Match against relevance tiers (first match wins, per PRD Section 5.3)
  6. Check redaction exclude_paths
  7. Detect language for compression applicability
- For `explain` with glob patterns, use `bmatcuk/doublestar.Glob()` to expand the pattern to actual files first, then explain each
- The rule trace is a slice of steps built during evaluation -- each step records the rule, whether it matched, and the outcome
- Use `bmatcuk/doublestar.Match()` for glob pattern matching (same engine used by the actual pipeline)
- The explain command needs access to the relevance matching logic from `internal/relevance/` (Phase 2 parallel task). If that's not ready, implement a standalone matcher here and refactor later, or mark as a dependency
- Language detection for compression: simple map of file extension to language name (`.ts` -> TypeScript, `.go` -> Go, etc.)
- This command is informational only -- it does not generate any output files

## Files to Create/Modify

- `internal/cli/profiles_lint.go` - Lint subcommand implementation
- `internal/cli/profiles_explain.go` - Explain subcommand implementation
- `internal/cli/profiles_lint_test.go` - Lint command tests
- `internal/cli/profiles_explain_test.go` - Explain command tests
- `internal/config/explain.go` - Explain engine: rule trace evaluation
- `internal/config/explain_test.go` - Explain logic tests

## Testing Requirements

### Lint
- Unit test: Clean config returns 0 errors, exit code 0
- Unit test: Config with invalid format returns error
- Unit test: Config with overlapping tiers returns warning
- Unit test: `--profile` flag limits linting to one profile
- Unit test: Output formatting matches expected structure
- Unit test: Exit code 1 when errors found, 0 when only warnings

### Explain
- Unit test: File in tier_1 shows correct tier assignment and pattern
- Unit test: File in ignore list shows EXCLUDED with correct rule
- Unit test: File in priority_files shows tier 0
- Unit test: File not matching any tier shows default tier (tier 2)
- Unit test: Redaction exclude_paths correctly reflected in output
- Unit test: Compression applicability shown for supported language
- Unit test: Compression inapplicable shown for unsupported language
- Unit test: Glob pattern input expands to multiple files
- Unit test: Nonexistent file still shows theoretical evaluation
- Unit test: Full rule trace includes all evaluation steps in order

## References

- [bmatcuk/doublestar](https://github.com/bmatcuk/doublestar)
- PRD Section 5.2 - `harvx profiles lint` and `harvx profiles explain <filepath>`
- PRD Section 5.3 - Relevance tier matching rules
- PRD Section 5.9 - Profile management subcommands
