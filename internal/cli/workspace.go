// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file implements the `harvx workspace` subcommand which renders workspace
// manifests into structured output for multi-repo context.
package cli

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/tokenizer"
	"github.com/harvx/harvx/internal/workflows"
)

// workspaceJSON is a local flag target for --json on the workspace command.
var workspaceJSON bool

// workspaceDeep is a local flag target for --deep on the workspace command.
var workspaceDeep bool

// workspaceCmd implements `harvx workspace` which renders workspace manifests.
var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Render workspace manifest for multi-repo context",
	Long: `Render a .harvx/workspace.toml manifest into structured output describing
related repositories and their relationships.

The workspace manifest is auto-detected by searching for .harvx/workspace.toml
in the current directory and parent directories.

Examples:
  # Render workspace to stdout
  harvx workspace --stdout

  # Deep mode with directory trees
  harvx workspace --deep --stdout

  # Machine-readable JSON metadata
  harvx workspace --json

  # XML output for Claude
  harvx workspace --target claude --stdout

  # Save to file
  harvx workspace -o workspace-context.md

  # Initialize a new workspace config
  harvx workspace init`,
	RunE: runWorkspace,
}

// workspaceInitCmd implements `harvx workspace init`.
var workspaceInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter workspace.toml configuration",
	Long: `Generate a starter .harvx/workspace.toml configuration file with
placeholder entries that you can edit to describe your workspace.`,
	RunE: runWorkspaceInit,
}

func init() {
	workspaceCmd.Flags().BoolVar(&workspaceJSON, "json", false, "Output machine-readable JSON metadata to stdout")
	workspaceCmd.Flags().BoolVar(&workspaceDeep, "deep", false, "Include expanded details: directory trees per repo")
	workspaceCmd.AddCommand(workspaceInitCmd)
	rootCmd.AddCommand(workspaceCmd)
}

// runWorkspace executes the workspace subcommand.
func runWorkspace(cmd *cobra.Command, args []string) error {
	fv := GlobalFlags()

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("workspace: resolving directory: %w", err)
	}

	// Auto-detect workspace config.
	configPath, err := config.DiscoverWorkspaceConfig(rootDir)
	if err != nil {
		return fmt.Errorf("workspace: discovering config: %w", err)
	}

	if configPath == "" {
		return fmt.Errorf("workspace: no .harvx/workspace.toml found\n\nTo create one, run:\n  harvx workspace init")
	}

	// Load and parse the config.
	cfg, err := config.LoadWorkspaceConfig(configPath)
	if err != nil {
		return fmt.Errorf("workspace: %w", err)
	}

	// The configDir is .harvx/, but we want the parent (repo root) for path resolution.
	configDir := filepath.Dir(filepath.Dir(configPath))

	// Validate and collect warnings.
	warnings := config.ValidateWorkspace(cfg)
	for _, w := range warnings {
		slog.Warn("workspace validation", "warning", w)
	}

	// Build token counter.
	tokenCount, err := buildWorkspaceTokenCounter(fv.Tokenizer)
	if err != nil {
		slog.Warn("workspace: could not create tokenizer, using estimator",
			"tokenizer", fv.Tokenizer,
			"error", err,
		)
		tokenCount = nil
	}

	opts := workflows.WorkspaceOptions{
		Config:       cfg,
		ConfigDir:    configDir,
		Deep:         workspaceDeep,
		Target:       fv.Target,
		TokenCounter: tokenCount,
	}

	result, err := workflows.GenerateWorkspace(opts)
	if err != nil {
		return fmt.Errorf("workspace: %w", err)
	}

	// Handle --json output.
	if workspaceJSON {
		return writeWorkspaceJSON(cmd, cfg, result, warnings)
	}

	// Route output.
	return writeWorkspaceOutput(cmd, fv, result)
}

// runWorkspaceInit generates a starter workspace.toml configuration.
func runWorkspaceInit(cmd *cobra.Command, args []string) error {
	fv := GlobalFlags()

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("workspace init: resolving directory: %w", err)
	}

	// Create .harvx directory if it doesn't exist.
	harvxDir := filepath.Join(rootDir, ".harvx")
	if err := os.MkdirAll(harvxDir, 0o755); err != nil {
		return fmt.Errorf("workspace init: creating .harvx directory: %w", err)
	}

	outputPath := filepath.Join(harvxDir, "workspace.toml")

	// Check if file already exists.
	if _, statErr := os.Stat(outputPath); statErr == nil {
		if !fv.Yes {
			return fmt.Errorf("workspace init: %s already exists (use --yes to overwrite)", outputPath)
		}
	}

	content := config.GenerateWorkspaceInit()

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("workspace init: writing config: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Created %s\nEdit the file to describe your workspace repositories.\n", outputPath)
	return nil
}

// buildWorkspaceTokenCounter creates a token counter function from the configured
// tokenizer name. Returns nil if the tokenizer cannot be created.
func buildWorkspaceTokenCounter(name string) (func(string) int, error) {
	if name == "none" || name == "" {
		return nil, nil
	}

	tok, err := tokenizer.NewTokenizer(name)
	if err != nil {
		return nil, fmt.Errorf("creating tokenizer %q: %w", name, err)
	}

	return tok.Count, nil
}

// writeWorkspaceJSON writes workspace metadata as JSON to stdout.
func writeWorkspaceJSON(cmd *cobra.Command, cfg *config.WorkspaceConfig, result *workflows.WorkspaceResult, warnings []string) error {
	repos := make([]string, 0, len(cfg.Workspace.Repos))
	for _, r := range cfg.Workspace.Repos {
		repos = append(repos, r.Name)
	}

	// Sort for determinism (repos may not be sorted in the original config).
	sort.Strings(repos)

	meta := workflows.WorkspaceJSON{
		Name:        cfg.Workspace.Name,
		Description: cfg.Workspace.Description,
		RepoCount:   result.RepoCount,
		TokenCount:  result.TokenCount,
		ContentHash: result.FormattedHash,
		Repos:       repos,
		Warnings:    warnings,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("workspace: marshaling JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// writeWorkspaceOutput routes workspace content to stdout or a file.
func writeWorkspaceOutput(cmd *cobra.Command, fv *config.FlagValues, result *workflows.WorkspaceResult) error {
	content := result.Content

	if fv.Stdout {
		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// Determine output path.
	outputPath := fv.Output
	if outputPath == config.DefaultOutput {
		outputPath = "harvx-workspace.md"
	}

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("workspace: writing output to %s: %w", outputPath, err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Workspace written to %s (%d tokens, %d repos, hash: %s)\n",
		outputPath, result.TokenCount, result.RepoCount, result.FormattedHash)
	return nil
}