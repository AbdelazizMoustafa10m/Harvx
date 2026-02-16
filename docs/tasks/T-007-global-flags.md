# T-007: Global Flags Implementation

**Priority:** Must Have
**Effort:** Medium (6-8hrs)
**Dependencies:** T-005, T-004
**Phase:** 1 - Foundation

---

## Description

Implement all global persistent flags on the root Cobra command that are shared across subcommands. This includes directory targeting, output path, filtering flags, format/target selection, logging verbosity, and all Phase 1-relevant flags from PRD Section 5.9. Flags are registered as Cobra persistent flags and their values are parsed into a structured config object for downstream consumption.

## User Story

As a developer, I want to customize Harvx's behavior through CLI flags like `--dir`, `--output`, `--include`, `--exclude`, and `--verbose` so that I can control context generation without editing config files.

## Acceptance Criteria

- [ ] All Phase 1 global flags are registered as persistent flags on the root command:
  - `-d, --dir <path>` -- Target directory (default: `.` i.e. current directory)
  - `-o, --output <path>` -- Output file path (default: `harvx-output.md`)
  - `-f, --filter <ext>` -- Filter by file extension (repeatable: `-f ts -f go`)
  - `--include <pattern>` -- Include glob pattern (repeatable)
  - `--exclude <pattern>` -- Exclude glob pattern (repeatable)
  - `--format <type>` -- Output format: `markdown`, `xml` (default: `markdown`)
  - `--target <preset>` -- LLM target: `claude`, `chatgpt`, `generic` (default: `generic`)
  - `--git-tracked-only` -- Only include files in git index (default: false)
  - `--skip-large-files <size>` -- Skip files larger than threshold (default: `1MB`)
  - `--stdout` -- Output to stdout instead of file (default: false)
  - `--line-numbers` -- Add line numbers to code blocks (default: false)
  - `--no-redact` -- Disable secret redaction (default: false, i.e. redaction ON)
  - `--fail-on-redaction` -- Exit 1 if secrets detected (default: false)
  - `--verbose` -- Debug-level logging (default: false)
  - `--quiet` -- Error-only logging (default: false)
  - `--yes` -- Skip confirmation prompts (default: false)
  - `--clear-cache` -- Clear cached state before running (default: false)
- [ ] `--verbose` and `--quiet` are mutually exclusive (error if both set)
- [ ] `--dir` is validated to exist and be a directory at command execution time
- [ ] `--skip-large-files` accepts human-readable sizes: `1MB`, `500KB`, `2mb`, etc.
- [ ] `--format` validates against allowed values (`markdown`, `xml`)
- [ ] `--target` validates against allowed values (`claude`, `chatgpt`, `generic`)
- [ ] `-f` / `--filter` strips leading dots if provided (e.g., `.ts` becomes `ts`)
- [ ] Environment variable overrides work for key flags (prefix `HARVX_`): `HARVX_DIR`, `HARVX_OUTPUT`, `HARVX_FORMAT`, `HARVX_TARGET`, `HARVX_VERBOSE`, `HARVX_QUIET`
- [ ] An `internal/config/flags.go` file defines a `FlagValues` struct that collects all parsed flag values
- [ ] A `BindFlags(cmd *cobra.Command) *FlagValues` function handles registration and parsing
- [ ] `PersistentPreRunE` on the root command resolves flag values and validates them
- [ ] `harvx --help` shows all flags with descriptions and defaults
- [ ] Unit tests cover flag parsing, validation, and mutual exclusion

## Technical Notes

- Cobra persistent flags are inherited by all subcommands. Register them in an `init()` function in `root.go` or in a dedicated `flags.go`.
- For repeatable flags (`--include`, `--exclude`, `-f`), use `cobra.StringArrayVar` (not `StringSlice`, which splits on commas).
- Size parsing for `--skip-large-files`: implement a simple parser that handles `KB`, `MB`, `GB` suffixes (case-insensitive). Store as `int64` bytes internally.
- Environment variable binding: for Phase 1, use simple `os.Getenv()` fallbacks. Viper integration comes in Phase 2 with the profile system.
  ```go
  if v := os.Getenv("HARVX_DIR"); v != "" && !cmd.Flags().Changed("dir") {
      dir = v
  }
  ```
- The `FlagValues` struct in `internal/config/flags.go`:
  ```go
  type FlagValues struct {
      Dir             string
      Output          string
      Filters         []string  // file extensions
      Includes        []string  // glob patterns
      Excludes        []string  // glob patterns
      Format          string
      Target          string
      GitTrackedOnly  bool
      SkipLargeFiles  int64     // bytes
      Stdout          bool
      LineNumbers     bool
      NoRedact        bool
      FailOnRedaction bool
      Verbose         bool
      Quiet           bool
      Yes             bool
      ClearCache      bool
  }
  ```
- Per PRD Section 5.9: "Progress output and logs go to stderr so stdout remains clean for piping." This is already handled by T-004 logging setup, but the `--stdout` flag will need to be respected by the output renderer (later tasks).
- Reference: PRD Section 5.9 (Global flags list)

## Files to Create/Modify

- `internal/config/flags.go` - FlagValues struct and size parser
- `internal/config/flags_test.go` - Unit tests for flag parsing/validation
- `internal/cli/root.go` - Register persistent flags, PersistentPreRunE validation

## Testing Requirements

- Unit test: all flags have correct defaults
- Unit test: `--verbose` and `--quiet` mutual exclusion produces error
- Unit test: `--dir` with non-existent path produces error
- Unit test: `--format xyz` (invalid) produces error
- Unit test: `--target xyz` (invalid) produces error
- Unit test: `-f .ts` strips dot to `ts`
- Unit test: `-f ts -f go` produces `["ts", "go"]`
- Unit test: `--skip-large-files 500KB` parses to 512000 bytes
- Unit test: `--skip-large-files 2MB` parses to 2097152 bytes
- Unit test: `--skip-large-files 1mb` (lowercase) parses correctly
- Unit test: environment variable `HARVX_DIR` is respected when flag not explicitly set
- Unit test: explicit flag value overrides environment variable
