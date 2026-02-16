package discovery

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
)

// GitTrackedFiles runs `git ls-files` in the given root directory and returns
// a set of file paths relative to the root that are tracked by Git. This is
// used to implement the --git-tracked-only flag, which restricts discovery to
// files in the Git index.
//
// The returned map uses relative paths (as output by git ls-files) as keys,
// with all values set to true for O(1) membership checks.
//
// Errors are returned when:
//   - The directory is not a Git repository (git ls-files fails).
//   - The git command is not found on PATH.
//
// An empty repository (no tracked files) returns an empty map and a nil error.
func GitTrackedFiles(root string) (map[string]bool, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = root

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed in %s: %w (is this a git repository?)", root, err)
	}

	files := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			files[line] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing git ls-files output: %w", err)
	}

	return files, nil
}
