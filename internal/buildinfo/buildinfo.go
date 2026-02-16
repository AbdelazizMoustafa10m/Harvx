// Package buildinfo holds build-time metadata injected via ldflags.
// These variables are set by the Makefile during compilation:
//
//	go build -ldflags "-X github.com/harvx/harvx/internal/buildinfo.Version=..."
package buildinfo

import "runtime"

// Build-time variables injected via ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	Date      = "unknown"
	GoVersion = "unknown"
)

// OS returns the operating system (from runtime.GOOS).
func OS() string {
	return runtime.GOOS
}

// Arch returns the architecture (from runtime.GOARCH).
func Arch() string {
	return runtime.GOARCH
}
