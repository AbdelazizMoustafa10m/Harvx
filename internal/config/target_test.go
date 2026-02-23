package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyTargetPreset_Claude verifies that the "claude" preset sets the
// expected format and token budget.
func TestApplyTargetPreset_Claude(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	err := ApplyTargetPreset(p, "claude")

	require.NoError(t, err)
	assert.Equal(t, "xml", p.Format)
	assert.Equal(t, 200000, p.MaxTokens)
}

// TestApplyTargetPreset_ChatGPT verifies that the "chatgpt" preset sets the
// expected format and token budget.
func TestApplyTargetPreset_ChatGPT(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	err := ApplyTargetPreset(p, "chatgpt")

	require.NoError(t, err)
	assert.Equal(t, "markdown", p.Format)
	assert.Equal(t, 128000, p.MaxTokens)
}

// TestApplyTargetPreset_Generic verifies that the "generic" preset sets the
// format to markdown and does not change the token budget.
func TestApplyTargetPreset_Generic(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	p.MaxTokens = 99999 // arbitrary value to verify it is untouched
	err := ApplyTargetPreset(p, "generic")

	require.NoError(t, err)
	assert.Equal(t, "markdown", p.Format)
	assert.Equal(t, 99999, p.MaxTokens, "generic preset must not change MaxTokens")
}

// TestApplyTargetPreset_Empty verifies that an empty target string is a no-op.
func TestApplyTargetPreset_Empty(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	origFormat := p.Format
	origMaxTokens := p.MaxTokens

	err := ApplyTargetPreset(p, "")

	require.NoError(t, err)
	assert.Equal(t, origFormat, p.Format, "empty target must not change Format")
	assert.Equal(t, origMaxTokens, p.MaxTokens, "empty target must not change MaxTokens")
}

// TestApplyTargetPreset_Unknown verifies that an unknown target name returns
// an error.
func TestApplyTargetPreset_Unknown(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	err := ApplyTargetPreset(p, "llama")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "llama")
}

// TestApplyTargetPreset_AllPresets runs all valid targets in a table-driven
// test to ensure no valid target returns an error.
func TestApplyTargetPreset_AllPresets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		target     string
		wantFormat string
	}{
		{"claude", "xml"},
		{"chatgpt", "markdown"},
		{"generic", "markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			t.Parallel()

			p := DefaultProfile()
			err := ApplyTargetPreset(p, tt.target)

			require.NoError(t, err)
			assert.Equal(t, tt.wantFormat, p.Format)
		})
	}
}
