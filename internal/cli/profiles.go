package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/harvx/harvx/internal/config"
	"github.com/spf13/cobra"
)

// profilesCmd is the parent command for all profile management subcommands.
// Running `harvx profiles` with no subcommand prints the help text.
var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage harvx configuration profiles",
	Long: `Profile management commands for harvx.

Use these subcommands to discover, create, and inspect your harvx configuration:

  list   Show all available profiles from all config sources
  init   Generate a starter harvx.toml in the current directory
  show   Display the fully resolved configuration for a named profile`,
	// No RunE: default Cobra behaviour will print help when no subcommand is given.
}

// profilesListCmd lists all profiles available in the current context.
var profilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available profiles from all config sources",
	Long: `List all profiles that harvx can resolve in the current directory.

Profiles are loaded from three sources (in precedence order):
  1. Built-in defaults
  2. Global config (~/.config/harvx/config.toml)
  3. Repository config (harvx.toml in or above the current directory)

The output shows each profile name, its source, any parent it extends, and a
brief description. Available starter templates are listed below the table.`,
	RunE: runProfilesList,
}

// profilesInitCmd generates a starter harvx.toml file.
var profilesInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter harvx.toml in the current directory",
	Long: `Generate a harvx.toml configuration file from a named template.

Without --template, the minimal "base" template is used. Specify --template
to use a framework-specific starter that includes sensible tier patterns and
ignore rules for that ecosystem.

If harvx.toml already exists in the current directory, the command returns an
error unless --yes is passed to allow overwriting.`,
	RunE: runProfilesInit,
}

// profilesShowCmd resolves and displays a named profile.
var profilesShowCmd = &cobra.Command{
	Use:               "show [profile]",
	Short:             "Show the resolved configuration for a profile",
	Long:              `Resolve the named profile (following its inheritance chain) and print the fully merged configuration.

The output is annotated TOML where each field carries an inline comment
indicating which configuration layer (default/global/repo/env/flag) provided
its value. Use --json to get machine-readable JSON output instead.

If no profile name is given, the active default profile is shown.`,
	Args:              cobra.MaximumNArgs(1),
	RunE:              runProfilesShow,
	ValidArgsFunction: completeProfileNames,
}

func init() {
	// Register flags on profilesInitCmd.
	profilesInitCmd.Flags().String("template", "base", "template name (run `harvx profiles list` to see options)")
	profilesInitCmd.Flags().StringP("output", "o", "harvx.toml", "path to write the generated config file")
	profilesInitCmd.Flags().Bool("yes", false, "overwrite existing harvx.toml without prompting")

	// Register flags on profilesShowCmd.
	profilesShowCmd.Flags().Bool("json", false, "output the resolved profile as JSON instead of TOML")

	// Register completions on profilesInitCmd --template flag.
	profilesInitCmd.RegisterFlagCompletionFunc("template", completeTemplateNames)

	// Assemble hierarchy.
	profilesCmd.AddCommand(profilesListCmd)
	profilesCmd.AddCommand(profilesInitCmd)
	profilesCmd.AddCommand(profilesShowCmd)

	rootCmd.AddCommand(profilesCmd)
}

// ── profiles list ──────────────────────────────────────────────────────────

// profileEntry is a row in the profiles list table.
type profileEntry struct {
	name        string
	source      string
	extends     string
	description string
}

// runProfilesList implements `harvx profiles list`.
func runProfilesList(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	entries, err := collectProfileEntries()
	if err != nil {
		return fmt.Errorf("collecting profiles: %w", err)
	}

	fmt.Fprintln(out, "Available Profiles:")
	fmt.Fprintln(out)

	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tSOURCE\tEXTENDS\tDESCRIPTION")
	for _, e := range entries {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", e.name, e.source, e.extends, e.description)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flushing table: %w", err)
	}

	// Templates section.
	fmt.Fprintln(out)
	templates := config.ListTemplates()
	names := make([]string, len(templates))
	for i, t := range templates {
		names[i] = t.Name
	}
	fmt.Fprintf(out, "Templates (use with `harvx profiles init --template <name>`):\n")
	fmt.Fprintf(out, "  %s\n", strings.Join(names, ", "))

	return nil
}

