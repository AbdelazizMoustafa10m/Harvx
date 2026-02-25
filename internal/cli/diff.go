package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/diff"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Generate differential output showing changes since last run or a git ref",
	Long: `Compare the current project state against a previous state and output only
what changed. Supports both cached-state diffing and git-ref diffing.

When run with no flags, compares against the cached state from the last
'harvx generate' run for the active profile.

Use --since to compare against a specific git ref (e.g., HEAD~1, a commit SHA).
Use --base and --head together to compare between two git refs (for PR reviews).`,
	Example: `  # Compare against cached state from last run
  harvx diff

  # Show changes since the last commit
  harvx diff --since HEAD~1

  # Show changes since a specific commit
  harvx diff --since abc1234

  # Compare two branches (PR review)
  harvx diff --base main --head feature-branch

  # Use a specific profile for cache lookup
  harvx diff --profile myprofile`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().String("since", "", "Git ref to diff against (e.g., HEAD~1, a commit SHA)")
	diffCmd.Flags().String("base", "", "Base git ref for PR review diffing (requires --head)")
	diffCmd.Flags().String("head", "", "Head git ref for PR review diffing (requires --base)")
	rootCmd.AddCommand(diffCmd)
}

// runDiff executes the diff command. It determines the diff mode from flags,
// runs the diff operation, and prints the change summary. The diff command is
// read-only: it does NOT save state to cache.
func runDiff(cmd *cobra.Command, _ []string) error {
	sinceRef, _ := cmd.Flags().GetString("since")
	baseRef, _ := cmd.Flags().GetString("base")
	headRef, _ := cmd.Flags().GetString("head")

	// Determine the diff mode and validate flag combinations.
	mode, err := diff.DetermineDiffMode(sinceRef, baseRef, headRef)
	if err != nil {
		return err
	}

	// Resolve the target directory.
	rootDir, err := filepath.Abs(flagValues.Dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	// Resolve the profile name.
	profileName := flagValues.Profile

	slog.Debug("diff command",
		"mode", mode.String(),
		"root", rootDir,
		"profile", profileName,
		"since", sinceRef,
		"base", baseRef,
		"head", headRef,
	)

	opts := diff.DiffOptions{
		Mode:        mode,
		RootDir:     rootDir,
		ProfileName: profileName,
		SinceRef:    sinceRef,
		BaseRef:     baseRef,
		HeadRef:     headRef,
	}

	output, err := diff.RunDiff(cmd.Context(), opts)
	if err != nil {
		// Provide a helpful message when no cached state exists.
		if errors.Is(err, diff.ErrNoState) {
			fmt.Fprintln(cmd.ErrOrStderr(), "No cached state found. Run `harvx generate` first, or use `--since <ref>` for git-based diffing.")
			return nil
		}
		return fmt.Errorf("running diff: %w", err)
	}

	// Print the change summary to stdout.
	fmt.Fprint(cmd.OutOrStdout(), output.Summary)

	return nil
}