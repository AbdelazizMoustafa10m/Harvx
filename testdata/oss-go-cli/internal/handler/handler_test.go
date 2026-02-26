package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/example/gosync/internal/config"
)

func TestNew_RequiresSource(t *testing.T) {
	_, err := New(Options{Destination: "/tmp/dst"})
	if err == nil {
		t.Fatal("expected error when source is empty")
	}
}

func TestNew_RequiresDestination(t *testing.T) {
	_, err := New(Options{Source: "/tmp/src"})
	if err == nil {
		t.Fatal("expected error when destination is empty")
	}
}

func TestHandler_DryRun(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a source file
	if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	h, err := New(Options{
		Source:      srcDir,
		Destination: dstDir,
		Config:      config.DefaultConfig(),
		DryRun:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should NOT exist in destination (dry run)
	dstFile := filepath.Join(dstDir, "test.txt")
	if _, err := os.Stat(dstFile); err == nil {
		t.Error("expected file to not exist in destination during dry run")
	}
}

func TestHandler_SyncFile(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	content := []byte("sync me please")
	if err := os.WriteFile(filepath.Join(srcDir, "data.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	h, err := New(Options{
		Source:      srcDir,
		Destination: dstDir,
		Config:      config.DefaultConfig(),
		DryRun:      false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := h.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dstDir, "data.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("expected %q, got %q", content, got)
	}
}