# T-024: Config Debug Command

**Priority:** Should Have
**Effort:** Small (4-6hrs)
**Dependencies:** T-017 (Multi-Source Config Merging), T-022 (profiles init/list/show)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Implement `harvx config debug` -- a diagnostic command that shows the fully resolved configuration with source annotations indicating where each value came from (built-in default, global config, repo config, environment variable, or CLI flag). This is the primary troubleshooting tool when users are confused about which configuration source is winning for a particular setting.

## User Story

As a developer debugging unexpected Harvx behavior, I want to see exactly which configuration source provides each setting value, so that I can quickly identify where a misconfiguration is coming from and fix it.

## Acceptance Criteria

- [ ] `harvx config debug` displays the complete resolved config with source annotations
- [ ] Each line shows: key, value, and source (where the value came from)
- [ ] Sources are labeled: `default`, `global (~/.config/harvx/config.toml)`, `repo (./harvx.toml)`, `env (HARVX_*)`, `flag (--*)`
- [ ] Shows which config files were loaded (and which were not found)
- [ ] Shows the active profile name and inheritance chain
- [ ] Shows environment variables that were detected and applied
- [ ] With `--json` flag: output as structured JSON for programmatic consumption
- [ ] With `--profile <name>`: show debug info for a specific profile
- [ ] Example output:
  ```
  Harvx Configuration Debug
  ==========================

  Config Files:
    Global: ~/.config/harvx/config.toml (not found)
    Repo:   ./harvx.toml (loaded)

  Active Profile: finvault (extends: default)

  Environment Variables:
    HARVX_MAX_TOKENS = 150000 (applied)
    HARVX_COMPRESS   = (not set)

  Resolved Configuration:
    KEY                              VALUE                    SOURCE
    output                           .harvx/finvault-ctx.md   repo
    format                           markdown                 default
    max_tokens                       150000                   env (HARVX_MAX_TOKENS)
    tokenizer                        o200k_base               repo
    compression                      true                     repo
    redaction.enabled                true                     default
    redaction.confidence_threshold   high                     default
    redaction.exclude_paths          [**/*test*/**]           repo
    target                           claude                   repo
    ignore                           [node_modules, dist...]  repo
    priority_files                   [CLAUDE.md, prisma/...]  repo
    relevance.tier_0                 [CLAUDE.md, *.config.*]  repo
    relevance.tier_1                 [app/api/**, lib/**]     repo
    relevance.tier_2                 [components/**, hooks/**] repo
    relevance.tier_3                 (not set)                -
    relevance.tier_4                 (not set)                -
    relevance.tier_5                 (not set)                -
  ```
- [ ] Exit code always 0 (debug is informational, never fails)

## Technical Notes

- This command consumes the source annotations from T-017's `ResolvedConfig`
- Use `text/tabwriter` for aligned columnar output
- The "Config Files" section uses the discovery results from T-018
- For JSON output, create a `DebugOutput` struct with fields for files, profile, env vars, and config entries, then marshal with `encoding/json`
- Environment variable detection: iterate known `HARVX_*` keys and check `os.Getenv()`, marking each as "applied" or "not set"
- For values that are slices (ignore, priority_files, tier patterns), show abbreviated form if too long (first 3 items + "..." + count)
- Color coding via lipgloss (if available): source labels in distinct colors to make scanning easier
- Register as `configCmd` parent with `debugCmd` child under root command
- This could also be `harvx debug config` -- follow whatever command hierarchy feels natural; the PRD says `harvx config debug`

## Files to Create/Modify

- `internal/cli/config_debug.go` - Config debug subcommand implementation
- `internal/cli/config_debug_test.go` - Command output tests
- `internal/config/debug.go` - Debug output generation logic (decoupled from CLI)
- `internal/config/debug_test.go` - Debug output unit tests

## Testing Requirements

- Unit test: Default-only config shows all values sourced from "default"
- Unit test: Repo config overrides show correct source
- Unit test: Env var override shows "env (HARVX_MAX_TOKENS)" source
- Unit test: CLI flag override shows "flag (--max-tokens)" source
- Unit test: Missing global config file noted as "not found"
- Unit test: Present repo config file noted as "loaded"
- Unit test: `--json` outputs valid JSON matching expected structure
- Unit test: Profile inheritance chain correctly displayed
- Unit test: Slice values abbreviated when longer than 3 items
- Unit test: All known HARVX_ env vars checked and reported

## References

- PRD Section 5.9 - `harvx config debug` -- "Show resolved config with source annotations"
- PRD Section 6.6 - Logging & Diagnostics
- PRD Section 6.6 - `HARVX_DEBUG=1` dumps effective resolved config
