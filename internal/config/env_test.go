package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestBuildEnvMap_Empty verifies that when no HARVX_* vars are set the
// returned map is empty.
func TestBuildEnvMap_Empty(t *testing.T) {
	// Not parallel: mutates environment.
	clearHarvxEnv(t)

	m := buildEnvMap()
	assert.Empty(t, m)
}

// TestBuildEnvMap_Format verifies that HARVX_FORMAT sets the "format" key.
func TestBuildEnvMap_Format(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvFormat, "xml")

	m := buildEnvMap()
	assert.Equal(t, "xml", m["format"])
}

// TestBuildEnvMap_MaxTokens verifies that HARVX_MAX_TOKENS is parsed as an
// integer.
func TestBuildEnvMap_MaxTokens(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvMaxTokens, "200000")

	m := buildEnvMap()
	assert.Equal(t, 200000, m["max_tokens"])
}

// TestBuildEnvMap_MaxTokens_Invalid verifies that a non-numeric
// HARVX_MAX_TOKENS value is silently skipped (not included in the map).
func TestBuildEnvMap_MaxTokens_Invalid(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvMaxTokens, "not-a-number")

	m := buildEnvMap()
	_, ok := m["max_tokens"]
	assert.False(t, ok, "invalid HARVX_MAX_TOKENS must not appear in the map")
}

// TestBuildEnvMap_Tokenizer verifies HARVX_TOKENIZER.
func TestBuildEnvMap_Tokenizer(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvTokenizer, "o200k_base")

	m := buildEnvMap()
	assert.Equal(t, "o200k_base", m["tokenizer"])
}

// TestBuildEnvMap_Output verifies HARVX_OUTPUT.
func TestBuildEnvMap_Output(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvOutput, "my-output.md")

	m := buildEnvMap()
	assert.Equal(t, "my-output.md", m["output"])
}

// TestBuildEnvMap_Target verifies HARVX_TARGET.
func TestBuildEnvMap_Target(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvTarget, "claude")

	m := buildEnvMap()
	assert.Equal(t, "claude", m["target"])
}

// TestBuildEnvMap_Compress verifies HARVX_COMPRESS parses a bool.
func TestBuildEnvMap_Compress(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvCompress, "true")

	m := buildEnvMap()
	assert.Equal(t, true, m["compression"])
}

// TestBuildEnvMap_Compress_False verifies HARVX_COMPRESS=false.
func TestBuildEnvMap_Compress_False(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvCompress, "false")

	m := buildEnvMap()
	assert.Equal(t, false, m["compression"])
}

// TestBuildEnvMap_Compress_Invalid verifies that an invalid bool is skipped.
func TestBuildEnvMap_Compress_Invalid(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvCompress, "maybe")

	m := buildEnvMap()
	_, ok := m["compression"]
	assert.False(t, ok, "invalid HARVX_COMPRESS must not appear in the map")
}

// TestBuildEnvMap_Redact verifies HARVX_REDACT.
func TestBuildEnvMap_Redact(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvRedact, "false")

	m := buildEnvMap()
	assert.Equal(t, false, m["redaction"])
}

// TestBuildEnvMap_LogFormat_NotInMap verifies that HARVX_LOG_FORMAT does not
// appear in the profile map (it is not a profile field).
func TestBuildEnvMap_LogFormat_NotInMap(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvLogFormat, "json")

	m := buildEnvMap()
	_, ok := m["log_format"]
	assert.False(t, ok, "HARVX_LOG_FORMAT must not appear in the profile map")
}

// TestBuildEnvMap_Profile_NotInMap verifies that HARVX_PROFILE does not appear
// in the profile map (it is handled separately during profile selection).
func TestBuildEnvMap_Profile_NotInMap(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvProfile, "myprofile")

	m := buildEnvMap()
	_, ok := m["profile"]
	assert.False(t, ok, "HARVX_PROFILE must not appear in the profile map")
}

// TestBuildEnvMap_AllFields verifies that all supported env vars are read when
// set simultaneously.
func TestBuildEnvMap_AllFields(t *testing.T) {
	clearHarvxEnv(t)

	t.Setenv(EnvFormat, "xml")
	t.Setenv(EnvMaxTokens, "50000")
	t.Setenv(EnvTokenizer, "o200k_base")
	t.Setenv(EnvOutput, "env-output.md")
	t.Setenv(EnvTarget, "chatgpt")
	t.Setenv(EnvCompress, "1")
	t.Setenv(EnvRedact, "0")

	m := buildEnvMap()

	assert.Equal(t, "xml", m["format"])
	assert.Equal(t, 50000, m["max_tokens"])
	assert.Equal(t, "o200k_base", m["tokenizer"])
	assert.Equal(t, "env-output.md", m["output"])
	assert.Equal(t, "chatgpt", m["target"])
	assert.Equal(t, true, m["compression"])
	assert.Equal(t, false, m["redaction"])
}

