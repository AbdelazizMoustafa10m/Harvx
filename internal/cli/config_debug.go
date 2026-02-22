// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
package cli

import (
	"fmt"

	"github.com/harvx/harvx/internal/config"
	"github.com/spf13/cobra"
)

// configCmd is the parent command for configuration-related subcommands.
// Running `harvx config` with no subcommand prints the help text.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long: `Configuration management commands for harvx.

Use these subcommands to inspect and debug your harvx configuration:

  debug  Show the fully resolved configuration with per-field source annotations`,
	// No RunE: default Cobra behaviour will print help when no subcommand is given.
}

// configDebugCmd shows the fully resolved configuration with source annotations.
var configDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Show resolved configuration with source annotations",
	Long: `Displays the complete resolved configuration showing exactly which source
(built-in default, global config, repo config, environment variable, or CLI flag)
provided each value. Useful for diagnosing unexpected configuration behavior.`,
	RunE: runConfigDebug,
}

func init() {
	// Register flags on configDebugCmd.
	configDebugCmd.Flags().Bool("json", false, "output as structured JSON")
	configDebugCmd.Flags().String("profile", "", "profile name to debug (default: active profile)")

	// Assemble hierarchy.
	configCmd.AddCommand(configDebugCmd)
	rootCmd.AddCommand(configCmd)
}

// runConfigDebug implements `harvx config debug`.
func runConfigDebug(cmd *cobra.Command, _ []string) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	profileName, _ := cmd.Flags().GetString("profile")

	out := cmd.OutOrStdout()

	result, err := config.BuildDebugOutput(config.DebugOptions{
		ProfileName: profileName,
		TargetDir:   ".",
	})
	if err != nil {
		return fmt.Errorf("building debug output: %w", err)
	}

	if asJSON {
		if err := config.FormatDebugOutputJSON(result, out); err != nil {
			return fmt.Errorf("formatting debug output as JSON: %w", err)
		}
		return nil
	}

	if err := config.FormatDebugOutput(result, out); err != nil {
		return fmt.Errorf("formatting debug output: %w", err)
	}
	return nil
}
