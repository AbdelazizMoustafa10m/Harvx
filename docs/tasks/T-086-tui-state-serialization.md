# T-086: TUI State Serialization to Profile TOML & Smart Default Launch

**Priority:** Must Have
**Effort:** Small (4-6hrs)
**Dependencies:** T-079 (TUI scaffold), T-080 (file tree model), T-083 (save action triggers serialization)
**Phase:** 5 - Interactive TUI

---

## Description

Implement the serialization of TUI file selection state into a reusable TOML profile, enabling users to save their interactive selections for later use in headless mode. Also implement the "smart default" behavior where running `harvx` with no arguments and no `harvx.toml` detected automatically launches the TUI, making first-time use discoverable and intuitive.

## User Story

As a developer who just explored my project in the TUI and selected the perfect set of files, I want to save that selection as a named profile so that I can reproduce the same context output later with `harvx --profile <name>` without opening the TUI again.

## Acceptance Criteria

- [ ] `internal/tui/serialize.go` implements `SerializeToProfile(name string, nodes []*filetree.Node, baseConfig config.ResolvedConfig) ([]byte, error)` that produces valid TOML
- [ ] Serialized profile includes:
  - Profile name as section header: `[profile.<name>]`
  - `extends = "default"` for inheritance
  - `priority_files` array from tier 0 included files
  - `relevance` section with tier assignments derived from included files' tiers
  - `ignore` patterns derived from explicitly excluded files/directories
  - `include` patterns for files that don't match any tier glob but were manually included
  - Preserved settings from the active profile (format, max_tokens, tokenizer, compression, redaction)
- [ ] Pattern minimization: if all files in a directory are included, use a directory glob (`dir/**`) instead of listing individual files
- [ ] Pattern minimization: if all files in a directory are excluded, use a single ignore entry instead of listing individual files
- [ ] Serialized TOML is valid and parseable by the existing config loader
- [ ] Round-trip test: serialize from TUI state, reload profile, run headless -- same files included
- [ ] Save flow: writes to `harvx.toml` in the project root. If file exists, appends/updates the profile section. If file doesn't exist, creates a new file with the profile.
- [ ] Smart default detection logic in `internal/cli/root.go`:
  1. Check `os.Args` -- if only the binary name (no subcommand, no flags)
  2. Check for `harvx.toml` in current dir and parent dirs up to filesystem root
  3. If no args AND no `harvx.toml` found, set `interactive = true`
  4. If `harvx.toml` exists, run headless generation with default profile as normal
- [ ] Smart default prints a one-line hint on first TUI launch: `Tip: Run 'harvx -i' to always open the interactive mode, or create a harvx.toml for headless use.`
- [ ] Unit tests for serialization, pattern minimization, and smart default detection

## Technical Notes

- Use `BurntSushi/toml` for TOML generation (same library used for parsing in the config package).
- Pattern minimization algorithm: walk the tree bottom-up. For each directory, if all children are included, replace with `dir/**` glob. If all children are excluded, add to ignore list. Only include explicit patterns for mixed directories.
- When appending to existing `harvx.toml`, parse the file first, add/update the profile section, and re-serialize. Use `toml.Encoder` to produce clean output.
- Careful with TOML table ordering: `[profile.<name>]` must come after any existing profiles but the order within the profile can follow the standard convention (output settings, then relevance, then ignore).
- Smart default detection must happen before Cobra command execution. Use Cobra's `PersistentPreRunE` on the root command, or check in the `RunE` of the root command.
- Reference: PRD Section 5.13 (TUI state serialization to profile TOML, Smart default)

## Files to Create/Modify

- `internal/tui/serialize.go` - Profile serialization from TUI state
- `internal/tui/serialize_test.go` - Serialization unit tests
- `internal/tui/patterns.go` - Pattern minimization algorithm
- `internal/tui/patterns_test.go` - Pattern minimization tests
- `internal/cli/root.go` - Smart default detection logic (modify)
- `internal/cli/root_test.go` - Smart default detection tests

## Testing Requirements

- Unit test: serialization produces valid TOML parseable by `BurntSushi/toml`
- Unit test: all included tier-0 files appear in `priority_files`
- Unit test: directory with all children included produces `dir/**` glob
- Unit test: directory with all children excluded produces single ignore entry
- Unit test: mixed directory produces individual file entries
- Round-trip test: serialize state -> parse profile -> resolve config -> same file set
- Unit test: smart default returns `interactive=true` when no args and no harvx.toml
- Unit test: smart default returns `interactive=false` when harvx.toml exists
- Unit test: smart default returns `interactive=false` when subcommand is present
- Unit test: smart default returns `interactive=false` when any flag is present
- Unit test: appending profile to existing harvx.toml preserves other profiles