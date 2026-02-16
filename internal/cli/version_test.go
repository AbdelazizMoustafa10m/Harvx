package cli

import (
	"bytes"
	"encoding/json"
	"runtime"
	"testing"

	"github.com/harvx/harvx/internal/buildinfo"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			found = true
			break
		}
	}
	assert.True(t, found, "version subcommand must be registered on root command")
}

func TestVersionCommandProperties(t *testing.T) {
	assert.Equal(t, "version", versionCmd.Use)
	assert.Equal(t, "Show version and build information", versionCmd.Short)
}

func TestVersionCommandHasJSONFlag(t *testing.T) {
	flag := versionCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "version command must have --json flag")
	assert.Equal(t, "false", flag.DefValue)
}

func TestVersionHumanOutput(t *testing.T) {
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "harvx version")
	assert.Contains(t, output, "commit:")
	assert.Contains(t, output, "built:")
	assert.Contains(t, output, "go version:")
	assert.Contains(t, output, "os/arch:")
}

func TestVersionHumanOutputContainsOSArch(t *testing.T) {
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, runtime.GOOS)
	assert.Contains(t, output, runtime.GOARCH)
}

func TestVersionHumanOutputContainsDefaultValues(t *testing.T) {
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, buildinfo.Version)
	assert.Contains(t, output, buildinfo.Commit)
	assert.Contains(t, output, buildinfo.Date)
}

func TestVersionJSONOutput(t *testing.T) {
	rootCmd.SetArgs([]string{"version", "--json"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	var info versionInfo
	err := json.Unmarshal(buf.Bytes(), &info)
	require.NoError(t, err, "version --json must output valid JSON")

	assert.Equal(t, buildinfo.Version, info.Version)
	assert.Equal(t, buildinfo.Commit, info.Commit)
	assert.Equal(t, buildinfo.Date, info.Date)
	assert.Equal(t, buildinfo.GoVersion, info.GoVersion)
	assert.Equal(t, runtime.GOOS, info.OS)
	assert.Equal(t, runtime.GOARCH, info.Arch)
}

func TestVersionJSONHasExpectedKeys(t *testing.T) {
	rootCmd.SetArgs([]string{"version", "--json"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	var raw map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &raw)
	require.NoError(t, err)

	expectedKeys := []string{"version", "commit", "date", "goVersion", "os", "arch"}
	for _, key := range expectedKeys {
		assert.Contains(t, raw, key, "JSON output must contain key: %s", key)
	}
	assert.Len(t, raw, len(expectedKeys), "JSON output should have exactly %d keys", len(expectedKeys))
}

func TestVersionInfoStruct(t *testing.T) {
	info := versionInfo{
		Version:   "1.0.0",
		Commit:    "abc1234",
		Date:      "2026-02-16T10:00:00Z",
		GoVersion: "go1.24.0",
		OS:        "linux",
		Arch:      "amd64",
	}

	data, err := json.Marshal(info)
	require.NoError(t, err)

	var decoded versionInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, info, decoded)
}
