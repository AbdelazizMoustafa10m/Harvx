// Package security_test provides golden tests for the secret detection corpus.
// The golden tests load fixture files from testdata/secrets/, run them through
// the StreamRedactor, and compare against .expected JSON files.
package security_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harvx/harvx/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensureFixtures regenerates the dynamic fixture files (those containing
// synthetic credentials assembled at runtime to avoid GitHub Push Protection).
// Static files (.expected JSON, README, etc.) are left untouched.
func ensureFixtures(t *testing.T) {
	t.Helper()
	fixtureDir := filepath.Join("..", "..", "testdata", "secrets")
	require.NoError(t, writeFixtures(fixtureDir),
		"regenerating fixture files with synthetic credentials")
}

// expectedRedaction describes a single expected match in a fixture file.
type expectedRedaction struct {
	Line       int    `json:"line"`
	RuleID     string `json:"rule_id"`
	SecretType string `json:"secret_type"`
	Confidence string `json:"confidence"`
}

// expectedResults is the structure of a .expected JSON file.
type expectedResults struct {
	ExpectedRedactions []expectedRedaction `json:"expected_redactions"`
}

// TestGoldenCorpus runs golden tests against all fixture files in testdata/secrets/.
// Each fixture file must have a corresponding .expected JSON file.
func TestGoldenCorpus(t *testing.T) {
	ensureFixtures(t)

	// Use ConfidenceLow to exercise all patterns.
	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceLow,
	}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	// Find all fixture files (not .expected files, not directories, not .gitkeep,
	// not patterns_corpus.go which is a Go source providing corpus data).
	fixtureDir := filepath.Join("..", "..", "testdata", "secrets")
	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)

	testedCount := 0
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

		fixtureName := name
		fixturePath := filepath.Join(fixtureDir, fixtureName)
		expectedPath := fixturePath + ".expected"

		// Skip fixtures without .expected files.
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Logf("skipping %s: no .expected file", fixtureName)
			continue
		}

		t.Run(fixtureName, func(t *testing.T) {
			// Load fixture content.
			content, err := os.ReadFile(fixturePath)
			require.NoError(t, err, "reading fixture file")

			// Load expected results.
			expectedData, err := os.ReadFile(expectedPath)
			require.NoError(t, err, "reading expected file")

			var expected expectedResults
			require.NoError(t, json.Unmarshal(expectedData, &expected), "parsing expected JSON")

			// Run the redactor.
			_, matches, err := redactor.Redact(context.Background(), string(content), fixturePath)
			require.NoError(t, err, "redacting fixture")

			// Build index of actual matches by (line, secretType).
			type matchKey struct {
				line       int
				secretType string
			}
			actualByKey := make(map[matchKey]bool)
			for _, m := range matches {
				actualByKey[matchKey{m.LineNumber, m.SecretType}] = true
			}

			// Check all expected redactions are present.
			for _, exp := range expected.ExpectedRedactions {
				key := matchKey{exp.Line, exp.SecretType}
				assert.True(t, actualByKey[key],
					"expected redaction not found: line=%d secret_type=%s rule_id=%s (fixture=%s)",
					exp.Line, exp.SecretType, exp.RuleID, fixtureName)
			}
		})
		testedCount++
	}

	require.Greater(t, testedCount, 0, "no fixture files were tested")
}

// TestFalsePositiveRate verifies that the false_positives.txt file
// has a false positive rate below 5%.
func TestFalsePositiveRate(t *testing.T) {
	ensureFixtures(t)

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceLow,
	}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	fixturePath := filepath.Join("..", "..", "testdata", "secrets", "false_positives.txt")
	content, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	_, matches, err := redactor.Redact(context.Background(), string(content), fixturePath)
	require.NoError(t, err)

	lines := strings.Split(string(content), "\n")
	nonEmptyLines := 0
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			nonEmptyLines++
		}
	}
	if nonEmptyLines == 0 {
		t.Skip("no non-empty non-comment lines in false_positives.txt")
	}

	falsePositiveRate := float64(len(matches)) / float64(nonEmptyLines)
	assert.Less(t, falsePositiveRate, 0.05,
		"false positive rate %.2f%% exceeds 5%% threshold (%d matches in %d non-empty lines): matches=%+v",
		falsePositiveRate*100, len(matches), nonEmptyLines, matches)
}

