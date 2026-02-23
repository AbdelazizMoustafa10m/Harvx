package security_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/security"
)

// BenchmarkFullCorpus benchmarks the full corpus processing time.
func BenchmarkFullCorpus(b *testing.B) {
	cfg := security.RedactionConfig{Enabled: true, ConfidenceThreshold: security.ConfidenceLow}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	// Regenerate dynamic fixture files.
	fixtureDir := filepath.Join("..", "..", "testdata", "secrets")
	if err := writeFixtures(fixtureDir); err != nil {
		b.Fatalf("writing fixtures: %v", err)
	}

	// Load all fixture files.
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		b.Fatalf("reading fixture dir: %v", err)
	}

	type fixture struct {
		name    string
		content string
	}
	var fixtures []fixture

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".expected") ||
			name == ".gitkeep" ||
			name == "patterns_corpus.go" ||
			name == "README.md" {
			continue
		}
		path := filepath.Join(fixtureDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			b.Fatalf("reading %s: %v", name, err)
		}
		fixtures = append(fixtures, fixture{name, string(data)})
	}

	if len(fixtures) == 0 {
		b.Fatal("no fixture files found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, fix := range fixtures {
			_, _, err := redactor.Redact(context.Background(), fix.content, fix.name)
			if err != nil {
				b.Fatalf("redacting %s: %v", fix.name, err)
			}
		}
	}
}

// BenchmarkSingleFixture benchmarks a single fixture file for fine-grained analysis.
func BenchmarkSingleFixture(b *testing.B) {
	cfg := security.RedactionConfig{Enabled: true, ConfidenceThreshold: security.ConfidenceLow}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	// Regenerate dynamic fixture files.
	fixtureDir := filepath.Join("..", "..", "testdata", "secrets")
	if err := writeFixtures(fixtureDir); err != nil {
		b.Fatalf("writing fixtures: %v", err)
	}

	fixturePath := filepath.Join(fixtureDir, "config.env")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		b.Fatalf("reading config.env: %v", err)
	}
	content := string(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := redactor.Redact(context.Background(), content, "config.env")
		if err != nil {
			b.Fatalf("redacting: %v", err)
		}
	}
}
