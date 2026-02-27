package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_ReturnsReport(t *testing.T) {
	dir := t.TempDir()

	report, err := Run(Options{Dir: dir})
	require.NoError(t, err)

	// filepath.Abs may resolve symlinks differently on macOS (/var vs /private/var),
	// so we compare the Abs of the temp dir rather than the raw string.
	absDir, absErr := filepath.Abs(dir)
	require.NoError(t, absErr)
	assert.Equal(t, absDir, report.Directory)
	assert.NotEmpty(t, report.Timestamp)
	assert.Len(t, report.Checks, 6, "expected 6 checks")

	// Verify all check names are present.
	names := make([]string, len(report.Checks))
	for i, c := range report.Checks {
		names[i] = c.Name
	}
	assert.Contains(t, names, "Git Repository")
	assert.Contains(t, names, "Large Binary Files")
	assert.Contains(t, names, "Oversized Text Files")
	assert.Contains(t, names, "Build Artifacts")
	assert.Contains(t, names, "Configuration")
	assert.Contains(t, names, "State Cache")
}

func TestRun_InvalidDir(t *testing.T) {
	// Even a nonexistent dir is resolved by filepath.Abs without error on most
	// systems, so the checks themselves will produce results rather than a
	// top-level error. We just verify no panic.
	report, err := Run(Options{Dir: "/nonexistent/path/12345"})
	require.NoError(t, err)
	assert.NotNil(t, report)
}

func TestCheckGitRepo_NotARepo(t *testing.T) {
	dir := t.TempDir()
	result := checkGitRepo(dir, false)

	assert.Equal(t, StatusWarn, result.Status)
	assert.Contains(t, result.Message, "Not a git repository")
}

func TestCheckGitRepo_ValidRepo(t *testing.T) {
	// Use the actual project root as a known git repo.
	// This test relies on running within the harvx repo.
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Walk up to find the .git directory.
	dir := wd
	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("not running inside a git repository")
		}
		dir = parent
	}

	result := checkGitRepo(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Contains(t, result.Message, "Git repository on branch")
	assert.NotEmpty(t, result.Details)
}

func TestCheckLargeBinaries_NoBinaries(t *testing.T) {
	dir := t.TempDir()
	// Create a small text file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644))

	result := checkLargeBinaries(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Equal(t, "No large binary files found", result.Message)
}

func TestCheckLargeBinaries_WithLargeBinary(t *testing.T) {
	dir := t.TempDir()
	// Create a binary file >1MB containing null bytes.
	data := make([]byte, 1_100_000)
	data[0] = 0x00 // null byte to trigger binary detection
	data[100] = 0x00
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.bin"), data, 0o644))

	result := checkLargeBinaries(dir, false)

	assert.Equal(t, StatusWarn, result.Status)
	assert.Contains(t, result.Message, "1 large binary file(s)")
	assert.Contains(t, result.Details[0], "big.bin")
}

func TestCheckLargeBinaries_FixCreatesHarvxignore(t *testing.T) {
	dir := t.TempDir()
	// Create a binary file >1MB.
	data := make([]byte, 1_100_000)
	data[0] = 0x00
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.bin"), data, 0o644))

	result := checkLargeBinaries(dir, true)

	assert.Equal(t, StatusWarn, result.Status)

	// Verify .harvxignore was created with the entry.
	content, err := os.ReadFile(filepath.Join(dir, ".harvxignore"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "big.bin")
}

func TestCheckOversizedTextFiles_NoOversized(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "small.txt"), []byte("hello"), 0o644))

	result := checkOversizedTextFiles(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Equal(t, "No oversized text files found", result.Message)
}

func TestCheckOversizedTextFiles_WithOversized(t *testing.T) {
	dir := t.TempDir()
	// Create a text file >500KB (all 'a' characters, no null bytes).
	data := make([]byte, 520_000)
	for i := range data {
		data[i] = 'a'
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "huge.txt"), data, 0o644))

	result := checkOversizedTextFiles(dir, false)

	assert.Equal(t, StatusWarn, result.Status)
	assert.Contains(t, result.Message, "1 text file(s) >500KB")
	assert.Contains(t, result.Details[0], "huge.txt")
}

func TestCheckBuildArtifacts_NoneFound(t *testing.T) {
	dir := t.TempDir()

	result := checkBuildArtifacts(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Equal(t, "No unignored build artifact directories detected", result.Message)
}

func TestCheckBuildArtifacts_FoundWithoutHarvxignore(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "node_modules"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "dist"), 0o755))

	result := checkBuildArtifacts(dir, false)

	assert.Equal(t, StatusWarn, result.Status)
	assert.Contains(t, result.Message, "node_modules")
	assert.Contains(t, result.Message, "dist")
}

