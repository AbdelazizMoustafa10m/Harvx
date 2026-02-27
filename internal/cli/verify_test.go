package cli

import (
	"bytes"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/workflows"
	"github.com/stretchr/testify/assert"
)

// resetVerifyFlags resets the package-level verify flag variables to their
// defaults. Call this in t.Cleanup after any test that mutates verify flags.
func resetVerifyFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		verifySampleSize = 10
		verifyPaths = nil
		verifyJSON = false
	})
}

func TestVerifyCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "verify" {
			found = true
			break
		}
	}
	assert.True(t, found, "verify command must be registered on root")
}

func TestVerifyCommandProperties(t *testing.T) {
	assert.Equal(t, "verify", verifyCmd.Use)
	assert.NotEmpty(t, verifyCmd.Short, "Short description must not be empty")
	assert.NotEmpty(t, verifyCmd.Long, "Long description must not be empty")
}

func TestVerifyCommandHasFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{name: "sample flag", flag: "sample", defValue: "10"},
		{name: "json flag", flag: "json", defValue: "false"},
		{name: "path flag", flag: "path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := verifyCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f, "verify command must have --%s flag", tt.flag)
			if tt.defValue != "" {
				assert.Equal(t, tt.defValue, f.DefValue,
					"--%s default value mismatch", tt.flag)
			}
		})
	}
}

func TestVerifyCommandInheritsGlobalFlags(t *testing.T) {
	globalFlags := []string{"dir", "output", "profile"}
	for _, name := range globalFlags {
		t.Run(name, func(t *testing.T) {
			flag := verifyCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, flag, "verify must inherit --%s from root", name)
		})
	}
}

func TestVerifyHelp(t *testing.T) {
	resetVerifyFlags(t)
	rootCmd.SetArgs([]string{"verify", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "verify")
	assert.Contains(t, output, "--sample")
	assert.Contains(t, output, "--path")
	assert.Contains(t, output, "--json")

	// Clean up help flag state.
	t.Cleanup(func() {
		if f := verifyCmd.Flags().Lookup("help"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
	})
}

func TestVerifyLongDescriptionContainsExamples(t *testing.T) {
	assert.Contains(t, verifyCmd.Long, "--sample",
		"verify long description should mention --sample")
	assert.Contains(t, verifyCmd.Long, "--path",
		"verify long description should mention --path")
	assert.Contains(t, verifyCmd.Long, "--json",
		"verify long description should mention --json")
	assert.Contains(t, verifyCmd.Long, "--profile",
		"verify long description should mention --profile")
}

func TestVerifyStatusLabel(t *testing.T) {
	tests := []struct {
		name   string
		status workflows.VerifyStatus
		want   string
	}{
		{name: "match", status: workflows.VerifyMatch, want: "[PASS]"},
		{name: "redaction diff", status: workflows.VerifyRedactionDiff, want: "[PASS]"},
		{name: "compression diff", status: workflows.VerifyCompressionDiff, want: "[PASS]"},
		{name: "unexpected diff", status: workflows.VerifyUnexpectedDiff, want: "[WARN]"},
		{name: "file changed", status: workflows.VerifyFileChanged, want: "[WARN]"},
		{name: "unknown status", status: workflows.VerifyStatus("UNKNOWN"), want: "[????]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifyStatusLabel(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero", n: 0, want: "0"},
		{name: "small number", n: 42, want: "42"},
		{name: "three digits", n: 999, want: "999"},
		{name: "one thousand", n: 1000, want: "1,000"},
		{name: "large number", n: 1234567, want: "1,234,567"},
		{name: "negative number", n: -1000, want: "-1,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNumber(tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPluralS(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  string
	}{
		{name: "zero", count: 0, want: "s"},
		{name: "one", count: 1, want: ""},
		{name: "two", count: 2, want: "s"},
		{name: "five", count: 5, want: "s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluralS(tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}