// TestAllPatternsExercised verifies that every built-in rule ID appears
// in at least one .expected file.
func TestAllPatternsExercised(t *testing.T) {
	ensureFixtures(t)

	// Collect all rule IDs from built-in patterns.
	registry := security.NewDefaultRegistry()
	ruleIDs := make(map[string]bool)
	for _, rule := range registry.Rules() {
		ruleIDs[rule.ID] = false // false = not yet seen in expected files
	}

	// Scan all .expected files for rule_id references.
	fixtureDir := filepath.Join("..", "..", "testdata", "secrets")
	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".expected") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fixtureDir, entry.Name()))
		require.NoError(t, err)

		var results expectedResults
		if err := json.Unmarshal(data, &results); err != nil {
			continue
		}
		for _, exp := range results.ExpectedRedactions {
			if _, exists := ruleIDs[exp.RuleID]; exists {
				ruleIDs[exp.RuleID] = true
			}
		}
	}

	// Assert every rule is exercised.
	for ruleID, seen := range ruleIDs {
		assert.True(t, seen, "rule %q is not exercised by any .expected file", ruleID)
	}
}

// TestFullCorpusPerformance asserts that the full corpus processes in under 500ms.
func TestFullCorpusPerformance(t *testing.T) {
	ensureFixtures(t)

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceLow,
	}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	fixtureDir := filepath.Join("..", "..", "testdata", "secrets")
	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)

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
		require.NoError(t, err)
		fixtures = append(fixtures, fixture{name, string(data)})
	}

	require.NotEmpty(t, fixtures, "no fixture files found for performance test")

	start := time.Now()
	for _, fix := range fixtures {
		_, _, err := redactor.Redact(context.Background(), fix.content, fix.name)
		require.NoError(t, err, "redacting %s", fix.name)
	}
	elapsed := time.Since(start)

	assert.Less(t, elapsed.Milliseconds(), int64(500),
		"full corpus processing took %dms, exceeds 500ms limit", elapsed.Milliseconds())
}

// ---------------------------------------------------------------------------
// Fixture generation helpers
// ---------------------------------------------------------------------------
//
// The testdata/secrets/ fixture files contain synthetic credentials that are
// assembled at runtime to avoid triggering GitHub Push Protection. The static
// files in the repository are placeholders; writeFixtures overwrites them
// with the real content before golden tests run.

// Synthetic credential fragments -- split at runtime.
var (
	fStripeSecretLive   = join("sk_liv", "e_abcdefghijklmnopqrstuvwx")
	fStripePubLive      = join("pk_liv", "e_abcdefghijklmnopqrstuvwx")
	fStripeRestrictLive = join("rk_liv", "e_abcdefghijklmnopqrstuvwx")
	fStripeSecretTest   = join("sk_tes", "t_ABCDEFGHIJKLMNOPQRSTUVabcdefghij")
	fStripePubTest      = join("pk_tes", "t_ABCDEFGHIJKLMNOPQRSTUVabcdefghij")
	fGHTokenP           = join("gh", "p_1A2B3C4D5E6F7G8H9I0J1K2L3M4N5O6P7Q8R")
	fGHTokenO           = join("gh", "o_1A2B3C4D5E6F7G8H9I0J1K2L3M4N5O6P7Q8R")
	fGHTokenS           = join("gh", "s_1A2B3C4D5E6F7G8H9I0J1K2L3M4N5O6P7Q8R")
	fGHTokenR           = join("gh", "r_1A2B3C4D5E6F7G8H9I0J1K2L3M4N5O6P7Q8R")
	fGHPAT              = join("github_pa", "t_11ABCDEFGHIJKLMNOPQRSTUVWXabcdefghij")
	fSlackBot           = join("xox", "b-123456789012-123456789012-abcdefghijklmnopqrstuvwx")
)

// writeFixtures writes all testdata/secrets fixture files that contain
// synthetic credentials into dir. Static .expected JSON files are left as-is.
func writeFixtures(dir string) error {
	files := map[string]string{
		"stripe_keys.txt":         fxStripeKeys(),
		"github_tokens.txt":       fxGitHubTokens(),
		"config.env":              fxConfigEnv(),
		"generic_assignments.txt": fxGenericAssignments(),
		"mixed_file.go":           fxMixedGo(),
		"mixed_file.ts":           fxMixedTS(),
		"mixed_file.py":           fxMixedPy(),
		"docker-compose.yml":      fxDockerCompose(),
		"false_positives.txt":     fxFalsePositives(),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}

	subdir := filepath.Join(dir, "test_fixtures")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(subdir, "sample_test.go"), []byte(fxSampleTestGo()), 0o644)
}

