// Package doctor provides diagnostic checks for repository health. It
// inspects a target directory for common issues that affect context generation
// quality, including git status, large binaries, oversized text files, build
// artifacts, configuration validity, and stale cache entries.
package doctor

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/discovery"
)

// Status represents the result status of a doctor check.
type Status string

const (
	// StatusPass indicates the check completed without issues.
	StatusPass Status = "pass"
	// StatusWarn indicates the check found a non-critical issue.
	StatusWarn Status = "warn"
	// StatusFail indicates the check found a critical issue.
	StatusFail Status = "fail"
)

// CheckResult is the outcome of a single diagnostic check.
type CheckResult struct {
	Name    string   `json:"name"`
	Status  Status   `json:"status"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

// DoctorReport is the full output of a doctor run.
type DoctorReport struct {
	Directory string        `json:"directory"`
	Timestamp string        `json:"timestamp"`
	Checks    []CheckResult `json:"checks"`
	HasFail   bool          `json:"has_fail"`
	HasWarn   bool          `json:"has_warn"`
}

// Options configures the doctor run.
type Options struct {
	// Dir is the target directory to diagnose.
	Dir string
	// Fix enables auto-remediation where possible (e.g. generating .harvxignore).
	Fix bool
}

// maxBinarySize is the threshold above which a binary file triggers a warning.
const maxBinarySize int64 = 1_048_576 // 1MB

// maxTextSize is the threshold above which a text file triggers a warning
// because it may blow token budgets.
const maxTextSize int64 = 512_000 // 500KB

// staleCacheThreshold is the age beyond which cache files are considered stale.
const staleCacheThreshold = 7 * 24 * time.Hour

// maxDetailEntries is the maximum number of file entries shown in details
// before truncation.
const maxDetailEntries = 10

// Run executes all diagnostic checks against the given directory and returns
// a DoctorReport summarising the findings. Each check runs independently;
// a failure in one check does not prevent subsequent checks from running.
func Run(opts Options) (*DoctorReport, error) {
	absDir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("resolving directory: %w", err)
	}

	report := &DoctorReport{
		Directory: absDir,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	checks := []func(string, bool) CheckResult{
		checkGitRepo,
		checkLargeBinaries,
		checkOversizedTextFiles,
		checkBuildArtifacts,
		checkConfig,
		checkStaleCache,
	}

	for _, check := range checks {
		result := check(absDir, opts.Fix)
		report.Checks = append(report.Checks, result)
		switch result.Status {
		case StatusFail:
			report.HasFail = true
		case StatusWarn:
			report.HasWarn = true
		}
	}

	return report, nil
}

// checkGitRepo reports git repository status (branch, HEAD SHA, clean/dirty).
func checkGitRepo(dir string, _ bool) CheckResult {
	result := CheckResult{Name: "Git Repository"}

	// Check if it's a git repo by looking for .git.
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		result.Status = StatusWarn
		result.Message = "Not a git repository"
		return result
	}

	var details []string

	// Get branch name.
	branch, err := gitCommand(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		result.Status = StatusWarn
		result.Message = "Git repository detected but could not read branch"
		return result
	}
	details = append(details, fmt.Sprintf("Branch: %s", branch))

	// Get HEAD SHA.
	sha, err := gitCommand(dir, "rev-parse", "--short", "HEAD")
	if err == nil {
		details = append(details, fmt.Sprintf("HEAD: %s", sha))
	}

	// Check if working tree is clean.
	status, err := gitCommand(dir, "status", "--porcelain")
	if err == nil {
		if status == "" {
			details = append(details, "Working tree: clean")
		} else {
			lines := strings.Split(strings.TrimSpace(status), "\n")
			details = append(details, fmt.Sprintf("Working tree: dirty (%d changed files)", len(lines)))
		}
	}

	result.Status = StatusPass
	result.Message = fmt.Sprintf("Git repository on branch %s", branch)
	result.Details = details
	return result
}

// checkLargeBinaries warns about binary files >1MB that may inflate context size.
func checkLargeBinaries(dir string, fix bool) CheckResult {
	result := CheckResult{Name: "Large Binary Files"}

	var largeBinaries []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // skip inaccessible entries
		}
		if info.IsDir() {
			if shouldSkipDir(filepath.Base(path)) {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() <= maxBinarySize {
			return nil
		}
		isBin, binErr := discovery.IsBinary(path)
		if binErr != nil {
			slog.Debug("binary check failed", "path", path, "err", binErr)
			return nil
		}
		if isBin {
			rel, _ := filepath.Rel(dir, path)
			largeBinaries = append(largeBinaries, rel)
		}
		return nil
	})
	if err != nil {
		result.Status = StatusWarn
		result.Message = fmt.Sprintf("Error scanning for binaries: %s", err)
		return result
	}

	if len(largeBinaries) == 0 {
		result.Status = StatusPass
		result.Message = "No large binary files found"
		return result
	}

	result.Status = StatusWarn
	result.Message = fmt.Sprintf("Found %d large binary file(s) >1MB", len(largeBinaries))
	result.Details = truncateDetails(largeBinaries, maxDetailEntries)

	if fix {
		writeHarvxignoreEntries(dir, largeBinaries)
	}

	return result
}

// checkOversizedTextFiles warns about text files >500KB that might blow token budgets.
func checkOversizedTextFiles(dir string, _ bool) CheckResult {
	result := CheckResult{Name: "Oversized Text Files"}

	var oversized []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			if shouldSkipDir(filepath.Base(path)) {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() <= maxTextSize {
			return nil
		}
		isBin, binErr := discovery.IsBinary(path)
		if binErr != nil || isBin {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		oversized = append(oversized, fmt.Sprintf("%s (%s)", rel, formatSize(info.Size())))
		return nil
	})
	if err != nil {
		result.Status = StatusWarn
		result.Message = fmt.Sprintf("Error scanning for oversized files: %s", err)
		return result
	}

	if len(oversized) == 0 {
		result.Status = StatusPass
		result.Message = "No oversized text files found"
		return result
	}

	result.Status = StatusWarn
	result.Message = fmt.Sprintf("Found %d text file(s) >500KB that may blow token budgets", len(oversized))
	result.Details = truncateDetails(oversized, maxDetailEntries)

	return result
}

// checkBuildArtifacts checks for common build artifact directories and whether
// a .harvxignore file exists to exclude them.
func checkBuildArtifacts(dir string, fix bool) CheckResult {
	result := CheckResult{Name: "Build Artifacts"}

	artifactDirs := []string{
		"node_modules",
		"dist",
		"target",
		"build",
		"__pycache__",
		".next",
		"out",
	}

	var found []string
	for _, d := range artifactDirs {
		fullPath := filepath.Join(dir, d)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			found = append(found, d)
		}
	}

	if len(found) == 0 {
		result.Status = StatusPass
		result.Message = "No unignored build artifact directories detected"
		return result
	}

	// Check if .harvxignore exists.
	harvxignorePath := filepath.Join(dir, ".harvxignore")
	if _, err := os.Stat(harvxignorePath); os.IsNotExist(err) {
		result.Status = StatusWarn
		result.Message = fmt.Sprintf("Build artifact directories found but no .harvxignore file: %s", strings.Join(found, ", "))
		result.Details = []string{
			"Consider creating a .harvxignore file to exclude build artifacts",
			"Run 'harvx doctor --fix' to auto-generate .harvxignore entries",
		}

		if fix {
			entries := make([]string, len(found))
			for i, d := range found {
				entries[i] = d + "/"
			}
			writeHarvxignoreFile(dir, entries)
			result.Details = append(result.Details, "Auto-generated .harvxignore with artifact entries")
		}
	} else {
		result.Status = StatusPass
		result.Message = fmt.Sprintf("Build artifact directories present (%s); .harvxignore exists", strings.Join(found, ", "))
	}

	return result
}

// checkConfig validates harvx.toml if present in the target directory.
func checkConfig(dir string, _ bool) CheckResult {
	result := CheckResult{Name: "Configuration"}

	configPath, err := config.DiscoverRepoConfig(dir)
	if err != nil {
		result.Status = StatusWarn
		result.Message = fmt.Sprintf("Error searching for config: %s", err)
		return result
	}

	if configPath == "" {
		result.Status = StatusPass
		result.Message = "No harvx.toml found (using defaults)"
		result.Details = []string{"Run 'harvx profiles init' to create a config file"}
		return result
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		result.Status = StatusFail
		result.Message = fmt.Sprintf("Invalid config: %s", err)
		return result
	}

	validationErrors := config.Validate(cfg)
	if len(validationErrors) == 0 {
		rel, _ := filepath.Rel(dir, configPath)
		if rel == "" {
			rel = configPath
		}
		result.Status = StatusPass
		result.Message = fmt.Sprintf("Config valid: %s", rel)
		return result
	}

	var errCount, warnCount int
	var details []string
	for _, ve := range validationErrors {
		if ve.Severity == "error" {
			errCount++
		} else {
			warnCount++
		}
		details = append(details, ve.Error())
	}

	if errCount > 0 {
		result.Status = StatusFail
	} else {
		result.Status = StatusWarn
	}
	result.Message = fmt.Sprintf("Config has %d error(s) and %d warning(s)", errCount, warnCount)
	result.Details = details

	return result
}

// checkStaleCache checks for stale cache files in .harvx/state/.
func checkStaleCache(dir string, _ bool) CheckResult {
	result := CheckResult{Name: "State Cache"}

	stateDir := filepath.Join(dir, ".harvx", "state")
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			result.Status = StatusPass
			result.Message = "No state cache directory"
			return result
		}
		result.Status = StatusWarn
		result.Message = fmt.Sprintf("Error reading state cache: %s", err)
		return result
	}

	if len(entries) == 0 {
		result.Status = StatusPass
		result.Message = "State cache is empty"
		return result
	}

	var staleFiles []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		age := time.Since(info.ModTime())
		if age > staleCacheThreshold {
			staleFiles = append(staleFiles, fmt.Sprintf("%s (age: %s)", entry.Name(), formatDuration(age)))
		}
	}

	if len(staleFiles) == 0 {
		result.Status = StatusPass
		result.Message = fmt.Sprintf("%d cache file(s), all fresh", len(entries))
		return result
	}

	result.Status = StatusWarn
	result.Message = fmt.Sprintf("%d stale cache file(s) older than 7 days", len(staleFiles))
	result.Details = staleFiles
	result.Details = append(result.Details, "Run 'harvx --clear-cache' to remove stale cache entries")

	return result
}

// shouldSkipDir reports whether a directory should be skipped during walks.
// Hidden directories (starting with '.') and common dependency/build dirs
// are skipped to avoid scanning irrelevant content.
func shouldSkipDir(base string) bool {
	if strings.HasPrefix(base, ".") {
		return true
	}
	switch base {
	case "node_modules", "vendor":
		return true
	}
	return false
}

// gitCommand runs a git command in the specified directory and returns
// the trimmed stdout output.
func gitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// writeHarvxignoreEntries appends entries to .harvxignore (or creates it).
func writeHarvxignoreEntries(dir string, entries []string) {
	path := filepath.Join(dir, ".harvxignore")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Warn("could not write .harvxignore", "err", err)
		return
	}
	defer f.Close()

	for _, entry := range entries {
		fmt.Fprintln(f, entry)
	}
	slog.Info("appended entries to .harvxignore", "count", len(entries))
}

// writeHarvxignoreFile creates a new .harvxignore with the given entries.
func writeHarvxignoreFile(dir string, entries []string) {
	path := filepath.Join(dir, ".harvxignore")
	content := "# Auto-generated by harvx doctor --fix\n" + strings.Join(entries, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		slog.Warn("could not create .harvxignore", "err", err)
		return
	}
	slog.Info("created .harvxignore", "entries", len(entries))
}

// truncateDetails returns at most limit entries from items, appending a
// summary line if truncation occurs.
func truncateDetails(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	result := make([]string, limit, limit+1)
	copy(result, items[:limit])
	result = append(result, fmt.Sprintf("... and %d more", len(items)-limit))
	return result
}

// formatSize formats bytes as a human-readable string.
func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// formatDuration formats a duration as a human-readable string using days
// when appropriate.
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	return d.Round(time.Minute).String()
}
