package cli

import (
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate LLM-optimized context from a codebase",
	Long: `Recursively discover files, apply filters, and produce a structured
context document optimized for large language models.

This is the primary workflow command. Running 'harvx' with no subcommand
is equivalent to running 'harvx generate'.`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().Bool("preview", false, "show file tree and token estimate without writing output")
	rootCmd.AddCommand(generateCmd)

	// Register completion for inherited persistent flags on the generate command.
	generateCmd.RegisterFlagCompletionFunc("tokenizer", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"cl100k_base", "o200k_base", "none"}, cobra.ShellCompDirectiveNoFileComp
	})
	generateCmd.RegisterFlagCompletionFunc("truncation-strategy", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"truncate", "skip"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func runGenerate(cmd *cobra.Command, args []string) error {
	return pipeline.Run(cmd.Context(), flagValues)
}
