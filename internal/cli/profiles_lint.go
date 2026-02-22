package cli

import (
	"fmt"
	"log/slog"

	"github.com/harvx/harvx/internal/config"
	"github.com/spf13/cobra"
)

// profilesLintCmd lints the harvx configuration for errors and warnings.
var profilesLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint the harvx configuration for errors and warnings",
	Long: `Run comprehensive validation and static analysis on the active harvx configuration.

Lint groups findings by severity (errors, warnings, info) and exits with code 1
if any errors are found. Warnings do not cause a non-zero exit.

Use --profile to restrict linting to a single named profile.`,
	RunE: runProfilesLint,
}

func init() {
	profilesLintCmd.Flags().String("profile", "", "lint only the specified profile name")
	profilesCmd.AddCommand(profilesLintCmd)
}

// runProfilesLint implements `harvx profiles lint`.
func runProfilesLint(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	profileFlag, _ := cmd.Flags().GetString("profile")

	// Discover the repo config path for display purposes.
	repoPath, err := config.DiscoverRepoConfig(".")
	if err != nil {
		slog.Debug("repo config discovery failed", "err", err)
	}

	var cfg *config.Config
	if repoPath == "" {
		fmt.Fprintln(out, "No harvx.toml found; using built-in defaults")
		cfg = &config.Config{Profile: map[string]*config.Profile{}}
	} else {
		cfg, err = config.LoadFromFile(repoPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		fmt.Fprintf(out, "Linting %s...\n", displayPath(repoPath))
	}

	// Filter to a single profile if requested.
	if profileFlag != "" {
		p, ok := cfg.Profile[profileFlag]
		if !ok {
			return fmt.Errorf("profile %q not found in configuration", profileFlag)
		}
		cfg = &config.Config{
			Profile: map[string]*config.Profile{
				profileFlag: p,
			},
		}
	}

	results := config.Lint(cfg)

	if len(results) == 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "No issues found.")
		return nil
	}

	// Partition results by severity.
	var errors, warnings, infos []config.LintResult
	for _, r := range results {
		switch r.Severity {
		case "error":
			errors = append(errors, r)
		case "warning":
			warnings = append(warnings, r)
		default:
			infos = append(infos, r)
		}
	}

	// Print each section.
	if len(errors) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Errors:")
		for _, r := range errors {
			fmt.Fprintf(out, "  X [%s] %s\n", r.Field, r.Message)
			if r.Suggest != "" {
				fmt.Fprintf(out, "    Fix: %s\n", r.Suggest)
			}
		}
	}

	if len(warnings) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Warnings:")
		for _, r := range warnings {
			fmt.Fprintf(out, "  ! [%s] %s\n", r.Field, r.Message)
			if r.Suggest != "" {
				fmt.Fprintf(out, "    Fix: %s\n", r.Suggest)
			}
		}
	}

	if len(infos) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Info:")
		for _, r := range infos {
			fmt.Fprintf(out, "  i [%s] %s\n", r.Field, r.Message)
			if r.Suggest != "" {
				fmt.Fprintf(out, "    Fix: %s\n", r.Suggest)
			}
		}
	}

	// Summary line.
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Result: %d error(s), %d warning(s), %d info\n",
		len(errors), len(warnings), len(infos))

	if len(errors) > 0 {
		return fmt.Errorf("lint: %d error(s) found in configuration", len(errors))
	}
	return nil
}
