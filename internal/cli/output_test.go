package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectPipe_RegularFile(t *testing.T) {
	t.Parallel()

	// A regular file is not a character device, so DetectPipe returns true
	// (it looks like a pipe/redirect from the fd perspective).
	f, err := os.CreateTemp(t.TempDir(), "pipe-test-*")
	require.NoError(t, err)
	defer f.Close()

	assert.True(t, DetectPipe(f), "regular file should be detected as piped (not a char device)")
}

func TestDetectPipe_ClosedFile(t *testing.T) {
	t.Parallel()

	// A closed file should return false (Stat fails).
	f, err := os.CreateTemp(t.TempDir(), "closed-test-*")
	require.NoError(t, err)
	f.Close()

	assert.False(t, DetectPipe(f), "closed file should return false on Stat failure")
}

func TestDetectOutputMode_StdoutFlagTrue(t *testing.T) {
	mode := DetectOutputMode(true)
	assert.True(t, mode.StdoutMode, "StdoutMode should be true when stdoutFlag is true")
}

func TestDetectOutputMode_StdoutFlagFalse(t *testing.T) {
	// Ensure HARVX_STDOUT is not set for this test.
	t.Setenv("HARVX_STDOUT", "")

	mode := DetectOutputMode(false)
	assert.False(t, mode.StdoutMode, "StdoutMode should be false when stdoutFlag is false and env is unset")
}

func TestDetectOutputMode_EnvOverride(t *testing.T) {
	t.Setenv("HARVX_STDOUT", "true")

	mode := DetectOutputMode(false)
	assert.True(t, mode.StdoutMode, "StdoutMode should be true when HARVX_STDOUT=true")
}

func TestDetectOutputMode_EnvNotTrue(t *testing.T) {
	t.Setenv("HARVX_STDOUT", "false")

	mode := DetectOutputMode(false)
	assert.False(t, mode.StdoutMode, "StdoutMode should be false when HARVX_STDOUT is not 'true'")
}

func TestDetectOutputMode_FlagTakesPrecedenceOverEnv(t *testing.T) {
	t.Setenv("HARVX_STDOUT", "false")

	mode := DetectOutputMode(true)
	assert.True(t, mode.StdoutMode, "stdoutFlag=true should take precedence over HARVX_STDOUT=false")
}

func TestShouldSuppressProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mode     OutputMode
		expected bool
	}{
		{
			name: "normal mode, terminal stdout",
			mode: OutputMode{
				StdoutMode:    false,
				IsPiped:       false,
				StderrIsPiped: false,
			},
			expected: false,
		},
		{
			name: "normal mode, piped stdout",
			mode: OutputMode{
				StdoutMode:    false,
				IsPiped:       true,
				StderrIsPiped: false,
			},
			expected: true,
		},
		{
			name: "stdout mode, stderr is terminal",
			mode: OutputMode{
				StdoutMode:    true,
				IsPiped:       false,
				StderrIsPiped: false,
			},
			expected: false,
		},
		{
			name: "stdout mode, stderr is piped",
			mode: OutputMode{
				StdoutMode:    true,
				IsPiped:       false,
				StderrIsPiped: true,
			},
			expected: true,
		},
		{
			name: "stdout mode, both piped",
			mode: OutputMode{
				StdoutMode:    true,
				IsPiped:       true,
				StderrIsPiped: true,
			},
			expected: true,
		},
		{
			name: "normal mode, both terminal",
			mode: OutputMode{
				StdoutMode:    false,
				IsPiped:       false,
				StderrIsPiped: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.mode.ShouldSuppressProgress())
		})
	}
}

func TestShouldDisableColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mode     OutputMode
		expected bool
	}{
		{
			name: "stderr is terminal",
			mode: OutputMode{
				StderrIsPiped: false,
			},
			expected: false,
		},
		{
			name: "stderr is piped",
			mode: OutputMode{
				StderrIsPiped: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.mode.ShouldDisableColor())
		})
	}
}

func TestMessageWriter_AlwaysReturnsStderr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode OutputMode
	}{
		{
			name: "normal mode",
			mode: OutputMode{StdoutMode: false},
		},
		{
			name: "stdout mode",
			mode: OutputMode{StdoutMode: true},
		},
		{
			name: "piped stdout mode",
			mode: OutputMode{StdoutMode: true, IsPiped: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := tt.mode.MessageWriter()
			assert.Equal(t, os.Stderr, w, "MessageWriter should always return os.Stderr")
		})
	}
}
