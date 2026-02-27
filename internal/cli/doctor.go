package cli

import (
	"os"

	"github.com/harvx/harvx/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorJSON bool
var doctorFix bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check for common repository issues",
	Long: `Run diagnostic checks against the current repository to identify
issues that may affect context generation quality.

Checks include:
  - Git repository status (branch, HEAD, clean/dirty)
  - Large binary files not excluded (>1MB)
  - Oversized text files that may blow token budgets (>500KB)
  - Build artifact directories without .harvxignore
  - Configuration validation (harvx.toml)
  - Stale cache files in .harvx/state/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := flagValues.Dir
		if dir == "" {
			dir = "."
		}

		report, err := doctor.Run(doctor.Options{
			Dir: dir,
			Fix: doctorFix,
		})
		if err != nil {
			return err
		}

		if doctorJSON {
			return doctor.FormatJSON(cmd.OutOrStdout(), report)
		}

		doctor.FormatText(cmd.OutOrStdout(), report)

		if report.HasFail {
			// Return exit code 1 via os.Exit rather than error to avoid
			// printing the error message twice (cobra + slog).
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results as JSON")
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Auto-fix detected issues (e.g. generate .harvxignore)")
	rootCmd.AddCommand(doctorCmd)
}
