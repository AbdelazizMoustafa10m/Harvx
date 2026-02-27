// Package cmd implements the CLI commands for gosync.
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/example/gosync/internal/config"
	"github.com/example/gosync/internal/handler"
)

var (
	cfgFile string
	verbose bool
	appVer  string
)

// SetVersion sets the application version string.
func SetVersion(v string) {
	appVer = v
}

var rootCmd = &cobra.Command{
	Use:   "gosync",
	Short: "A fast file synchronization tool",
	Long: `gosync synchronizes files between local directories with support
for watch mode, ignore patterns, and dry-run previews.`,
	Version: appVer,
}

var syncCmd = &cobra.Command{
	Use:   "sync [source] [destination]",
	Short: "Synchronize two directories",
	Args:  cobra.ExactArgs(2),
	RunE:  runSync,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	syncCmd.Flags().Bool("watch", false, "enable watch mode")
	syncCmd.Flags().Bool("dry-run", false, "preview changes without applying")
	syncCmd.Flags().StringSlice("ignore", nil, "patterns to ignore")

	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	src, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving source path: %w", err)
	}

	dst, err := filepath.Abs(args[1])
	if err != nil {
		return fmt.Errorf("resolving destination path: %w", err)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	watch, _ := cmd.Flags().GetBool("watch")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	h, err := handler.New(handler.Options{
		Source:      src,
		Destination: dst,
		Config:      cfg,
		Watch:       watch,
		DryRun:      dryRun,
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("creating sync handler: %w", err)
	}

	return h.Run(ctx)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}