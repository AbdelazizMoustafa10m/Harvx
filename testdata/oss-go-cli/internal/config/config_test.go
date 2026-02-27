package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Sync.MaxFileSize != "100MB" {
		t.Errorf("expected MaxFileSize 100MB, got %s", cfg.Sync.MaxFileSize)
	}
	if !cfg.Sync.Checksum {
		t.Error("expected Checksum to be true by default")
	}
	if len(cfg.Sync.Ignore) == 0 {
		t.Error("expected default ignore patterns")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.toml")

	content := `
[sync]
ignore = ["*.log", "tmp/"]
max_file_size = "50MB"
checksum = false

[logging]
level = "debug"
format = "json"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Sync.MaxFileSize != "50MB" {
		t.Errorf("expected MaxFileSize 50MB, got %s", cfg.Sync.MaxFileSize)
	}
	if cfg.Sync.Checksum {
		t.Error("expected Checksum to be false")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.Logging.Level)
	}
}

func TestLoad_InvalidLevel(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.toml")

	content := `[logging]
level = "invalid"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("expected default config when no file found, got error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil default config")
	}
}