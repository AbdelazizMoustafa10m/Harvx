package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// clearHarvxEnvForBenchmark unsets all HARVX_* environment variables.
// It does not use t.Setenv because testing.B does not support it.
func clearHarvxEnvForBenchmark() {
	for _, name := range []string{
		EnvProfile, EnvMaxTokens, EnvFormat, EnvTokenizer,
		EnvOutput, EnvTarget, EnvLogFormat, EnvCompress, EnvRedact,
	} {
		os.Unsetenv(name)
	}
}

// BenchmarkConfigResolve measures the cost of config resolution across
// different source configurations.
func BenchmarkConfigResolve(b *testing.B) {
	b.Run("defaults-only", func(b *testing.B) {
		clearHarvxEnvForBenchmark()

		dir := b.TempDir()
		globalPath := filepath.Join(dir, "nonexistent.toml")
		opts := ResolveOptions{
			TargetDir:        dir,
			GlobalConfigPath: globalPath,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Resolve(opts)
		}
	})

	b.Run("single-file", func(b *testing.B) {
		clearHarvxEnvForBenchmark()

		dir := b.TempDir()
		tomlContent := `
[profile.default]
format = "markdown"
max_tokens = 100000
tokenizer = "cl100k_base"
compression = false
redaction = true
output = "harvx-output.md"
ignore = ["node_modules", "dist", ".git"]
`
		tomlPath := filepath.Join(dir, "harvx.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), 0o644); err != nil {
			b.Fatal(err)
		}

		opts := ResolveOptions{
			TargetDir:        dir,
			GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Resolve(opts)
		}
	})

	b.Run("multi-source", func(b *testing.B) {
		clearHarvxEnvForBenchmark()

		globalDir := b.TempDir()
		globalContent := `
[profile.default]
tokenizer = "o200k_base"
format = "markdown"
output = "global-output.md"
`
		globalPath := filepath.Join(globalDir, "global.toml")
		if err := os.WriteFile(globalPath, []byte(globalContent), 0o644); err != nil {
			b.Fatal(err)
		}

		repoDir := b.TempDir()
		repoContent := `
[profile.default]
format = "xml"
max_tokens = 150000
compression = true
`
		repoPath := filepath.Join(repoDir, "harvx.toml")
		if err := os.WriteFile(repoPath, []byte(repoContent), 0o644); err != nil {
			b.Fatal(err)
		}

		opts := ResolveOptions{
			TargetDir:        repoDir,
			GlobalConfigPath: globalPath,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Resolve(opts)
		}
	})

	b.Run("ten-profiles", func(b *testing.B) {
		clearHarvxEnvForBenchmark()

		dir := b.TempDir()

		// Build a config with 10 named profiles.
		var sb strings.Builder
		sb.WriteString("[profile.default]\nformat = \"markdown\"\nmax_tokens = 128000\n\n")
		for i := 1; i <= 9; i++ {
			sb.WriteString(fmt.Sprintf("[profile.profile%d]\nextends = \"default\"\nmax_tokens = %d\n\n",
				i, 50000+i*10000))
		}

		tomlPath := filepath.Join(dir, "harvx.toml")
		if err := os.WriteFile(tomlPath, []byte(sb.String()), 0o644); err != nil {
			b.Fatal(err)
		}

		opts := ResolveOptions{
			ProfileName:      "profile5",
			TargetDir:        dir,
			GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Resolve(opts)
		}
	})
}

// BenchmarkConfigValidate measures the cost of config validation.
func BenchmarkConfigValidate(b *testing.B) {
	b.Run("clean-config", func(b *testing.B) {
		cfg, err := LoadFromString(`
[profile.default]
format = "markdown"
max_tokens = 128000
tokenizer = "cl100k_base"
compression = false
redaction = true
output = "harvx-output.md"
`, "bench")
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Validate(cfg)
		}
	})

	b.Run("complex-config", func(b *testing.B) {
		cfg, err := LoadFromString(`
[profile.default]
format = "markdown"
max_tokens = 128000
tokenizer = "cl100k_base"
compression = false
redaction = true
output = "harvx-output.md"
ignore = ["node_modules", "dist", ".git", "coverage", "__pycache__", ".next"]
priority_files = ["README.md", "go.mod", "package.json"]
include = ["**/*.go", "**/*.ts"]

[profile.default.relevance]
tier_0 = ["package.json", "tsconfig.json", "go.mod", "Makefile"]
tier_1 = ["src/**", "internal/**", "cmd/**"]
tier_2 = ["components/**", "utils/**", "services/**"]
tier_3 = ["**/*_test.go", "**/*.test.ts", "**/*.spec.ts"]
tier_4 = ["**/*.md", "docs/**", "README*"]
tier_5 = [".github/**", "**/*.lock"]

[profile.default.redaction_config]
enabled = true
exclude_paths = ["testdata/**", "docs/examples/**"]
confidence_threshold = "high"

[profile.staging]
extends = "default"
format = "xml"
max_tokens = 200000
tokenizer = "o200k_base"
target = "claude"
output = ".harvx/staging.md"

[profile.ci]
extends = "default"
max_tokens = 64000
compression = true
`, "bench")
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Validate(cfg)
		}
	})
}
