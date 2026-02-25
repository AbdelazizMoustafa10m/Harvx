package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harvx/harvx/internal/diff"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command Registration Tests ---

func TestCacheCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "cache" {
			found = true
			break
		}
	}
	assert.True(t, found, "cache command must be registered on root")
}

func TestCacheCommandProperties(t *testing.T) {
	assert.Equal(t, "cache", cacheCmd.Use)
	assert.Contains(t, cacheCmd.Short, "Manage")
	assert.NotEmpty(t, cacheCmd.Long)
}

func TestCacheClearSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range cacheCmd.Commands() {
		if cmd.Use == "clear" {
			found = true
			break
		}
	}
	assert.True(t, found, "cache clear subcommand must be registered on cache")
}

func TestCacheShowSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range cacheCmd.Commands() {
		if cmd.Use == "show" {
			found = true
			break
		}
	}
	assert.True(t, found, "cache show subcommand must be registered on cache")
}

// --- Flag Tests ---

func TestCacheClearHasProfileFlag(t *testing.T) {
	flag := cacheClearCmd.Flags().Lookup("profile")
	require.NotNil(t, flag, "cache clear must have --profile flag")
	assert.Equal(t, "", flag.DefValue)
	assert.Equal(t, "p", flag.Shorthand)
}

func TestCacheShowHasJSONFlag(t *testing.T) {
	flag := cacheShowCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "cache show must have --json flag")
	assert.Equal(t, "false", flag.DefValue)
}

// --- Help Output Tests ---

func TestCacheHelpOutput(t *testing.T) {
	rootCmd.SetArgs([]string{"cache", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "cache")
	assert.Contains(t, output, "clear")
	assert.Contains(t, output, "show")
}

func TestCacheClearHelpOutput(t *testing.T) {
	rootCmd.SetArgs([]string{"cache", "clear", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "clear")
	assert.Contains(t, output, "--profile")
}

func TestCacheShowHelpOutput(t *testing.T) {
	rootCmd.SetArgs([]string{"cache", "show", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "show")
	assert.Contains(t, output, "--json")
}

// --- Helper: withFlagValues temporarily overrides flagValues.Dir and Profile ---

func withFlagValues(t *testing.T, dir, profile string) func() {
	t.Helper()
	origDir := flagValues.Dir
	origProfile := flagValues.Profile
	flagValues.Dir = dir
	flagValues.Profile = profile
	return func() {
		flagValues.Dir = origDir
		flagValues.Profile = origProfile
	}
}

// createTestState creates a state snapshot file in .harvx/state/ for testing.
func createTestState(t *testing.T, rootDir, profileName string, fileCount int) {
	t.Helper()

	snap := diff.NewStateSnapshot(profileName, rootDir, "main", "a1b2c3d4e5f6g7h8")
	for i := range fileCount {
		snap.AddFile(
			filepath.Join("src", "file"+string(rune('a'+i))+".go"),
			diff.FileState{
				Size:         int64(100 + i),
				ContentHash:  uint64(0xdeadbeef + i),
				ModifiedTime: "2026-02-15T14:30:00Z",
			},
		)
	}

	cache := diff.NewStateCache(profileName)
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)
}

// --- cache clear Tests ---

func TestCacheClear_AllState(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	// Create state for two profiles.
	createTestState(t, dir, "default", 3)
	createTestState(t, dir, "finvault", 5)

	// Verify state directory exists.
	stateDir := filepath.Join(dir, ".harvx", "state")
	_, err := os.Stat(stateDir)
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	cacheClearCmd.SetOut(buf)
	defer cacheClearCmd.SetOut(nil)

	err = runCacheClear(cacheClearCmd, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Cleared all cached state from .harvx/state/")

	// Verify state directory is gone.
	_, err = os.Stat(stateDir)
	assert.True(t, os.IsNotExist(err), "state directory should be removed")
}

func TestCacheClear_SpecificProfile(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	// Create state for two profiles.
	createTestState(t, dir, "default", 3)
	createTestState(t, dir, "finvault", 5)

	buf := new(bytes.Buffer)
	cacheClearCmd.SetOut(buf)
	defer cacheClearCmd.SetOut(nil)

	// Set the profile flag.
	cacheClearCmd.Flags().Set("profile", "finvault")
	defer cacheClearCmd.Flags().Set("profile", "")

	err := runCacheClear(cacheClearCmd, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Cleared cached state for profile 'finvault'")

	// finvault state should be gone.
	finvaultCache := diff.NewStateCache("finvault")
	assert.False(t, finvaultCache.HasState(dir), "finvault state should be removed")

	// default state should still exist.
	defaultCache := diff.NewStateCache("default")
	assert.True(t, defaultCache.HasState(dir), "default state should still exist")
}

func TestCacheClear_NoStateExists(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	buf := new(bytes.Buffer)
	cacheClearCmd.SetOut(buf)
	defer cacheClearCmd.SetOut(nil)

	// Reset profile flag to empty.
	cacheClearCmd.Flags().Set("profile", "")

	err := runCacheClear(cacheClearCmd, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "No cached state found.")
}

func TestCacheClear_NoStateForProfile(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	buf := new(bytes.Buffer)
	cacheClearCmd.SetOut(buf)
	defer cacheClearCmd.SetOut(nil)

	cacheClearCmd.Flags().Set("profile", "nonexistent")
	defer cacheClearCmd.Flags().Set("profile", "")

	err := runCacheClear(cacheClearCmd, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "No cached state found.")
}

// --- cache show Tests ---

func TestCacheShow_NoState(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	// Ensure json flag is false.
	cacheShowCmd.Flags().Set("json", "false")

	err := runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "No cached state found. Run 'harvx generate' to create state.")
}

func TestCacheShow_TableFormat(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	createTestState(t, dir, "default", 3)
	createTestState(t, dir, "finvault", 5)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "false")

	err := runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Cached State Summary (.harvx/state/):")
	assert.Contains(t, output, "PROFILE")
	assert.Contains(t, output, "GENERATED")
	assert.Contains(t, output, "BRANCH")
	assert.Contains(t, output, "HEAD")
	assert.Contains(t, output, "FILES")
	assert.Contains(t, output, "default")
	assert.Contains(t, output, "finvault")
	assert.Contains(t, output, "main")
	assert.Contains(t, output, "a1b2c3d") // Truncated SHA
	assert.Contains(t, output, "Total: 2 profiles cached")
}

func TestCacheShow_TableSortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	// Create profiles in reverse alphabetical order.
	createTestState(t, dir, "zeta", 1)
	createTestState(t, dir, "alpha", 2)
	createTestState(t, dir, "middle", 3)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "false")

	err := runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	output := buf.String()

	// Check that profiles appear in alphabetical order.
	alphaIdx := bytes.Index([]byte(output), []byte("alpha"))
	middleIdx := bytes.Index([]byte(output), []byte("middle"))
	zetaIdx := bytes.Index([]byte(output), []byte("zeta"))

	assert.Greater(t, middleIdx, alphaIdx, "alpha should appear before middle")
	assert.Greater(t, zetaIdx, middleIdx, "middle should appear before zeta")
	assert.Contains(t, output, "Total: 3 profiles cached")
}

