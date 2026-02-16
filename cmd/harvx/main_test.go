package main

import (
	"testing"

	"github.com/harvx/harvx/internal/buildinfo"
)

func TestBuildMetadataDefaults(t *testing.T) {
	// Verify build-time ldflags variables in internal/buildinfo have
	// sensible defaults when not injected via -ldflags (i.e., during go test).
	if buildinfo.Version == "" {
		t.Error("Version should not be empty")
	}
	if buildinfo.Commit == "" {
		t.Error("Commit should not be empty")
	}
	if buildinfo.Date == "" {
		t.Error("Date should not be empty")
	}
	if buildinfo.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
}
