package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCommand creates a fresh Cobra command with flags bound for testing.
// Using a fresh command avoids shared state between tests.
func newTestCommand() (*cobra.Command, *FlagValues) {
	cmd := &cobra.Command{
		Use:           "test",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	fv := BindFlags(cmd)
	return cmd, fv
}

func TestFlagDefaults(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	assert.Equal(t, ".", fv.Dir)
	assert.Equal(t, DefaultOutput, fv.Output)
	assert.Nil(t, fv.Filters)
	assert.Nil(t, fv.Includes)
	assert.Nil(t, fv.Excludes)
	assert.Equal(t, "markdown", fv.Format)
	assert.Equal(t, "generic", fv.Target)
	assert.False(t, fv.GitTrackedOnly)
	assert.False(t, fv.Stdout)
	assert.False(t, fv.LineNumbers)
	assert.False(t, fv.NoRedact)
	assert.False(t, fv.FailOnRedaction)
	assert.False(t, fv.Verbose)
	assert.False(t, fv.Quiet)
	assert.False(t, fv.Yes)
	assert.False(t, fv.ClearCache)
}

func TestVerboseQuietMutualExclusion(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--verbose", "--quiet"})
	require.NoError(t, cmd.Execute())

	// Both flags are set; validation should catch this.
	err := ValidateFlags(fv, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestDirNonExistentPath(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--dir", "/nonexistent/path/that/does/not/exist"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--dir")
}

func TestDirNotADirectory(t *testing.T) {
	// Create a temporary file (not a directory).
	tmp := t.TempDir()
	f := filepath.Join(tmp, "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("hello"), 0o644))

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--dir", f})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestDirValidDirectory(t *testing.T) {
	tmp := t.TempDir()

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--dir", tmp})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, tmp, fv.Dir)
}

func TestFormatInvalid(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--format", "xyz"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--format")
	assert.Contains(t, err.Error(), "xyz")
}

func TestFormatValidValues(t *testing.T) {
	tests := []string{"markdown", "xml"}
	for _, format := range tests {
		t.Run(format, func(t *testing.T) {
			cmd, fv := newTestCommand()
			cmd.SetArgs([]string{"--format", format})
			require.NoError(t, cmd.Execute())

			err := ValidateFlags(fv, cmd)
			require.NoError(t, err)
			assert.Equal(t, format, fv.Format)
		})
	}
}

func TestTargetInvalid(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--target", "xyz"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--target")
	assert.Contains(t, err.Error(), "xyz")
}

func TestTargetValidValues(t *testing.T) {
	tests := []string{"claude", "chatgpt", "generic"}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			cmd, fv := newTestCommand()
			cmd.SetArgs([]string{"--target", target})
			require.NoError(t, cmd.Execute())

			err := ValidateFlags(fv, cmd)
			require.NoError(t, err)
			assert.Equal(t, target, fv.Target)
		})
	}
}

func TestFilterStripsDots(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"-f", ".ts"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	require.Len(t, fv.Filters, 1)
	assert.Equal(t, "ts", fv.Filters[0])
}

func TestFilterMultipleValues(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"-f", "ts", "-f", "go"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	require.Len(t, fv.Filters, 2)
	assert.Equal(t, "ts", fv.Filters[0])
	assert.Equal(t, "go", fv.Filters[1])
}

func TestFilterMixedDotsAndBare(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"-f", ".py", "-f", "rs"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	require.Len(t, fv.Filters, 2)
	assert.Equal(t, "py", fv.Filters[0])
	assert.Equal(t, "rs", fv.Filters[1])
}

func TestSkipLargeFilesDefault(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, int64(1*1024*1024), fv.SkipLargeFiles)
}

func TestEnvDirOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HARVX_DIR", tmp)

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, tmp, fv.Dir)
}

func TestExplicitFlagOverridesEnv(t *testing.T) {
	tmp1 := t.TempDir()
	tmp2 := t.TempDir()
	t.Setenv("HARVX_DIR", tmp1)

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--dir", tmp2})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, tmp2, fv.Dir, "explicit --dir flag should override HARVX_DIR env var")
}

func TestEnvVerboseOverride(t *testing.T) {
	t.Setenv("HARVX_VERBOSE", "1")

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	// Need to call ValidateFlags which applies env overrides
	// First reset skipLargeFilesRaw since tests share state
	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.True(t, fv.Verbose)
}

func TestEnvFormatOverride(t *testing.T) {
	t.Setenv("HARVX_FORMAT", "xml")

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, "xml", fv.Format)
}

