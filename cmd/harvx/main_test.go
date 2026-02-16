package main

import "testing"

func TestBuildMetadataDefaults(t *testing.T) {
	// Verify build-time ldflags variables have sensible defaults
	// when not injected via -ldflags (i.e., during go test).
	if version == "" {
		t.Error("version should not be empty")
	}
	if commit == "" {
		t.Error("commit should not be empty")
	}
	if date == "" {
		t.Error("date should not be empty")
	}
	if goVersion == "" {
		t.Error("goVersion should not be empty")
	}
}
