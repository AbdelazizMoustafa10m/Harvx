# T-095: Doctor Command and README Documentation

**Priority:** Should Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-005 (Cobra CLI), T-015 (File Discovery), T-020 (Config Validation)
**Phase:** 6 - Polish & Distribution

---

## Description

Implement the `harvx doctor` diagnostic command and write comprehensive README documentation with canonical usage recipes per persona. The doctor command checks for common repository issues (large binaries not ignored, huge files, missing config, stale cache), while the README provides installation instructions, quickstart guides, and persona-specific workflows (Alex: daily chat user, Zizo: pipeline integrator, Jordan: CI integrator).

## User Story

As a developer, I want `harvx doctor` to identify potential issues with my repository setup so that I can fix problems before they affect context generation.

As a new user, I want clear documentation with real-world examples so that I can start using Harvx effectively within minutes.

## Acceptance Criteria

### `harvx doctor` Command

- [ ] Registered as `harvx doctor` subcommand via Cobra
- [ ] Checks for large binary files not in `.gitignore` (warns if >1MB binaries found in tracked files)
- [ ] Checks for oversized text files that might blow token budgets (warns if >500KB text files)
- [ ] Checks for missing `.harvxignore` when common build artifacts are detected (node_modules/, dist/, target/)
- [ ] Validates `harvx.toml` if present (delegates to config validation engine from T-020)
- [ ] Checks for stale cache files in `.harvx/state/`
- [ ] Reports git repository status (branch, HEAD SHA, clean/dirty)
- [ ] Outputs results as a checklist with pass/warn/fail indicators
- [ ] Supports `--json` flag for machine-readable output
- [ ] Supports `--fix` flag that auto-generates `.harvxignore` entries for detected issues
- [ ] Exit code 0 if all checks pass, exit code 1 if any failures detected

### README Documentation

- [ ] Installation section: go install, binary download, homebrew (future)
- [ ] Quickstart: zero-config usage (`harvx` in any directory)
- [ ] Configuration guide: harvx.toml basics, profile system overview
- [ ] Persona recipes from PRD Section 5.9:
  - Alex (quick use): `harvx` / `harvx -i` / `harvx --compress`
  - Zizo (pipeline): `harvx brief --profile finvault && harvx review-slice --base main --head HEAD`
  - Jordan (CI): `harvx --profile ci --fail-on-redaction --output-metadata --quiet`
- [ ] Command reference: all subcommands with usage examples
- [ ] Profile template examples
- [ ] Claude Code hook setup example
- [ ] Comparison with alternatives (Repomix, code2prompt, etc.)

## Technical Notes

- Use `spf13/cobra` for the doctor subcommand
- Binary detection reuses logic from `internal/discovery/binary.go` (T-013)
- Config validation reuses `internal/config/validate.go` (T-020)
- Doctor output uses `charmbracelet/lipgloss` for styled terminal output (pass=green, warn=amber, fail=red)
- README follows standard Go project conventions with badges (Go version, license, release)
- Reference: PRD Sections 5.9 (doctor command), 8.1 (helpful errors), 10 (roadmap Phase 6)

## Files to Create/Modify

- `internal/cli/doctor.go` - Doctor subcommand implementation
- `internal/doctor/checks.go` - Individual check functions
- `internal/doctor/reporter.go` - Output formatting (text and JSON)
- `README.md` - Comprehensive project documentation
- `internal/cli/root.go` - Register doctor subcommand

## Testing Requirements

- Unit test: each doctor check function with mock filesystem
- Unit test: JSON output format validation
- Unit test: exit code behavior (0 for pass, 1 for failure)
- Integration test: run doctor on testdata/ sample repos
- README: verify all command examples are syntactically valid
