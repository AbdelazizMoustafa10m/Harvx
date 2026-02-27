// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
package cli

import (
	"io"
	"os"
)

// OutputMode describes the current output routing for the CLI.
// When stdout mode is active, context output goes to stdout and all
// user-facing messages (progress, warnings, summaries) go to stderr.
type OutputMode struct {
	// StdoutMode is true when --stdout or HARVX_STDOUT=true is active.
	StdoutMode bool

	// IsPiped is true when stdout is a pipe (not a terminal).
	IsPiped bool

	// StderrIsPiped is true when stderr is a pipe (not a terminal).
	StderrIsPiped bool
}

// DetectPipe checks whether the given *os.File is a pipe (not a terminal).
// Returns true when the file descriptor is not a character device (i.e.,
// output is being piped to another process or redirected to a file).
// If Stat fails (e.g., fd is closed), it returns false.
func DetectPipe(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

// DetectOutputMode determines the current output mode based on the --stdout
// flag, HARVX_STDOUT env var, and terminal detection.
func DetectOutputMode(stdoutFlag bool) OutputMode {
	stdoutMode := stdoutFlag
	if !stdoutMode && os.Getenv("HARVX_STDOUT") == "true" {
		stdoutMode = true
	}

	return OutputMode{
		StdoutMode:    stdoutMode,
		IsPiped:       DetectPipe(os.Stdout),
		StderrIsPiped: DetectPipe(os.Stderr),
	}
}

// ShouldSuppressProgress returns true when progress output (bars, spinners)
// should be suppressed. This happens when stdout is piped (auto-detected)
// or when --stdout mode is active (all progress must go to stderr, and if
// stderr is also piped, suppress entirely).
func (m OutputMode) ShouldSuppressProgress() bool {
	if m.StdoutMode {
		// In stdout mode, progress would go to stderr.
		// If stderr is piped, suppress entirely.
		return m.StderrIsPiped
	}
	// In normal mode, suppress if stdout is piped.
	return m.IsPiped
}

// ShouldDisableColor returns true when ANSI color output should be disabled.
// Color is disabled when stderr is piped (since all user-facing output in
// stdout mode goes to stderr, color detection uses stderr).
func (m OutputMode) ShouldDisableColor() bool {
	return m.StderrIsPiped
}

// MessageWriter returns the writer for user-facing messages (progress,
// warnings, summaries). In stdout mode, this is always stderr so that
// stdout is clean for piped output. In normal mode, it returns stderr
// as well (slog already writes to stderr).
func (m OutputMode) MessageWriter() io.Writer {
	return os.Stderr
}
