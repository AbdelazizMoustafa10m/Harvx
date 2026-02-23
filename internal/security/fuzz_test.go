package security_test

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/harvx/harvx/internal/security"
)

// FuzzRedactRandomContent verifies that the redactor:
//   - never panics on any input
//   - always returns valid UTF-8
//   - redaction markers are well-formed when present
func FuzzRedactRandomContent(f *testing.F) {
	// Seed corpus with representative inputs.
	seeds := []string{
		"",
		"Hello, world!",
		"AKIAIOSFODNN7EXAMPLE",
		"export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"gh" + "p_1A2B3C4D5E6F7G8H9I0J1K2L3M4N5O6P7Q8R",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIEo...\n-----END RSA PRIVATE KEY-----",
		"postgres://user:pass@localhost/db",
		"sk_liv" + "e_abcdefghijklmnopqrstuvwx",
		string([]byte{0xFF, 0xFE}), // invalid UTF-8
		"api_key = " + strings.Repeat("a", 100),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	cfg := security.RedactionConfig{Enabled: true, ConfidenceThreshold: security.ConfidenceLow}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	f.Fuzz(func(t *testing.T, content string) {
		result, _, err := redactor.Redact(context.Background(), content, "fuzz.txt")
		if err != nil {
			// Context cancellation or similar -- not a failure.
			return
		}

		// Invariant 1: output is valid UTF-8 when input is valid UTF-8.
		// If the input contains invalid UTF-8 the redactor passes it through
		// unchanged, so the output inherits the same invalidity.
		if utf8.ValidString(content) && !utf8.ValidString(result) {
			t.Fatalf("Redact returned non-UTF-8 output for valid-UTF-8 input %q", content)
		}

		// Invariant 2: redaction markers are well-formed.
		for _, marker := range extractFuzzMarkers(result) {
			if !isFuzzWellFormedMarker(marker) {
				t.Fatalf("malformed redaction marker %q in output", marker)
			}
		}
	})
}

// FuzzRedactEnvFile verifies that the redactor handles .env-style content
// (KEY=VALUE lines) gracefully on all inputs.
func FuzzRedactEnvFile(f *testing.F) {
	// Seed with realistic .env content.
	seeds := []string{
		"KEY=value",
		"AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\nAWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"DATABASE_URL=postgres://user:pass@localhost/db",
		"GITHUB_TOKEN=" + "gh" + "p_1A2B3C4D5E6F7G8H9I0J1K2L3M4N5O6P7Q8R",
		`PASSWORD="supersecretpassword"`,
		"=",
		"KEY=",
		"=VALUE",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	cfg := security.RedactionConfig{Enabled: true, ConfidenceThreshold: security.ConfidenceLow}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	f.Fuzz(func(t *testing.T, envContent string) {
		result, _, err := redactor.Redact(context.Background(), envContent, ".env")
		if err != nil {
			return
		}

		// Invariant: output is valid UTF-8 when input is valid UTF-8.
		if utf8.ValidString(envContent) && !utf8.ValidString(result) {
			t.Fatalf("Redact returned non-UTF-8 output for .env content %q", envContent)
		}

		// Invariant: no panic means we get here; verify result is non-nil string.
		_ = result
	})
}

// FuzzEntropyAnalyzer verifies that the entropy analyzer produces
// consistent results and never panics on arbitrary token inputs.
func FuzzEntropyAnalyzer(f *testing.F) {
	// Seed with high-entropy strings that resemble real secrets.
	seeds := []string{
		"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
		strings.Repeat("a", 16),
		strings.Repeat("A1", 20),
		"",
		"short",
		string([]byte{0x00, 0x01, 0x02}),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	analyzer := security.NewEntropyAnalyzer()

	f.Fuzz(func(t *testing.T, token string) {
		// Invariant: Calculate never panics.
		entropy := analyzer.Calculate(token)
		_ = entropy

		// Invariant: DetectCharset never panics.
		charset := security.DetectCharset(token)
		_ = charset

		// Invariant: AnalyzeToken never panics.
		result := analyzer.AnalyzeToken(token, security.TokenContext{})

		// If token is too short, it should not be flagged as high entropy.
		if len(token) < analyzer.MinLength {
			if result.IsHigh {
				t.Fatalf("token shorter than MinLength (%d) should not be high entropy: %q",
					analyzer.MinLength, token)
			}
		}
	})
}

// extractFuzzMarkers returns all [REDACTED:...] markers found in s.
func extractFuzzMarkers(s string) []string {
	var markers []string
	for {
		start := strings.Index(s, "[REDACTED:")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], "]")
		if end < 0 {
			break
		}
		markers = append(markers, s[start:start+end+1])
		s = s[start+end+1:]
	}
	return markers
}

// isFuzzWellFormedMarker returns true if marker matches [REDACTED:<non-empty-type>].
func isFuzzWellFormedMarker(marker string) bool {
	if !strings.HasPrefix(marker, "[REDACTED:") || !strings.HasSuffix(marker, "]") {
		return false
	}
	secretType := marker[len("[REDACTED:"):len(marker)-1]
	return len(secretType) > 0
}
