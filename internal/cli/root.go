// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// The root command defined here is the entry point for all subcommands and
// handles cross-cutting concerns like logging initialization and error handling.
package cli

import (
	"log/slog"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "harvx",
	Short: "Harvest your context.",
	Long: `Harvx packages codebases into LLM-optimized context documents.

It walks your repository, applies intelligent filtering, relevance sorting,
secret redaction, and optional tree-sitter compression to produce a single
context file optimized for large language models like Claude, ChatGPT, and others.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		quiet, _ := cmd.Flags().GetBool("quiet")

		level := config.ResolveLogLevel(verbose, quiet)
		format := config.ResolveLogFormat()
		config.SetupLogging(level, format)

		slog.Debug("logging initialized", "level", level, "format", format)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress all output except errors")
}

// Execute runs the root command and returns an appropriate exit code.
// It returns pipeline.ExitSuccess (0) on success, pipeline.ExitError (1) on failure.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		return int(pipeline.ExitError)
	}
	return int(pipeline.ExitSuccess)
}

// RootCmd returns the root cobra.Command for use in testing and subcommand registration.
func RootCmd() *cobra.Command {
	return rootCmd
}