func TestCheckBuildArtifacts_FoundWithHarvxignore(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harvxignore"), []byte("node_modules/\n"), 0o644))

	result := checkBuildArtifacts(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Contains(t, result.Message, ".harvxignore exists")
}

func TestCheckBuildArtifacts_FixCreatesHarvxignore(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "node_modules"), 0o755))

	result := checkBuildArtifacts(dir, true)

	assert.Equal(t, StatusWarn, result.Status)
	assert.Contains(t, result.Details, "Auto-generated .harvxignore with artifact entries")

	content, err := os.ReadFile(filepath.Join(dir, ".harvxignore"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "node_modules/")
}

func TestCheckConfig_NoConfig(t *testing.T) {
	dir := t.TempDir()
	// Create .git so DiscoverRepoConfig stops searching here.
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	result := checkConfig(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Contains(t, result.Message, "No harvx.toml found")
}

func TestCheckConfig_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	// Create .git so DiscoverRepoConfig stops here.
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	validConfig := `[profile.default]
format = "markdown"
max_tokens = 128000
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(validConfig), 0o644))

	result := checkConfig(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Contains(t, result.Message, "Config valid")
}

func TestCheckConfig_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	invalidConfig := `this is not valid TOML {{{{`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(invalidConfig), 0o644))

	result := checkConfig(dir, false)

	assert.Equal(t, StatusFail, result.Status)
	assert.Contains(t, result.Message, "Invalid config")
}

func TestCheckStaleCache_NoCache(t *testing.T) {
	dir := t.TempDir()

	result := checkStaleCache(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Equal(t, "No state cache directory", result.Message)
}

func TestCheckStaleCache_EmptyCache(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".harvx", "state"), 0o755))

	result := checkStaleCache(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Equal(t, "State cache is empty", result.Message)
}

func TestCheckStaleCache_FreshFiles(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".harvx", "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "default.json"), []byte("{}"), 0o644))

	result := checkStaleCache(dir, false)

	assert.Equal(t, StatusPass, result.Status)
	assert.Contains(t, result.Message, "1 cache file(s), all fresh")
}

func TestCheckStaleCache_StaleFiles(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".harvx", "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	filePath := filepath.Join(stateDir, "old-profile.json")
	require.NoError(t, os.WriteFile(filePath, []byte("{}"), 0o644))

	// Set modification time to 10 days ago.
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(filePath, oldTime, oldTime))

	result := checkStaleCache(dir, false)

	assert.Equal(t, StatusWarn, result.Status)
	assert.Contains(t, result.Message, "1 stale cache file(s)")
	assert.Contains(t, result.Details[0], "old-profile.json")
}

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name string
		base string
		want bool
	}{
		{name: "hidden dir", base: ".git", want: true},
		{name: "hidden config", base: ".config", want: true},
		{name: "node_modules", base: "node_modules", want: true},
		{name: "vendor", base: "vendor", want: true},
		{name: "src", base: "src", want: false},
		{name: "internal", base: "internal", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldSkipDir(tt.base))
		})
	}
}

func TestTruncateDetails(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		limit int
		want  int
		last  string
	}{
		{
			name:  "under limit",
			items: []string{"a", "b", "c"},
			limit: 5,
			want:  3,
		},
		{
			name:  "at limit",
			items: []string{"a", "b", "c"},
			limit: 3,
			want:  3,
		},
		{
			name:  "over limit",
			items: []string{"a", "b", "c", "d", "e"},
			limit: 3,
			want:  4,
			last:  "... and 2 more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateDetails(tt.items, tt.limit)
			assert.Len(t, result, tt.want)
			if tt.last != "" {
				assert.Equal(t, tt.last, result[len(result)-1])
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "bytes", bytes: 500, want: "500B"},
		{name: "kilobytes", bytes: 2048, want: "2.0KB"},
		{name: "megabytes", bytes: 1_572_864, want: "1.5MB"},
		{name: "zero", bytes: 0, want: "0B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatSize(tt.bytes))
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{name: "days", duration: 10 * 24 * time.Hour, want: "10d"},
		{name: "one day", duration: 24 * time.Hour, want: "1d"},
		{name: "hours", duration: 5 * time.Hour, want: "5h0m0s"},
		{name: "minutes", duration: 30 * time.Minute, want: "30m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatDuration(tt.duration))
		})
	}
}

func TestRun_HasFailAndHasWarn(t *testing.T) {
	dir := t.TempDir()
	// Create .git so config discovery stops here.
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	// Create an invalid config to trigger a fail.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte("{{invalid"), 0o644))

	report, err := Run(Options{Dir: dir})
	require.NoError(t, err)

	assert.True(t, report.HasFail, "expected HasFail=true for invalid config")
}