// clearHarvxEnv unsets all HARVX_* environment variables for the duration of
// the test, restoring them on cleanup via t.Setenv semantics.
func clearHarvxEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		EnvProfile, EnvMaxTokens, EnvFormat, EnvTokenizer,
		EnvOutput, EnvTarget, EnvLogFormat, EnvCompress, EnvRedact,
		"HARVX_VERBOSE", "HARVX_QUIET", "HARVX_NO_REDACT",
		"HARVX_FAIL_ON_REDACTION", "HARVX_STDOUT", "HARVX_DIR",
	} {
		t.Setenv(name, "")
	}
}

// ---------------------------------------------------------------------------
// parseBoolEnv tests (T-069)
// ---------------------------------------------------------------------------

func TestParseBoolEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		// Truthy values
		{"true", true, false},
		{"True", true, false},
		{"TRUE", true, false},
		{"1", true, false},
		{"yes", true, false},
		{"Yes", true, false},
		{"YES", true, false},
		// Falsy values
		{"false", false, false},
		{"False", false, false},
		{"FALSE", false, false},
		{"0", false, false},
		{"no", false, false},
		{"No", false, false},
		{"NO", false, false},
		// With whitespace
		{" true ", true, false},
		{" false ", false, false},
		// Invalid
		{"maybe", false, true},
		{"abc", false, true},
		{"2", false, true},
		{"", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := parseBoolEnv(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// applyEnvOverrides tests (T-069)
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides_Profile(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_PROFILE", "myprofile")

	fv := &FlagValues{Profile: "default"}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	assert.Equal(t, "myprofile", fv.Profile)
}

func TestApplyEnvOverrides_MaxTokens(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_MAX_TOKENS", "50000")

	fv := &FlagValues{}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	assert.Equal(t, 50000, fv.MaxTokens)
}

func TestApplyEnvOverrides_MaxTokens_Invalid(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_MAX_TOKENS", "abc")

	fv := &FlagValues{}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	// Invalid value should be ignored (stays 0).
	assert.Equal(t, 0, fv.MaxTokens)
}

func TestApplyEnvOverrides_Tokenizer(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_TOKENIZER", "o200k_base")

	fv := &FlagValues{Tokenizer: "cl100k_base"}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	assert.Equal(t, "o200k_base", fv.Tokenizer)
}

func TestApplyEnvOverrides_Compress(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_COMPRESS", "yes")

	fv := &FlagValues{}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	assert.True(t, fv.Compress)
}

func TestApplyEnvOverrides_Redact_False(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_REDACT", "false")

	fv := &FlagValues{}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	assert.True(t, fv.NoRedact, "HARVX_REDACT=false should set NoRedact=true")
}

func TestApplyEnvOverrides_Redact_True(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_REDACT", "true")

	fv := &FlagValues{NoRedact: true}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	assert.False(t, fv.NoRedact, "HARVX_REDACT=true should set NoRedact=false")
}

func TestApplyEnvOverrides_Stdout(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_STDOUT", "1")

	fv := &FlagValues{}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	assert.True(t, fv.Stdout)
}

func TestApplyEnvOverrides_Stdout_YesUppercase(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_STDOUT", "YES")

	fv := &FlagValues{}
	cmd := newTestCmd(fv)
	applyEnvOverrides(fv, cmd)
	assert.True(t, fv.Stdout)
}

func TestApplyEnvOverrides_CLIFlagTakesPrecedence(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv("HARVX_FORMAT", "xml")

	fv := &FlagValues{Format: "markdown"}
	cmd := newTestCmd(fv)
	// Simulate the flag being explicitly set on CLI by parsing args.
	cmd.SetArgs([]string{"--format", "markdown"})
	cmd.Execute()
	applyEnvOverrides(fv, cmd)
	assert.Equal(t, "markdown", fv.Format, "CLI flag should take precedence over env var")
}

// newTestCmd creates a minimal cobra.Command with the standard flags registered
// for testing applyEnvOverrides. Flags are registered but not marked as Changed
// unless explicitly set via cmd.Flags().Set().
func newTestCmd(fv *FlagValues) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	pf := cmd.PersistentFlags()
	pf.StringVar(&fv.Dir, "dir", ".", "")
	pf.StringVar(&fv.Output, "output", "", "")
	pf.StringVar(&fv.Format, "format", "markdown", "")
	pf.StringVar(&fv.Target, "target", "generic", "")
	pf.StringVar(&fv.Profile, "profile", "default", "")
	pf.StringVar(&fv.Tokenizer, "tokenizer", "cl100k_base", "")
	pf.IntVar(&fv.MaxTokens, "max-tokens", 0, "")
	pf.BoolVar(&fv.Verbose, "verbose", false, "")
	pf.BoolVar(&fv.Quiet, "quiet", false, "")
	pf.BoolVar(&fv.NoRedact, "no-redact", false, "")
	pf.BoolVar(&fv.FailOnRedaction, "fail-on-redaction", false, "")
	pf.BoolVar(&fv.Compress, "compress", false, "")
	pf.BoolVar(&fv.Stdout, "stdout", false, "")
	return cmd
}
