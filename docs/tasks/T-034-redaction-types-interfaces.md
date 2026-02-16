# T-034: Redaction Core Types, Interfaces, and Pattern Registry

**Priority:** Must Have
**Effort:** Medium (6-8hrs)
**Dependencies:** None (standalone foundation; assumes Go module and project structure from Phase 1 exist)
**Phase:** 3 - Security

---

## Description

Define the core data types, interfaces, and pattern registry that form the foundation of the secret detection and redaction subsystem. This includes the `Redactor` interface, `RedactionRule` struct, `RedactionResult` and `RedactionSummary` types, confidence levels, and the `PatternRegistry` that holds compiled regex patterns. This task produces no executable detection logic -- it establishes the contracts that all subsequent security tasks implement against.

## User Story

As a developer building the Harvx redaction pipeline, I want well-defined interfaces and data types so that the detection patterns, entropy analyzer, streaming filter, and report generator can all be developed and tested independently against stable contracts.

## Acceptance Criteria

- [ ] `Redactor` interface defined with `Redact(ctx context.Context, content string, filePath string) (string, []RedactionMatch, error)` method
- [ ] `RedactionRule` struct with: `ID`, `Description`, `Regex` (compiled `*regexp.Regexp`), `Keywords` (for pre-filtering), `SecretType` string, `Confidence` level, `Entropy` threshold (optional)
- [ ] `Confidence` type defined as `string` with constants: `ConfidenceHigh`, `ConfidenceMedium`, `ConfidenceLow`
- [ ] `RedactionMatch` struct with: `RuleID`, `SecretType`, `Confidence`, `FilePath`, `LineNumber`, `StartCol`, `EndCol`, `Replacement` string (e.g., `[REDACTED:aws_access_key]`)
- [ ] `RedactionSummary` struct with: total count, count by type (map), count by confidence, file count
- [ ] `RedactionConfig` struct matching profile TOML schema: `Enabled` bool, `ExcludePaths` []string, `ConfidenceThreshold` Confidence, `CustomPatterns` []CustomPatternConfig
- [ ] `PatternRegistry` struct that stores `[]RedactionRule` with methods: `Register(rule RedactionRule)`, `Rules() []RedactionRule`, `RulesByConfidence(c Confidence) []RedactionRule`
- [ ] `NewDefaultRegistry()` constructor that returns an empty registry (patterns registered in T-035)
- [ ] Replacement format helper: `FormatReplacement(secretType string) string` returns `[REDACTED:<type>]`
- [ ] All types have JSON struct tags for report serialization
- [ ] Unit tests for all type constructors, registry operations, and replacement formatting
- [ ] GoDoc comments on all exported types and methods

## Technical Notes

- Place all files under `internal/security/`
- Use Go's standard `regexp` package (RE2 engine) for pattern compilation. The RE2 engine does not support lookaheads but provides O(n) runtime guarantees which is critical for scanning untrusted input at scale. Gitleaks itself uses Go's standard regexp package.
- Do NOT use `regexp2` (dlclark/regexp2) -- it adds backtracking which removes the linear-time guarantee. All patterns must be expressible in RE2 syntax.
- The `Keywords` field on `RedactionRule` enables a two-phase detection approach: first check if any keyword exists in the line (fast string search), then apply the regex only on matching lines. This is the same optimization gitleaks uses.
- `PatternRegistry` should be safe for concurrent read access (patterns are registered at init time, then read-only during scanning). No mutex needed if registration happens before scanning begins.
- Reference: Gitleaks rule structure at https://github.com/gitleaks/gitleaks/blob/master/config/gitleaks.toml

## Files to Create/Modify

- `internal/security/types.go` - Core types: `Confidence`, `RedactionMatch`, `RedactionSummary`, `RedactionConfig`
- `internal/security/rule.go` - `RedactionRule` struct and `FormatReplacement` helper
- `internal/security/registry.go` - `PatternRegistry` struct and methods
- `internal/security/redactor.go` - `Redactor` interface definition
- `internal/security/types_test.go` - Tests for types, constructors, JSON serialization
- `internal/security/registry_test.go` - Tests for registry operations
- `internal/security/rule_test.go` - Tests for rule creation and replacement formatting

## Testing Requirements

- Unit tests for `FormatReplacement` with various secret types (aws_access_key, private_key_block, github_token, etc.)
- Unit tests for `PatternRegistry.Register` and `PatternRegistry.Rules`
- Unit tests for `PatternRegistry.RulesByConfidence` filtering
- Unit tests verifying JSON marshaling of `RedactionMatch` and `RedactionSummary`
- Unit tests for `RedactionConfig` zero-value defaults (Enabled=true by default)
- Verify that registering a rule with an invalid regex returns an error at construction time
- Table-driven tests for all confidence level constants
