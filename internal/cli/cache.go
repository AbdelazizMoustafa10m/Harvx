package cli

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/diff"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage Harvx state cache",
	Long: `Inspect and manage Harvx's cached state snapshots.

State snapshots are stored in .harvx/state/ and track file hashes, sizes,
and git metadata from previous 'harvx generate' runs. Use 'cache show' to
inspect cached state and 'cache clear' to remove it.`,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear cached state",
	Long: `Remove cached state snapshots from .harvx/state/.

By default, clears all cached state for all profiles. Use --profile to
clear state for a specific profile only.`,
	Example: `  # Clear all cached state
  harvx cache clear

  # Clear state for a specific profile
  harvx cache clear --profile finvault

  # Clear state in a specific directory
  harvx cache clear -d /path/to/project`,
	RunE: runCacheClear,
}

var cacheShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show cached state summary",
	Long: `Display a summary of all cached state snapshots.

Lists each cached profile with its last generation timestamp, git branch,
HEAD SHA, and file count. Use --json for machine-readable output.`,
	Example: `  # Show cached state summary
  harvx cache show

  # Show as JSON
  harvx cache show --json

  # Show state for a specific directory
  harvx cache show -d /path/to/project`,
	RunE: runCacheShow,
}

func init() {
	cacheClearCmd.Flags().StringP("profile", "p", "", "Clear state for a specific profile")
	cacheShowCmd.Flags().Bool("json", false, "Output as JSON")
	cacheCmd.AddCommand(cacheClearCmd, cacheShowCmd)
	rootCmd.AddCommand(cacheCmd)
}

// stateBasePath returns the path to the .harvx/state/ directory under rootDir.
func stateBasePath(rootDir string) string {
	return filepath.Join(rootDir, ".harvx", "state")
}

// runCacheClear executes the cache clear subcommand. It clears either all
// cached state or state for a specific profile, depending on the --profile flag.
func runCacheClear(cmd *cobra.Command, _ []string) error {
	rootDir, err := filepath.Abs(flagValues.Dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	profileName, _ := cmd.Flags().GetString("profile")

	slog.Debug("cache clear",
		"root", rootDir,
		"profile", profileName,
	)

	if profileName != "" {
		return clearProfileState(cmd, rootDir, profileName)
	}
	return clearAllState(cmd, rootDir)
}

// clearProfileState removes the cached state file for a single profile.
func clearProfileState(cmd *cobra.Command, rootDir, profileName string) error {
	cache := diff.NewStateCache(profileName)

	if !cache.HasState(rootDir) {
		fmt.Fprintln(cmd.OutOrStdout(), "No cached state found.")
		return nil
	}

	if err := cache.ClearState(rootDir); err != nil {
		return fmt.Errorf("clearing state for profile %q: %w", profileName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Cleared cached state for profile '%s'\n", profileName)
	return nil
}

// clearAllState removes the entire .harvx/state/ directory.
func clearAllState(cmd *cobra.Command, rootDir string) error {
	dir := stateBasePath(rootDir)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Fprintln(cmd.OutOrStdout(), "No cached state found.")
		return nil
	}

	// Use a dummy cache instance to call ClearAllState.
	cache := diff.NewStateCache("default")
	if err := cache.ClearAllState(rootDir); err != nil {
		return fmt.Errorf("clearing all cached state: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Cleared all cached state from .harvx/state/")
	return nil
}

// cacheShowEntry holds the metadata for a single cached profile, used for
// both table and JSON output in the cache show subcommand.
type cacheShowEntry struct {
	Name        string `json:"name"`
	GeneratedAt string `json:"generated_at"`
	GitBranch   string `json:"git_branch"`
	GitHeadSHA  string `json:"git_head_sha"`
	FileCount   int    `json:"file_count"`
	StateFile   string `json:"state_file"`
}

// cacheShowOutput is the top-level JSON structure for cache show --json.
type cacheShowOutput struct {
	CacheDir string           `json:"cache_dir"`
	Profiles []cacheShowEntry `json:"profiles"`
}

// runCacheShow executes the cache show subcommand. It reads all state files
// from .harvx/state/ and displays their metadata.
func runCacheShow(cmd *cobra.Command, _ []string) error {
	rootDir, err := filepath.Abs(flagValues.Dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")

	slog.Debug("cache show",
		"root", rootDir,
		"json", jsonFlag,
	)

	dir := stateBasePath(rootDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(cmd.OutOrStdout(), "No cached state found. Run 'harvx generate' to create state.")
			return nil
		}
		return fmt.Errorf("reading state directory: %w", err)
	}

	// Collect profile entries from .json files.
	var profiles []cacheShowEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			slog.Debug("skipping unreadable state file",
				"path", filePath,
				"error", readErr,
			)
			continue
		}

		snap, parseErr := diff.ParseStateSnapshot(data)
		if parseErr != nil {
			slog.Debug("skipping unparseable state file",
				"path", filePath,
				"error", parseErr,
			)
			continue
		}

		profileName := strings.TrimSuffix(entry.Name(), ".json")

		// Truncate HEAD SHA to 7 characters for display.
		headSHA := snap.GitHeadSHA
		if len(headSHA) > 7 {
			headSHA = headSHA[:7]
		}

		// Build the relative state file path for display.
		relStateFile := filepath.Join(".harvx", "state", entry.Name())

		profiles = append(profiles, cacheShowEntry{
			Name:        profileName,
			GeneratedAt: snap.GeneratedAt,
			GitBranch:   snap.GitBranch,
			GitHeadSHA:  headSHA,
			FileCount:   len(snap.Files),
			StateFile:   relStateFile,
		})
	}

	if len(profiles) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No cached state found. Run 'harvx generate' to create state.")
		return nil
	}

	// Sort profiles alphabetically by name.
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	if jsonFlag {
		return renderCacheShowJSON(cmd, dir, rootDir, profiles)
	}
	return renderCacheShowTable(cmd, profiles)
}

// renderCacheShowJSON outputs the cache show data as formatted JSON.
func renderCacheShowJSON(cmd *cobra.Command, dir, rootDir string, profiles []cacheShowEntry) error {
	// Make the cache_dir relative to rootDir for cleaner output.
	relDir, err := filepath.Rel(rootDir, dir)
	if err != nil {
		relDir = dir
	}

	output := cacheShowOutput{
		CacheDir: relDir,
		Profiles: profiles,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache show JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// renderCacheShowTable outputs the cache show data as a formatted table.
func renderCacheShowTable(cmd *cobra.Command, profiles []cacheShowEntry) error {
	out := cmd.OutOrStdout()

	fmt.Fprintln(out, "Cached State Summary (.harvx/state/):")
	fmt.Fprintln(out)

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PROFILE\tGENERATED\tBRANCH\tHEAD\tFILES")

	for _, p := range profiles {
		// Format GeneratedAt from RFC3339 to a more readable format.
		displayTime := formatGeneratedAt(p.GeneratedAt)

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n",
			p.Name,
			displayTime,
			p.GitBranch,
			p.GitHeadSHA,
			p.FileCount,
		)
	}

	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flushing table writer: %w", err)
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Total: %d profiles cached\n", len(profiles))

	return nil
}

// formatGeneratedAt converts an RFC3339 timestamp to a human-readable
// format (2006-01-02 15:04:05). If parsing fails, the raw string is returned.
func formatGeneratedAt(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.Format("2006-01-02 15:04:05")
}
