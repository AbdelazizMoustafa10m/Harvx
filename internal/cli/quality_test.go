package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetQualityFlags resets the package-level quality flag variables to their
// defaults. Call this in t.Cleanup after any test that mutates quality flags.
func resetQualityFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		qualityJSON = false
		qualityQuestionsPath = ""
		qualityInitOutput = ".harvx/golden-questions.toml"
		qualityInitYes = false
	})
}

func TestQualityCmd_Registration(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "quality" {
			found = true
			break
		}
	}
	assert.True(t, found, "quality command must be registered on rootCmd")
}

func TestQualityCmd_Alias(t *testing.T) {
	t.Parallel()

	require.Contains(t, qualityCmd.Aliases, "qa",
		"quality command must have 'qa' alias")
}

func TestQualityCmd_Properties(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "quality", qualityCmd.Use)
	assert.NotEmpty(t, qualityCmd.Short, "Short description must not be empty")
	assert.NotEmpty(t, qualityCmd.Long, "Long description must not be empty")
	assert.NotNil(t, qualityCmd.RunE, "RunE must be set")
}

func TestQualityCmd_Flags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{name: "json flag", flag: "json", defValue: "false"},
		{name: "questions flag", flag: "questions", defValue: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := qualityCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "quality command must have --%s flag", tt.flag)
			assert.Equal(t, tt.defValue, f.DefValue,
				"--%s default value mismatch", tt.flag)
		})
	}
}

func TestQualityInitCmd_Registration(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range qualityCmd.Commands() {
		if cmd.Name() == "init" {
			found = true
			break
		}
	}
	assert.True(t, found, "init must be registered as a subcommand of quality")
}

func TestQualityInitCmd_Flags(t *testing.T) {
	t.Parallel()

	f := qualityInitCmd.Flags().Lookup("yes")
	require.NotNil(t, f, "quality init must have --yes flag")
	assert.Equal(t, "false", f.DefValue, "--yes default value must be false")

	o := qualityInitCmd.Flags().Lookup("output")
	require.NotNil(t, o, "quality init must have --output flag")
	assert.Equal(t, ".harvx/golden-questions.toml", o.DefValue, "--output default value mismatch")
}

func TestQualityCmd_HelpContainsExamples(t *testing.T) {
	t.Parallel()

	assert.Contains(t, qualityCmd.Long, "harvx quality",
		"long description should contain example 'harvx quality'")
	assert.Contains(t, qualityCmd.Long, "--questions",
		"long description should mention --questions flag")
	assert.Contains(t, qualityCmd.Long, "--json",
		"long description should mention --json flag")
	assert.Contains(t, qualityCmd.Long, "quality init",
		"long description should mention 'quality init' subcommand")
}

func TestQualityStatusLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "[PASS]", qualityStatusLabel(true))
	assert.Equal(t, "[MISS]", qualityStatusLabel(false))
}

func TestQualityCmd_GlobalFlagInheritance(t *testing.T) {
	// InheritedFlags() modifies cobra internal state (parent flag merging),
	// so this test must not run in parallel with other cobra tests.
	globalFlags := []string{"dir", "profile"}
	for _, name := range globalFlags {
		flag := qualityCmd.InheritedFlags().Lookup(name)
		assert.NotNil(t, flag, "quality must inherit --%s from root", name)
	}
}