// collectProfileEntries builds the list of profile rows by querying all config
// sources. Missing or unreadable config files are silently skipped.
func collectProfileEntries() ([]profileEntry, error) {
	var entries []profileEntry

	// Built-in "default" is always present.
	entries = append(entries, profileEntry{
		name:        "default",
		source:      "built-in",
		extends:     "-",
		description: "Built-in defaults for any repository",
	})

	// Collect profiles from both global and repo config files.
	allProfiles, err := loadAllConfigProfiles()
	if err != nil {
		return nil, err
	}

	// Append profiles found in config files; skip "default" since it is
	// already listed as built-in above.
	for _, ep := range allProfiles {
		if ep.name == "default" {
			continue
		}
		entries = append(entries, ep)
	}

	return entries, nil
}

// loadAllConfigProfiles loads profiles from global and repo config files. It
// returns deduplicated entries annotated with their source. Missing files are
// silently ignored.
func loadAllConfigProfiles() ([]profileEntry, error) {
	var entries []profileEntry
	seen := make(map[string]bool)

	// Helper to process a config file.
	addFromFile := func(path, sourceLabel string) {
		cfg, err := config.LoadFromFile(path)
		if err != nil {
			slog.Debug("skipping config file", "path", path, "err", err)
			return
		}
		for name, p := range cfg.Profile {
			if seen[name] {
				continue
			}
			seen[name] = true

			extends := "-"
			if p.Extends != nil && *p.Extends != "" {
				extends = *p.Extends
			}

			// Build a short description from the config source.
			description := fmt.Sprintf("Loaded from %s", displayPath(path))

			entries = append(entries, profileEntry{
				name:        name,
				source:      sourceLabel,
				extends:     extends,
				description: description,
			})
		}
	}

	// Global config.
	globalPath, err := config.DiscoverGlobalConfig()
	if err != nil {
		slog.Debug("global config discovery failed", "err", err)
	} else if globalPath != "" {
		addFromFile(globalPath, "global")
	}

	// Repo config.
	repoPath, err := config.DiscoverRepoConfig(".")
	if err != nil {
		slog.Debug("repo config discovery failed", "err", err)
	} else if repoPath != "" {
		addFromFile(repoPath, "repo")
	}

	return entries, nil
}

// displayPath converts an absolute path to a shorter relative or tilde-prefixed
// path for display purposes.
func displayPath(path string) string {
	rel, err := filepath.Rel(".", path)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return "./" + rel
	}
	// Try ~/ prefix for paths under HOME.
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// ── profiles init ──────────────────────────────────────────────────────────

// runProfilesInit implements `harvx profiles init`.
func runProfilesInit(cmd *cobra.Command, _ []string) error {
	templateName, _ := cmd.Flags().GetString("template")
	outputPath, _ := cmd.Flags().GetString("output")
	yes, _ := cmd.Flags().GetBool("yes")

	// Also check the global --yes flag.
	if !yes {
		yes = flagValues != nil && flagValues.Yes
	}

	// Derive project name from the current working directory.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}
	projectName := filepath.Base(cwd)

	// Check if the output file already exists.
	if _, statErr := os.Stat(outputPath); statErr == nil {
		if !yes {
			return fmt.Errorf("%s already exists; use --yes to overwrite", outputPath)
		}
		slog.Debug("overwriting existing config file", "path", outputPath)
	}

	// Render the template.
	content, err := config.RenderTemplate(templateName, projectName)
	if err != nil {
		return fmt.Errorf("loading template %q: %w", templateName, err)
	}

	// Write the file.
	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Created %s (template: %s)\n", outputPath, templateName)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Next steps:")
	fmt.Fprintln(out, "  1. Review and customize the profile settings")
	fmt.Fprintln(out, "  2. Run `harvx profiles lint` to validate")
	fmt.Fprintln(out, "  3. Run `harvx preview` to see what would be included")

	return nil
}

