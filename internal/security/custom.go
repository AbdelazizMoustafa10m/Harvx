// Package security - this file provides helpers for compiling user-defined
// redaction patterns from profile configuration into RedactionRule values
// that can be registered in the PatternRegistry.
package security

import "fmt"

// CompileCustomPattern compiles a single user-defined pattern definition into
// a RedactionRule. It returns an error if the regex is invalid.
//
// The id, description, pattern, secretType, confidence, and keywords parameters
// correspond to the fields of a CustomPatternDefinition from the config package.
//
// This function is the integration point between the config layer (TOML parsing)
// and the security layer (rule execution). The pipeline calls this for each
// custom pattern after config resolution.
func CompileCustomPattern(id, description, pattern, secretType string, confidence Confidence, keywords []string) (RedactionRule, error) {
	if id == "" {
		return RedactionRule{}, fmt.Errorf("custom pattern has empty id")
	}
	if pattern == "" {
		return RedactionRule{}, fmt.Errorf("custom pattern %q has empty regex", id)
	}
	rule, err := NewRedactionRule(id, description, pattern, keywords, secretType, confidence, 0)
	if err != nil {
		return RedactionRule{}, fmt.Errorf("custom pattern %q: %w", id, err)
	}
	return rule, nil
}
