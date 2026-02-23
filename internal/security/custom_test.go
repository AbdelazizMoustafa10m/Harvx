package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileCustomPattern_ValidPattern(t *testing.T) {
	t.Parallel()

	rule, err := CompileCustomPattern(
		"my-token",
		"My company token",
		`MY_TOKEN_[A-Z0-9]{16}`,
		"my_company_token",
		ConfidenceHigh,
		[]string{"token"},
	)

	require.NoError(t, err)
	assert.Equal(t, "my-token", rule.ID)
	assert.Equal(t, "My company token", rule.Description)
	assert.Equal(t, "my_company_token", rule.SecretType)
	assert.Equal(t, ConfidenceHigh, rule.Confidence)
	assert.Equal(t, []string{"token"}, rule.Keywords)
	assert.NotNil(t, rule.Regex)
}

func TestCompileCustomPattern_EmptyID(t *testing.T) {
	t.Parallel()

	_, err := CompileCustomPattern(
		"",
		"desc",
		`[A-Z]+`,
		"type",
		ConfidenceMedium,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty id")
}

func TestCompileCustomPattern_EmptyPattern(t *testing.T) {
	t.Parallel()

	_, err := CompileCustomPattern(
		"my-rule",
		"desc",
		"",
		"type",
		ConfidenceHigh,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty regex")
}

func TestCompileCustomPattern_InvalidRegex(t *testing.T) {
	t.Parallel()

	_, err := CompileCustomPattern(
		"my-rule",
		"desc",
		`[invalid(`,
		"type",
		ConfidenceHigh,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "my-rule")
}

func TestCompileCustomPattern_NoKeywords(t *testing.T) {
	t.Parallel()

	rule, err := CompileCustomPattern(
		"no-keywords",
		"no keywords pattern",
		`SECRET_[A-Z]{8}`,
		"my_secret",
		ConfidenceLow,
		nil,
	)

	require.NoError(t, err)
	assert.Nil(t, rule.Keywords)
}

func TestCompileCustomPattern_MatchesContent(t *testing.T) {
	t.Parallel()

	rule, err := CompileCustomPattern(
		"internal-key",
		"Internal API key",
		`(INTERNAL_[A-Z0-9]{8})`,
		"internal_key",
		ConfidenceHigh,
		nil,
	)
	require.NoError(t, err)

	// Verify the compiled regex actually matches expected content.
	match := rule.Regex.FindString("api_key = INTERNAL_ABCD1234")
	assert.NotEmpty(t, match)
}

// TestCompileCustomPattern_TableDriven exercises the happy-path and error
// paths using a single table-driven test. This complements the individual
// tests above with additional boundary cases.
func TestCompileCustomPattern_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		id          string
		description string
		pattern     string
		secretType  string
		confidence  Confidence
		keywords    []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid high-confidence pattern",
			id:          "myco-key",
			description: "Company API key",
			pattern:     `MYCO_[A-Z0-9]{32}`,
			secretType:  "myco_api_key",
			confidence:  ConfidenceHigh,
			keywords:    []string{"MYCO_"},
			wantErr:     false,
		},
		{
			name:        "valid medium-confidence pattern with anchors",
			id:          "bearer-token",
			description: "Bearer token",
			pattern:     `(?i)bearer\s+([a-zA-Z0-9._-]{20,})`,
			secretType:  "bearer_token",
			confidence:  ConfidenceMedium,
			keywords:    nil,
			wantErr:     false,
		},
		{
			name:        "valid low-confidence pattern",
			id:          "generic-hex",
			description: "Generic hex string",
			pattern:     `[0-9a-f]{40}`,
			secretType:  "hex_secret",
			confidence:  ConfidenceLow,
			keywords:    nil,
			wantErr:     false,
		},
		{
			name:        "empty id produces error",
			id:          "",
			description: "no id",
			pattern:     `[A-Z]+`,
			secretType:  "some_type",
			confidence:  ConfidenceMedium,
			keywords:    nil,
			wantErr:     true,
			errContains: "empty id",
		},
		{
			name:        "empty pattern produces error",
			id:          "my-rule",
			description: "no pattern",
			pattern:     "",
			secretType:  "some_type",
			confidence:  ConfidenceHigh,
			keywords:    nil,
			wantErr:     true,
			errContains: "empty regex",
		},
		{
			name:        "invalid regex produces error",
			id:          "bad-regex",
			description: "unclosed bracket",
			pattern:     `[invalid(`,
			secretType:  "some_type",
			confidence:  ConfidenceHigh,
			keywords:    nil,
			wantErr:     true,
			errContains: "bad-regex",
		},
		{
			name:        "multiple keywords stored on rule",
			id:          "multi-kw",
			description: "pattern with multiple keywords",
			pattern:     `TOKEN_[A-Z0-9]{16}`,
			secretType:  "internal_token",
			confidence:  ConfidenceHigh,
			keywords:    []string{"TOKEN_", "token_"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rule, err := CompileCustomPattern(
				tt.id, tt.description, tt.pattern, tt.secretType, tt.confidence, tt.keywords,
			)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.id, rule.ID)
			assert.Equal(t, tt.secretType, rule.SecretType)
			assert.Equal(t, tt.confidence, rule.Confidence)
			assert.NotNil(t, rule.Regex, "compiled regex must not be nil")

			if tt.keywords != nil {
				assert.Equal(t, tt.keywords, rule.Keywords)
			}
		})
	}
}

// TestCompileCustomPattern_RegexMatchesExpectedContent verifies that compiled
// patterns from CompileCustomPattern actually match the content they are
// designed to detect. This provides end-to-end coverage of the compilation
// path all the way through to regex matching.
func TestCompileCustomPattern_RegexMatchesExpectedContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		pattern        string
		matchContent   string
		noMatchContent string
	}{
		{
			name:           "company API key format",
			pattern:        `MYCO_[A-Z0-9]{32}`,
			matchContent:   "api_key = MYCO_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
			noMatchContent: "api_key = OTHER_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
		},
		{
			name:           "database token format",
			pattern:        `DB_TOKEN_[a-f0-9]{24}`,
			matchContent:   "token: DB_TOKEN_abcdef1234567890abcdef12",
			noMatchContent: "token: OTHER_TOKEN_abcdef1234567890abcdef12",
		},
		{
			name:           "anchored prefix",
			pattern:        `^SECRET=(.+)`,
			matchContent:   "SECRET=mysecretvalue",
			noMatchContent: "NOT_SECRET=mysecretvalue",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rule, err := CompileCustomPattern(
				"test-rule",
				"test pattern",
				tt.pattern,
				"test_secret",
				ConfidenceHigh,
				nil,
			)
			require.NoError(t, err)
			require.NotNil(t, rule.Regex)

			// Should match.
			assert.NotEmpty(t, rule.Regex.FindString(tt.matchContent),
				"regex %q must match: %q", tt.pattern, tt.matchContent)

			// Should not match.
			assert.Empty(t, rule.Regex.FindString(tt.noMatchContent),
				"regex %q must not match: %q", tt.pattern, tt.noMatchContent)
		})
	}
}
