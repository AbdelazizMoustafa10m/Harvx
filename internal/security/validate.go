package security

import "strings"

// ValidateJWT returns true when match is a structurally valid JSON Web Token.
//
// A valid JWT consists of exactly three base64url-encoded segments separated
// by dots (header.payload.signature). Each segment must be non-empty. The
// function does NOT decode or verify the segments; it only validates that the
// surface structure matches the JWT specification.
//
// This validator is intended for use by the T-036 redactor to reduce false
// positives from the "jwt-token" rule.  The rule regex guarantees the eyJ
// prefix and minimum segment lengths; this function adds the dot-count check.
//
// Examples:
//
//	ValidateJWT("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.abc123def456")  // true
//	ValidateJWT("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0")               // false -- only 2 segments
//	ValidateJWT("eyJhbGciOiJIUzI1NiJ9..abc123def456")                       // false -- empty segment
func ValidateJWT(match string) bool {
	parts := strings.Split(match, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
	}
	return true
}

// ValidateAWSKeyID returns true when match is a structurally valid AWS Access
// Key ID.
//
// A valid AWS Access Key ID:
//   - Starts with one of the five known prefixes: AKIA, ASIA, ABIA, ACCA, A3T
//   - Is exactly 20 characters in total (4-char prefix + 16 uppercase
//     alphanumeric chars, or 3-char prefix A3T + 17 uppercase alphanumeric)
//
// This validator is intended for use by the T-036 redactor alongside the
// "aws-access-key-id" rule to reject accidental matches produced by the
// broad [A-Z0-9]{16} suffix.
//
// Examples:
//
//	ValidateAWSKeyID("AKIAIOSFODNN7EXAMPLE")  // true
//	ValidateAWSKeyID("ASIA1234567890ABCDEF")  // true
//	ValidateAWSKeyID("AKIAIOSFODNN7EXAM")     // false -- too short
//	ValidateAWSKeyID("ABCD1234567890ABCDEF")  // false -- unknown prefix
func ValidateAWSKeyID(match string) bool {
	if len(match) != 20 {
		return false
	}
	// 4-character prefixes: AKIA, ASIA, ABIA, ACCA
	prefix4 := match[:4]
	switch prefix4 {
	case "AKIA", "ASIA", "ABIA", "ACCA":
		return true
	}
	// 3-character prefix: A3T
	if match[:3] == "A3T" {
		return true
	}
	return false
}
