# T-090: Shell Completion Generation & Man Page Generation

**Priority:** Must Have
**Effort:** Medium (6-8hrs)
**Dependencies:** T-001 (Cobra CLI framework set up with all subcommands registered)
**Phase:** 6 - Polish & Distribution

---

## Description

Implement shell completion generation for bash, zsh, fish, and PowerShell via the `harvx completion <shell>` subcommand, including intelligent completions for dynamic values (profile names, format options, target presets). Also implement man page generation using Cobra's `doc` package, producing installable man pages that cover all commands and flags.

## User Story

As a developer who uses Harvx frequently, I want shell tab-completion for commands, flags, and profile names so that I can work faster and discover features without consulting documentation.

## Acceptance Criteria

- [ ] `harvx completion bash` outputs valid bash completion script
- [ ] `harvx completion zsh` outputs valid zsh completion script
- [ ] `harvx completion fish` outputs valid fish completion script
- [ ] `harvx completion powershell` outputs valid PowerShell completion script
- [ ] Each completion subcommand includes usage instructions in its `Long` description (how to install for that shell)
- [ ] Intelligent completions for dynamic values:
  - `harvx --profile <TAB>` lists available profile names from loaded config
  - `harvx --format <TAB>` lists `markdown`, `xml`
  - `harvx --target <TAB>` lists `claude`, `chatgpt`, `generic`
  - `harvx --tokenizer <TAB>` lists `cl100k_base`, `o200k_base`, `none`
  - `harvx profiles show <TAB>` lists available profile names
  - `harvx completion <TAB>` lists `bash`, `zsh`, `fish`, `powershell`
- [ ] Dynamic profile completion uses `ValidArgsFunction` that reads `harvx.toml` at completion time
- [ ] Man pages generated for all commands: `harvx(1)`, `harvx-generate(1)`, `harvx-brief(1)`, `harvx-preview(1)`, `harvx-profiles(1)`, etc.
- [ ] Man pages include: name, synopsis, description, options, examples, see also
- [ ] Man page generation command: `harvx docs man --output-dir ./man/` (hidden command for build process)
- [ ] Man pages are included in GoReleaser archives (from T-088)
- [ ] Makefile targets: `make completions` (generate all 4 scripts), `make man` (generate man pages)
- [ ] Generated completion scripts are stored in `completions/` directory for inclusion in archives

## Technical Notes

- Cobra has built-in support for shell completions via `GenBashCompletionV2`, `GenZshCompletion`, `GenFishCompletion`, `GenPowerShellCompletion`. Use V2 for bash (better completions).
- For intelligent completions, use `cobra.ValidArgsFunction` on commands and `cmd.RegisterFlagCompletionFunc` for flags. These are portable across all shells.
- Profile name completion function:
  ```go
  func profileCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
      // Load config, extract profile names
      cfg, err := config.LoadConfig()
      if err != nil {
          return nil, cobra.ShellCompDirectiveNoFileComp
      }
      return cfg.ProfileNames(), cobra.ShellCompDirectiveNoFileComp
  }
  ```
- Man page generation uses `github.com/spf13/cobra/doc` package:
  ```go
  header := &doc.GenManHeader{Title: "HARVX", Section: "1"}
  doc.GenManTree(rootCmd, header, "./man/")
  ```
- Man pages should be generated as part of the build process (Makefile target) and included in release archives.
- Installation instructions per shell should be included in each completion command's help text.
- Reference: PRD Section 5.9 (shell completions), Cobra completion docs (https://cobra.dev/docs/how-to-guides/shell-completion/)

## Files to Create/Modify

- `internal/cli/completion.go` - Completion subcommand with bash/zsh/fish/powershell
- `internal/cli/docs.go` - Hidden `docs man` subcommand for man page generation
- `internal/cli/completion_funcs.go` - Dynamic completion functions (profiles, formats, targets)
- `internal/cli/root.go` - Register flag completion functions (modify)
- `Makefile` - Add `completions` and `man` targets (modify)
- `.goreleaser.yaml` - Include completions and man pages in archives (modify)
- `completions/.gitkeep` - Directory for generated completion scripts
- `man/.gitkeep` - Directory for generated man pages

## Testing Requirements

- Unit test: bash completion output contains expected command names
- Unit test: profile completion function returns profiles from test config
- Unit test: format flag completion returns exactly `markdown`, `xml`
- Unit test: target flag completion returns exactly `claude`, `chatgpt`, `generic`
- Unit test: man page generation produces files for all registered commands
- Integration test: generated bash completion script is valid (`bash -n` check)
- Integration test: generated zsh completion script is valid
- Verify man pages render correctly with `man -l ./man/harvx.1`