func TestCacheShow_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	createTestState(t, dir, "default", 3)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "true")
	defer cacheShowCmd.Flags().Set("json", "false")

	err := runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	var result cacheShowOutput
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "cache show --json must output valid JSON")

	assert.Equal(t, filepath.Join(".harvx", "state"), result.CacheDir)
	require.Len(t, result.Profiles, 1)

	p := result.Profiles[0]
	assert.Equal(t, "default", p.Name)
	assert.NotEmpty(t, p.GeneratedAt)
	assert.Equal(t, "main", p.GitBranch)
	assert.Equal(t, "a1b2c3d", p.GitHeadSHA) // Truncated to 7 chars
	assert.Equal(t, 3, p.FileCount)
	assert.Equal(t, filepath.Join(".harvx", "state", "default.json"), p.StateFile)
}

func TestCacheShow_JSONMultipleProfiles(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	createTestState(t, dir, "alpha", 2)
	createTestState(t, dir, "beta", 4)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "true")
	defer cacheShowCmd.Flags().Set("json", "false")

	err := runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	var result cacheShowOutput
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	require.Len(t, result.Profiles, 2)
	assert.Equal(t, "alpha", result.Profiles[0].Name)
	assert.Equal(t, "beta", result.Profiles[1].Name)
	assert.Equal(t, 2, result.Profiles[0].FileCount)
	assert.Equal(t, 4, result.Profiles[1].FileCount)
}

func TestCacheShow_JSONHasExpectedKeys(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	createTestState(t, dir, "default", 1)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "true")
	defer cacheShowCmd.Flags().Set("json", "false")

	err := runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "cache_dir")
	assert.Contains(t, raw, "profiles")

	profiles, ok := raw["profiles"].([]interface{})
	require.True(t, ok)
	require.Len(t, profiles, 1)

	profile, ok := profiles[0].(map[string]interface{})
	require.True(t, ok)

	expectedKeys := []string{"name", "generated_at", "git_branch", "git_head_sha", "file_count", "state_file"}
	for _, key := range expectedKeys {
		assert.Contains(t, profile, key, "profile JSON must contain key: %s", key)
	}
}

func TestCacheShow_EmptyStateDirectory(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	// Create the state directory but put no JSON files in it.
	stateDir := filepath.Join(dir, ".harvx", "state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "false")

	err = runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "No cached state found. Run 'harvx generate' to create state.")
}

func TestCacheShow_SkipsNonJSONFiles(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	createTestState(t, dir, "default", 2)

	// Create a non-JSON file that should be skipped.
	stateDir := filepath.Join(dir, ".harvx", "state")
	err := os.WriteFile(filepath.Join(stateDir, "readme.txt"), []byte("not a state file"), 0644)
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "true")
	defer cacheShowCmd.Flags().Set("json", "false")

	err = runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	var result cacheShowOutput
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Only the valid state file should appear.
	require.Len(t, result.Profiles, 1)
	assert.Equal(t, "default", result.Profiles[0].Name)
}

