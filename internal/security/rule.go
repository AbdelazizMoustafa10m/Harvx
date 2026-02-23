package security

import (
	"fmt"
	"regexp"
)

// RedactionRule defines a single detection rule used by the Redactor. Rules
// are registered in a PatternRegistry at initialisation time (T-035) and are
// treated as immutable during the redaction pass.
//
// A rule matches a file line when:
//  1. The Regex matches a substring of that line, AND
//  2. At least one keyword in Keywords appears in the line (case-insensitive),
//     OR Keywords is empty (the regex is authoritative on its own).
//
// If EntropyThreshold is non-zero, the matched substring is also tested with
// a Shannon-entropy calculation; values below the threshold are discarded.
type RedactionRule struct {
	// ID is the unique identifier for the rule (e.g. "aws-access-key").
	// It appears in RedactionMatch.RuleID in output reports.
	ID string

	// Description is a short human-readable label shown in CLI reports and
	// verbose output (e.g. "AWS Access Key ID (AKIA…)").
	Description string

	// Regex is the compiled RE2 regular expression used to locate candidate
	// secrets. At least one capture group is required; the first capture
	// group is treated as the secret value to be redacted.
	//
	// Must not be nil; rules with a nil Regex are silently skipped by the
	// registry scan loop.
	Regex *regexp.Regexp

	// Keywords is an optional list of strings that must appear in the same
	// line as the Regex match (case-insensitive substring check). An empty
	// slice means the Regex alone determines a match.
	//
	// Keywords are used to reduce false positives for generic patterns such
	// as high-entropy strings.
	Keywords []string

	// SecretType is the category label used when building the replacement
	// string (e.g. "aws_access_key"). Use FormatReplacement to produce the
	// canonical replacement value.
	SecretType string

	// Confidence is the certainty level assigned to all matches produced by
	// this rule.
	Confidence Confidence

	// EntropyThreshold is the minimum Shannon entropy (in bits per character)
	// that the matched value must reach for the match to be accepted. A value
	// of 0 disables the entropy check entirely.
	//
	// Typical values: 3.5 (moderate), 4.5 (high). The practical maximum for
	// printable ASCII is approximately 6.5.
	EntropyThreshold float64
}

// NewRedactionRule creates a RedactionRule by compiling the given RE2 pattern.
// Returns an error if pattern is not a valid RE2 regular expression.
// This is the preferred way to create a RedactionRule when you have a pattern
// string; it ensures the regex is valid at construction time rather than
// panicking later with regexp.MustCompile.
func NewRedactionRule(id, description, pattern string, keywords []string, secretType string, confidence Confidence, entropyThreshold float64) (RedactionRule, error) {
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return RedactionRule{}, fmt.Errorf("compiling pattern for rule %q: %w", id, err)
	}
	return RedactionRule{
		ID:               id,
		Description:      description,
		Regex:            rx,
		Keywords:         keywords,
		SecretType:       secretType,
		Confidence:       confidence,
		EntropyThreshold: entropyThreshold,
	}, nil
}

// FormatReplacement returns the canonical redaction placeholder string for
// the given secret type. The format is "[REDACTED:<type>]".
//
// Examples:
//
//	FormatReplacement("aws_access_key")  → "[REDACTED:aws_access_key]"
//	FormatReplacement("github_token")    → "[REDACTED:github_token]"
//	FormatReplacement("")                → "[REDACTED:]"
func FormatReplacement(secretType string) string {
	return fmt.Sprintf("[REDACTED:%s]", secretType)
}
