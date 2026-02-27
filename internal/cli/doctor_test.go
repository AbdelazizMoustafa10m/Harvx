package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "doctor" {
			found = true
			break
		}
	}
	assert.True(t, found, "doctor command should be registered on root")
}

func TestDoctorCmd_HasFlags(t *testing.T) {
	jsonFlag := doctorCmd.Flags().Lookup("json")
	require.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)

	fixFlag := doctorCmd.Flags().Lookup("fix")
	require.NotNil(t, fixFlag)
	assert.Equal(t, "false", fixFlag.DefValue)
}

func TestDoctorCmd_TextOutput(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(nil)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetErr(nil)
	rootCmd.SetArgs([]string{"doctor", "--dir", dir})
	defer rootCmd.SetArgs(nil)
	t.Cleanup(func() { flagValues.Dir = "." })

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Harvx Doctor")
	assert.Contains(t, output, "[PASS]")
}

func TestDoctorCmd_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(nil)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetErr(nil)
	rootCmd.SetArgs([]string{"doctor", "--json", "--dir", dir})
	defer rootCmd.SetArgs(nil)
	t.Cleanup(func() { flagValues.Dir = "." })

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"directory"`)
	assert.Contains(t, output, `"checks"`)
}
