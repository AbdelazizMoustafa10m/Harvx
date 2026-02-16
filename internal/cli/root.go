// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// The root command defined here is the entry point for all subcommands and
// handles cross-cutting concerns like logging initialization and error handling.
package cli

import (
	"errors"
	"log/slog"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/spf13/cobra"
)

// flagValues holds the parsed global flag values, populated by config.BindFlags
// during command initialization and validated in PersistentPreRunE.
var flagValues *config.FlagValues

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
		// Validate all global flags.
		if err := config.ValidateFlags(flagValues, cmd); err != nil {
			return err
		}

		// Initialize logging with validated flag values.
		level := config.ResolveLogLevel(flagValues.Verbose, flagValues.Quiet)
		format := config.ResolveLogFormat()
		config.SetupLogging(level, format)

		slog.Debug("logging initialized", "level", level, "format", format)
		return nil
	},
	// When no subcommand is given, delegate to the generate command.
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGenerate(cmd, args)
	},
}

func init() {
	flagValues = config.BindFlags(rootCmd)

	// Register flag completion functions for flags with fixed valid values.
	// These enable intelligent tab completion (e.g., --format <TAB>).
	rootCmd.RegisterFlagCompletionFunc("format", completeFormat)
	rootCmd.RegisterFlagCompletionFunc("target", completeTarget)
}

// completeFormat returns the valid values for the --format flag.
func completeFormat(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"markdown", "xml"}, cobra.ShellCompDirectiveNoFileComp
}

// completeTarget returns the valid values for the --target flag.
func completeTarget(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"claude", "chatgpt", "generic"}, cobra.ShellCompDirectiveNoFileComp
}

// Execute runs the root command and returns an appropriate exit code.
// If the error is a *pipeline.HarvxError, its Code is used.
// Generic errors return ExitError (1). Nil returns ExitSuccess (0).
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		return extractExitCode(err)
	}
	return int(pipeline.ExitSuccess)
}

// extractExitCode determines the process exit code from an error.
// If the error is a *pipeline.HarvxError, its Code field is used.
// Otherwise, ExitError (1) is returned for any non-nil error.
func extractExitCode(err error) int {
	if err == nil {
		return int(pipeline.ExitSuccess)
	}
	var harvxErr *pipeline.HarvxError
	if errors.As(err, &harvxErr) {
		return harvxErr.Code
	}
	return int(pipeline.ExitError)
}

// RootCmd returns the root cobra.Command for use in testing and subcommand registration.
func RootCmd() *cobra.Command {
	return rootCmd
}

// GlobalFlags returns the parsed global flag values. This is available after
// PersistentPreRunE has run. Subcommands use this to access shared configuration.
func GlobalFlags() *config.FlagValues {
	return flagValues
}
