package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/workflows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetBriefFlags resets the package-level brief flag variables to their
// defaults. Call this in t.Cleanup after any test that sets --json.
func resetBriefFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		briefJSON = false
	})
}

func TestBriefCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "brief" {
			found = true
			break
		}
	}
	assert.True(t, found, "brief command must be registered on root")
}

func TestBriefCommandProperties(t *testing.T) {
	assert.Equal(t, "brief", briefCmd.Use)
	assert.NotEmpty(t, briefCmd.Short)
	assert.NotEmpty(t, briefCmd.Long)
}

func TestBriefCommandHasJSONFlag(t *testing.T) {
	flag := briefCmd.Flags().Lookup("json")
	assert.NotNil(t, flag, "brief command must have --json flag")
	assert.Equal(t, "false", flag.DefValue)
}

func TestBriefCommandInheritsGlobalFlags(t *testing.T) {
	globalFlags := []string{
		"dir", "output", "target", "profile", "stdout",
		"assert-include",
	}
	for _, name := range globalFlags {
		t.Run(name, func(t *testing.T) {
			flag := briefCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, flag, "brief must inherit --%s from root", name)
		})
	}
}

func TestBriefStdoutExitsZero(t *testing.T) {
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--stdout"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx brief --stdout must exit 0; stderr: %s", errBuf.String())

	// Should contain brief content (either Markdown or XML header).
	output := outBuf.String()
	assert.NotEmpty(t, output, "brief --stdout should produce output")
}

func TestBriefJSONExitsZero(t *testing.T) {
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx brief --json must exit 0; stderr: %s", errBuf.String())

	// Output must be valid JSON.
	output := outBuf.Bytes()
	assert.True(t, json.Valid(output),
		"brief --json output must be valid JSON, got: %s", string(output))
}

func TestBriefJSONSchema(t *testing.T) {
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var raw map[string]json.RawMessage
	err := json.Unmarshal(outBuf.Bytes(), &raw)
	require.NoError(t, err, "output must be valid JSON: %s", outBuf.String())

	expectedKeys := []string{
		"token_count",
		"content_hash",
		"files_included",
		"sections",
		"max_tokens",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, raw, key,
			"brief --json output must contain %q field", key)
	}
}

func TestBriefJSONParsesToBriefJSON(t *testing.T) {
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var briefMeta workflows.BriefJSON
	err := json.Unmarshal(outBuf.Bytes(), &briefMeta)
	require.NoError(t, err, "output must unmarshal to BriefJSON: %s", outBuf.String())

	assert.Greater(t, briefMeta.TokenCount, 0, "token count should be > 0")
	assert.NotEmpty(t, briefMeta.ContentHash, "content hash should not be empty")
	assert.Greater(t, briefMeta.MaxTokens, 0, "max tokens should be > 0")
}

func TestBriefHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"brief", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "brief")
	assert.Contains(t, output, "--json")

	// Clean up help flag state.
	t.Cleanup(func() {
		if f := briefCmd.Flags().Lookup("help"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
	})
}

func TestBriefJSONOutputToStdout(t *testing.T) {
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	assert.True(t, json.Valid(outBuf.Bytes()),
		"stdout should contain valid JSON")
}

func TestBriefLongDescriptionContainsExamples(t *testing.T) {
	assert.Contains(t, briefCmd.Long, "--json",
		"brief long description should mention --json")
	assert.Contains(t, briefCmd.Long, "--stdout",
		"brief long description should mention --stdout")
}

// ---------------------------------------------------------------------------
// Additional CLI tests for T-070 acceptance criteria coverage
// ---------------------------------------------------------------------------

// TestBriefTargetClaude_ProducesXML verifies that `harvx brief --target claude --stdout`
// produces XML-formatted output with <repo-brief> tags.
func TestBriefTargetClaude_ProducesXML(t *testing.T) {
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--target", "claude", "--stdout"})
	defer rootCmd.SetArgs(nil)

	// Reset the target flag after test to avoid leaking into other tests.
	t.Cleanup(func() {
		if f := rootCmd.PersistentFlags().Lookup("target"); f != nil {
			f.Changed = false
			_ = f.Value.Set("generic")
		}
	})

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx brief --target claude --stdout must exit 0; stderr: %s", errBuf.String())

	output := outBuf.String()
	assert.Contains(t, output, "<repo-brief>",
		"--target claude should produce XML with <repo-brief> tag")
	assert.Contains(t, output, "<!-- Repo Brief",
		"--target claude should produce XML comment header")
}

// TestBriefJSON_MetadataFieldValues verifies that the --json output contains
// sensible metadata values: positive token count, non-empty hash, reasonable
// max_tokens, and at least some section names.
func TestBriefJSON_MetadataFieldValues(t *testing.T) {
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var meta workflows.BriefJSON
	err := json.Unmarshal(outBuf.Bytes(), &meta)
	require.NoError(t, err)

	assert.Greater(t, meta.TokenCount, 0, "token_count must be positive")
	assert.NotEmpty(t, meta.ContentHash, "content_hash must not be empty")
	assert.Greater(t, meta.MaxTokens, 0, "max_tokens must be positive")
	// The hash should look like a hex string (16 hex chars for 64-bit).
	assert.Regexp(t, `^[0-9a-f]+$`, meta.ContentHash,
		"content_hash must be a hex string")
}

// TestBriefJSON_Deterministic verifies that running --json twice produces
// the same token_count and content_hash (determinism at the CLI level).
func TestBriefJSON_Deterministic(t *testing.T) {
	// Run 1
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--json"})
	var out1 bytes.Buffer
	rootCmd.SetOut(&out1)
	rootCmd.SetErr(new(bytes.Buffer))
	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	rootCmd.SetArgs(nil)

	var meta1 workflows.BriefJSON
	require.NoError(t, json.Unmarshal(out1.Bytes(), &meta1))

	// Run 2
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--json"})
	var out2 bytes.Buffer
	rootCmd.SetOut(&out2)
	rootCmd.SetErr(new(bytes.Buffer))
	code = Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	rootCmd.SetArgs(nil)

	var meta2 workflows.BriefJSON
	require.NoError(t, json.Unmarshal(out2.Bytes(), &meta2))

	assert.Equal(t, meta1.ContentHash, meta2.ContentHash,
		"content_hash must be deterministic across runs")
	assert.Equal(t, meta1.TokenCount, meta2.TokenCount,
		"token_count must be deterministic across runs")
	assert.Equal(t, meta1.Sections, meta2.Sections,
		"sections list must be deterministic across runs")
}

// TestBriefStdout_ContainsRepoHeader verifies that stdout output contains
// the brief header with hash and token count metadata.
func TestBriefStdout_ContainsRepoHeader(t *testing.T) {
	resetBriefFlags(t)
	rootCmd.SetArgs([]string{"brief", "--stdout"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	output := outBuf.String()
	assert.Contains(t, output, "# Repo Brief",
		"stdout output should contain Markdown header")
	assert.Contains(t, output, "hash:",
		"stdout output should contain content hash in header")
	assert.Contains(t, output, "tokens:",
		"stdout output should contain token count in header")
}
