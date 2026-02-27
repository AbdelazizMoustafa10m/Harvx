//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// harvxBinary is the path to the compiled harvx binary.
// It is built once via TestMain and reused by all tests.
var harvxBinary string

// TestMain compiles the harvx binary before running integration tests.
func TestMain(m *testing.M) {
	// Build the binary once.
	binary, err := buildHarvx()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build harvx binary: %v\n", err)
		os.Exit(1)
	}
	harvxBinary = binary

	os.Exit(m.Run())
}

// buildHarvx compiles the harvx binary into a temporary directory
// and returns its path.
func buildHarvx() (string, error) {
	dir, err := os.MkdirTemp("", "harvx-integration-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	binaryName := "harvx"
	if runtime.GOOS == "windows" {
		binaryName = "harvx.exe"
	}
	binaryPath := filepath.Join(dir, binaryName)

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/harvx/")
	cmd.Dir = projectRoot()
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %s: %w", string(output), err)
	}

	return binaryPath, nil
}

// projectRoot returns the absolute path to the project root.
func projectRoot() string {
	// Navigate from tests/integration/ up to project root.
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// sampleRepoDir returns the absolute path to testdata/sample-repo.
func sampleRepoDir() string {
	return filepath.Join(projectRoot(), "testdata", "sample-repo")
}

// runHarvx executes the harvx binary with the given arguments and environment
// variables. Returns stdout, stderr, and the exit code.
func runHarvx(t *testing.T, args []string, env ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(harvxBinary, args...)
	cmd.Dir = sampleRepoDir()

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Build environment: inherit current env + add overrides.
	cmdEnv := os.Environ()
	// Clear any existing HARVX_ vars from parent process.
	filtered := make([]string, 0, len(cmdEnv))
	for _, e := range cmdEnv {
		if !strings.HasPrefix(e, "HARVX_") {
			filtered = append(filtered, e)
		}
	}
	cmdEnv = append(filtered, env...)
	cmd.Env = cmdEnv

	err := cmd.Run()

	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run harvx: %v", err)
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// runHarvxInDir is like runHarvx but runs in a specified directory.
func runHarvxInDir(t *testing.T, dir string, args []string, env ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(harvxBinary, args...)
	cmd.Dir = dir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	cmdEnv := os.Environ()
	filtered := make([]string, 0, len(cmdEnv))
	for _, e := range cmdEnv {
		if !strings.HasPrefix(e, "HARVX_") {
			filtered = append(filtered, e)
		}
	}
	cmdEnv = append(filtered, env...)
	cmd.Env = cmdEnv

	err := cmd.Run()

	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run harvx: %v", err)
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// setupGitRepo creates a temporary git repository with the sample-repo files
// for tests that need git history (like review-slice).
func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Copy sample-repo files to temp dir.
	err := copyDir(sampleRepoDir(), dir)
	require.NoError(t, err, "copying sample-repo to temp dir")

	// Initialize git repo.
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	// Initial commit with all files.
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "initial commit")

	return dir
}

// setupGitRepoWithChange creates a git repo with two commits.
// The second commit modifies a specific file.
func setupGitRepoWithChange(t *testing.T) (dir string, baseRef string) {
	t.Helper()

	dir = setupGitRepo(t)

	// Record the base ref.
	baseRef = getGitRef(t, dir)

	// Make a change and commit it.
	changePath := filepath.Join(dir, "src", "auth", "middleware.go")
	content, err := os.ReadFile(changePath)
	require.NoError(t, err)

	newContent := strings.Replace(string(content), "checking auth", "checking auth v2", 1)
	err = os.WriteFile(changePath, []byte(newContent), 0o644)
	require.NoError(t, err)

	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "update auth middleware")

	return dir, baseRef
}

// runGit runs a git command in the given directory.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s failed: %s", strings.Join(args, " "), string(output))

	return strings.TrimSpace(string(output))
}

// getGitRef returns the current HEAD commit SHA.
func getGitRef(t *testing.T, dir string) string {
	t.Helper()
	return runGit(t, dir, "rev-parse", "HEAD")
}

// copyDir recursively copies a directory, including hidden files and directories.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}
