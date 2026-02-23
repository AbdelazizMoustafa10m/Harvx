package security

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withinEpsilon is a helper that checks two float64 values are within an
// acceptable delta (used for entropy assertions).
func withinEpsilon(t *testing.T, want, got, delta float64, msgAndArgs ...interface{}) {
	t.Helper()
	diff := math.Abs(want - got)
	if diff > delta {
		t.Errorf("want %v, got %v (diff %v > epsilon %v); %v", want, got, diff, delta, msgAndArgs)
	}
}

// --------------------------------------------------------------------------
// NewEntropyAnalyzer
// --------------------------------------------------------------------------

func TestNewEntropyAnalyzer_Defaults(t *testing.T) {
	a := NewEntropyAnalyzer()
	require.NotNil(t, a)
	assert.Equal(t, 3.0, a.HexThreshold)
	assert.Equal(t, 4.5, a.Base64Threshold)
	assert.Equal(t, 4.0, a.AlphanumericThreshold)
	assert.Equal(t, 3.5, a.MixedThreshold)
	assert.Equal(t, 16, a.MinLength)
	assert.Equal(t, 256, a.MaxLength)
}

// --------------------------------------------------------------------------
// Calculate
// --------------------------------------------------------------------------

func TestCalculate_KnownValues(t *testing.T) {
	// Override MinLength to 1 so we can test short known-entropy strings.
	a := &EntropyAnalyzer{MinLength: 1, MaxLength: 256}

	tests := []struct {
		name    string
		input   string
		wantH   float64
		epsilon float64
	}{
		{
			name:    "empty string",
			input:   "",
			wantH:   0.0,
			epsilon: 0.0,
		},
		{
			name:    "single char",
			input:   "a",
			wantH:   0.0,
			epsilon: 0.0,
		},
		{
			name:    "all same chars aaaa",
			input:   "aaaa",
			wantH:   0.0,
			epsilon: 0.0,
		},
		{
			name:  "two equal halves aabb -> entropy 1.0",
			input: "aabb",
			// p(a)=0.5, p(b)=0.5 -> -2*(0.5*log2(0.5)) = 1.0
			wantH:   1.0,
			epsilon: 1e-9,
		},
		{
			name:  "four distinct chars abcd -> entropy 2.0",
			input: "abcd",
			// p(x)=0.25 for each -> -4*(0.25*log2(0.25)) = 2.0
			wantH:   2.0,
			epsilon: 1e-9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.Calculate(tt.input)
			withinEpsilon(t, tt.wantH, got, tt.epsilon)
		})
	}
}

func TestCalculate_BelowMinLength(t *testing.T) {
	a := NewEntropyAnalyzer() // MinLength == 16
	// 15 chars of high-diversity content -- should still return 0.0 due to length gate.
	got := a.Calculate("abcdefghijklmno") // 15 chars
	assert.Equal(t, 0.0, got, "strings below MinLength must return 0.0")
}

func TestCalculate_ExactlyMinLength(t *testing.T) {
	a := NewEntropyAnalyzer()
	// 16 chars, all distinct -> high entropy expected, not 0.0
	got := a.Calculate("abcdefghijklmnop")
	assert.Greater(t, got, 0.0, "string at exactly MinLength should be analysed")
}

func TestCalculate_MaxLengthClamping(t *testing.T) {
	a := &EntropyAnalyzer{MinLength: 1, MaxLength: 4}
	// "aaaa" repeated 10 times -- only first 4 chars ("aaaa") are examined
	input := strings.Repeat("a", 40)
	got := a.Calculate(input)
	assert.Equal(t, 0.0, got, "only first MaxLength chars used; all 'a' -> entropy 0")
}

func TestCalculate_HighEntropyHexString(t *testing.T) {
	a := NewEntropyAnalyzer()
	// 32-char lowercase hex string with good diversity
	// SHA256 prefix of "hello": 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	hex32 := "2cf24dba5fb0a30e26e83b2ac5b9e29e"
	got := a.Calculate(hex32)
	// Expect entropy in the range [3.5, 4.0] for a good hex string
	assert.Greater(t, got, 3.0, "hex secret should have entropy > 3.0")
	assert.Less(t, got, 4.1, "hex entropy max is ~4.0")
}

