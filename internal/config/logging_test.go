package config

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		quiet    bool
		envDebug string
		want     slog.Level
	}{
		{
			name: "default is info",
			want: slog.LevelInfo,
		},
		{
			name:    "verbose sets debug",
			verbose: true,
			want:    slog.LevelDebug,
		},
		{
			name:  "quiet sets error",
			quiet: true,
			want:  slog.LevelError,
		},
		{
			name:    "verbose wins over quiet",
			verbose: true,
			quiet:   true,
			want:    slog.LevelDebug,
		},
		{
			name:     "HARVX_DEBUG overrides default",
			envDebug: "1",
			want:     slog.LevelDebug,
		},
		{
			name:     "HARVX_DEBUG overrides quiet",
			quiet:    true,
			envDebug: "1",
			want:     slog.LevelDebug,
		},
		{
			name:     "HARVX_DEBUG non-1 value ignored",
			envDebug: "true",
			want:     slog.LevelInfo,
		},
		{
			name:     "HARVX_DEBUG empty string ignored",
			envDebug: "",
			want:     slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env for this test case, restore after.
			if tt.envDebug != "" {
				t.Setenv("HARVX_DEBUG", tt.envDebug)
			} else {
				// Ensure HARVX_DEBUG is unset for tests that don't use it.
				t.Setenv("HARVX_DEBUG", "")
			}

			got := ResolveLogLevel(tt.verbose, tt.quiet)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveLogFormat(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		want    string
	}{
		{
			name: "default is text",
			want: "text",
		},
		{
			name:   "json format from env",
			envVal: "json",
			want:   "json",
		},
		{
			name:   "JSON uppercase from env",
			envVal: "JSON",
			want:   "json",
		},
		{
			name:   "Json mixed case from env",
			envVal: "Json",
			want:   "json",
		},
		{
			name:   "non-json value returns text",
			envVal: "yaml",
			want:   "text",
		},
		{
			name:   "empty string returns text",
			envVal: "",
			want:   "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HARVX_LOG_FORMAT", tt.envVal)
			got := ResolveLogFormat()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetupLogging_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	SetupLoggingWithWriter(slog.LevelInfo, "text", &buf)

	slog.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
	// Text handler uses key=value format, not JSON.
	assert.NotContains(t, output, `"msg"`)
}

func TestSetupLogging_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	SetupLoggingWithWriter(slog.LevelInfo, "json", &buf)

	slog.Info("test message", "key", "value")

	output := buf.String()

	// Verify it is valid JSON.
	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "JSON log output should be valid JSON")

	assert.Equal(t, "test message", parsed["msg"])
	assert.Equal(t, "value", parsed["key"])
	assert.Equal(t, "INFO", parsed["level"])
}

func TestSetupLogging_JSONFormatCaseInsensitive(t *testing.T) {
	var buf bytes.Buffer
	SetupLoggingWithWriter(slog.LevelInfo, "JSON", &buf)

	slog.Info("case test")

	output := buf.String()

	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "JSON handler should be selected case-insensitively")
	assert.Equal(t, "case test", parsed["msg"])
}

func TestSetupLogging_LevelFiltering(t *testing.T) {
	tests := []struct {
		name       string
		level      slog.Level
		logFunc    func(msg string, args ...any)
		msg        string
		shouldShow bool
	}{
		{
			name:       "info shows at info level",
			level:      slog.LevelInfo,
			logFunc:    slog.Info,
			msg:        "info-msg",
			shouldShow: true,
		},
		{
			name:       "debug hidden at info level",
			level:      slog.LevelInfo,
			logFunc:    slog.Debug,
			msg:        "debug-msg",
			shouldShow: false,
		},
		{
			name:       "debug shows at debug level",
			level:      slog.LevelDebug,
			logFunc:    slog.Debug,
			msg:        "debug-msg",
			shouldShow: true,
		},
		{
			name:       "info hidden at error level",
			level:      slog.LevelError,
			logFunc:    slog.Info,
			msg:        "info-msg",
			shouldShow: false,
		},
		{
			name:       "warn hidden at error level",
			level:      slog.LevelError,
			logFunc:    slog.Warn,
			msg:        "warn-msg",
			shouldShow: false,
		},
		{
			name:       "error shows at error level",
			level:      slog.LevelError,
			logFunc:    slog.Error,
			msg:        "error-msg",
			shouldShow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			SetupLoggingWithWriter(tt.level, "text", &buf)

			tt.logFunc(tt.msg)

			output := buf.String()
			if tt.shouldShow {
				assert.Contains(t, output, tt.msg)
			} else {
				assert.NotContains(t, output, tt.msg)
			}
		})
	}
}