func TestCacheShow_SkipsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	createTestState(t, dir, "valid", 1)

	// Create a malformed JSON state file.
	stateDir := filepath.Join(dir, ".harvx", "state")
	err := os.WriteFile(filepath.Join(stateDir, "broken.json"), []byte("{invalid json"), 0644)
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "true")
	defer cacheShowCmd.Flags().Set("json", "false")

	err = runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	var result cacheShowOutput
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Only the valid state file should appear.
	require.Len(t, result.Profiles, 1)
	assert.Equal(t, "valid", result.Profiles[0].Name)
}

func TestCacheShow_SHATruncation(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	createTestState(t, dir, "default", 1)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "true")
	defer cacheShowCmd.Flags().Set("json", "false")

	err := runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	var result cacheShowOutput
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	require.Len(t, result.Profiles, 1)
	assert.Len(t, result.Profiles[0].GitHeadSHA, 7, "HEAD SHA should be truncated to 7 characters")
}

func TestCacheShow_ShortSHANotTruncated(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	// Create a snapshot with a short SHA (less than 7 chars).
	snap := diff.NewStateSnapshot("default", dir, "main", "abc")
	snap.AddFile("test.go", diff.FileState{Size: 42, ContentHash: 0xbeef, ModifiedTime: "2026-02-15T14:30:00Z"})
	cache := diff.NewStateCache("default")
	err := cache.SaveState(dir, snap)
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "true")
	defer cacheShowCmd.Flags().Set("json", "false")

	err = runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	var result cacheShowOutput
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	require.Len(t, result.Profiles, 1)
	assert.Equal(t, "abc", result.Profiles[0].GitHeadSHA, "short SHA should not be truncated")
}

func TestCacheShow_GeneratedAtFormatting(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	// Create a snapshot with a known timestamp to verify formatting.
	snap := diff.NewStateSnapshot("default", dir, "main", "a1b2c3d4e5f6g7h8")
	snap.GeneratedAt = "2026-02-15T14:30:00Z"
	snap.AddFile("test.go", diff.FileState{Size: 42, ContentHash: 0xbeef, ModifiedTime: "2026-02-15T14:30:00Z"})
	cache := diff.NewStateCache("default")
	err := cache.SaveState(dir, snap)
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	cacheShowCmd.SetOut(buf)
	defer cacheShowCmd.SetOut(nil)

	cacheShowCmd.Flags().Set("json", "false")

	err = runCacheShow(cacheShowCmd, nil)
	require.NoError(t, err)

	output := buf.String()
	// The table output should format RFC3339 as "2006-01-02 15:04:05".
	assert.Contains(t, output, "2026-02-15 14:30:00", "table should contain formatted timestamp")
	assert.NotContains(t, output, "T14:30:00Z", "table should not contain raw RFC3339 format")
}

// --- cache clear respects -d flag Tests ---

func TestCacheClear_RespectsDir(t *testing.T) {
	dir := t.TempDir()
	cleanup := withFlagValues(t, dir, "default")
	defer cleanup()

	createTestState(t, dir, "default", 2)

	// Verify state exists.
	cache := diff.NewStateCache("default")
	assert.True(t, cache.HasState(dir))

	buf := new(bytes.Buffer)
	cacheClearCmd.SetOut(buf)
	defer cacheClearCmd.SetOut(nil)

	cacheClearCmd.Flags().Set("profile", "")

	err := runCacheClear(cacheClearCmd, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Cleared all cached state from .harvx/state/")
	assert.False(t, cache.HasState(dir))
}

// --- formatGeneratedAt Tests ---

func TestFormatGeneratedAt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid RFC3339",
			input: "2026-02-15T14:30:00Z",
			want:  "2026-02-15 14:30:00",
		},
		{
			name:  "RFC3339 with timezone offset",
			input: "2026-02-15T14:30:00+05:30",
			want:  "2026-02-15 14:30:00",
		},
		{
			name:  "invalid string returns raw",
			input: "not-a-date",
			want:  "not-a-date",
		},
		{
			name:  "empty string returns empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatGeneratedAt(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- stateBasePath Tests ---

func TestStateBasePath(t *testing.T) {
	got := stateBasePath("/tmp/project")
	want := filepath.Join("/tmp/project", ".harvx", "state")
	assert.Equal(t, want, got)
}

// --- Inherited Global Flags ---

func TestCacheSubcommandsInheritGlobalFlags(t *testing.T) {
	globalFlags := []string{"dir", "verbose", "quiet"}
	for _, name := range globalFlags {
		t.Run("clear/"+name, func(t *testing.T) {
			flag := cacheClearCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, flag, "cache clear must inherit --%s from root", name)
		})
		t.Run("show/"+name, func(t *testing.T) {
			flag := cacheShowCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, flag, "cache show must inherit --%s from root", name)
		})
	}
}
