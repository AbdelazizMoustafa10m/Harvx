package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSource_String verifies the human-readable names for every Source constant.
func TestSource_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		source Source
		want   string
	}{
		{SourceDefault, "default"},
		{SourceGlobal, "global"},
		{SourceRepo, "repo"},
		{SourceEnv, "env"},
		{SourceFlag, "flag"},
		{Source(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.source.String())
		})
	}
}

// TestSource_Precedence verifies the Source iota ordering matches the intended
// precedence: Default < Global < Repo < Env < Flag.
func TestSource_Precedence(t *testing.T) {
	t.Parallel()

	assert.Less(t, int(SourceDefault), int(SourceGlobal))
	assert.Less(t, int(SourceGlobal), int(SourceRepo))
	assert.Less(t, int(SourceRepo), int(SourceEnv))
	assert.Less(t, int(SourceEnv), int(SourceFlag))
}

// TestSourceMap_KeyAssignment verifies that SourceMap accepts keys and values
// as a plain map.
func TestSourceMap_KeyAssignment(t *testing.T) {
	t.Parallel()

	sm := make(SourceMap)
	sm["format"] = SourceFlag
	sm["max_tokens"] = SourceEnv
	sm["output"] = SourceDefault

	assert.Equal(t, SourceFlag, sm["format"])
	assert.Equal(t, SourceEnv, sm["max_tokens"])
	assert.Equal(t, SourceDefault, sm["output"])
}
