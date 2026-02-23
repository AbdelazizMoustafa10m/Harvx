package security

// PatternRegistry stores the ordered set of RedactionRules used by the
// Redactor. Rules are registered once at program initialisation (via
// NewDefaultRegistry, which pre-loads all built-in patterns defined in
// patterns.go) and thereafter treated as read-only.
//
// Concurrent read access (Rules, RulesByConfidence) is safe without
// additional locking because writes only occur at init time before any
// goroutines start reading.
type PatternRegistry struct {
	rules []RedactionRule
}

// NewEmptyRegistry returns a PatternRegistry with no rules registered.
// Use this when you want full control over which rules are active (e.g.,
// in tests or when building a custom-only redactor without built-in patterns).
//
// Callers must register all desired rules via Register before handing the
// registry to a Redactor implementation.
func NewEmptyRegistry() *PatternRegistry {
	return &PatternRegistry{}
}

// NewDefaultRegistry returns a PatternRegistry pre-loaded with all built-in
// detection rules defined in patterns.go (T-035). Custom rules can be appended
// via Register after construction.
//
// The registry is safe for concurrent read access (Rules, RulesByConfidence)
// once returned; all writes happen before the function returns.
//
// Usage:
//
//	r := security.NewDefaultRegistry()
//	r.Register(myCustomRule) // optional: add project-specific rules
//	rules := r.Rules()
func NewDefaultRegistry() *PatternRegistry {
	r := &PatternRegistry{}
	registerBuiltinPatterns(r)
	return r
}

// Register appends rule to the registry. Callers are responsible for ensuring
// that rule.ID is unique across all registered rules; duplicate IDs are
// permitted by the registry itself but will produce duplicate matches in
// Redactor output.
//
// Register must only be called before any goroutine begins calling Rules or
// RulesByConfidence; it is not safe for concurrent use with readers.
func (r *PatternRegistry) Register(rule RedactionRule) {
	r.rules = append(r.rules, rule)
}

// Rules returns a copy of all registered rules in registration order.
//
// The returned slice is a shallow copy; mutating the slice does not affect
// the registry, but mutating a RedactionRule's fields (e.g. replacing the
// Regex) will affect future reads. Callers must not modify rule fields.
func (r *PatternRegistry) Rules() []RedactionRule {
	out := make([]RedactionRule, len(r.rules))
	copy(out, r.rules)
	return out
}

// RulesByConfidence returns a copy of all rules whose Confidence field equals
// c, in registration order. If no rules match c, an empty (non-nil) slice is
// returned.
func (r *PatternRegistry) RulesByConfidence(c Confidence) []RedactionRule {
	out := make([]RedactionRule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.Confidence == c {
			out = append(out, rule)
		}
	}
	return out
}
