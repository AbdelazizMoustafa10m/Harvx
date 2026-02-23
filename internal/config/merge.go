package config

// mergeProfile creates a new Profile by applying override on top of base.
// The merge rules are:
//   - String scalars: use override if non-empty; otherwise keep base.
//   - Int scalars: use override if non-zero; otherwise keep base.
//   - Bool scalars: always use override (false is a valid override value).
//   - Slice fields (Ignore, PriorityFiles, Include): use override slice if
//     it is non-nil and non-empty; otherwise keep base slice.
//   - RelevanceConfig: each tier is replaced independently (non-nil, non-empty
//     child tier replaces the parent tier).
//   - RedactionConfig: merged field-by-field with the same scalar/slice rules.
//
// Neither base nor override is mutated. A fresh Profile is always returned.
// The Extends field is always cleared on the returned profile.
func mergeProfile(base, override *Profile) *Profile {
	result := &Profile{
		// Scalar: string
		Output:    mergeString(base.Output, override.Output),
		Format:    mergeString(base.Format, override.Format),
		Tokenizer: mergeString(base.Tokenizer, override.Tokenizer),
		Target:    mergeString(base.Target, override.Target),

		// Scalar: int
		MaxTokens: mergeInt(base.MaxTokens, override.MaxTokens),

		// Scalar: bool -- override always wins (false is meaningful)
		Compression: override.Compression,
		Redaction:   override.Redaction,

		// Slices: child replaces parent entirely when non-nil and non-empty
		Ignore:        mergeSlice(base.Ignore, override.Ignore),
		PriorityFiles: mergeSlice(base.PriorityFiles, override.PriorityFiles),
		Include:       mergeSlice(base.Include, override.Include),

		// Nested structs
		Relevance:       mergeRelevance(base.Relevance, override.Relevance),
		RedactionConfig: mergeRedactionConfig(base.RedactionConfig, override.RedactionConfig),

		// Extends is always cleared after merge (profile is fully resolved)
		Extends: nil,
	}
	return result
}

// mergeString returns override if non-empty, otherwise base.
func mergeString(base, override string) string {
	if override != "" {
		return override
	}
	return base
}

// mergeInt returns override if non-zero, otherwise base.
func mergeInt(base, override int) int {
	if override != 0 {
		return override
	}
	return base
}

// mergeSlice returns a copy of override if it is non-nil and non-empty,
// otherwise returns a copy of base. Copies are made at the boundary to
// prevent callers from sharing slice backing arrays (DC-1).
func mergeSlice(base, override []string) []string {
	if len(override) > 0 {
		result := make([]string, len(override))
		copy(result, override)
		return result
	}
	if len(base) > 0 {
		result := make([]string, len(base))
		copy(result, base)
		return result
	}
	return nil
}

// mergeRelevance merges two RelevanceConfig values. Each tier is independent:
// if the override tier is non-empty it fully replaces the base tier.
func mergeRelevance(base, override RelevanceConfig) RelevanceConfig {
	return RelevanceConfig{
		Tier0: mergeSlice(base.Tier0, override.Tier0),
		Tier1: mergeSlice(base.Tier1, override.Tier1),
		Tier2: mergeSlice(base.Tier2, override.Tier2),
		Tier3: mergeSlice(base.Tier3, override.Tier3),
		Tier4: mergeSlice(base.Tier4, override.Tier4),
		Tier5: mergeSlice(base.Tier5, override.Tier5),
	}
}

// mergeRedactionConfig merges two RedactionConfig values field-by-field.
// Enabled always uses override (false is a valid explicit value).
// ConfidenceThreshold uses override if non-empty.
// ExcludePaths uses override slice if non-nil and non-empty.
func mergeRedactionConfig(base, override RedactionConfig) RedactionConfig {
	return RedactionConfig{
		Enabled:             override.Enabled,
		ExcludePaths:        mergeSlice(base.ExcludePaths, override.ExcludePaths),
		ConfidenceThreshold: mergeString(base.ConfidenceThreshold, override.ConfidenceThreshold),
	}
}
