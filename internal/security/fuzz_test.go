package security_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"unicode/utf8"

	"github.com/harvx/harvx/internal/security"
)

// markerRe matches well-formed [REDACTED:type] markers where the type portion
// contains only ASCII letters, digits, and underscores.
var markerRe = regexp.MustCompile(`\[REDACTED:[a-zA-Z0-9_]+\]`)

// broadMarkerRe matches any [REDACTED:...] substring, including potentially
// malformed ones. Used to find candidates that are then validated against markerRe.
var broadMarkerRe = regexp.MustCompile(`\[REDACTED:[^\]]*\]`)

// assertValidUTF8 is a test helper that fails t if s is not valid UTF-8.
func assertValidUTF8(t *testing.T, s, label string) {
	t.Helper()
	if !utf8.ValidString(s) {
		t.Fatalf("%s is not valid UTF-8", label)
	}
}

// assertWellFormedMarkers is a test helper that extracts all [REDACTED:...]
// substrings from s and verifies each one matches the expected pattern.
func assertWellFormedMarkers(t *testing.T, s string) {
	t.Helper()
	// Find all candidate markers with a broad search, then verify each one.
	for _, marker := range broadMarkerRe.FindAllString(s, -1) {
		if !markerRe.MatchString(marker) {
			t.Fatalf("malformed redaction marker %q in output", marker)
		}
	}
}

// FuzzRedactContent verifies that the StreamRedactor never panics, always
// returns valid UTF-8, produces non-negative length output, and emits only
// well-formed [REDACTED:type] markers.
func FuzzRedactContent(f *testing.F) {
	seeds := []string{
		"password=mysecret123",
		"AKIA1234567890ABCDEF",
		"normal code without secrets",
		"ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefg",
		"sk_" + "live_51234567890abcdefghijklmnop",
		"",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJBAK...\n-----END RSA PRIVATE KEY-----",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceLow,
	}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	f.Fuzz(func(t *testing.T, input string) {
		result, _, err := redactor.Redact(context.Background(), input, "test.go")
		if err != nil {
			return
		}

		// Invariant 1: output is valid UTF-8.
		assertValidUTF8(t, result, "Redact output")

		// Invariant 2: output length is non-negative.
		if len(result) < 0 {
			t.Fatal("output length is negative")
		}

		// Invariant 3: all [REDACTED:...] markers are well-formed.
		assertWellFormedMarkers(t, result)
	})
}

// FuzzRedactHighEntropy verifies that the entropy analyzer never panics and
// returns consistent, valid results for arbitrary inputs.
func FuzzRedactHighEntropy(f *testing.F) {
	seeds := []string{
		"aB3dE5gH7jK9mN1pQ3sT5v",
		"0123456789abcdef0123456789abcdef",
		"aaaaaaaaaaaaaaaaaaaaaaaaa",
		"ABCDEFGHIJKLMNOPqrstuvwxyz012345",
		"",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	analyzer := security.NewEntropyAnalyzer()

	f.Fuzz(func(t *testing.T, input string) {
		// Invariant 1: Calculate does not panic and returns >= 0.0.
		entropy := analyzer.Calculate(input)
		if entropy < 0.0 {
			t.Fatalf("Calculate returned negative entropy %f for input %q", entropy, input)
		}

		// Invariant 2: AnalyzeToken does not panic and returns a valid EntropyResult.
		result := analyzer.AnalyzeToken(input, security.TokenContext{})
		if result.Entropy < 0.0 {
			t.Fatalf("AnalyzeToken returned negative entropy %f for input %q", result.Entropy, input)
		}
		// Confidence must be one of the known values.
		switch result.Confidence {
		case security.ConfidenceHigh, security.ConfidenceMedium, security.ConfidenceLow:
			// valid
		default:
			t.Fatalf("AnalyzeToken returned unknown confidence %q for input %q", result.Confidence, input)
		}

		// Invariant 3: DetectCharset does not panic.
		_ = security.DetectCharset(input)
	})
}

// FuzzRedactEnvFile verifies that the redactor handles .env-style content
// gracefully on arbitrary inputs, never panics, and returns valid UTF-8.
func FuzzRedactEnvFile(f *testing.F) {
	seeds := []string{
		"API_KEY=sk_" + "live_4242424242424242",
		"DATABASE_URL=postgres://user:pass@host:5432/db",
		"SECRET=",
		"NORMAL_VAR=hello_world",
		"AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceLow,
	}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	f.Fuzz(func(t *testing.T, input string) {
		// Wrap the fuzzed string as env-like content.
		envContent := fmt.Sprintf("ENV_VAR=%s", input)
		result, _, err := redactor.Redact(context.Background(), envContent, ".env")
		if err != nil {
			return
		}

		// Invariant 1: no panic (reaching here proves it).

		// Invariant 2: output is valid UTF-8.
		assertValidUTF8(t, result, "Redact .env output")
	})
}

// FuzzRedactMixedContent verifies that the redactor handles a combination of
// code and secret content without panics, returning valid UTF-8 and
// well-formed markers.
func FuzzRedactMixedContent(f *testing.F) {
	f.Add("func main() {", "password=secret123")
	f.Add("// comment", "AKIA1234567890ABCDEF")
	f.Add("\u3053\u3093\u306b\u3061\u306f\u4e16\u754c", "token=abc123def456ghi789")
	f.Add("", "")

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceLow,
	}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	f.Fuzz(func(t *testing.T, code, secret string) {
		mixed := code + "\n" + secret
		result, _, err := redactor.Redact(context.Background(), mixed, "test.go")
		if err != nil {
			return
		}

		// Invariant 1: no panic (reaching here proves it).

		// Invariant 2: output is valid UTF-8.
		assertValidUTF8(t, result, "Redact mixed output")

		// Invariant 3: all [REDACTED:...] markers are well-formed.
		assertWellFormedMarkers(t, result)
	})
}