// ── profiles show ──────────────────────────────────────────────────────────

// runProfilesShow implements `harvx profiles show [profile]`.
func runProfilesShow(cmd *cobra.Command, args []string) error {
	asJSON, _ := cmd.Flags().GetBool("json")

	// Determine which profile to show.
	profileName := "default"
	if len(args) > 0 {
		profileName = args[0]
	}

	// Resolve the profile through the full 5-layer pipeline.
	resolved, err := config.Resolve(config.ResolveOptions{
		ProfileName: profileName,
		TargetDir:   ".",
	})
	if err != nil {
		// If the profile is not found, provide a helpful list of available names.
		available, listErr := availableProfileNames()
		if listErr == nil && len(available) > 0 {
			return fmt.Errorf("%w\n\nAvailable profiles: %s", err, strings.Join(available, ", "))
		}
		return err
	}

	// Compute the inheritance chain for display.
	chain, err := resolveChainForShow(profileName)
	if err != nil {
		// Non-fatal: fall back to just the requested profile name.
		slog.Debug("chain resolution for show failed", "err", err)
		chain = []string{profileName}
	}

	out := cmd.OutOrStdout()

	if asJSON {
		jsonStr, err := config.ShowProfileJSON(resolved.Profile)
		if err != nil {
			return fmt.Errorf("serializing profile to JSON: %w", err)
		}
		fmt.Fprintln(out, jsonStr)
		return nil
	}

	tomlStr := config.ShowProfile(config.ShowOptions{
		Profile:     resolved.Profile,
		Sources:     resolved.Sources,
		ProfileName: profileName,
		Chain:       chain,
	})
	fmt.Fprint(out, tomlStr)
	return nil
}

// resolveChainForShow loads the repo/global config files and calls
// ResolveProfile to get the inheritance chain. Returns just [profileName] on
// any error so that the show command can still function without it.
func resolveChainForShow(profileName string) ([]string, error) {
	profiles := make(map[string]*config.Profile)

	// Load repo config profiles.
	repoPath, err := config.DiscoverRepoConfig(".")
	if err == nil && repoPath != "" {
		cfg, loadErr := config.LoadFromFile(repoPath)
		if loadErr == nil {
			for name, p := range cfg.Profile {
				profiles[name] = p
			}
		}
	}

	// Load global config profiles (do not overwrite repo values).
	globalPath, err := config.DiscoverGlobalConfig()
	if err == nil && globalPath != "" {
		cfg, loadErr := config.LoadFromFile(globalPath)
		if loadErr == nil {
			for name, p := range cfg.Profile {
				if _, exists := profiles[name]; !exists {
					profiles[name] = p
				}
			}
		}
	}

	res, err := config.ResolveProfile(profileName, profiles)
	if err != nil {
		return nil, err
	}
	return res.Chain, nil
}

// availableProfileNames returns the names of all profiles from all config
// sources for use in error messages.
func availableProfileNames() ([]string, error) {
	entries, err := collectProfileEntries()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}
	return names, nil
}

// ── shell completions ──────────────────────────────────────────────────────

// completeProfileNames is a Cobra ValidArgsFunction that returns all known
// profile names from both config sources. Errors are silently swallowed so
// completions degrade gracefully.
func completeProfileNames(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	names, err := availableProfileNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Filter by the prefix already typed.
	var filtered []string
	for _, n := range names {
		if strings.HasPrefix(n, toComplete) {
			filtered = append(filtered, n)
		}
	}
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// completeTemplateNames returns the list of available template names for
// `--template` flag completion.
func completeTemplateNames(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	templates := config.ListTemplates()
	var names []string
	for _, t := range templates {
		if strings.HasPrefix(t.Name, toComplete) {
			names = append(names, t.Name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
