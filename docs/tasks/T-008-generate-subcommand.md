# T-008: Generate Subcommand (harvx generate / harvx gen)

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-005, T-007
**Phase:** 1 - Foundation

---

## Description

Implement the `harvx generate` subcommand (with alias `harvx gen`) that serves as the explicit context generation command. Also wire the root command's `RunE` to delegate to generate, so that `harvx` (no subcommand) and `harvx generate` behave identically. This task creates the command skeleton and the high-level pipeline orchestration stub that later tasks will fill in with real discovery, filtering, and rendering logic.

## User Story

As a developer, I want to run `harvx generate` (or just `harvx`) to produce a context file so that I have a single, memorable command for the primary workflow.

## Acceptance Criteria

- [ ] `harvx generate` subcommand is registered with alias `gen`
- [ ] Running `harvx gen` is equivalent to `harvx generate`
- [ ] Running `harvx` (no subcommand) delegates to the generate logic
- [ ] The generate command accepts and respects all global flags from T-007
- [ ] The generate command has its own `--preview` flag (show file tree and token estimate without writing output; stub for now)
- [ ] `internal/cli/generate.go` defines the command
- [ ] `internal/pipeline/pipeline.go` defines a `Run(ctx context.Context, cfg *config.FlagValues) error` function stub that the generate command calls
- [ ] The pipeline stub:
  1. Logs "Starting Harvx context generation..." at info level
  2. Logs the resolved configuration (dir, output, format, target) at debug level
  3. Returns nil (success) -- actual implementation comes in later tasks
- [ ] When the pipeline returns an error, the generate command returns it to the root command for proper exit code handling
- [ ] `context.Context` is threaded from `cmd.Context()` into the pipeline for cancellation support (Ctrl+C)
- [ ] Help text for `harvx generate --help` clearly describes the command's purpose
- [ ] `harvx help generate` works (Cobra default behavior)
- [ ] Unit tests verify command registration, alias, and flag inheritance

## Technical Notes

- Cobra alias pattern:
  ```go
  var generateCmd = &cobra.Command{
      Use:     "generate",
      Aliases: []string{"gen"},
      Short:   "Generate LLM-optimized context from a codebase",
      Long:    `Recursively discover files, apply filters, and produce a structured context document.`,
      RunE: func(cmd *cobra.Command, args []string) error {
          ctx := cmd.Context()
          flags := config.GetFlagValues(cmd)
          return pipeline.Run(ctx, flags)
      },
  }
  ```
- To make `harvx` (no subcommand) run generate, set `rootCmd.RunE` to the same function. Alternatively, use Cobra's `rootCmd.AddCommand(generateCmd)` and set `rootCmd.RunE` to call `generateCmd.RunE`.
- The pipeline stub in `internal/pipeline/pipeline.go` is intentionally minimal:
  ```go
  func Run(ctx context.Context, cfg *config.FlagValues) error {
      slog.Info("Starting Harvx context generation",
          "dir", cfg.Dir,
          "output", cfg.Output,
          "format", cfg.Format,
      )
      // TODO: Implement discovery, filtering, rendering pipeline
      return nil
  }
  ```
- Per PRD Section 5.9: "Root command: `harvx` (runs context generation with auto-detected or default profile)" and "`harvx generate` (alias: `harvx gen`) -- Explicit generation command (same as root)."
- Per PRD Section 6.7: "Only `internal/pipeline` orchestrates multiple layers." The pipeline package is the central coordinator.
- Reference: PRD Sections 5.9, 6.3, 6.7

## Files to Create/Modify

- `internal/cli/generate.go` - Generate subcommand definition
- `internal/cli/generate_test.go` - Unit tests
- `internal/cli/root.go` - Wire root RunE to generate, register generate subcommand
- `internal/pipeline/pipeline.go` - Pipeline Run stub

## Testing Requirements

- Unit test: `generate` command is registered on root
- Unit test: `gen` alias resolves to the generate command
- Unit test: running `harvx generate` calls the pipeline stub (mock or capture slog output)
- Unit test: `--preview` flag is available on the generate command
- Unit test: root command with no subcommand delegates to generate logic
- Unit test: context cancellation is propagated (pass a cancelled context, verify early return)
