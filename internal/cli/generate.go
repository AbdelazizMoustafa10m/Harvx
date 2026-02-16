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
}

func runGenerate(cmd *cobra.Command, args []string) error {
	return pipeline.Run(cmd.Context(), flagValues)
}