func fxStripeKeys() string {
	return "# Stripe API Keys\n" +
		"# Live keys (MUST be redacted)\n" +
		"STRIPE_SECRET_KEY=" + fStripeSecretLive + "\n" +
		"STRIPE_PUBLISHABLE_KEY=" + fStripePubLive + "\n" +
		"STRIPE_RESTRICTED_KEY=" + fStripeRestrictLive + "\n" +
		"# Test keys (should NOT be redacted by stripe-live-key)\n" +
		"STRIPE_TEST_SECRET=" + fStripeSecretTest + "\n" +
		"STRIPE_TEST_PK=" + fStripePubTest + "\n"
}

func fxGitHubTokens() string {
	return "# GitHub tokens for CI/CD\n" +
		"# Classic PAT for GitHub Actions\n" +
		"GITHUB_TOKEN=" + fGHTokenP + "\n" +
		"GITHUB_OAUTH_TOKEN=" + fGHTokenO + "\n" +
		"# Server-to-server token\n" +
		"GITHUB_APP_TOKEN=" + fGHTokenS + "\n" +
		"GITHUB_REFRESH_TOKEN=" + fGHTokenR + "\n" +
		"# Fine-grained PAT (new format)\n" +
		"GITHUB_PAT=" + fGHPAT + "\n"
}

func fxConfigEnv() string {
	return "# Application environment configuration\n" +
		"# Generated: 2024-01-01\n" +
		"# WARNING: Synthetic credentials for testing only\n" +
		"\n" +
		"# AWS\n" +
		"AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\n" +
		"AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\n" +
		"\n" +
		"# GitHub\n" +
		"GITHUB_TOKEN=" + fGHTokenP + "\n" +
		"\n" +
		"# Database\n" +
		"DATABASE_URL=postgres://appuser:apppassword123@db.internal:5432/production\n" +
		"\n" +
		"# Stripe\n" +
		"STRIPE_SECRET_KEY=" + fStripeSecretLive + "\n" +
		"\n" +
		"# OpenAI\n" +
		"OPENAI_KEY=sk-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij\n" +
		"\n" +
		"# Slack\n" +
		"SLACK_BOT_TOKEN=" + fSlackBot + "\n" +
		"\n" +
		"# Sendgrid\n" +
		"SENDGRID_API_KEY=" + join("SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij") + "\n" +
		"\n" +
		"# JWT\n" +
		`JWT_SECRET="mysupersecretjwtkey123456789012"` + "\n" +
		"\n" +
		"# Hex\n" +
		"SECRET_KEY=a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6\n"
}

func fxGenericAssignments() string {
	return "# YAML config\n" +
		"database:\n" +
		`  password: "mysupersecretpassword123"` + "\n" +
		`  token: "myauthtoken12345678"` + "\n" +
		"\n" +
		"# TOML config\n" +
		"[api]\n" +
		"api_key = ABCDEFGHIJKLMNOPQRSTUVWXYZabcde\n" +
		"\n" +
		"# .properties format\n" +
		`app.password="supersecretpass1"` + "\n" +
		`app.secret="mysecretvalue123"` + "\n" +
		"\n" +
		"# Slack token\n" +
		"SLACK_BOT_TOKEN=" + fSlackBot + "\n" +
		"\n" +
		"# Twilio\n" +
		"TWILIO_AUTH_TOKEN=" + join("S", "K1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d") + "\n" +
		"\n" +
		"# SendGrid\n" +
		"SENDGRID_API_KEY=" + join("SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij") + "\n" +
		"\n" +
		"# Hex secret\n" +
		"secret = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6\n"
}

func fxMixedGo() string {
	return `package config

import (
	"os"
)

// Config holds application configuration.
type Config struct {
	AWSKey    string
	AWSSecret string
	DBConn    string
}

const (
	// DefaultAWSKey is the fallback key for development.
	// WARNING: Do not use in production!
	defaultAWSKey    = "AKIAIOSFODNN7EXAMPLE"
	defaultAWSSecret = "aws_secret_access_key"
)

// NewConfigFromEnv loads configuration from environment variables.
// In development, it falls back to hardcoded values for convenience.
func NewConfigFromEnv() *Config {
	key := os.Getenv("AWS_ACCESS_KEY_ID")
	if key == "" {
		key = "AKIAI44QH8DHBEXAMPLE"
	}

	dbConn := os.Getenv("DATABASE_URL")
	if dbConn == "" {
		dbConn = "postgres://devuser:devpass123@localhost:5432/devdb"
	}

	return &Config{
		AWSKey: key,
		DBConn: dbConn,
	}
}

// getStripeKey returns the Stripe secret key.
func getStripeKey() string {
	return "` + fStripeSecretLive + `"
}
`
}

