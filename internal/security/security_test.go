package security_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/harvx/harvx/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Confidence constants
// ---------------------------------------------------------------------------

func TestConfidenceConstants(t *testing.T) {
	assert.Equal(t, security.Confidence("high"), security.ConfidenceHigh)
	assert.Equal(t, security.Confidence("medium"), security.ConfidenceMedium)
	assert.Equal(t, security.Confidence("low"), security.ConfidenceLow)
}

// ---------------------------------------------------------------------------
// FormatReplacement
// ---------------------------------------------------------------------------

func TestFormatReplacement(t *testing.T) {
	tests := []struct {
		name       string
		secretType string
		want       string
	}{
		{
			name:       "aws access key",
			secretType: "aws_access_key",
			want:       "[REDACTED:aws_access_key]",
		},
		{
			name:       "github token",
			secretType: "github_token",
			want:       "[REDACTED:github_token]",
		},
		{
			name:       "empty type",
			secretType: "",
			want:       "[REDACTED:]",
		},
		{
			name:       "type with spaces",
			secretType: "my custom secret",
			want:       "[REDACTED:my custom secret]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.FormatReplacement(tt.secretType)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// PatternRegistry
// ---------------------------------------------------------------------------

func TestNewDefaultRegistry_HasBuiltinRules(t *testing.T) {
	r := security.NewDefaultRegistry()
	require.NotNil(t, r)
	// After T-035 the default registry is pre-loaded with all built-in
	// detection rules; it must never be empty.
	assert.NotEmpty(t, r.Rules(), "default registry must be pre-loaded with built-in rules")
}

func TestPatternRegistry_Register(t *testing.T) {
	// Use an empty registry so we get a deterministic rule count.
	r := security.NewEmptyRegistry()

	rule := security.RedactionRule{
		ID:         "test-rule",
		SecretType: "test_secret",
		Confidence: security.ConfidenceHigh,
		Regex:      regexp.MustCompile(`TEST[A-Z0-9]{16}`),
	}

	r.Register(rule)

	rules := r.Rules()
	require.Len(t, rules, 1)
	assert.Equal(t, "test-rule", rules[0].ID)
	assert.Equal(t, "test_secret", rules[0].SecretType)
	assert.Equal(t, security.ConfidenceHigh, rules[0].Confidence)
}

func TestPatternRegistry_Register_MultipleRules(t *testing.T) {
	// Use an empty registry so we can assert the exact registration order
	// without accounting for built-in rules.
	r := security.NewEmptyRegistry()

	ruleA := security.RedactionRule{ID: "rule-a", Confidence: security.ConfidenceHigh}
	ruleB := security.RedactionRule{ID: "rule-b", Confidence: security.ConfidenceMedium}
	ruleC := security.RedactionRule{ID: "rule-c", Confidence: security.ConfidenceLow}

	r.Register(ruleA)
	r.Register(ruleB)
	r.Register(ruleC)

	rules := r.Rules()
	require.Len(t, rules, 3)
	// Registration order must be preserved.
	assert.Equal(t, "rule-a", rules[0].ID)
	assert.Equal(t, "rule-b", rules[1].ID)
	assert.Equal(t, "rule-c", rules[2].ID)
}

func TestPatternRegistry_Rules_ReturnsCopy(t *testing.T) {
	// Use an empty registry so [0] is the rule we add, not a built-in rule.
	r := security.NewEmptyRegistry()
	r.Register(security.RedactionRule{ID: "original"})

	copy1 := r.Rules()
	copy1[0] = security.RedactionRule{ID: "mutated"}

	copy2 := r.Rules()
	assert.Equal(t, "original", copy2[0].ID, "mutating returned slice must not affect registry")
}

func TestPatternRegistry_RulesByConfidence(t *testing.T) {
	// Use an empty registry so RulesByConfidence returns only the rules we
	// register here, without built-in rules interfering with the exact ID lists.
	r := security.NewEmptyRegistry()

	r.Register(security.RedactionRule{ID: "h1", Confidence: security.ConfidenceHigh})
	r.Register(security.RedactionRule{ID: "m1", Confidence: security.ConfidenceMedium})
	r.Register(security.RedactionRule{ID: "h2", Confidence: security.ConfidenceHigh})
	r.Register(security.RedactionRule{ID: "l1", Confidence: security.ConfidenceLow})

	tests := []struct {
		name       string
		confidence security.Confidence
		wantIDs    []string
	}{
		{
			name:       "high confidence",
			confidence: security.ConfidenceHigh,
			wantIDs:    []string{"h1", "h2"},
		},
		{
			name:       "medium confidence",
			confidence: security.ConfidenceMedium,
			wantIDs:    []string{"m1"},
		},
		{
			name:       "low confidence",
			confidence: security.ConfidenceLow,
			wantIDs:    []string{"l1"},
		},
		{
			name:       "no match returns empty non-nil slice",
			confidence: security.Confidence("unknown"),
			wantIDs:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.RulesByConfidence(tt.confidence)
			require.NotNil(t, got, "RulesByConfidence must return non-nil slice")
			ids := make([]string, len(got))
			for i, rule := range got {
				ids[i] = rule.ID
			}
			assert.Equal(t, tt.wantIDs, ids)
		})
	}
}

func TestPatternRegistry_RulesByConfidence_ReturnsCopy(t *testing.T) {
	// Use an empty registry so [0] is the rule we add, not a built-in rule.
	r := security.NewEmptyRegistry()
	r.Register(security.RedactionRule{ID: "h1", Confidence: security.ConfidenceHigh})

	copy1 := r.RulesByConfidence(security.ConfidenceHigh)
	copy1[0] = security.RedactionRule{ID: "mutated"}

	copy2 := r.RulesByConfidence(security.ConfidenceHigh)
	assert.Equal(t, "h1", copy2[0].ID, "mutating returned slice must not affect registry")
}

// ---------------------------------------------------------------------------
// RedactionRule fields
// ---------------------------------------------------------------------------

func TestRedactionRule_Fields(t *testing.T) {
	rx := regexp.MustCompile(`(?i)secret[_-]?key\s*[:=]\s*([A-Za-z0-9+/]{20,})`)
	rule := security.RedactionRule{
		ID:               "generic-secret-key",
		Description:      "Generic secret key assignment",
		Regex:            rx,
		Keywords:         []string{"secret", "key"},
		SecretType:       "generic_secret_key",
		Confidence:       security.ConfidenceMedium,
		EntropyThreshold: 3.5,
	}

	assert.Equal(t, "generic-secret-key", rule.ID)
	assert.Equal(t, "Generic secret key assignment", rule.Description)
	assert.Equal(t, rx, rule.Regex)
	assert.Equal(t, []string{"secret", "key"}, rule.Keywords)
	assert.Equal(t, "generic_secret_key", rule.SecretType)
	assert.Equal(t, security.ConfidenceMedium, rule.Confidence)
	assert.Equal(t, 3.5, rule.EntropyThreshold)
}

func TestRedactionRule_ZeroEntropyThreshold_MeansDisabled(t *testing.T) {
	rule := security.RedactionRule{
		EntropyThreshold: 0,
	}
	// A zero EntropyThreshold documents that the entropy check is disabled.
	// There is no behaviour to test here beyond the field existing as float64.
	assert.Equal(t, float64(0), rule.EntropyThreshold)
}

// ---------------------------------------------------------------------------
// RedactionMatch JSON tags (compile-time check via struct literal)
// ---------------------------------------------------------------------------

func TestRedactionMatch_StructFields(t *testing.T) {
	m := security.RedactionMatch{
		RuleID:      "r1",
		SecretType:  "aws_access_key",
		Confidence:  security.ConfidenceHigh,
		FilePath:    "config/secrets.yaml",
		LineNumber:  42,
		StartCol:    10,
		EndCol:      30,
		Replacement: "[REDACTED:aws_access_key]",
	}

	assert.Equal(t, "r1", m.RuleID)
	assert.Equal(t, "aws_access_key", m.SecretType)
	assert.Equal(t, security.ConfidenceHigh, m.Confidence)
	assert.Equal(t, "config/secrets.yaml", m.FilePath)
	assert.Equal(t, 42, m.LineNumber)
	assert.Equal(t, 10, m.StartCol)
	assert.Equal(t, 30, m.EndCol)
	assert.Equal(t, "[REDACTED:aws_access_key]", m.Replacement)
}

// ---------------------------------------------------------------------------
// RedactionSummary
// ---------------------------------------------------------------------------

func TestRedactionSummary_StructFields(t *testing.T) {
	s := security.RedactionSummary{
		TotalCount: 5,
		ByType: map[string]int{
			"aws_access_key": 3,
			"github_token":   2,
		},
		ByConfidence: map[security.Confidence]int{
			security.ConfidenceHigh:   4,
			security.ConfidenceMedium: 1,
		},
		FileCount: 2,
	}

	assert.Equal(t, 5, s.TotalCount)
	assert.Equal(t, 3, s.ByType["aws_access_key"])
	assert.Equal(t, 2, s.ByType["github_token"])
	assert.Equal(t, 4, s.ByConfidence[security.ConfidenceHigh])
	assert.Equal(t, 1, s.ByConfidence[security.ConfidenceMedium])
	assert.Equal(t, 2, s.FileCount)
}

// ---------------------------------------------------------------------------
// RedactionConfig
// ---------------------------------------------------------------------------

func TestRedactionConfig_StructFields(t *testing.T) {
	cfg := security.RedactionConfig{
		Enabled:             true,
		ExcludePaths:        []string{"testdata/**", "docs/**"},
		ConfidenceThreshold: security.ConfidenceMedium,
		CustomPatterns: []security.CustomPatternConfig{
			{
				ID:          "my-company-key",
				Description: "My company API key",
				Pattern:     `MCK[A-Za-z0-9]{32}`,
				SecretType:  "my_company_key",
				Confidence:  security.ConfidenceHigh,
			},
		},
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, []string{"testdata/**", "docs/**"}, cfg.ExcludePaths)
	assert.Equal(t, security.ConfidenceMedium, cfg.ConfidenceThreshold)
	require.Len(t, cfg.CustomPatterns, 1)
	assert.Equal(t, "my-company-key", cfg.CustomPatterns[0].ID)
	assert.Equal(t, "MCK[A-Za-z0-9]{32}", cfg.CustomPatterns[0].Pattern)
}

// ---------------------------------------------------------------------------
// RedactionConfig zero-value
// ---------------------------------------------------------------------------

func TestRedactionConfig_ZeroValue(t *testing.T) {
	var cfg security.RedactionConfig

	assert.False(t, cfg.Enabled, "Enabled must be false by default (zero value of bool)")
	assert.Nil(t, cfg.ExcludePaths, "ExcludePaths must be nil by default")
	assert.Equal(t, security.Confidence(""), cfg.ConfidenceThreshold, "ConfidenceThreshold must be empty string by default")
	assert.Nil(t, cfg.CustomPatterns, "CustomPatterns must be nil by default")
}

// ---------------------------------------------------------------------------
// RedactionMatch JSON marshaling
// ---------------------------------------------------------------------------

func TestRedactionMatch_JSONMarshal(t *testing.T) {
	tests := []struct {
		name      string
		match     security.RedactionMatch
		wantKeys  []string
		wantPairs map[string]interface{}
	}{
		{
			name: "fully populated match",
			match: security.RedactionMatch{
				RuleID:      "aws-access-key",
				SecretType:  "aws_access_key",
				Confidence:  security.ConfidenceHigh,
				FilePath:    "config/secrets.yaml",
				LineNumber:  7,
				StartCol:    4,
				EndCol:      24,
				Replacement: "[REDACTED:aws_access_key]",
			},
			wantKeys: []string{
				"rule_id",
				"secret_type",
				"confidence",
				"file_path",
				"line_number",
				"start_col",
				"end_col",
				"replacement",
			},
			wantPairs: map[string]interface{}{
				"rule_id":     "aws-access-key",
				"secret_type": "aws_access_key",
				"confidence":  "high",
				"file_path":   "config/secrets.yaml",
				"line_number": float64(7),
				"start_col":   float64(4),
				"end_col":     float64(24),
				"replacement": "[REDACTED:aws_access_key]",
			},
		},
		{
			name:  "zero value match uses snake_case keys",
			match: security.RedactionMatch{},
			wantKeys: []string{
				"rule_id",
				"secret_type",
				"confidence",
				"file_path",
				"line_number",
				"start_col",
				"end_col",
				"replacement",
			},
			wantPairs: map[string]interface{}{
				"rule_id":     "",
				"secret_type": "",
				"confidence":  "",
				"file_path":   "",
				"line_number": float64(0),
				"start_col":   float64(0),
				"end_col":     float64(0),
				"replacement": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.match)
			require.NoError(t, err)

			var got map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &got))

			// Verify all expected keys are present with the right names (snake_case).
			for _, key := range tt.wantKeys {
				assert.Contains(t, got, key, "JSON output must contain key %q", key)
			}

			// Verify actual field values round-trip correctly.
			for key, wantVal := range tt.wantPairs {
				assert.Equal(t, wantVal, got[key], "key %q has unexpected value", key)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RedactionSummary JSON marshaling
// ---------------------------------------------------------------------------

func TestRedactionSummary_JSONMarshal(t *testing.T) {
	tests := []struct {
		name      string
		summary   security.RedactionSummary
		wantKeys  []string
		wantPairs map[string]interface{}
	}{
		{
			name: "populated summary",
			summary: security.RedactionSummary{
				TotalCount: 3,
				ByType: map[string]int{
					"aws_access_key": 2,
					"github_token":   1,
				},
				ByConfidence: map[security.Confidence]int{
					security.ConfidenceHigh:   2,
					security.ConfidenceMedium: 1,
				},
				FileCount: 2,
			},
			wantKeys: []string{"total_count", "by_type", "by_confidence", "file_count"},
			wantPairs: map[string]interface{}{
				"total_count": float64(3),
				"file_count":  float64(2),
			},
		},
		{
			name:     "zero value summary",
			summary:  security.RedactionSummary{},
			wantKeys: []string{"total_count", "by_type", "by_confidence", "file_count"},
			wantPairs: map[string]interface{}{
				"total_count": float64(0),
				"file_count":  float64(0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.summary)
			require.NoError(t, err)

			var got map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &got))

			// All top-level keys must be snake_case.
			for _, key := range tt.wantKeys {
				assert.Contains(t, got, key, "JSON output must contain key %q", key)
			}

			// Scalar fields must round-trip correctly.
			for key, wantVal := range tt.wantPairs {
				assert.Equal(t, wantVal, got[key], "key %q has unexpected value", key)
			}
		})
	}
}

func TestRedactionSummary_JSONMarshal_NestedMaps(t *testing.T) {
	summary := security.RedactionSummary{
		TotalCount: 5,
		ByType: map[string]int{
			"aws_access_key": 3,
			"github_token":   2,
		},
		ByConfidence: map[security.Confidence]int{
			security.ConfidenceHigh:   4,
			security.ConfidenceMedium: 1,
		},
		FileCount: 2,
	}

	data, err := json.Marshal(summary)
	require.NoError(t, err)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &got))

	// Verify by_type nested map serializes with the correct string keys and counts.
	byType, ok := got["by_type"].(map[string]interface{})
	require.True(t, ok, "by_type must unmarshal as a JSON object")
	assert.Equal(t, float64(3), byType["aws_access_key"], "by_type[aws_access_key] must be 3")
	assert.Equal(t, float64(2), byType["github_token"], "by_type[github_token] must be 2")

	// Verify by_confidence nested map serializes with Confidence string keys.
	byConf, ok := got["by_confidence"].(map[string]interface{})
	require.True(t, ok, "by_confidence must unmarshal as a JSON object")
	assert.Equal(t, float64(4), byConf["high"], "by_confidence[high] must be 4")
	assert.Equal(t, float64(1), byConf["medium"], "by_confidence[medium] must be 1")
}

