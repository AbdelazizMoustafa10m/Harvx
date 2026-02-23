package security

import (
	"math"
	"strings"
)

// CharacterSet identifies the character composition of a string.
type CharacterSet int

const (
	// CharsetUnknown is returned when the string is empty or classification
	// cannot be determined.
	CharsetUnknown CharacterSet = iota

	// CharsetHex indicates the string contains only hexadecimal characters
	// [0-9a-fA-F].
	CharsetHex

	// CharsetBase64 indicates the string contains only characters from the
	// standard or URL-safe base64 alphabet [A-Za-z0-9+/=_-].
	CharsetBase64

	// CharsetAlphanumeric indicates the string contains only ASCII letters
	// and digits [A-Za-z0-9].
	CharsetAlphanumeric

	// CharsetMixed indicates the string contains special characters beyond
	// the base64 / alphanumeric sets.
	CharsetMixed
)

// EntropyResult holds the result of analysing a token for information entropy.
type EntropyResult struct {
	// Entropy is the Shannon entropy of the token in bits per character.
	Entropy float64

	// Charset is the detected character set of the token.
	Charset CharacterSet

	// IsHigh reports whether the entropy exceeds the threshold for the
	// detected character set (after any context-based threshold adjustment).
	IsHigh bool

	// Confidence is the suggested confidence level for a secret match based
	// on the entropy signal and surrounding context.
	Confidence Confidence
}

// TokenContext provides contextual information about where a candidate token
// appears. It is used to boost detection confidence when the token is
// surrounded by suspicious variable names or keywords.
type TokenContext struct {
	// VariableName is the name of the variable or key that holds the token
	// (e.g. "API_KEY", "password", "token"). May be empty.
	VariableName string

	// FilePath is the path of the file that contains the token
	// (e.g. "config/settings.go"). May be empty.
	FilePath string

	// LineContent is the full text of the line that contains the token.
	// May be empty.
	LineContent string
}

// EntropyAnalyzer calculates and evaluates Shannon entropy of strings to
// detect high-entropy values that may be secrets.
//
// All threshold fields are public so callers can tune sensitivity. The zero
// value is not useful; create instances with NewEntropyAnalyzer.
type EntropyAnalyzer struct {
	// HexThreshold is the minimum entropy (bits/char) required for a hex
	// string to be considered high-entropy. Default 3.0 (max theoretical ~4.0).
	HexThreshold float64

	// Base64Threshold is the minimum entropy required for a base64 string.
	// Default 4.5 (max theoretical ~6.0).
	Base64Threshold float64

	// AlphanumericThreshold is the minimum entropy required for an
	// alphanumeric string. Default 4.0 (max theoretical ~5.7).
	AlphanumericThreshold float64

	// MixedThreshold is the minimum entropy required for a mixed-character
	// string. Default 3.5.
	MixedThreshold float64

	// MinLength is the minimum number of characters a string must have for
	// entropy analysis to proceed. Shorter strings are skipped. Default 16.
	MinLength int

	// MaxLength is the maximum number of characters used for entropy
	// calculation. Only the first MaxLength characters are examined.
	// Default 256.
	MaxLength int
}

// NewEntropyAnalyzer returns an EntropyAnalyzer initialised with default
// thresholds calibrated to minimise false positives on source code.
func NewEntropyAnalyzer() *EntropyAnalyzer {
	return &EntropyAnalyzer{
		HexThreshold:          3.0,
		Base64Threshold:       4.5,
		AlphanumericThreshold: 4.0,
		MixedThreshold:        3.5,
		MinLength:             16,
		MaxLength:             256,
	}
}

// Calculate computes the Shannon entropy of s in bits per character.
//
//	H = -sum(p_i * log2(p_i))
//
// An empty string or a string with a single unique character returns 0.0.
// If s has fewer runes than a.MinLength, 0.0 is returned immediately.
// If s has more runes than a.MaxLength, only the first a.MaxLength runes are
// examined. Length comparisons are rune-based so that multi-byte Unicode
// characters are counted correctly.
func (a *EntropyAnalyzer) Calculate(s string) float64 {
	// Convert to rune slice once so that all length operations are rune-based
	// and we never split a multi-byte character during clamping.
	runes := []rune(s)
	n := len(runes)

	if n < a.MinLength {
		return 0.0
	}

	// Clamp to MaxLength.
	if n > a.MaxLength {
		runes = runes[:a.MaxLength]
		n = a.MaxLength
	}

	if n <= 1 {
		return 0.0
	}

	// Count rune frequencies using a fixed-size array for ASCII (fast path) and
	// a map for non-ASCII runes. Most secrets are pure ASCII, so the array path
	// dominates.
	var asciiFreq [128]int
	var extraFreq map[rune]int

	for _, r := range runes {
		if r < 128 {
			asciiFreq[r]++
		} else {
			if extraFreq == nil {
				extraFreq = make(map[rune]int)
			}
			extraFreq[r]++
		}
	}

	fTotal := float64(n)
	var h float64

	for _, count := range asciiFreq {
		if count == 0 {
			continue
		}
		p := float64(count) / fTotal
		h -= p * math.Log2(p)
	}
	for _, count := range extraFreq {
		p := float64(count) / fTotal
		h -= p * math.Log2(p)
	}

	return h
}