func fxMixedTS() string {
	return `// Configuration module for MyApp
// WARNING: These are synthetic test credentials

interface Config {
  awsAccessKeyId: string;
  awsSecretKey: string;
  githubToken: string;
  stripeKey: string;
}

const devConfig: Config = {
  awsAccessKeyId: 'AKIAIOSFODNN7EXAMPLE',
  aws_secret_key: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY', // THIS IS SYNTHETIC
  githubToken: '` + fGHTokenP + `',
  stripeKey: '` + fStripeSecretLive + `',
};

const apiKey = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef';

export const getJwtToken = (): string => {
  return 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.ABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890ab';
};
`
}

func fxMixedPy() string {
	return `# Configuration for MyApp (SYNTHETIC CREDENTIALS - DO NOT USE IN PRODUCTION)
import os

# AWS Configuration
AWS_ACCESS_KEY_ID = 'AKIAIOSFODNN7EXAMPLE'
AWS_SECRET_ACCESS_KEY = os.environ.get('AWS_SECRET', 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY')

# GitHub
GITHUB_TOKEN = '` + fGHTokenP + `'

# Database
DATABASE_URL = 'postgres://admin:adminpass123@db.example.com:5432/myapp'

# Stripe
STRIPE_API_KEY = '` + fStripeSecretLive + `'

# OpenAI
OPENAI_KEY = 'sk-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij'

# JWT Secret
jwt_secret = 'mysupersecretjwtkey123456789'

# Hex secret
SECRET_KEY = 'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6'
`
}

func fxDockerCompose() string {
	return `version: '3.8'

services:
  app:
    image: myapp:latest
    environment:
      AWS_ACCESS_KEY_ID: AKIAIOSFODNN7EXAMPLE
      AWS_SECRET_ACCESS_KEY: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
      DATABASE_URL: postgres://dbuser:dbpassword123@postgres:5432/appdb
      GITHUB_TOKEN: ` + fGHTokenP + `

  postgres:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: "mypostgrespassword1"
      POSTGRES_USER: dbuser
`
}

func fxFalsePositives() string {
	return `# False positive candidates - strings that resemble secrets but should NOT trigger
# any built-in secret detection rules (regex or entropy-based).
# All strings are either too short, have wrong format, or have low entropy.

# Stripe TEST keys: _test_ prefix not _live_, and values are short + low entropy
STRIPE_TEST=` + join("sk_tes", "t_aaaaaaaaa") + `
STRIPE_TEST2=` + join("pk_tes", "t_aaaaaaaaa") + `

# Too short to match minimum lengths for AWS access key ID (needs 20 chars total)
TOOSHORT_AKID=AKIAIOSFODNN7

# GitHub token with wrong prefix (ghz not in gh[pors]), also too short for pattern
NOT_GH_SHORT=ghz_1A2B3C4D5E

# Wrong prefix for AWS access key ID (BKIA not a valid prefix)
NOT_AWS=BKIAIOSFODNN7EXAMPLE

# Placeholder values - no assignment operator and value
YOUR_API_KEY_HERE
PLACEHOLDER_TOKEN

# UUIDs - structured separators break entropy tokens into short pieces
user_id=550e8400-e29b

# Version strings - no secret context
APP_VERSION=v1.2.3-beta.1
BUILD_TAG=release-2024-01

# Short tokens well below minimum length
short_token=abc123
tiny_ref=deadbeef
`
}

func fxSampleTestGo() string {
	return `package mypackage_test

import (
	"testing"
)

// TestSecretDetection verifies that the redactor handles various inputs.
// The credentials in this file are synthetic test fixtures.
func TestSecretDetection(t *testing.T) {
	// These are test-only values used in assertions - they are NOT real secrets
	const expectedAWSKey = "AKIAIOSFODNN7EXAMPLE"
	const expectedGHToken = "` + fGHTokenP + `"

	// These should be found by the redactor when scanning test fixture files
	if expectedAWSKey == "" {
		t.Fatal("expected AWS key to be non-empty")
	}
	if expectedGHToken == "" {
		t.Fatal("expected GitHub token to be non-empty")
	}
}
`
}
