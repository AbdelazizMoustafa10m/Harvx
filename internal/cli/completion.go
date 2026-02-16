// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// completionCmd generates shell completion scripts for Bash, Zsh, Fish, and
// PowerShell. When run without arguments, it displays installation instructions
// for each supported shell.
var completionCmd = &cobra.Command{
	Use:       "completion [bash|zsh|fish|powershell]",
	Short:     "Generate shell completion scripts",
	Long:      completionLongHelp,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	RunE:      runCompletion,
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

// completionLongHelp contains installation instructions for each shell.
const completionLongHelp = `Generate shell completion scripts for harvx.

To load completions:

Bash:
  # Load completions in the current shell session:
  $ source <(harvx completion bash)

  # Load completions for every new session (Linux):
  $ harvx completion bash > /etc/bash_completion.d/harvx

  # Load completions for every new session (macOS):
  $ harvx completion bash > $(brew --prefix)/etc/bash_completion.d/harvx

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Load completions for every new session:
  $ harvx completion zsh > "${fpath[1]}/_harvx"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ harvx completion fish > ~/.config/fish/completions/harvx.fish

PowerShell:
  # Load completions in the current shell session:
  PS> harvx completion powershell | Out-String | Invoke-Expression

  # Load completions for every new session:
  PS> harvx completion powershell >> $PROFILE
`

// runCompletion generates a shell completion script for the specified shell.
// If no shell argument is provided, it prints the help text with installation
// instructions and returns nil.
func runCompletion(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	out := cmd.OutOrStdout()

	switch args[0] {
	case "bash":
		return cmd.Root().GenBashCompletionV2(out, true)
	case "zsh":
		return cmd.Root().GenZshCompletion(out)
	case "fish":
		return cmd.Root().GenFishCompletion(out, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(out)
	default:
		return fmt.Errorf("unsupported shell: %s", args[0])
	}
}