// DetectCharset classifies the character composition of s.
//
// The classification hierarchy, from most to least restrictive, is:
//  1. CharsetHex -- only [0-9a-fA-F]
//  2. CharsetAlphanumeric -- only [A-Za-z0-9] (broader than hex)
//  3. CharsetBase64 -- only [A-Za-z0-9+/=_-] (broader than alphanumeric)
//  4. CharsetMixed -- anything else
//
// An empty string returns CharsetUnknown.
func DetectCharset(s string) CharacterSet {
	if s == "" {
		return CharsetUnknown
	}

	isHex := true
	isAlphanumeric := true
	isBase64 := true

	for _, r := range s {
		// Hex: [0-9a-fA-F]
		hexChar := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		if !hexChar {
			isHex = false
		}

		// Alphanumeric: [A-Za-z0-9]
		alphanumChar := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if !alphanumChar {
			isAlphanumeric = false
		}

		// Base64 (standard + URL-safe): [A-Za-z0-9+/=_-]
		base64Char := alphanumChar || r == '+' || r == '/' || r == '=' || r == '_' || r == '-'
		if !base64Char {
			isBase64 = false
		}

		// Short-circuit: if none of the sets apply, it is mixed.
		if !isHex && !isAlphanumeric && !isBase64 {
			return CharsetMixed
		}
	}

	switch {
	case isHex:
		return CharsetHex
	case isAlphanumeric:
		return CharsetAlphanumeric
	case isBase64:
		return CharsetBase64
	default:
		return CharsetMixed
	}
}

// IsHighEntropy reports whether s has entropy above the threshold for the
// given character set. The decision uses the analyzer's configured thresholds.
//
// If s is shorter than a.MinLength, the function always returns false.
func (a *EntropyAnalyzer) IsHighEntropy(s string, charset CharacterSet) bool {
	entropy := a.Calculate(s)
	if entropy == 0.0 {
		return false
	}
	threshold := a.thresholdFor(charset)
	return entropy >= threshold
}

// AnalyzeToken performs a full entropy analysis of token and returns an
// EntropyResult with entropy value, charset, high-entropy flag, and suggested
// confidence level.
//
// Context boosting: when ctx.VariableName or ctx.LineContent contains one of
// the suspicious keywords (key, secret, token, password, api, auth,
// credential), the effective threshold is lowered by 0.5 to catch borderline
// cases.
//
// Confidence assignment:
//   - High entropy + suspicious context -> ConfidenceMedium
//   - High entropy without context       -> ConfidenceLow
//   - Low entropy                        -> ConfidenceLow
func (a *EntropyAnalyzer) AnalyzeToken(token string, ctx TokenContext) EntropyResult {
	charset := DetectCharset(token)
	entropy := a.Calculate(token)

	threshold := a.thresholdFor(charset)
	suspicious := isSuspiciousContext(ctx)
	if suspicious {
		threshold -= 0.5
	}

	isHigh := entropy > 0.0 && entropy >= threshold

	var confidence Confidence
	switch {
	case isHigh && suspicious:
		confidence = ConfidenceMedium
	default:
		confidence = ConfidenceLow
	}

	return EntropyResult{
		Entropy:    entropy,
		Charset:    charset,
		IsHigh:     isHigh,
		Confidence: confidence,
	}
}

// thresholdFor returns the entropy threshold for the given CharacterSet.
func (a *EntropyAnalyzer) thresholdFor(charset CharacterSet) float64 {
	switch charset {
	case CharsetHex:
		return a.HexThreshold
	case CharsetBase64:
		return a.Base64Threshold
	case CharsetAlphanumeric:
		return a.AlphanumericThreshold
	case CharsetMixed:
		return a.MixedThreshold
	default:
		return a.MixedThreshold
	}
}

// suspiciousKeywords is the set of lower-case keywords that indicate a
// variable or line is likely to hold a credential.
var suspiciousKeywords = []string{
	"key", "secret", "token", "password", "api", "auth", "credential",
}

// isSuspiciousContext returns true when ctx.VariableName or ctx.LineContent
// contains at least one suspicious keyword (case-insensitive).
func isSuspiciousContext(ctx TokenContext) bool {
	lowerVar := strings.ToLower(ctx.VariableName)
	lowerLine := strings.ToLower(ctx.LineContent)

	for _, kw := range suspiciousKeywords {
		if strings.Contains(lowerVar, kw) || strings.Contains(lowerLine, kw) {
			return true
		}
	}
	return false
}
