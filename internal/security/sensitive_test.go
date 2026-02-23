package security_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/harvx/harvx/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSensitiveFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		// .env files
		{name: "root .env", filePath: ".env", want: true},
		{name: "nested .env", filePath: "config/.env", want: true},
		{name: "deep nested .env", filePath: "services/api/config/.env", want: true},
		{name: ".env.local root", filePath: ".env.local", want: true},
		{name: ".env.local nested", filePath: "config/.env.local", want: true},
		{name: ".env.production", filePath: "config/.env.production", want: true},
		{name: "prefixed env file", filePath: "app.env", want: true},

		// PEM and key files
		{name: "pem file root", filePath: "server.pem", want: true},
		{name: "pem file nested", filePath: "certs/server.pem", want: true},
		{name: "key file root", filePath: "private.key", want: true},
		{name: "key file nested", filePath: "ssl/server.key", want: true},
		{name: "p12 file", filePath: "keystore.p12", want: true},
		{name: "pfx file", filePath: "certificate.pfx", want: true},

		// New certificate/key patterns (T-038)
		{name: "jks file", filePath: "keystore.jks", want: true},
		{name: "keystore file", filePath: "app.keystore", want: true},

		// SSH private key files (T-038)
		{name: "id_rsa root", filePath: "id_rsa", want: true},
		{name: "id_rsa nested", filePath: ".ssh/id_rsa", want: true},
		{name: "id_dsa file", filePath: "id_dsa", want: true},
		{name: "id_ecdsa file", filePath: "id_ecdsa", want: true},
		{name: "id_ed25519 file", filePath: "id_ed25519", want: true},

		// Auth configuration files (T-038)
		{name: ".htpasswd root", filePath: ".htpasswd", want: true},
		{name: ".htpasswd nested", filePath: "config/.htpasswd", want: true},
		{name: ".netrc root", filePath: ".netrc", want: true},
		{name: ".npmrc root", filePath: ".npmrc", want: true},

		// Encrypted/signed files (T-038)
		{name: "gpg file", filePath: "signing.gpg", want: true},
		{name: "asc file", filePath: "key.asc", want: true},

		// Secret/credential/password patterns
		{name: "file with secret in name", filePath: "my_secret_config.json", want: true},
		{name: "file with credential in name", filePath: "credentials.json", want: true},
		{name: "file with password in name", filePath: "passwords.txt", want: true},
		{name: "lowercase secret in name", filePath: "my_secrets.yaml", want: true},
		{name: ".env.example is sensitive", filePath: ".env.example", want: true},

		// Non-sensitive files
		{name: "regular go file", filePath: "main.go", want: false},
		{name: "regular config yaml", filePath: "config.yaml", want: false},
		{name: "readme", filePath: "README.md", want: false},
		{name: "source code", filePath: "internal/server/server.go", want: false},
		{name: "json file", filePath: "package.json", want: false},
		{name: "toml file", filePath: "config.toml", want: false},
		{name: "env in path component not name", filePath: "envsetup/main.go", want: false},
		{name: "environment.go not sensitive", filePath: "environment.go", want: false},
		{name: "pkg/keys/handler.go not sensitive", filePath: "pkg/keys/handler.go", want: false},
		{name: "docs/security-overview.md not sensitive", filePath: "docs/security-overview.md", want: false},

		// Backslash paths (Windows-style normalization)
		{name: "windows path pem", filePath: `certs\server.pem`, want: true},
		{name: "windows path env", filePath: `config\.env`, want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := security.IsSensitiveFile(tt.filePath)
			assert.Equal(t, tt.want, got, "IsSensitiveFile(%q)", tt.filePath)
		})
	}
}

func TestSensitiveFilePatterns(t *testing.T) {
	t.Parallel()

	patterns := security.SensitiveFilePatterns()
	assert.NotEmpty(t, patterns, "SensitiveFilePatterns must not be empty")

	// Verify independence: mutating returned slice does not affect future calls.
	patterns[0] = "mutated"
	patterns2 := security.SensitiveFilePatterns()
	assert.NotEqual(t, "mutated", patterns2[0],
		"SensitiveFilePatterns must return an independent copy")

	// Verify expected patterns are present.
	expected := []string{
		".env",
		"*.pem",
		"*.key",
		"*.jks",
		"id_rsa",
		".htpasswd",
		"*.gpg",
		"*secret*",
	}
	patternSet := make(map[string]bool, len(patterns2))
	for _, p := range patterns2 {
		patternSet[p] = true
	}
	for _, exp := range expected {
		assert.True(t, patternSet[exp], "SensitiveFilePatterns should contain %q", exp)
	}
}

func TestWarnIfSensitiveFile_Suppressed(t *testing.T) {
	t.Parallel()

	// Use a custom slog handler to capture log records.
	var records []slog.Record
	var mu sync.Mutex
	handler := &captureHandler{records: &records, mu: &mu}
	logger := slog.New(handler)

	// With suppressWarning=true, no warning should be emitted.
	security.WarnIfSensitiveFile(".env", true, logger)

	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, records, "WarnIfSensitiveFile should not emit when suppressWarning=true")
}

func TestWarnIfSensitiveFile_EmitsWarn(t *testing.T) {
	t.Parallel()

	var records []slog.Record
	var mu sync.Mutex
	handler := &captureHandler{records: &records, mu: &mu}
	logger := slog.New(handler)

	security.WarnIfSensitiveFile("config/.env", false, logger)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, records, 1, "WarnIfSensitiveFile should emit exactly one warning")
	assert.Equal(t, slog.LevelWarn, records[0].Level)
	assert.Equal(t, "sensitive file included by profile override", records[0].Message)

	// Verify path and matched_pattern attrs are present.
	attrs := make(map[string]string)
	records[0].Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	assert.Equal(t, "config/.env", attrs["path"])
	assert.NotEmpty(t, attrs["matched_pattern"])
}

func TestWarnIfSensitiveFile_NonSensitiveNoWarn(t *testing.T) {
	t.Parallel()

	var records []slog.Record
	var mu sync.Mutex
	handler := &captureHandler{records: &records, mu: &mu}
	logger := slog.New(handler)

	security.WarnIfSensitiveFile("main.go", false, logger)

	mu.Lock()
	defer mu.Unlock()
	assert.Empty(t, records, "WarnIfSensitiveFile should not emit for non-sensitive files")
}

func TestWarnIfSensitiveFile_EmitsOnce(t *testing.T) {
	t.Parallel()

	var records []slog.Record
	var mu sync.Mutex
	handler := &captureHandler{records: &records, mu: &mu}
	logger := slog.New(handler)

	// "secret.env" could match both "*secret*" and "*.env" -- should only warn once.
	security.WarnIfSensitiveFile("secret.env", false, logger)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, records, 1, "WarnIfSensitiveFile must emit exactly one warning per file")
}

// captureHandler is a slog.Handler that captures all log records.
type captureHandler struct {
	records *[]slog.Record
	mu      *sync.Mutex
	attrs   []slog.Attr
	groups  []string
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	// Prepend pre-attached attrs to the record.
	if len(h.attrs) > 0 {
		r.AddAttrs(h.attrs...)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	*h.records = append(*h.records, r)
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &captureHandler{records: h.records, mu: h.mu, attrs: newAttrs, groups: h.groups}
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &captureHandler{records: h.records, mu: h.mu, attrs: h.attrs, groups: newGroups}
}
