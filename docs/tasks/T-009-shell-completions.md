# T-009: Shell Completions (harvx completion)

**Priority:** Should Have
**Effort:** Small (2-4hrs)
**Dependencies:** T-005
**Phase:** 1 - Foundation

---

## Description

Implement the `harvx completion` subcommand that generates shell completion scripts for Bash, Zsh, Fish, and PowerShell. Also register `ValidArgsFunction` and flag completion functions for flags with fixed valid values (`--format`, `--target`) to enable intelligent tab completion.

## User Story

As a developer who uses Harvx daily, I want tab completion for commands and flags so that I can type faster and discover available options without consulting the docs.

## Acceptance Criteria

- [ ] `harvx completion bash` outputs a valid Bash completion script to stdout
- [ ] `harvx completion zsh` outputs a valid Zsh completion script to stdout
- [ ] `harvx completion fish` outputs a valid Fish completion script to stdout
- [ ] `harvx completion powershell` outputs a valid PowerShell completion script to stdout
- [ ] Running `harvx completion` with no argument shows usage help explaining how to install completions for each shell
- [ ] The help text includes installation instructions for each shell:
  - Bash: `harvx completion bash > /etc/bash_completion.d/harvx` or `source <(harvx completion bash)`
  - Zsh: `harvx completion zsh > "${fpath[1]}/_harvx"`
  - Fish: `harvx completion fish > ~/.config/fish/completions/harvx.fish`
  - PowerShell: `harvx completion powershell | Out-String | Invoke-Expression`
- [ ] `--format` flag has completion for: `markdown`, `xml`
- [ ] `--target` flag has completion for: `claude`, `chatgpt`, `generic`
- [ ] Subcommand names complete correctly (e.g., `harvx ge<TAB>` completes to `harvx generate`)
- [ ] Unit tests verify that completion scripts are generated without error for all four shells

## Technical Notes

- Cobra has built-in completion generation. The standard pattern:
  ```go
  var completionCmd = &cobra.Command{
      Use:   "completion [bash|zsh|fish|powershell]",
      Short: "Generate shell completion scripts",
      Long:  `Generate shell completion scripts for Harvx. ...`,
      ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
      Args:  cobra.ExactArgs(1),
      RunE: func(cmd *cobra.Command, args []string) error {
          switch args[0] {
          case "bash":
              return rootCmd.GenBashCompletionV2(os.Stdout, true)
          case "zsh":
              return rootCmd.GenZshCompletion(os.Stdout)
          case "fish":
              return rootCmd.GenFishCompletion(os.Stdout, true)
          case "powershell":
              return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
          }
          return fmt.Errorf("unsupported shell: %s", args[0])
      },
  }
  ```
- For flag value completion, use `cmd.RegisterFlagCompletionFunc`:
  ```go
  rootCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
      return []string{"markdown", "xml"}, cobra.ShellCompDirectiveNoFileComp
  })
  ```
- Reference: Cobra shell completion docs at https://cobra.dev/docs/how-to-guides/shell-completion/
- Use `GenBashCompletionV2` (not the deprecated `GenBashCompletion`) for modern Bash completion with descriptions.
- Per PRD Section 5.9: "Intelligent completions: `harvx --profile <TAB>` lists available profiles, `--format <TAB>` lists formats, `--target <TAB>` lists presets." Profile completion will be added when the profile system is implemented.
- Reference: PRD Section 5.9

## Files to Create/Modify

- `internal/cli/completion.go` - Completion subcommand
- `internal/cli/completion_test.go` - Unit tests
- `internal/cli/root.go` - Register completion subcommand, register flag completion functions

## Testing Requirements

- Unit test: `harvx completion bash` generates non-empty output without error
- Unit test: `harvx completion zsh` generates non-empty output without error
- Unit test: `harvx completion fish` generates non-empty output without error
- Unit test: `harvx completion powershell` generates non-empty output without error
- Unit test: `harvx completion invalid` returns an error
- Unit test: `harvx completion` with no args returns an error (exact args = 1)
- Unit test: `--format` flag completion returns `["markdown", "xml"]`
- Unit test: `--target` flag completion returns `["claude", "chatgpt", "generic"]`
