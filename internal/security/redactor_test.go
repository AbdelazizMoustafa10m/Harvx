package security_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// newEnabledRedactor returns a StreamRedactor with all built-in rules and
// redaction enabled at the given confidence threshold.
func newEnabledRedactor(threshold security.Confidence) *security.StreamRedactor {
	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: threshold,
	}
	return security.NewStreamRedactor(nil, nil, cfg)
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewStreamRedactor_NilRegistryUsesDefault(t *testing.T) {
	t.Parallel()
	r := security.NewStreamRedactor(nil, nil, security.RedactionConfig{Enabled: true})
	require.NotNil(t, r)
}

func TestNewStreamRedactor_CustomPatternAppended(t *testing.T) {
	t.Parallel()
	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceHigh,
		CustomPatterns: []security.CustomPatternConfig{
			{
				ID:         "custom-test",
				Pattern:    `(MYTEST[A-Z0-9]{10})`,
				SecretType: "my_test_key",
				Confidence: security.ConfidenceHigh,
			},
		},
	}
	r := security.NewStreamRedactor(nil, nil, cfg)
	require.NotNil(t, r)

	content := "config.key = MYTEST1234567890"
	redacted, matches, err := r.Redact(context.Background(), content, "config.go")
	require.NoError(t, err)
	assert.Contains(t, redacted, "[REDACTED:my_test_key]")
	assert.Len(t, matches, 1)
	assert.Equal(t, "custom-test", matches[0].RuleID)
}

func TestNewStreamRedactor_InvalidCustomPatternSkipped(t *testing.T) {
	t.Parallel()
	cfg := security.RedactionConfig{
		Enabled: true,
		CustomPatterns: []security.CustomPatternConfig{
			{
				ID:         "bad-pattern",
				Pattern:    `(invalid[`,
				SecretType: "bad",
				Confidence: security.ConfidenceHigh,
			},
		},
	}
	// Should not panic; invalid pattern is silently skipped.
	r := security.NewStreamRedactor(nil, nil, cfg)
	require.NotNil(t, r)
}

// ---------------------------------------------------------------------------
// Disabled redaction
// ---------------------------------------------------------------------------

func TestRedact_DisabledReturnsUnchanged(t *testing.T) {
	t.Parallel()
	cfg := security.RedactionConfig{Enabled: false}
	r := security.NewStreamRedactor(nil, nil, cfg)

	content := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
	redacted, matches, err := r.Redact(context.Background(), content, "config.env")
	require.NoError(t, err)
	assert.Equal(t, content, redacted)
	assert.Nil(t, matches)
}

// ---------------------------------------------------------------------------
// Path exclusion
// ---------------------------------------------------------------------------

