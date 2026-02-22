package config

import "fmt"

// ValidationError describes a single configuration validation problem. It
// carries a severity level, the field path where the problem was detected,
// a human-readable message explaining what is wrong, and an optional
// suggestion describing how to fix the issue.
//
// ValidationError implements the error interface so individual results can be
// returned as standard Go errors when only one result is relevant.
type ValidationError struct {
	// Severity is either "error" (hard error, config is unusable) or
	// "warning" (soft issue, config may still be used with caution).
	Severity string

	// Field is the dotted path of the configuration field that caused the
	// issue, e.g. "profile.finvault.format" or "profile.work.relevance.tier_0[2]".
	Field string

	// Message describes what went wrong in plain English.
	Message string

	// Suggest is an optional actionable fix suggestion presented to the user.
	// It may be empty when no specific remediation is obvious.
	Suggest string
}

// Error implements the error interface. It returns a formatted string that
// includes the severity, field, message, and suggestion when present.
func (e ValidationError) Error() string {
	if e.Suggest != "" {
		return fmt.Sprintf("[%s] %s: %s (suggestion: %s)", e.Severity, e.Field, e.Message, e.Suggest)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Severity, e.Field, e.Message)
}

// LintResult extends ValidationError with a short machine-readable code that
// identifies the class of lint issue. Codes are stable across harvx releases
// and are suitable for programmatic filtering (e.g. --ignore-lint=complexity).
type LintResult struct {
	ValidationError

	// Code is a kebab-case identifier for the lint rule that fired.
	// Examples: "unreachable-tier", "no-ext-match", "complexity".
	Code string
}
