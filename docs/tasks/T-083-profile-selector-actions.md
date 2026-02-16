# T-083: Profile Selector & Action Keybindings

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-079 (TUI scaffold), T-082 (stats panel for profile display)
**Phase:** 5 - Interactive TUI

---

## Description

Implement the profile selector component that allows quick-switching between available profiles using the Tab key, and wire up all action keybindings for the TUI: Enter to generate, `p` to preview, `s` to save selection as profile, `e` to export to clipboard, and `q`/Esc to quit. Each action integrates with the core pipeline or produces appropriate TUI state changes. This task also includes the profile display header showing the active profile name and key settings.

## User Story

As a developer with multiple Harvx profiles, I want to quickly switch between profiles in the TUI using Tab so that I can compare how different profiles affect my file selection and token budget without restarting.

## Acceptance Criteria

- [ ] `internal/tui/profile/model.go` defines `Model` implementing `tea.Model`
- [ ] Tab key cycles forward through available profiles, Shift+Tab cycles backward
- [ ] Profile list is loaded from the resolved config (global + local `harvx.toml`)
- [ ] Switching profiles triggers: (1) tree re-evaluation (tier assignments change), (2) stats recalculation (budget changes), (3) `ProfileChangedMsg` sent to root
- [ ] Active profile displayed in stats panel header: `Profile: finvault | Target: claude`
- [ ] Profile name styled with a distinct background color to stand out
- [ ] Action keybindings implemented in root model `Update()`:
  - `Enter`: Generate output with current selection. Shows a "Generating..." overlay, runs pipeline, then exits TUI with success message.
  - `p`: Preview -- shows a temporary overlay with output summary (file count, token count, tier breakdown) without generating file
  - `s`: Save as profile -- prompts for profile name via text input, serializes current selection to TOML, saves to `harvx.toml`
  - `e`: Export to clipboard -- copies the list of selected file paths to system clipboard
  - `q` / `Esc`: Quit without generating (exit code 0)
- [ ] During generate, a spinner overlay blocks input and shows progress
- [ ] Save-as-profile flow: text input component appears, user types name, Enter confirms, Esc cancels
- [ ] Clipboard export uses `atotto/clipboard` (cross-platform clipboard access)
- [ ] All actions show brief status messages (toast-style) that auto-dismiss after 2 seconds
- [ ] Unit tests for profile cycling, action dispatch, and save serialization

## Technical Notes

- Profile switching must recompute tier assignments for all loaded tree nodes. This means the relevance sorter from the core pipeline needs to be callable per-node.
- For clipboard support, use `github.com/atotto/clipboard` which provides cross-platform clipboard access (macOS pbcopy, Linux xclip/xsel, Windows clip). If clipboard is unavailable, show an error message.
- Save-to-profile serialization: collect all included file paths, compute the minimal set of include/exclude glob patterns that reproduce the selection, and write them to a new `[profile.<name>]` section in `harvx.toml`.
- For the text input (save-as-profile name), use `charmbracelet/bubbles/textinput`.
- Generate action should run the pipeline in a `tea.Cmd` (goroutine) so the UI remains responsive during generation.
- Toast messages: implement as a timed message that appears in the status bar area, dismissed by a `tea.Tick` after 2 seconds.
- Reference: PRD Section 5.13 (Profile selector, Actions)

## Files to Create/Modify

- `internal/tui/profile/model.go` - Profile selector model with cycling logic
- `internal/tui/profile/view.go` - Profile display rendering
- `internal/tui/actions.go` - Action handlers (generate, preview, save, export, quit)
- `internal/tui/overlay.go` - Overlay components (spinner, text input, preview summary)
- `internal/tui/toast.go` - Toast message component with auto-dismiss
- `internal/tui/serialize.go` - TUI selection to TOML serialization
- `internal/tui/app.go` - Updated root model to wire actions and profile selector
- `internal/tui/profile/model_test.go` - Profile cycling tests
- `internal/tui/actions_test.go` - Action handler tests
- `internal/tui/serialize_test.go` - Serialization tests

## Testing Requirements

- Unit test: Tab cycles through profiles in order, wraps around at end
- Unit test: Shift+Tab cycles backward through profiles
- Unit test: switching profile sends `ProfileChangedMsg`
- Unit test: Enter action produces pipeline execution command
- Unit test: `p` action produces preview overlay display
- Unit test: `s` action activates text input overlay
- Unit test: text input Enter with valid name triggers TOML serialization
- Unit test: text input Esc cancels save flow
- Unit test: `e` action calls clipboard write with selected paths
- Unit test: `q` produces `tea.Quit`
- Unit test: toast message auto-dismisses after tick
- Unit test: serialization produces valid TOML with include patterns