package diff

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// GitChangeType represents the type of change to a file in a git diff.
type GitChangeType int

const (
	// GitAdded indicates a file was added.
	GitAdded GitChangeType = iota
	// GitModified indicates a file was modified.
	GitModified
	// GitDeleted indicates a file was deleted.
	GitDeleted
	// GitRenamed indicates a file was renamed.
	GitRenamed
)

// String returns the human-readable name of the change type.
func (t GitChangeType) String() string {
	switch t {
	case GitAdded:
		return "added"
	case GitModified:
		return "modified"
	case GitDeleted:
		return "deleted"
	case GitRenamed:
		return "renamed"
	default:
		return "unknown"
	}
}

// GitFileChange represents a single file change from a git diff.
type GitFileChange struct {
	// Path is the relative file path.
	Path string

	// OldPath is the previous path, non-empty only for renames.
	OldPath string

	// Status is the type of change.
	Status GitChangeType
}

// GitDiffer provides git-aware diffing capabilities by shelling out to git CLI.
type GitDiffer struct{}

// NewGitDiffer creates a new GitDiffer.
func NewGitDiffer() *GitDiffer {
	return &GitDiffer{}
}

// GetCurrentBranch returns the current git branch name for the repository at
// rootDir. If the repository is in detached HEAD state, an empty string is
// returned. Returns ErrNotGitRepo if rootDir is not inside a git repository
// and ErrGitNotFound if the git executable is not on PATH.
func (d *GitDiffer) GetCurrentBranch(ctx context.Context, rootDir string) (string, error) {
	out, err := runGit(ctx, rootDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}

	// Detached HEAD returns the literal string "HEAD".
	if out == "HEAD" {
		return "", nil
	}

	return out, nil
}

// GetHeadSHA returns the short (7-character) SHA of the HEAD commit for the
// repository at rootDir.
func (d *GitDiffer) GetHeadSHA(ctx context.Context, rootDir string) (string, error) {
	out, err := runGit(ctx, rootDir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting HEAD SHA: %w", err)
	}

	return out, nil
}

// ValidateRef checks whether the given git ref can be resolved in the
// repository at rootDir. Returns nil if the ref is valid, or an error wrapping
// ErrInvalidRef if it cannot be resolved.
func (d *GitDiffer) ValidateRef(ctx context.Context, rootDir, ref string) error {
	_, err := runGit(ctx, rootDir, "rev-parse", "--verify", ref)
	if err != nil {
		// If the underlying error is ErrGitNotFound or ErrNotGitRepo, preserve
		// those sentinel errors rather than masking them as ErrInvalidRef.
		if errors.Is(err, ErrGitNotFound) || errors.Is(err, ErrNotGitRepo) {
			return fmt.Errorf("validating ref %q: %w", ref, err)
		}
		return fmt.Errorf("ref %q: %w", ref, ErrInvalidRef)
	}

	return nil
}

// GetChangedFiles returns the list of changed files between two git refs in
// the repository at rootDir. Both refs are validated before running the diff.
func (d *GitDiffer) GetChangedFiles(ctx context.Context, rootDir, baseRef, headRef string) ([]GitFileChange, error) {
	if err := d.ValidateRef(ctx, rootDir, baseRef); err != nil {
		return nil, fmt.Errorf("validating base ref: %w", err)
	}

	if err := d.ValidateRef(ctx, rootDir, headRef); err != nil {
		return nil, fmt.Errorf("validating head ref: %w", err)
	}

	out, err := runGit(ctx, rootDir, "diff", "--name-status", baseRef+".."+headRef)
	if err != nil {
		return nil, fmt.Errorf("getting changed files %s..%s: %w", baseRef, headRef, err)
	}

	return parseNameStatus(out)
}

// GetChangedFilesSince returns the list of changed files between sinceRef and
// HEAD in the repository at rootDir. The sinceRef is validated before running
// the diff.
func (d *GitDiffer) GetChangedFilesSince(ctx context.Context, rootDir, sinceRef string) ([]GitFileChange, error) {
	if err := d.ValidateRef(ctx, rootDir, sinceRef); err != nil {
		return nil, fmt.Errorf("validating since ref: %w", err)
	}

	out, err := runGit(ctx, rootDir, "diff", "--name-status", sinceRef+"..HEAD")
	if err != nil {
		return nil, fmt.Errorf("getting changed files since %s: %w", sinceRef, err)
	}

	return parseNameStatus(out)
}

// BuildDiffResultFromGit converts a slice of GitFileChange into a DiffResult.
// Renamed files are treated as a deletion of the old path and an addition of
// the new path. All result slices are sorted alphabetically.
func BuildDiffResultFromGit(changes []GitFileChange) *DiffResult {
	result := &DiffResult{}

	for _, c := range changes {
		switch c.Status {
		case GitAdded:
			result.Added = append(result.Added, c.Path)
		case GitModified:
			result.Modified = append(result.Modified, c.Path)
		case GitDeleted:
			result.Deleted = append(result.Deleted, c.Path)
		case GitRenamed:
			result.Deleted = append(result.Deleted, c.OldPath)
			result.Added = append(result.Added, c.Path)
		}
	}

	sort.Strings(result.Added)
	sort.Strings(result.Modified)
	sort.Strings(result.Deleted)

	return result
}

// runGit executes a git command and returns trimmed stdout.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir, "--no-pager"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check for git not found.
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("running git: %w", ErrGitNotFound)
		}

		// Check for not a git repository.
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "not a git repository") {
			return "", fmt.Errorf("%s: %w", strings.TrimSpace(stderrStr), ErrNotGitRepo)
		}

		return "", fmt.Errorf("git %s: %w: %s", args[0], err, stderrStr)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// parseNameStatus parses the output of `git diff --name-status`. Each line has
// the format "<status>\t<path>" or "<status>\t<old-path>\t<new-path>" for
// renames. Status codes: A=Added, M=Modified, D=Deleted, R<score>=Renamed,
// C<score>=Copied (treated as Added). Empty output returns nil, nil.
func parseNameStatus(output string) ([]GitFileChange, error) {
	if output == "" {
		return nil, nil
	}

	lines := strings.Split(output, "\n")
	changes := make([]GitFileChange, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			return nil, fmt.Errorf("malformed name-status line: %q", line)
		}

		statusStr := parts[0]
		var change GitFileChange

		switch {
		case statusStr == "A":
			change = GitFileChange{
				Path:   parts[1],
				Status: GitAdded,
			}
		case statusStr == "M":
			change = GitFileChange{
				Path:   parts[1],
				Status: GitModified,
			}
		case statusStr == "D":
			change = GitFileChange{
				Path:   parts[1],
				Status: GitDeleted,
			}
		case strings.HasPrefix(statusStr, "R"):
			if len(parts) < 3 {
				return nil, fmt.Errorf("malformed rename line (expected 3 fields): %q", line)
			}
			change = GitFileChange{
				Path:    parts[2],
				OldPath: parts[1],
				Status:  GitRenamed,
			}
		case strings.HasPrefix(statusStr, "C"):
			// Copy is treated as Added.
			if len(parts) < 3 {
				return nil, fmt.Errorf("malformed copy line (expected 3 fields): %q", line)
			}
			change = GitFileChange{
				Path:   parts[2],
				Status: GitAdded,
			}
		default:
			return nil, fmt.Errorf("unknown git status code %q in line: %q", statusStr, line)
		}

		changes = append(changes, change)
	}

	return changes, nil
}