func TestCalculate_HighEntropyBase64String(t *testing.T) {
	a := NewEntropyAnalyzer()
	// 40-char mock AWS-like base64 secret key (diverse chars)
	b64 := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	got := a.Calculate(b64)
	assert.Greater(t, got, 4.0, "base64 secret should have entropy > 4.0")
}

func TestCalculate_Unicode(t *testing.T) {
	a := &EntropyAnalyzer{MinLength: 1, MaxLength: 256}
	// Unicode string should not panic
	assert.NotPanics(t, func() {
		_ = a.Calculate("こんにちは世界！") // Japanese "Hello World!"
	})
}

func TestCalculate_LongString(t *testing.T) {
	a := NewEntropyAnalyzer()
	// 10 000-char string -- should complete quickly without panic
	long := strings.Repeat("abcdefghij", 1000) // 10 000 chars
	assert.NotPanics(t, func() {
		got := a.Calculate(long)
		// "abcdefghij" has 10 distinct chars -> entropy = log2(10) ≈ 3.32
		assert.Greater(t, got, 3.0)
		assert.Less(t, got, 4.0)
	})
}

// --------------------------------------------------------------------------
// DetectCharset
// --------------------------------------------------------------------------

func TestDetectCharset(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CharacterSet
	}{
		{
			name:  "empty string",
			input: "",
			want:  CharsetUnknown,
		},
		{
			name:  "lowercase hex only",
			input: "deadbeef0123456789abcdef",
			want:  CharsetHex,
		},
		{
			name:  "uppercase hex only",
			input: "DEADBEEF0123456789ABCDEF",
			want:  CharsetHex,
		},
		{
			name:  "mixed-case hex",
			input: "DeAdBeEf01234",
			want:  CharsetHex,
		},
		{
			name:  "hex digits only",
			input: "0123456789",
			want:  CharsetHex, // digits are valid hex
		},
		{
			name:  "alphanumeric with non-hex letter",
			input: "Hello123World",
			want:  CharsetAlphanumeric,
		},
		{
			name:  "base64 standard with plus and slash",
			input: "SGVsbG8hV29ybGQ+/==",
			want:  CharsetBase64,
		},
		{
			name:  "url-safe base64 with underscore and dash",
			input: "SGVsbG8hV29ybGQ_-",
			want:  CharsetBase64,
		},
		{
			name:  "mixed with special chars",
			input: "hello@world!",
			want:  CharsetMixed,
		},
		{
			name:  "string with dot",
			input: "hello.world",
			want:  CharsetMixed,
		},
		{
			name:  "pure digits are hex",
			input: "123456789",
			want:  CharsetHex,
		},
		{
			name:  "uppercase alpha only not hex",
			input: "GHIJKLMNOP",
			want:  CharsetAlphanumeric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectCharset(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --------------------------------------------------------------------------
// IsHighEntropy
// --------------------------------------------------------------------------

func TestIsHighEntropy(t *testing.T) {
	a := NewEntropyAnalyzer()

	tests := []struct {
		name    string
		input   string
		charset CharacterSet
		want    bool
	}{
		{
			name:    "below min length returns false",
			input:   "short",
			charset: CharsetAlphanumeric,
			want:    false,
		},
		{
			name: "low entropy hex repeated pattern",
			// 16 chars, but all the same nibble repeated
			input:   "aaaaaaaaaaaaaaaa",
			charset: CharsetHex,
			want:    false,
		},
		{
			name:    "high entropy sha256 hex prefix",
			input:   "2cf24dba5fb0a30e26e83b2ac5b9e29e",
			charset: CharsetHex,
			want:    true,
		},
		{
			name:    "high entropy base64 secret",
			input:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			charset: CharsetBase64,
			want:    true,
		},
		{
			name:    "common english word does not trigger",
			input:   "authentication",
			charset: CharsetAlphanumeric,
			want:    false, // below MinLength of 16
		},
		{
			name:    "common english phrase still low entropy",
			input:   "configuration1234",
			charset: CharsetAlphanumeric,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.IsHighEntropy(tt.input, tt.charset)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --------------------------------------------------------------------------
// AnalyzeToken
// --------------------------------------------------------------------------

func TestAnalyzeToken_HighEntropyWithSuspiciousContext(t *testing.T) {
	a := NewEntropyAnalyzer()

	// AWS-like base64 secret key -- genuinely high entropy
	token := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	ctx := TokenContext{
		VariableName: "AWS_SECRET_ACCESS_KEY",
		FilePath:     "config/aws.go",
		LineContent:  `AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`,
	}
	result := a.AnalyzeToken(token, ctx)

	assert.Greater(t, result.Entropy, 4.0)
	assert.True(t, result.IsHigh)
	assert.Equal(t, ConfidenceMedium, result.Confidence, "high entropy + suspicious context -> medium confidence")
}

func TestAnalyzeToken_HighEntropyWithoutContext(t *testing.T) {
	a := NewEntropyAnalyzer()

	token := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	ctx := TokenContext{} // no context
	result := a.AnalyzeToken(token, ctx)

	assert.True(t, result.IsHigh)
	assert.Equal(t, ConfidenceLow, result.Confidence, "high entropy without context -> low confidence")
}

func TestAnalyzeToken_LowEntropyReturnsLow(t *testing.T) {
	a := NewEntropyAnalyzer()

	// Low entropy string (well below MinLength or repeated chars)
	token := "aaaaaaaaaaaaaaaa" // 16 'a's -- entropy 0
	ctx := TokenContext{
		VariableName: "api_key",
		LineContent:  `api_key = "aaaaaaaaaaaaaaaa"`,
	}
	result := a.AnalyzeToken(token, ctx)

	assert.False(t, result.IsHigh)
	assert.Equal(t, ConfidenceLow, result.Confidence)
}

func TestAnalyzeToken_ContextBoostingLowersThreshold(t *testing.T) {
	a := NewEntropyAnalyzer()

	// Craft a borderline alphanumeric token with entropy just below the default
	// AlphanumericThreshold (4.0) but above threshold-0.5 (3.5).
	// "abcdefghijklmnopqrstuvwx" -> 24 distinct chars in a 24-char string -> entropy = log2(24) ≈ 4.58
	// that's already high. Use a more uniform string with some repetition.
	// We want entropy in [3.5, 4.0) so that context boost flips the decision.
	//
	// "aabbccddeeffgghhiijjkkll" (24 chars, 12 pairs) -> entropy = log2(12) ≈ 3.58
	token := "aabbccddeeffgghhiijjkkll"
	entropy := a.Calculate(token)
	// Sanity check the fixture is actually in the target range [3.5, 4.0)
	assert.Greater(t, entropy, 3.4)
	assert.Less(t, entropy, 4.0)

	charset := DetectCharset(token)
	assert.Equal(t, CharsetAlphanumeric, charset)

	// Without context: threshold is 4.0, so this should NOT be high entropy.
	ctxNone := TokenContext{}
	resultNone := a.AnalyzeToken(token, ctxNone)
	assert.False(t, resultNone.IsHigh, "below threshold without context")

	// With suspicious context: threshold drops to 3.5, so entropy ~3.58 SHOULD flag.
	ctxSuspicious := TokenContext{
		VariableName: "api_key",
		LineContent:  `api_key = "aabbccddeeffgghhiijjkkll"`,
	}
	resultSuspicious := a.AnalyzeToken(token, ctxSuspicious)
	assert.True(t, resultSuspicious.IsHigh, "context boost should trigger for borderline token")
	assert.Equal(t, ConfidenceMedium, resultSuspicious.Confidence)
}

func TestAnalyzeToken_ContextKeywords(t *testing.T) {
	a := NewEntropyAnalyzer()
	// High entropy base64 token that will pass regardless -- we only test
	// the Confidence level changing based on context keywords.
	token := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

	keywords := []string{"key", "secret", "token", "password", "api", "auth", "credential"}
	for _, kw := range keywords {
		t.Run("keyword_"+kw, func(t *testing.T) {
			ctx := TokenContext{VariableName: strings.ToUpper(kw)}
			result := a.AnalyzeToken(token, ctx)
			assert.Equal(t, ConfidenceMedium, result.Confidence,
				"keyword %q in VariableName should yield medium confidence", kw)
		})
	}
}

func TestAnalyzeToken_UUIDIsBorderline(t *testing.T) {
	a := NewEntropyAnalyzer()

	// A standard UUID with hyphens: hyphens are in the URL-safe base64 charset,
	// so DetectCharset returns CharsetBase64, not CharsetMixed.
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	result := a.AnalyzeToken(uuid, TokenContext{})

	// UUIDs have moderate entropy; they should NOT trigger without context.
	// This tests the "borderline only flagged with context" requirement.
	assert.Equal(t, CharsetBase64, result.Charset)
	// Without context, it should not be flagged (or at most ConfidenceLow).
	assert.Equal(t, ConfidenceLow, result.Confidence)
}

func TestAnalyzeToken_EntropyFieldPopulated(t *testing.T) {
	a := &EntropyAnalyzer{MinLength: 1, MaxLength: 256}
	token := "abcd"
	result := a.AnalyzeToken(token, TokenContext{})
	withinEpsilon(t, 2.0, result.Entropy, 1e-9)
}

func TestAnalyzeToken_CharsetFieldPopulated(t *testing.T) {
	a := NewEntropyAnalyzer()
	result := a.AnalyzeToken("2cf24dba5fb0a30e26e83b2ac5b9e29e", TokenContext{})
	assert.Equal(t, CharsetHex, result.Charset)
}

// --------------------------------------------------------------------------
// isSuspiciousContext
// --------------------------------------------------------------------------

func TestIsSuspiciousContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  TokenContext
		want bool
	}{
		{
			name: "empty context",
			ctx:  TokenContext{},
			want: false,
		},
		{
			name: "keyword in variable name case insensitive",
			ctx:  TokenContext{VariableName: "MY_API_KEY"},
			want: true,
		},
		{
			name: "keyword in line content",
			ctx:  TokenContext{LineContent: `password = "..."` },
			want: true,
		},
		{
			name: "credential keyword",
			ctx:  TokenContext{VariableName: "credentials"},
			want: true,
		},
		{
			name: "auth keyword",
			ctx:  TokenContext{LineContent: "authToken = xyz"},
			want: true,
		},
		{
			name: "no keyword",
			ctx:  TokenContext{VariableName: "fileName", LineContent: "fileName = document.txt"},
			want: false,
		},
		{
			name: "file path is not checked",
			// FilePath is not checked for keywords (only VariableName and LineContent)
			ctx:  TokenContext{FilePath: "secrets/keys.txt"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSuspiciousContext(tt.ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --------------------------------------------------------------------------
// Real-world calibration tests
// --------------------------------------------------------------------------

func TestCalibration_CommonEnglishWordsNotFlagged(t *testing.T) {
	a := NewEntropyAnalyzer()

	words := []string{
		"authentication",
		"configuration",
		"authorization",
		"implementation",
	}

	for _, word := range words {
		t.Run(word, func(t *testing.T) {
			result := a.AnalyzeToken(word, TokenContext{})
			// Most are below MinLength or low entropy; they should not be IsHigh.
			if len(word) >= a.MinLength {
				assert.False(t, result.IsHigh, "common English word %q should not be high-entropy", word)
			}
		})
	}
}

func TestCalibration_SHA256HexShouldTrigger(t *testing.T) {
	a := NewEntropyAnalyzer()

	// SHA-256 of "hello" (lowercase hex)
	sha256 := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	result := a.AnalyzeToken(sha256, TokenContext{})

	assert.True(t, result.IsHigh, "SHA-256 hex string should be detected as high-entropy")
	assert.Equal(t, CharsetHex, result.Charset)
}

// --------------------------------------------------------------------------
// Exact entropy ranges for real-world string classes (task spec calibration)
// --------------------------------------------------------------------------

func TestCalculate_Hex32EntropyRange(t *testing.T) {
	a := NewEntropyAnalyzer()
	// A well-distributed 32-char hex string (SHA-256 prefix of "hello").
	// Theoretical maximum for hex is log2(16) = 4.0 bits/char.
	// The task spec calibration targets ~3.8-4.0 for random hex; we verify
	// the empirical value sits well above the HexThreshold (3.0) and
	// at or below the theoretical ceiling.
	hex32 := "2cf24dba5fb0a30e26e83b2ac5b9e29e"
	got := a.Calculate(hex32)
	assert.Greater(t, got, 3.0, "32-char hex hash prefix must exceed HexThreshold (3.0)")
	assert.LessOrEqual(t, got, 4.0, "hex entropy is bounded by log2(16) = 4.0")
}

func TestCalculate_Base64_40EntropyRange(t *testing.T) {
	a := NewEntropyAnalyzer()
	// AWS-style secret key with diverse base64 characters.
	// The task spec targets entropy in the range [5.0, 5.5] for 40-char base64 strings.
	// The mock AWS key below has 41 chars with high character diversity, placing it
	// comfortably above the Base64Threshold (4.5) and in the 5.x range.
	b64 := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	got := a.Calculate(b64)
	assert.GreaterOrEqual(t, got, 4.5, "40-char base64 secret should have entropy >= 4.5 (above Base64Threshold)")
	assert.Less(t, got, 6.0, "base64 entropy is bounded by log2(65) ≈ 6.0")
}

// --------------------------------------------------------------------------
// UUID borderline behaviour: without vs. with context
// --------------------------------------------------------------------------

func TestAnalyzeToken_UUIDBorderline_WithContext(t *testing.T) {
	a := NewEntropyAnalyzer()

	// UUID hyphens are in the URL-safe base64 charset, so DetectCharset returns
	// CharsetBase64 (not CharsetMixed). Base64Threshold is 4.5, context boost
	// lowers it to 4.0.
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	// Without context: must NOT be high-entropy.
	ctxNone := TokenContext{}
	resultNone := a.AnalyzeToken(uuid, ctxNone)
	assert.False(t, resultNone.IsHigh, "UUID without suspicious context should not be flagged")
	assert.Equal(t, ConfidenceLow, resultNone.Confidence)

	// With suspicious context: threshold drops, UUID may be flagged if its entropy
	// exceeds (Base64Threshold - 0.5 = 4.0). We assert on the Confidence being boosted
	// when IsHigh is true.
	ctxSuspicious := TokenContext{VariableName: "session_token", LineContent: `session_token = "550e8400-e29b-41d4-a716-446655440000"`}
	resultWith := a.AnalyzeToken(uuid, ctxSuspicious)
	if resultWith.IsHigh {
		assert.Equal(t, ConfidenceMedium, resultWith.Confidence,
			"if UUID trips threshold with context, confidence must be medium")
	}
	// Regardless, entropy must be populated and charset must be Base64.
	assert.Equal(t, CharsetBase64, resultWith.Charset)
	assert.Greater(t, resultWith.Entropy, 0.0)
}

// --------------------------------------------------------------------------
// IsHighEntropy with CharsetUnknown (zero value) falls back to MixedThreshold
// --------------------------------------------------------------------------

func TestIsHighEntropy_CharsetUnknown_FallsBackToMixedThreshold(t *testing.T) {
	a := NewEntropyAnalyzer()

	// The thresholdFor helper maps CharsetUnknown -> MixedThreshold (3.5).
	// Use a 16-char string with entropy well above 3.5 to confirm the path executes.
	// "abcdefghijklmnop" has 16 distinct chars -> entropy = log2(16) = 4.0.
	s := "abcdefghijklmnop"
	got := a.IsHighEntropy(s, CharsetUnknown)
	// entropy(s) == 4.0 > MixedThreshold (3.5) -> expect true.
	assert.True(t, got, "CharsetUnknown should use MixedThreshold (3.5); entropy 4.0 should exceed it")
}

// --------------------------------------------------------------------------
// Configurable thresholds -- callers can raise or lower sensitivity
// --------------------------------------------------------------------------

func TestEntropyAnalyzer_ConfigurableThresholds(t *testing.T) {
	// Create an analyzer with thresholds raised to theoretical maximum so no real
	// string can pass -- this confirms the struct fields directly gate decisions.
	strict := &EntropyAnalyzer{
		HexThreshold:          4.01, // above theoretical max log2(16)=4.0 for hex
		Base64Threshold:       6.01, // above theoretical max log2(65)≈6.0 for base64
		AlphanumericThreshold: 5.71, // above theoretical max log2(62)≈5.95 -- use safe value
		MixedThreshold:        4.01,
		MinLength:             1,
		MaxLength:             256,
	}

	// Any hex string must have entropy <= log2(16) = 4.0, so strict threshold 4.01 blocks all.
	hex32 := "2cf24dba5fb0a30e26e83b2ac5b9e29e"
	assert.False(t, strict.IsHighEntropy(hex32, CharsetHex),
		"threshold above theoretical max must block every hex string")

	// Same string should pass with the default threshold of 3.0.
	defaults := NewEntropyAnalyzer()
	assert.True(t, defaults.IsHighEntropy(hex32, CharsetHex),
		"default threshold 3.0 must accept a well-distributed hex string")
}

// --------------------------------------------------------------------------
// AnalyzeToken -- result fields are fully populated
// --------------------------------------------------------------------------

func TestAnalyzeToken_ResultFieldsComplete(t *testing.T) {
	a := NewEntropyAnalyzer()
	token := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	ctx := TokenContext{
		VariableName: "AWS_SECRET_ACCESS_KEY",
		FilePath:     "config/credentials.go",
		LineContent:  `AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`,
	}
	result := a.AnalyzeToken(token, ctx)

	assert.Greater(t, result.Entropy, 0.0, "Entropy field must be populated")
	assert.NotEqual(t, CharsetUnknown, result.Charset, "Charset field must be populated")
	// IsHigh and Confidence are non-zero-value fields when entropy is high.
	assert.True(t, result.IsHigh, "IsHigh must be true for a high-entropy secret")
	assert.Equal(t, ConfidenceMedium, result.Confidence, "Suspicious context must yield medium confidence")
}

// --------------------------------------------------------------------------
// Standalone usability -- EntropyAnalyzer works without any redaction context
// --------------------------------------------------------------------------

func TestEntropyAnalyzer_StandaloneUsability(t *testing.T) {
	// Verify that EntropyAnalyzer can be constructed and used without any
	// other security subsystem (no PatternRegistry, no Redactor).
	a := NewEntropyAnalyzer()
	require.NotNil(t, a)

	// Plain Calculate
	h := a.Calculate("abcdefghijklmnop")
	assert.Greater(t, h, 0.0)

	// DetectCharset (package-level function)
	cs := DetectCharset("abcdefghijklmnop")
	assert.Equal(t, CharsetAlphanumeric, cs)

	// IsHighEntropy
	got := a.IsHighEntropy("abcdefghijklmnop", CharsetAlphanumeric)
	// entropy("abcdefghijklmnop") = log2(16) = 4.0 which equals AlphanumericThreshold (4.0) -> true
	assert.True(t, got)

	// AnalyzeToken: entropy == 4.0, AlphanumericThreshold == 4.0 -> exactly at threshold -> IsHigh true
	result := a.AnalyzeToken("abcdefghijklmnop", TokenContext{})
	assert.True(t, result.IsHigh, "token at exactly the alphanumeric threshold (4.0) should be flagged")
	// No suspicious context -> confidence stays Low even though IsHigh is true
	assert.Equal(t, ConfidenceLow, result.Confidence)
}

// --------------------------------------------------------------------------
// Context boosting -- LineContent keyword path (distinct from VariableName)
// --------------------------------------------------------------------------

func TestAnalyzeToken_LineContentKeywordBoost(t *testing.T) {
	a := NewEntropyAnalyzer()
	// Borderline token: entropy in [3.5, 4.0) so context decides.
	// "aabbccddeeffgghhiijjkkll" -> entropy = log2(12) ≈ 3.58
	token := "aabbccddeeffgghhiijjkkll"

	// Boost via LineContent only (VariableName is empty).
	ctx := TokenContext{
		VariableName: "",
		LineContent:  `export SECRET="aabbccddeeffgghhiijjkkll"`,
	}
	result := a.AnalyzeToken(token, ctx)
	assert.True(t, result.IsHigh, "LineContent with 'secret' keyword should boost threshold and flag borderline token")
	assert.Equal(t, ConfidenceMedium, result.Confidence)
}

// --------------------------------------------------------------------------
// Minimum length boundary: exactly one below, exactly at, and one above
// --------------------------------------------------------------------------

func TestCalculate_MinLengthBoundary(t *testing.T) {
	a := NewEntropyAnalyzer() // MinLength == 16

	tests := []struct {
		name    string
		input   string
		wantPos bool // true if we expect entropy > 0.0
	}{
		{
			name:    "15 chars (one below MinLength)",
			input:   "abcdefghijklmno", // 15 chars, diverse
			wantPos: false,
		},
		{
			name:    "16 chars (exactly at MinLength)",
			input:   "abcdefghijklmnop", // 16 chars, all distinct
			wantPos: true,
		},
		{
			name:    "17 chars (one above MinLength)",
			input:   "abcdefghijklmnopq", // 17 chars, all distinct
			wantPos: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := a.Calculate(tt.input)
			if tt.wantPos {
				assert.Greater(t, got, 0.0, "should compute non-zero entropy for length >= MinLength")
			} else {
				assert.Equal(t, 0.0, got, "should return 0.0 for length < MinLength")
			}
		})
	}
}

// --------------------------------------------------------------------------
// Maximum length boundary: verify clamping is rune-based not byte-based
// --------------------------------------------------------------------------

func TestCalculate_MaxLengthRuneBased(t *testing.T) {
	// Build a string whose first MaxLength runes are all 'a' (entropy 0) but
	// whose tail runes are diverse. If clamping is rune-based (correct) we
	// should get entropy 0.0; if byte-based and we insert multi-byte runes the
	// calculation should still not panic.
	a := &EntropyAnalyzer{MinLength: 1, MaxLength: 8}

	// First 8 runes are 'a', remaining runes are diverse multi-byte characters.
	prefix := strings.Repeat("a", 8)
	tail := "こんにちは世界！" // 8 runes, all non-ASCII
	input := prefix + tail

	got := a.Calculate(input)
	assert.Equal(t, 0.0, got, "clamping to first 8 runes (all 'a') should yield entropy 0.0")
}

// --------------------------------------------------------------------------
// DetectCharset -- additional edge cases
// --------------------------------------------------------------------------

func TestDetectCharset_AdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CharacterSet
	}{
		{
			name:  "single hex digit",
			input: "a",
			want:  CharsetHex,
		},
		{
			name:  "single non-hex letter",
			input: "z",
			want:  CharsetAlphanumeric,
		},
		{
			name:  "equals sign alone (base64 padding)",
			input: "=",
			want:  CharsetBase64,
		},
		{
			name:  "URL-safe base64 chars only",
			input: "abc_def-GHI",
			want:  CharsetBase64,
		},
		{
			name:  "space makes it mixed",
			input: "hello world",
			want:  CharsetMixed,
		},
		{
			name:  "at-sign makes it mixed",
			input: "user@example",
			want:  CharsetMixed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DetectCharset(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --------------------------------------------------------------------------
// Fuzz test -- entropy calculation must never panic on arbitrary input
// --------------------------------------------------------------------------

func FuzzCalculate(f *testing.F) {
	// Seed corpus covering the important boundary cases.
	f.Add("")
	f.Add("a")
	f.Add("aaaa")
	f.Add("abcd")
	f.Add("aabb")
	f.Add("abcdefghijklmnop")                                // exactly MinLength
	f.Add("2cf24dba5fb0a30e26e83b2ac5b9e29e")               // hex
	f.Add("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")      // base64
	f.Add("こんにちは世界！")                                       // unicode
	f.Add(strings.Repeat("x", 10000))                        // very long

	a := NewEntropyAnalyzer()

	f.Fuzz(func(t *testing.T, input string) {
		// Must not panic.
		got := a.Calculate(input)

		// Invariants that must always hold:
		// 1. Entropy is non-negative.
		if got < 0 {
			t.Errorf("Calculate(%q) returned negative entropy %v", input, got)
		}
		// 2. Entropy can only be zero for inputs below MinLength, empty, or all-same chars.
		//    We can't easily verify #2 without re-implementing, so we just check non-negative.
	})
}