// ---------------------------------------------------------------------------
// NewRedactionRule constructor
// ---------------------------------------------------------------------------

func TestNewRedactionRule_ValidPattern(t *testing.T) {
	tests := []struct {
		name             string
		id               string
		description      string
		pattern          string
		keywords         []string
		secretType       string
		confidence       security.Confidence
		entropyThreshold float64
	}{
		{
			name:             "simple literal pattern",
			id:               "test-literal",
			description:      "matches TEST prefix",
			pattern:          `TEST[A-Z0-9]{16}`,
			keywords:         []string{"TEST"},
			secretType:       "test_key",
			confidence:       security.ConfidenceHigh,
			entropyThreshold: 0,
		},
		{
			name:             "capture group pattern",
			id:               "aws-access-key",
			description:      "AWS access key ID",
			pattern:          `(AKIA[0-9A-Z]{16})`,
			keywords:         []string{"aws", "key"},
			secretType:       "aws_access_key",
			confidence:       security.ConfidenceHigh,
			entropyThreshold: 3.5,
		},
		{
			name:             "no keywords",
			id:               "generic-token",
			description:      "Generic bearer token",
			pattern:          `Bearer\s+([A-Za-z0-9\-._~+/]+=*)`,
			keywords:         nil,
			secretType:       "bearer_token",
			confidence:       security.ConfidenceMedium,
			entropyThreshold: 4.0,
		},
		{
			name:             "empty keywords slice",
			id:               "api-key",
			description:      "Generic API key",
			pattern:          `[Aa][Pp][Ii][_-]?[Kk][Ee][Yy]\s*[:=]\s*([A-Za-z0-9]{20,})`,
			keywords:         []string{},
			secretType:       "api_key",
			confidence:       security.ConfidenceLow,
			entropyThreshold: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := security.NewRedactionRule(
				tt.id,
				tt.description,
				tt.pattern,
				tt.keywords,
				tt.secretType,
				tt.confidence,
				tt.entropyThreshold,
			)

			require.NoError(t, err)
			assert.Equal(t, tt.id, rule.ID)
			assert.Equal(t, tt.description, rule.Description)
			assert.NotNil(t, rule.Regex, "Regex must not be nil for a valid pattern")
			assert.Equal(t, tt.pattern, rule.Regex.String(), "compiled regex must preserve original pattern string")
			assert.Equal(t, tt.keywords, rule.Keywords)
			assert.Equal(t, tt.secretType, rule.SecretType)
			assert.Equal(t, tt.confidence, rule.Confidence)
			assert.Equal(t, tt.entropyThreshold, rule.EntropyThreshold)
		})
	}
}

func TestNewRedactionRule_InvalidPattern(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		pattern string
	}{
		{
			name:    "unclosed character class",
			id:      "bad-rule-1",
			pattern: `(invalid[`,
		},
		{
			name:    "unclosed group",
			id:      "bad-rule-2",
			pattern: `(unclosed`,
		},
		{
			name:    "reversed character range",
			id:      "bad-rule-3",
			pattern: `[z-a]`,
		},
		{
			name:    "lookahead not supported in RE2",
			id:      "bad-rule-4",
			pattern: `(?=lookahead)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := security.NewRedactionRule(
				tt.id,
				"test description",
				tt.pattern,
				nil,
				"test_type",
				security.ConfidenceHigh,
				0,
			)

			require.Error(t, err, "NewRedactionRule must return an error for invalid pattern %q", tt.pattern)
			assert.Contains(t, err.Error(), tt.id, "error message must mention the rule ID")
		})
	}
}
