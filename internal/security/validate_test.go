package security_test

import (
	"testing"

	"github.com/harvx/harvx/internal/security"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// validateJWT
// ---------------------------------------------------------------------------

func TestValidateJWT(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid JWT with three non-empty segments",
			input: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			want:  true,
		},
		{
			name:  "valid JWT with minimal segments",
			input: "eyJhbGciOiJub25lIn0.eyJzdWIiOiJ1c2VyIn0.abc123",
			want:  true,
		},
		{
			name:  "only two segments (missing signature)",
			input: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0",
			want:  false,
		},
		{
			name:  "four segments (not a valid JWT)",
			input: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.sig.extra",
			want:  false,
		},
		{
			name:  "empty header segment",
			input: ".eyJzdWIiOiJ1c2VyIn0.abc123",
			want:  false,
		},
		{
			name:  "empty payload segment",
			input: "eyJhbGciOiJIUzI1NiJ9..abc123",
			want:  false,
		},
		{
			name:  "empty signature segment",
			input: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.",
			want:  false,
		},
		{
			name:  "plain string with no dots",
			input: "notajwtatall",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.ValidateJWT(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// validateAWSKeyID
// ---------------------------------------------------------------------------

func TestValidateAWSKeyID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "AKIA prefix exactly 20 chars",
			input: "AKIAIOSFODNN7EXAMPLE",
			want:  true,
		},
		{
			name:  "ASIA prefix exactly 20 chars",
			input: "ASIAIOSFODNN7EXAMPLE",
			want:  true,
		},
		{
			name:  "ABIA prefix exactly 20 chars",
			input: "ABIAIOSFODNN7EXAMPLE",
			want:  true,
		},
		{
			name:  "ACCA prefix exactly 20 chars",
			input: "ACCAIOSFODNN7EXAMPLE",
			want:  true,
		},
		{
			name:  "A3T prefix exactly 20 chars",
			input: "A3T00000000000000001",
			want:  true,
		},
		{
			name:  "too short (19 chars)",
			input: "AKIAIOSFODNN7EXAMPL",
			want:  false,
		},
		{
			name:  "too long (21 chars)",
			input: "AKIAIOSFODNN7EXAMPLEX",
			want:  false,
		},
		{
			name:  "unknown prefix ABCD",
			input: "ABCD1234567890ABCDEF",
			want:  false,
		},
		{
			name:  "lowercase AKIA",
			input: "akiaiosfodnn7example",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "A3T prefix too short (< 20 chars total)",
			input: "A3T000000000000000",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.ValidateAWSKeyID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