func TestEnvTargetOverride(t *testing.T) {
	t.Setenv("HARVX_TARGET", "claude")

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, "claude", fv.Target)
}

func TestIncludeExcludePatterns(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{
		"--include", "**/*.go",
		"--include", "**/*.ts",
		"--exclude", "vendor/**",
	})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*.go", "**/*.ts"}, fv.Includes)
	assert.Equal(t, []string{"vendor/**"}, fv.Excludes)
}

func TestBooleanFlags(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{
		"--git-tracked-only",
		"--stdout",
		"--line-numbers",
		"--no-redact",
		"--fail-on-redaction",
		"--yes",
		"--clear-cache",
	})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)

	assert.True(t, fv.GitTrackedOnly)
	assert.True(t, fv.Stdout)
	assert.True(t, fv.LineNumbers)
	assert.True(t, fv.NoRedact)
	assert.True(t, fv.FailOnRedaction)
	assert.True(t, fv.Yes)
	assert.True(t, fv.ClearCache)
}

// --- ParseSize tests ---

func TestParseSizeKB(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"500KB", 500 * 1024},
		{"500kb", 500 * 1024},
		{"500Kb", 500 * 1024},
		{"1KB", 1024},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSize(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSizeMB(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1MB", 1 * 1024 * 1024},
		{"2MB", 2 * 1024 * 1024},
		{"1mb", 1 * 1024 * 1024},
		{"2mb", 2 * 1024 * 1024},
		{"1Mb", 1 * 1024 * 1024},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSize(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSizeGB(t *testing.T) {
	result, err := ParseSize("1GB")
	require.NoError(t, err)
	assert.Equal(t, int64(1024*1024*1024), result)
}

func TestParseSizePlainBytes(t *testing.T) {
	result, err := ParseSize("4096")
	require.NoError(t, err)
	assert.Equal(t, int64(4096), result)
}

func TestParseSizeEmpty(t *testing.T) {
	_, err := ParseSize("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseSizeInvalid(t *testing.T) {
	_, err := ParseSize("abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid size")
}

func TestParseSizeNegative(t *testing.T) {
	_, err := ParseSize("-5MB")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-negative")
}

func TestSkipLargeFiles500KB(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--skip-large-files", "500KB"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, int64(500*1024), fv.SkipLargeFiles)
}

func TestSkipLargeFiles2MB(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--skip-large-files", "2MB"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, int64(2*1024*1024), fv.SkipLargeFiles)
}

func TestSkipLargeFilesLowercase(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--skip-large-files", "1mb"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, int64(1*1024*1024), fv.SkipLargeFiles)
}

// --- T-033: Token counting flag tests ---

func TestTokenizerDefault(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, "cl100k_base", fv.Tokenizer)
}

func TestTokenizerValidValues(t *testing.T) {
	tests := []string{"cl100k_base", "o200k_base", "none"}
	for _, enc := range tests {
		t.Run(enc, func(t *testing.T) {
			cmd, fv := newTestCommand()
			cmd.SetArgs([]string{"--tokenizer", enc})
			require.NoError(t, cmd.Execute())

			err := ValidateFlags(fv, cmd)
			require.NoError(t, err)
			assert.Equal(t, enc, fv.Tokenizer)
		})
	}
}

func TestTokenizerInvalidValue(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--tokenizer", "gpt2"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--tokenizer")
	assert.Contains(t, err.Error(), "gpt2")
}

func TestMaxTokensDefault(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, 0, fv.MaxTokens)
}

func TestMaxTokensExplicit(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--max-tokens", "200000"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, 200000, fv.MaxTokens)
}

func TestTruncationStrategyDefault(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, "skip", fv.TruncationStrategy)
}

func TestTruncationStrategyValidValues(t *testing.T) {
	tests := []string{"truncate", "skip"}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			cmd, fv := newTestCommand()
			cmd.SetArgs([]string{"--truncation-strategy", s})
			require.NoError(t, cmd.Execute())

			err := ValidateFlags(fv, cmd)
			require.NoError(t, err)
			assert.Equal(t, s, fv.TruncationStrategy)
		})
	}
}

func TestTruncationStrategyInvalidValue(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--truncation-strategy", "head"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--truncation-strategy")
	assert.Contains(t, err.Error(), "head")
}

func TestTokenCountOnlyDefault(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.False(t, fv.TokenCountOnly)
}

func TestTokenCountOnlyFlag(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--token-count"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.True(t, fv.TokenCountOnly)
}

func TestTopFilesDefault(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, 0, fv.TopFiles)
}

func TestTopFilesExplicit(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--top-files", "20"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, 20, fv.TopFiles)
}

// --- T-040: Redaction flags ---

func TestRedactionReportFlagDefault(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, "", fv.RedactionReport)
}

func TestRedactionReportFlagExplicit(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--redaction-report", "my-report.json"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, "my-report.json", fv.RedactionReport)
}

func TestEnvNoRedactOverride(t *testing.T) {
	t.Setenv("HARVX_NO_REDACT", "1")

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.True(t, fv.NoRedact)
}

func TestEnvNoRedactNotOverriddenByExplicitFlag(t *testing.T) {
	t.Setenv("HARVX_NO_REDACT", "1")

	cmd, fv := newTestCommand()
	// Explicit --no-redact=false should override env (flag wins because it was explicitly set).
	// In this test, the explicit flag sets it to true which aligns with env so just test env works.
	cmd.SetArgs([]string{"--no-redact"})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.True(t, fv.NoRedact)
}

func TestEnvFailOnRedactionOverride(t *testing.T) {
	t.Setenv("HARVX_FAIL_ON_REDACTION", "1")

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.True(t, fv.FailOnRedaction)
}

func TestNoRedactAndFailOnRedactionBothSet_NoError(t *testing.T) {
	// Both flags set should produce a warning (via slog), NOT an error.
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--no-redact", "--fail-on-redaction"})
	require.NoError(t, cmd.Execute())

	err := ValidateFlags(fv, cmd)
	// No error expected -- only a slog warning
	require.NoError(t, err)
	assert.True(t, fv.NoRedact)
	assert.True(t, fv.FailOnRedaction)
}

// --- T-040: additional coverage ---

// TestRedactionFlags_TableDriven exercises all redaction-related boolean flags
// and their defaults in a single table-driven test.
func TestRedactionFlags_TableDriven(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		wantNoRedact      bool
		wantFailOnRedact  bool
		wantRedactReport  string
	}{
		{
			name:             "defaults: all redaction flags off",
			args:             []string{},
			wantNoRedact:     false,
			wantFailOnRedact: false,
			wantRedactReport: "",
		},
		{
			name:             "no-redact only",
			args:             []string{"--no-redact"},
			wantNoRedact:     true,
			wantFailOnRedact: false,
			wantRedactReport: "",
		},
		{
			name:             "fail-on-redaction only",
			args:             []string{"--fail-on-redaction"},
			wantNoRedact:     false,
			wantFailOnRedact: true,
			wantRedactReport: "",
		},
		{
			name:             "redaction-report with explicit path",
			args:             []string{"--redaction-report", "report.json"},
			wantNoRedact:     false,
			wantFailOnRedact: false,
			wantRedactReport: "report.json",
		},
		{
			name:             "both no-redact and fail-on-redaction",
			args:             []string{"--no-redact", "--fail-on-redaction"},
			wantNoRedact:     true,
			wantFailOnRedact: true,
			wantRedactReport: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cmd, fv := newTestCommand()
			cmd.SetArgs(tt.args)
			require.NoError(t, cmd.Execute())

			skipLargeFilesRaw = "1MB"
			err := ValidateFlags(fv, cmd)
			require.NoError(t, err)

			assert.Equal(t, tt.wantNoRedact, fv.NoRedact, "NoRedact mismatch")
			assert.Equal(t, tt.wantFailOnRedact, fv.FailOnRedaction, "FailOnRedaction mismatch")
			assert.Equal(t, tt.wantRedactReport, fv.RedactionReport, "RedactionReport mismatch")
		})
	}
}

// TestEnvNoRedactValues exercises different truthy and falsy values for
// HARVX_NO_REDACT to verify that only "1" activates the flag.
func TestEnvNoRedactValues(t *testing.T) {
	tests := []struct {
		envVal       string
		wantNoRedact bool
	}{
		{envVal: "1", wantNoRedact: true},
		{envVal: "0", wantNoRedact: false},
		{envVal: "true", wantNoRedact: false}, // only "1" is accepted
		{envVal: "yes", wantNoRedact: false},
		{envVal: "", wantNoRedact: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("HARVX_NO_REDACT="+tt.envVal, func(t *testing.T) {
			t.Setenv("HARVX_NO_REDACT", tt.envVal)

			cmd, fv := newTestCommand()
			cmd.SetArgs([]string{})
			require.NoError(t, cmd.Execute())

			skipLargeFilesRaw = "1MB"
			err := ValidateFlags(fv, cmd)
			require.NoError(t, err)

			assert.Equal(t, tt.wantNoRedact, fv.NoRedact,
				"HARVX_NO_REDACT=%q: unexpected NoRedact value", tt.envVal)
		})
	}
}

// TestEnvFailOnRedactionValues exercises different truthy and falsy values for
// HARVX_FAIL_ON_REDACTION.
func TestEnvFailOnRedactionValues(t *testing.T) {
	tests := []struct {
		envVal          string
		wantFailOnRedact bool
	}{
		{envVal: "1", wantFailOnRedact: true},
		{envVal: "0", wantFailOnRedact: false},
		{envVal: "true", wantFailOnRedact: false},
		{envVal: "", wantFailOnRedact: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("HARVX_FAIL_ON_REDACTION="+tt.envVal, func(t *testing.T) {
			t.Setenv("HARVX_FAIL_ON_REDACTION", tt.envVal)

			cmd, fv := newTestCommand()
			cmd.SetArgs([]string{})
			require.NoError(t, cmd.Execute())

			skipLargeFilesRaw = "1MB"
			err := ValidateFlags(fv, cmd)
			require.NoError(t, err)

			assert.Equal(t, tt.wantFailOnRedact, fv.FailOnRedaction,
				"HARVX_FAIL_ON_REDACTION=%q: unexpected FailOnRedaction value", tt.envVal)
		})
	}
}

// TestExplicitFlagOverridesEnvForNoRedact verifies that an explicit --no-redact
// CLI flag takes precedence over the HARVX_NO_REDACT env var. When the explicit
// flag is set (to true), the env var value is irrelevant; the flag wins.
func TestExplicitFlagOverridesEnvForNoRedact(t *testing.T) {
	// Set env to "0" (falsy) and also pass --no-redact explicitly.
	// The explicit flag should win and NoRedact should be true.
	t.Setenv("HARVX_NO_REDACT", "0")

	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--no-redact"})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.True(t, fv.NoRedact,
		"explicit --no-redact must win regardless of HARVX_NO_REDACT env value")
}

// TestRedactionReportFlagWithCustomPath exercises the --redaction-report flag
// with several custom path values.
func TestRedactionReportFlagWithCustomPath(t *testing.T) {
	tests := []struct {
		name       string
		reportPath string
	}{
		{name: "simple filename", reportPath: "my-report.json"},
		{name: "relative subdirectory", reportPath: "reports/redact.json"},
		{name: "txt extension", reportPath: "out/redact.txt"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cmd, fv := newTestCommand()
			cmd.SetArgs([]string{"--redaction-report", tt.reportPath})
			require.NoError(t, cmd.Execute())

			skipLargeFilesRaw = "1MB"
			err := ValidateFlags(fv, cmd)
			require.NoError(t, err)

			assert.Equal(t, tt.reportPath, fv.RedactionReport)
		})
	}
}

// TestNoRedactFlag_DefaultIsFalse verifies that --no-redact defaults to false,
// meaning redaction is enabled by default.
func TestNoRedactFlag_DefaultIsFalse(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)

	assert.False(t, fv.NoRedact,
		"redaction must be enabled by default (NoRedact=false)")
}

// TestFailOnRedactionFlag_DefaultIsFalse verifies that --fail-on-redaction
// defaults to false (CI enforcement mode is opt-in).
func TestFailOnRedactionFlag_DefaultIsFalse(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)

	assert.False(t, fv.FailOnRedaction,
		"--fail-on-redaction must default to false (CI mode is opt-in)")
}

// --- T-056: Split flag tests ---

func TestSplitFlagDefault(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, 0, fv.Split)
}

func TestSplitFlagExplicit(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--split", "50000"})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, 50000, fv.Split)
}

func TestSplitFlagNegativeRejected(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--split", "-100"})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--split")
	assert.Contains(t, err.Error(), "non-negative")
}

func TestSplitFlagZeroIsValid(t *testing.T) {
	cmd, fv := newTestCommand()
	cmd.SetArgs([]string{"--split", "0"})
	require.NoError(t, cmd.Execute())

	skipLargeFilesRaw = "1MB"
	err := ValidateFlags(fv, cmd)
	require.NoError(t, err)
	assert.Equal(t, 0, fv.Split)
}