func TestSetupLogging_StderrNotStdout(t *testing.T) {
	// Capture the original stderr and stdout.
	origStderr := os.Stderr
	origStdout := os.Stdout
	defer func() {
		os.Stderr = origStderr
		os.Stdout = origStdout
	}()

	// Create pipes for stderr and stdout.
	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)
	defer stderrR.Close()

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)
	defer stdoutR.Close()

	os.Stderr = stderrW
	os.Stdout = stdoutW

	// SetupLogging (not WithWriter) should use os.Stderr.
	SetupLogging(slog.LevelInfo, "text")
	slog.Info("stderr test message")

	// Close writers so reads complete.
	stderrW.Close()
	stdoutW.Close()

	// Read both outputs.
	var stderrBuf, stdoutBuf bytes.Buffer
	_, _ = stderrBuf.ReadFrom(stderrR)
	_, _ = stdoutBuf.ReadFrom(stdoutR)

	assert.Contains(t, stderrBuf.String(), "stderr test message",
		"log output should appear on stderr")
	assert.Empty(t, stdoutBuf.String(),
		"log output should NOT appear on stdout")
}

func TestSetupLogging_Idempotent(t *testing.T) {
	// Calling SetupLogging multiple times should not panic or cause issues.
	var buf1, buf2 bytes.Buffer

	SetupLoggingWithWriter(slog.LevelInfo, "text", &buf1)
	slog.Info("first setup")

	SetupLoggingWithWriter(slog.LevelDebug, "json", &buf2)
	slog.Debug("second setup")

	assert.Contains(t, buf1.String(), "first setup",
		"first logger should have received first message")
	assert.Contains(t, buf2.String(), "second setup",
		"second logger should have received second message")

	// First buffer should NOT contain the second message.
	assert.NotContains(t, buf1.String(), "second setup",
		"first logger should not receive messages after reconfiguration")
}

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	SetupLoggingWithWriter(slog.LevelInfo, "json", &buf)

	logger := NewLogger("discovery")
	logger.Info("walking directory", "root", "/tmp/repo")

	output := buf.String()

	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, "discovery", parsed["component"],
		"component attribute should be set")
	assert.Equal(t, "walking directory", parsed["msg"])
	assert.Equal(t, "/tmp/repo", parsed["root"])
}

func TestNewLogger_MultipleComponents(t *testing.T) {
	var buf bytes.Buffer
	SetupLoggingWithWriter(slog.LevelInfo, "text", &buf)

	logDiscovery := NewLogger("discovery")
	logSecurity := NewLogger("security")

	logDiscovery.Info("discovery event")
	logSecurity.Info("security event")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 2, "should have exactly two log lines")

	assert.Contains(t, lines[0], "component=discovery")
	assert.Contains(t, lines[0], "discovery event")
	assert.Contains(t, lines[1], "component=security")
	assert.Contains(t, lines[1], "security event")
}

func TestNewLogger_InheritsLevel(t *testing.T) {
	var buf bytes.Buffer
	SetupLoggingWithWriter(slog.LevelError, "text", &buf)

	logger := NewLogger("cli")
	logger.Info("should be hidden")
	logger.Error("should be visible")

	output := buf.String()
	assert.NotContains(t, output, "should be hidden",
		"info messages should be filtered at error level")
	assert.Contains(t, output, "should be visible",
		"error messages should pass at error level")
}
