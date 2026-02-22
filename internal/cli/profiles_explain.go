package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/harvx/harvx/internal/config"
	"github.com/spf13/cobra"
)

// profilesExplainCmd shows how the active profile processes a specific file.
var profilesExplainCmd = &cobra.Command{
	Use:   "explain <filepath>",
	Short: "Show how the active profile processes a file",
	Long: `Simulate the discovery pipeline for a given file path and show the full
rule trace: which ignore patterns, include filters, and relevance tiers apply.

The command is informational only -- it does not generate any output files.

Pass a glob pattern (e.g. "src/**/*.ts") to explain multiple matching files.
Use --profile to explain against a specific named profile.`,
	Args: cobra.ExactArgs(1),
	RunE: runProfilesExplain,
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	},
}

func init() {
	profilesExplainCmd.Flags().String("profile", "", "profile name to explain against")
	profilesCmd.AddCommand(profilesExplainCmd)
}

// runProfilesExplain implements `harvx profiles explain <filepath>`.
func runProfilesExplain(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	profileFlag, _ := cmd.Flags().GetString("profile")
	out := cmd.OutOrStdout()

	// Resolve the profile through the full multi-source pipeline.
	resolveOpts := config.ResolveOptions{TargetDir: "."}
	if profileFlag != "" {
		resolveOpts.ProfileName = profileFlag
	}
	resolved, err := config.Resolve(resolveOpts)
	if err != nil {
		return fmt.Errorf("resolving profile: %w", err)
	}

	profileName := resolved.ProfileName

	// Determine whether filePath is a glob pattern.
	isGlob := strings.ContainsAny(filePath, "*?[{")

	if isGlob {
		// Expand the glob pattern against the current directory.
		matches, err := doublestar.Glob(os.DirFS("."), filePath, doublestar.WithFilesOnly())
		if err != nil {
			return fmt.Errorf("expanding glob %q: %w", filePath, err)
		}
		if len(matches) == 0 {
			fmt.Fprintf(out, "No files matched glob pattern %q\n", filePath)
			return nil
		}
		for i, match := range matches {
			if i > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, strings.Repeat("-", 60))
				fmt.Fprintln(out)
			}
			result := config.ExplainFile(match, profileName, resolved.Profile)
			printExplainResult(out, result)
		}
		return nil
	}

	// Single file path.
	result := config.ExplainFile(filePath, profileName, resolved.Profile)
	printExplainResult(out, result)
	return nil
}

// printExplainResult formats and writes a single ExplainResult to w.
func printExplainResult(w io.Writer, result config.ExplainResult) {
	printTo(w, result)
}

// printTo writes the formatted ExplainResult to any io.Writer.
func printTo(w io.Writer, result config.ExplainResult) {
	// Header: file path being explained.
	fmt.Fprintf(w, "Explaining: %s\n", result.FilePath)

	// Profile line.
	if result.Extends != "" {
		fmt.Fprintf(w, "Profile: %s (extends: %s)\n", result.ProfileName, result.Extends)
	} else {
		fmt.Fprintf(w, "Profile: %s\n", result.ProfileName)
	}
	fmt.Fprintln(w)

	if result.Included {
		fmt.Fprintf(w, "  Status:     INCLUDED\n")
		fmt.Fprintf(w, "  Tier:       %s\n", formatTier(result.Tier))
		if result.TierPattern != "" {
			tierLabel := "priority_files"
			if !result.IsPriority {
				// Find which tier name the pattern came from via the trace.
				for _, step := range result.Trace {
					if step.Matched && strings.HasPrefix(step.Rule, "Relevance ") {
						tierLabel = strings.TrimPrefix(step.Rule, "Relevance ")
						break
					}
				}
			}
			fmt.Fprintf(w, "  Matched by: %s pattern %q\n", tierLabel, result.TierPattern)
		}
		fmt.Fprintf(w, "  Redaction:  %s\n", formatRedaction(result))
		fmt.Fprintf(w, "  Compress:   %s\n", formatCompression(result.Compression))
		fmt.Fprintf(w, "  Priority:   %s\n", formatPriority(result.IsPriority))
	} else {
		fmt.Fprintf(w, "  Status:     EXCLUDED\n")
		fmt.Fprintf(w, "  Excluded by: %s\n", result.ExcludedBy)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Rule trace:")
	for _, step := range result.Trace {
		fmt.Fprintf(w, "  %d. %s: %s\n", step.StepNum, step.Rule, step.Outcome)
	}
}

// formatTier returns a human-readable string for the given tier number.
func formatTier(tier int) string {
	switch tier {
	case 0:
		return "0 (critical priority)"
	case 1:
		return "1 (high priority)"
	case 2:
		return "2 (medium priority)"
	case 3:
		return "3 (low priority)"
	case 4:
		return "4 (documentation)"
	case 5:
		return "5 (lowest priority)"
	default:
		return "untiered (default inclusion)"
	}
}

// formatRedaction returns a human-readable redaction status string.
func formatRedaction(result config.ExplainResult) string {
	if !result.Included {
		return "n/a (file excluded)"
	}
	// We derive whether redaction is active from RedactionOn and check the
	// profile's Redaction field via the result. The ExplainResult stores the
	// final RedactionOn value which accounts for ExcludePaths matching.
	if result.RedactionOn {
		return "enabled (not in exclude_paths)"
	}
	// RedactionOn is false -- could be profile disabled or excluded by paths.
	// We cannot distinguish without the profile, so report generically.
	return "disabled"
}

// formatCompression returns a human-readable compression support string.
func formatCompression(lang string) string {
	if lang != "" {
		return fmt.Sprintf("yes (%s supported)", lang)
	}
	return "no (file type not supported)"
}

// formatPriority returns a human-readable priority status string.
func formatPriority(isPriority bool) string {
	if isPriority {
		return "in priority_files"
	}
	return "not in priority_files"
}