func TestRedact_ExcludedPathReturnsUnchanged(t *testing.T) {
	t.Parallel()
	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceLow,
		ExcludePaths:        []string{"**/testdata/**", "**/*fixtures*/**"},
	}
	r := security.NewStreamRedactor(nil, nil, cfg)

	content := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"

	tests := []struct {
		name     string
		filePath string
		excluded bool
	}{
		{name: "matches testdata pattern", filePath: "testdata/secrets/config.env", excluded: true},
		{name: "matches fixtures pattern", filePath: "internal/fixtures/creds.yaml", excluded: true},
		{name: "not excluded", filePath: "internal/config/settings.go", excluded: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			redacted, matches, err := r.Redact(context.Background(), content, tt.filePath)
			require.NoError(t, err)
			if tt.excluded {
				assert.Equal(t, content, redacted, "excluded path must return content unchanged")
				assert.Nil(t, matches)
			} else {
				assert.NotEqual(t, content, redacted, "non-excluded path must be redacted")
				assert.NotEmpty(t, matches)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Empty content
// ---------------------------------------------------------------------------

func TestRedact_EmptyContent(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceLow)
	redacted, matches, err := r.Redact(context.Background(), "", "file.go")
	require.NoError(t, err)
	assert.Equal(t, "", redacted)
	assert.Empty(t, matches)
}

// ---------------------------------------------------------------------------
// AWS Access Key ID
// ---------------------------------------------------------------------------

func TestRedact_AWSAccessKeyID(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	content := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
	redacted, matches, err := r.Redact(context.Background(), content, "config.env")
	require.NoError(t, err)
	assert.Contains(t, redacted, "[REDACTED:aws_access_key_id]")
	assert.NotContains(t, redacted, "AKIAIOSFODNN7EXAMPLE")
	require.Len(t, matches, 1)
	assert.Equal(t, "aws-access-key-id", matches[0].RuleID)
	assert.Equal(t, security.ConfidenceHigh, matches[0].Confidence)
	assert.Equal(t, "config.env", matches[0].FilePath)
	assert.Equal(t, 1, matches[0].LineNumber)
}

// ---------------------------------------------------------------------------
// GitHub Token
// ---------------------------------------------------------------------------

func TestRedact_GitHubClassicToken(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	content := "github_token = " + "gh" + "p_1234567890123456789012345678901234AB"
	redacted, matches, err := r.Redact(context.Background(), content, "config.go")
	require.NoError(t, err)
	assert.Contains(t, redacted, "[REDACTED:github_token]")
	assert.NotContains(t, redacted, "ghp_")
	require.NotEmpty(t, matches)
	assert.Equal(t, "github-classic-token", matches[0].RuleID)
}

// ---------------------------------------------------------------------------
// Private key block (multi-line)
// ---------------------------------------------------------------------------

func TestRedact_PrivateKeyBlock(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	content := strings.Join([]string{
		"# RSA Private Key",
		"-----BEGIN RSA PRIVATE KEY-----",
		"MIIEowIBAAKCAQEA2a2rwplBQLF29amygykEMmYz0+Kcj3bKBp29VJTzk7M99l2S",
		"3Ld7EMjSbJU7QSZSaHhMD/SgbYZqLhELuA3GYpfFXFHEYUyNHJWVzuT/9VjZ",
		"-----END RSA PRIVATE KEY-----",
		"# end of key",
	}, "\n")

	redacted, matches, err := r.Redact(context.Background(), content, "key.pem")
	require.NoError(t, err)

	// The BEGIN line should be replaced with the block marker.
	assert.Contains(t, redacted, "[REDACTED:private_key_block]")
	// The key material lines should be blanked out.
	assert.NotContains(t, redacted, "MIIEowIBAAKCAQEA")
	// Non-secret lines should be preserved.
	assert.Contains(t, redacted, "# RSA Private Key")
	assert.Contains(t, redacted, "# end of key")

	// There must be exactly one match (for the block).
	require.Len(t, matches, 1)
	assert.Equal(t, "private-key-block", matches[0].RuleID)
	assert.Equal(t, "private_key_block", matches[0].SecretType)
	assert.Equal(t, 2, matches[0].LineNumber) // 1-based: BEGIN is on line 2
}

// ---------------------------------------------------------------------------
// Connection string (medium confidence)
// ---------------------------------------------------------------------------

func TestRedact_ConnectionString(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceMedium)

	content := `DATABASE_URL=postgres://user:password@localhost:5432/mydb`
	redacted, matches, err := r.Redact(context.Background(), content, "config.env")
	require.NoError(t, err)
	assert.Contains(t, redacted, "[REDACTED:connection_string]")
	assert.NotContains(t, redacted, "postgres://user:password")
	require.NotEmpty(t, matches)
	assert.Equal(t, "connection-string", matches[0].RuleID)
}

// ---------------------------------------------------------------------------
// Confidence threshold filtering
// ---------------------------------------------------------------------------

func TestRedact_ConfidenceThreshold(t *testing.T) {
	t.Parallel()

	// Low-confidence rule: password assignment.
	content := `password = "SuperSecretPassword123"`

	tests := []struct {
		name      string
		threshold security.Confidence
		wantMatch bool
	}{
		{name: "low threshold finds low-confidence matches", threshold: security.ConfidenceLow, wantMatch: true},
		{name: "medium threshold skips low-confidence matches", threshold: security.ConfidenceMedium, wantMatch: false},
		{name: "high threshold skips low-confidence matches", threshold: security.ConfidenceHigh, wantMatch: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newEnabledRedactor(tt.threshold)
			_, matches, err := r.Redact(context.Background(), content, "config.go")
			require.NoError(t, err)
			if tt.wantMatch {
				assert.NotEmpty(t, matches, "expected at least one match at threshold %s", tt.threshold)
			} else {
				assert.Empty(t, matches, "expected no matches at threshold %s", tt.threshold)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Heightened scanning mode
// ---------------------------------------------------------------------------

func TestRedact_HeightenedScanningForDotEnv(t *testing.T) {
	t.Parallel()

	// Use medium threshold -- normally would skip low-confidence rules.
	// In heightened mode (.env file), threshold is lowered to low, so
	// password-assignment (low confidence) should also fire.
	r := newEnabledRedactor(security.ConfidenceMedium)

	content := `password = "SuperSecretPassword123"`

	t.Run("env file triggers heightened mode", func(t *testing.T) {
		t.Parallel()
		_, matches, err := r.Redact(context.Background(), content, ".env")
		require.NoError(t, err)
		assert.NotEmpty(t, matches, ".env file should trigger heightened mode and find low-confidence matches")
	})

	t.Run("go file does not trigger heightened mode", func(t *testing.T) {
		t.Parallel()
		_, matches, err := r.Redact(context.Background(), content, "config.go")
		require.NoError(t, err)
		assert.Empty(t, matches, ".go file should not trigger heightened mode; no low-confidence match expected")
	})
}

func TestRedact_HeightenedScanningForPEMFile(t *testing.T) {
	t.Parallel()

	// A medium-threshold redactor should still see high-confidence rules in
	// heightened mode -- the lowering only affects the threshold floor.
	r := newEnabledRedactor(security.ConfidenceHigh)

	content := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"

	_, matches, err := r.Redact(context.Background(), content, "server.pem")
	require.NoError(t, err)
	// High-confidence AWS key should be detected regardless.
	assert.NotEmpty(t, matches)
}

// ---------------------------------------------------------------------------
// Multiple secrets on one line
// ---------------------------------------------------------------------------

func TestRedact_MultipleSecretsOnOneLine(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	// Two AWS keys on the same line (both exactly 20 chars: AKIA + 16 alphanums).
	content := "export A=AKIAIOSFODNN7EXAMPLE export B=AKIAABCDEFGH12345678"
	redacted, matches, err := r.Redact(context.Background(), content, "setup.sh")
	require.NoError(t, err)
	// Both keys should be redacted.
	assert.NotContains(t, redacted, "AKIAIOSFODNN7EXAMPLE")
	assert.NotContains(t, redacted, "AKIAABCDEFGH12345678")
	assert.GreaterOrEqual(t, len(matches), 2)
}

// ---------------------------------------------------------------------------
// Idempotency
// ---------------------------------------------------------------------------

func TestRedact_Idempotent(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	original := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"

	// First pass.
	redacted1, matches1, err := r.Redact(context.Background(), original, "config.env")
	require.NoError(t, err)
	require.NotEmpty(t, matches1)

	// Second pass on already-redacted content.
	redacted2, matches2, err := r.Redact(context.Background(), redacted1, "config.env")
	require.NoError(t, err)

	// Output should be identical.
	assert.Equal(t, redacted1, redacted2, "redacting twice must produce the same output")
	// No new matches on the second pass (already-redacted markers should not re-trigger).
	assert.Empty(t, matches2, "second pass must find no new matches")
}

// ---------------------------------------------------------------------------
// Context preservation
// ---------------------------------------------------------------------------

func TestRedact_ContextPreservation(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	content := `  aws_key = "AKIAIOSFODNN7EXAMPLE"  # production key`
	redacted, _, err := r.Redact(context.Background(), content, "settings.py")
	require.NoError(t, err)

	// Indentation and surrounding text must be preserved.
	assert.Contains(t, redacted, `  aws_key = "`)
	assert.Contains(t, redacted, `"  # production key`)
	assert.Contains(t, redacted, "[REDACTED:aws_access_key_id]")
	assert.NotContains(t, redacted, "AKIAIOSFODNN7EXAMPLE")
}

// ---------------------------------------------------------------------------
// Multi-line content with line numbers
// ---------------------------------------------------------------------------

func TestRedact_LineNumbers(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	lines := []string{
		"package main",
		"",
		`const awsKey = "AKIAIOSFODNN7EXAMPLE"`,
		"// more code",
		`const githubToken = "` + "gh" + `p_1234567890123456789012345678901234AB"`,
	}
	content := strings.Join(lines, "\n")

	_, matches, err := r.Redact(context.Background(), content, "secrets.go")
	require.NoError(t, err)
	require.Len(t, matches, 2)

	// Verify line numbers match the actual positions.
	assert.Equal(t, 3, matches[0].LineNumber, "AWS key should be on line 3")
	assert.Equal(t, 5, matches[1].LineNumber, "GitHub token should be on line 5")
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestRedact_ContextCancellation(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	// Build a large file (> 100 lines to trigger the cancellation check).
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString(fmt.Sprintf("line %d: some innocuous content\n", i))
	}
	content := sb.String()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, err := r.Redact(ctx, content, "large.go")
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Summary method
// ---------------------------------------------------------------------------

func TestStreamRedactor_Summary(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	// Before any calls, summary should be zero.
	initial := r.Summary()
	assert.Equal(t, 0, initial.TotalCount)
	assert.Equal(t, 0, initial.FileCount)

	// Redact a file with one match.
	_, matches, err := r.Redact(context.Background(), "key=AKIAIOSFODNN7EXAMPLE", "f1.env")
	require.NoError(t, err)
	require.Len(t, matches, 1)

	summary := r.Summary()
	assert.Equal(t, 1, summary.TotalCount)
	assert.Equal(t, 1, summary.FileCount)
	assert.Equal(t, 1, summary.ByType["aws_access_key_id"])
	assert.Equal(t, 1, summary.ByConfidence[security.ConfidenceHigh])
}

func TestStreamRedactor_Summary_IsCopy(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)
	_, _, _ = r.Redact(context.Background(), "key=AKIAIOSFODNN7EXAMPLE", "f1.env")

	s1 := r.Summary()
	// Mutate the returned summary.
	s1.ByType["aws_access_key_id"] = 9999
	s1.ByConfidence[security.ConfidenceHigh] = 9999

	// The internal state must not be affected.
	s2 := r.Summary()
	assert.Equal(t, 1, s2.ByType["aws_access_key_id"], "mutating returned summary must not affect internal state")
	assert.Equal(t, 1, s2.ByConfidence[security.ConfidenceHigh])
}

func TestStreamRedactor_Summary_FileWithNoMatches(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	_, matches, err := r.Redact(context.Background(), "no secrets here", "clean.go")
	require.NoError(t, err)
	assert.Empty(t, matches)

	summary := r.Summary()
	assert.Equal(t, 0, summary.FileCount, "files with no matches must not increment file count")
}

// ---------------------------------------------------------------------------
// Custom registry
// ---------------------------------------------------------------------------

func TestStreamRedactor_CustomRegistry(t *testing.T) {
	t.Parallel()

	reg := security.NewEmptyRegistry()
	rule, err := security.NewRedactionRule(
		"my-rule",
		"Test rule",
		`(MYTOKEN[A-Z0-9]{8})`,
		[]string{"MYTOKEN"},
		"my_token",
		security.ConfidenceHigh,
		0,
	)
	require.NoError(t, err)
	reg.Register(rule)

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceHigh,
	}
	r := security.NewStreamRedactor(reg, nil, cfg)

	content := "auth = MYTOKEN12345678"
	redacted, matches, err := r.Redact(context.Background(), content, "config.go")
	require.NoError(t, err)
	assert.Contains(t, redacted, "[REDACTED:my_token]")
	require.Len(t, matches, 1)
	assert.Equal(t, "my-rule", matches[0].RuleID)
}

// ---------------------------------------------------------------------------
// Regex with entropy threshold
// ---------------------------------------------------------------------------

func TestRedact_EntropyThresholdSkipsLowEntropyMatches(t *testing.T) {
	t.Parallel()

	// Build a rule with a high entropy threshold to ensure low-entropy values
	// are skipped even if the pattern matches.
	reg := security.NewEmptyRegistry()
	rx := regexp.MustCompile(`(?i)api_key\s*=\s*['"]?([A-Za-z0-9]{16,})['"]?`)
	rule := security.RedactionRule{
		ID:               "high-entropy-api-key",
		Regex:            rx,
		Keywords:         []string{"api_key"},
		SecretType:       "api_key",
		Confidence:       security.ConfidenceHigh,
		EntropyThreshold: 3.5, // "aaaaaaaaaaaaaaaa" (entropy ~0) won't pass; diverse strings will.
	}
	reg.Register(rule)

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceHigh,
	}
	r := security.NewStreamRedactor(reg, nil, cfg)

	tests := []struct {
		name      string
		content   string
		wantMatch bool
	}{
		{
			name:      "low entropy value skipped",
			content:   "api_key = 'aaaaaaaaaaaaaaaa'", // very low entropy (all same char)
			wantMatch: false,
		},
		{
			// 20 distinct alphanumeric chars -> entropy close to log2(20) ≈ 4.32 bits/char,
			// which exceeds the rule threshold of 3.5.
			name:      "high entropy value matched",
			content:   "api_key = 'aB3dE5fG7hI9jK1lM2nO'", // 20 distinct chars
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, matches, err := r.Redact(context.Background(), tt.content, "config.yaml")
			require.NoError(t, err)
			if tt.wantMatch {
				assert.NotEmpty(t, matches)
			} else {
				assert.Empty(t, matches)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Stripe live key
// ---------------------------------------------------------------------------

func TestRedact_StripeLiveKey(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	content := "STRIPE_SECRET_KEY=" + "sk_liv" + "e_51NxExampleKeyThatIsLongEnough1234567"
	redacted, matches, err := r.Redact(context.Background(), content, "env.go")
	require.NoError(t, err)
	assert.Contains(t, redacted, "[REDACTED:stripe_api_key]")
	assert.NotContains(t, redacted, "sk_live_")
	require.NotEmpty(t, matches)
}

// ---------------------------------------------------------------------------
// PEM block spans full file
// ---------------------------------------------------------------------------

func TestRedact_PEMBlockSpansMultipleLines(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	// 10-line key material block.
	pemLines := []string{
		"-----BEGIN PRIVATE KEY-----",
	}
	for i := 0; i < 10; i++ {
		pemLines = append(pemLines, fmt.Sprintf("BASE64LINE%d==", i))
	}
	pemLines = append(pemLines, "-----END PRIVATE KEY-----")

	content := strings.Join(pemLines, "\n")
	redacted, matches, err := r.Redact(context.Background(), content, "private.pem")
	require.NoError(t, err)

	assert.Contains(t, redacted, "[REDACTED:private_key_block]")
	// All key material lines must be removed.
	for i := 0; i < 10; i++ {
		assert.NotContains(t, redacted, fmt.Sprintf("BASE64LINE%d", i))
	}
	require.Len(t, matches, 1)
	assert.Equal(t, 1, matches[0].LineNumber) // BEGIN is on line 1
}

// ---------------------------------------------------------------------------
// Concurrent use
// ---------------------------------------------------------------------------

func TestStreamRedactor_ConcurrentUse(t *testing.T) {
	t.Parallel()
	r := newEnabledRedactor(security.ConfidenceHigh)

	const goroutines = 20
	done := make(chan struct{}, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			content := fmt.Sprintf("line %d: AWS_KEY=AKIAIOSFODNN7EXAMPLE", n)
			_, _, err := r.Redact(context.Background(), content, "concurrent.go")
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", n, err)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	summary := r.Summary()
	assert.Equal(t, goroutines, summary.TotalCount)
	assert.Equal(t, goroutines, summary.FileCount)
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func buildBenchmarkContent(lines int, secretEveryN int) string {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		if i%secretEveryN == 0 {
			// AWS access key format: AKIA + 16 uppercase alphanumeric chars = 20 chars total.
			suffix := fmt.Sprintf("%016X", uint64(i+1))
			sb.WriteString(fmt.Sprintf("const key%d = \"AKIA%s\"\n", i, suffix))
		} else {
			sb.WriteString(fmt.Sprintf("// line %d: normal Go comment with no secrets here\n", i))
		}
	}
	return sb.String()
}

func BenchmarkRedact_Small100Lines(b *testing.B) {
	r := newEnabledRedactor(security.ConfidenceHigh)
	content := buildBenchmarkContent(100, 50) // 2 secrets
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = r.Redact(ctx, content, "bench.go")
	}
}

func BenchmarkRedact_Medium1000Lines(b *testing.B) {
	r := newEnabledRedactor(security.ConfidenceHigh)
	content := buildBenchmarkContent(1000, 100) // 10 secrets
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = r.Redact(ctx, content, "bench.go")
	}
}

func BenchmarkRedact_Large10000Lines(b *testing.B) {
	r := newEnabledRedactor(security.ConfidenceHigh)
	content := buildBenchmarkContent(10000, 200) // 50 secrets
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = r.Redact(ctx, content, "bench.go")
	}
}

func BenchmarkRedact_NoSecrets(b *testing.B) {
	r := newEnabledRedactor(security.ConfidenceHigh)
	content := buildBenchmarkContent(1000, 999999) // effectively 0 secrets
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = r.Redact(ctx, content, "bench.go")
	}
}
