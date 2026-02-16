# T-016: Configuration Types, Defaults, and TOML Loading

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-001 (Project Scaffolding), T-002 (Cobra CLI Setup)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Define all configuration structs for the Harvx profile system and implement TOML parsing with `BurntSushi/toml`. This task establishes the foundational data model that every other profile task depends on. It includes the Go struct hierarchy for profiles, the built-in default profile, and the low-level TOML decoding logic.

## User Story

As a developer, I want Harvx to have a well-defined configuration schema with sensible defaults so that the tool works out of the box on any repository without requiring a config file.

## Acceptance Criteria

- [ ] `internal/config/types.go` defines all configuration structs:
  - `Config` (top-level): holds map of named profiles, global settings
  - `Profile`: output, tokenizer, relevance, ignore, priority_files, include, compression, redaction, target
  - `OutputConfig`: file path, format (markdown/xml/plain), max_tokens
  - `RedactionConfig`: enabled, exclude_paths, confidence_threshold
  - `RelevanceConfig`: tier_0 through tier_5 (each `[]string` of glob patterns)
- [ ] `internal/config/defaults.go` defines the built-in `default` profile:
  - `output = "harvx-output.md"`, `format = "markdown"`, `max_tokens = 128000`
  - `tokenizer = "cl100k_base"`, `compression = false`, `redaction = true`
  - `ignore` includes: `node_modules`, `dist`, `.git`, `coverage`, `__pycache__`, `.next`, `target`, `vendor`
  - Default relevance tiers per PRD Section 5.3
- [ ] `internal/config/loader.go` implements `LoadFromFile(path string) (*Config, error)` using `BurntSushi/toml`
- [ ] Parsing handles the `[profile.<name>]` table structure from the PRD example
- [ ] Nested tables like `[profile.<name>.relevance]` and `[profile.<name>.redaction]` decode correctly
- [ ] Unknown keys produce a warning (not an error) for forward compatibility
- [ ] Invalid TOML syntax returns a clear error with file path and line number
- [ ] Unit tests achieve 90%+ coverage for types, defaults, and loader
- [ ] All fields have `toml` struct tags matching the PRD field names

## Technical Notes

- Use `github.com/BurntSushi/toml` v1.5.0 for TOML parsing -- it supports TOML v1.0 spec and provides `toml.MetaData` for detecting unknown keys
- Do NOT use Viper/koanf in this task -- raw TOML loading is separate from multi-source merging (that is T-017)
- Use `toml.DecodeFile()` for file loading and `toml.Decode()` for string-based loading (useful for tests)
- The `MetaData.Undecoded()` method can detect unknown keys for warning messages
- Profile names are case-sensitive strings used as map keys
- The `extends` field is stored as a `string` pointer (`*string`) -- nil means no inheritance
- Use Go 1.22+ features (range over int, etc.) where appropriate
- Relevance tier arrays are `[]string` containing glob patterns (validated later in T-022)
- The `target` field is an enum-like string: `"claude"`, `"chatgpt"`, `"generic"`, or empty

## Files to Create/Modify

- `internal/config/types.go` - All configuration struct definitions with toml tags
- `internal/config/defaults.go` - Built-in default profile and default tier assignments
- `internal/config/loader.go` - TOML file loading and decoding
- `internal/config/loader_test.go` - Unit tests for TOML loading
- `internal/config/types_test.go` - Unit tests for struct defaults and field validation
- `testdata/config/valid.toml` - Valid test configuration (mirrors PRD example)
- `testdata/config/minimal.toml` - Minimal valid configuration
- `testdata/config/invalid_syntax.toml` - Malformed TOML for error testing
- `testdata/config/unknown_keys.toml` - TOML with extra keys for warning testing

## Testing Requirements

- Unit test: Load the PRD example `harvx.toml` and verify all fields decode correctly
- Unit test: Load minimal config (just `[profile.default]`) and verify defaults fill in
- Unit test: Malformed TOML returns error with line number
- Unit test: Unknown keys produce warnings but do not fail
- Unit test: Empty file returns empty config (no error)
- Unit test: Nested redaction and relevance configs decode into correct structs
- Unit test: `extends` field parsed correctly (string pointer)
- Unit test: Default profile values match PRD specification
- Golden test: Encode default profile back to TOML and verify round-trip

## References

- [BurntSushi/toml v1.5.0](https://github.com/BurntSushi/toml)
- [BurntSushi/toml pkg.go.dev](https://pkg.go.dev/github.com/BurntSushi/toml)
- PRD Section 5.2 (Profile System) - Example Configuration
- PRD Section 5.3 (Relevance-Based File Sorting) - Default Tier Assignments
