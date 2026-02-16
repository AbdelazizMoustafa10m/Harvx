# T-079: Bubble Tea Application Scaffold & Elm Architecture

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-001, T-003 (project structure and central data types must exist)
**Phase:** 5 - Interactive TUI

---

## Description

Set up the foundational Bubble Tea application structure following the Elm architecture (Model-Update-View). This creates the top-level TUI entry point, defines the root model with sub-models for each panel (file tree, stats, profile selector), establishes the message types for inter-component communication, and wires up the `--interactive` / `-i` flag on the CLI. The TUI is a presentation layer only -- it calls the same core pipeline library as the headless CLI.

## User Story

As a developer, I want `harvx -i` to launch an interactive terminal interface so that I can visually explore and select files for context generation without memorizing glob patterns.

## Acceptance Criteria

- [ ] `internal/tui/app.go` defines the root `Model` struct implementing `tea.Model` with `Init()`, `Update()`, and `View()` methods
- [ ] Root model contains sub-models: `fileTree`, `statsPanel`, `profileSelector`, `helpOverlay`
- [ ] `internal/tui/messages.go` defines all message types for TUI communication: `FileToggledMsg`, `TokenCountUpdatedMsg`, `ProfileChangedMsg`, `GenerateRequestedMsg`, `ErrorMsg`, `WindowSizeMsg`
- [ ] `internal/tui/app.go` includes a `New(cfg config.ResolvedConfig, pipeline *pipeline.Pipeline) Model` constructor that accepts the resolved config and pipeline reference
- [ ] The root `Update()` dispatches messages to appropriate sub-models and handles global keys (`q`/`Esc` to quit, `?` for help, `Enter` to generate, `Tab` for profile switch)
- [ ] The root `View()` composes sub-model views into a multi-panel layout (file tree on left, stats on right)
- [ ] `internal/cli/root.go` registers `--interactive` / `-i` flag
- [ ] When `-i` is passed, the CLI creates a `tea.Program` and runs the TUI instead of headless generation
- [ ] Smart default: when `harvx` is run with no arguments, no subcommand, and no `harvx.toml` in the directory tree, the TUI launches automatically
- [ ] `tea.Program` is created with `tea.WithAltScreen()` and `tea.WithMouseCellMotion()` options
- [ ] Ctrl+C and `q` cleanly exit the program with exit code 0
- [ ] Window resize events (`tea.WindowSizeMsg`) are handled and propagated to all sub-models
- [ ] Unit tests verify message routing, key handling, and constructor behavior

## Technical Notes

- Use `charmbracelet/bubbletea` v1.x (latest stable, v1.2+). Bubble Tea v2 is still in RC as of February 2026 and not recommended for production use yet. If v2 reaches stable during development, migration is straightforward (Init returns `(Model, Cmd)` instead of just `Cmd`, and import path changes).
- The Elm architecture pattern: `Init` sets up initial state and optional commands, `Update` handles messages and returns new state + commands, `View` renders state to a string.
- The TUI must NOT import or duplicate pipeline logic. It uses the same `pipeline.Pipeline` engine as headless mode. The TUI is purely a presentation layer.
- Use `tea.WithAltScreen()` to render in the alternate screen buffer (preserves terminal history on exit).
- Use `tea.WithMouseCellMotion()` to enable mouse wheel scrolling in the file tree.
- Reference: PRD Section 5.13, Section 6.2 (`internal/tui/`)

## Files to Create/Modify

- `internal/tui/app.go` - Root Bubble Tea application model, constructor, Init/Update/View
- `internal/tui/messages.go` - All TUI message types
- `internal/tui/keys.go` - Key binding definitions using `charmbracelet/bubbles/key`
- `internal/tui/app_test.go` - Unit tests for message routing and key handling
- `internal/cli/root.go` - Add `--interactive` / `-i` flag, smart default detection
- `internal/cli/interactive.go` - TUI launch logic (create Program, run, handle result)

## Testing Requirements

- Unit test: root model dispatches `FileToggledMsg` to file tree sub-model
- Unit test: root model dispatches `tea.WindowSizeMsg` to all sub-models
- Unit test: pressing `q` produces `tea.Quit` command
- Unit test: pressing `?` toggles help overlay visibility
- Unit test: pressing `Enter` produces `GenerateRequestedMsg`
- Unit test: constructor validates non-nil pipeline reference
- Unit test: smart default detection logic (no args + no harvx.toml = interactive)