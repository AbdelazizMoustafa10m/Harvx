# T-019: Profile Inheritance with Deep Merge

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-016 (Config Types & Defaults), T-017 (Multi-Source Config Merging)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Implement the `extends` field for profile inheritance. A profile can extend another profile using `extends = "default"` or `extends = "parent-profile"`, which deep-merges the parent's settings as the base with the child's settings overriding. This supports multi-level inheritance chains (up to 3 levels before warning) and handles circular dependency detection.

## User Story

As a developer managing multiple profiles for different workflows (review, session, full context), I want to extend a base profile and only override the settings that differ, so that I avoid duplicating configuration across profiles.

## Acceptance Criteria

- [ ] `internal/config/profile.go` implements `ResolveProfile(name string, profiles map[string]*Profile) (*Profile, error)`:
  - If profile has `extends` field, recursively resolve the parent first
  - Deep merge: child values override parent values
  - For map fields (relevance tiers): child replaces parent entirely (not concatenated)
  - For slice fields (ignore, priority_files, include): child replaces parent entirely
  - For scalar fields (output, format, max_tokens, etc.): child overrides parent
  - For nested struct fields (redaction): merge field-by-field (child fields override parent fields)
- [ ] The built-in `default` profile is always available as a base, even if not explicitly defined in config
- [ ] Circular inheritance is detected and returns a clear error:
  - e.g., "circular profile inheritance: a -> b -> a"
- [ ] Inheritance depth exceeding 3 levels emits a warning:
  - e.g., "profile 'deep-child' has 4 levels of inheritance; consider flattening"
- [ ] Missing parent profile returns a clear error:
  - e.g., "profile 'custom' extends 'nonexistent', but 'nonexistent' is not defined"
- [ ] Self-referential extends (`extends = "self"`) is detected as circular
- [ ] After resolution, the `extends` field is cleared (resolved profile is fully flattened)
- [ ] Unit tests cover all merge behaviors with 90%+ coverage
- [ ] The resolution result includes the inheritance chain for debugging (e.g., `["finvault", "default"]`)

## Technical Notes

- Deep merge implementation: write a custom `mergeProfile(base, override *Profile) *Profile` function
  - Use reflection-free approach: explicitly merge each field (safer and more readable)
  - For pointer fields: if override is non-nil, use it; else keep base
  - For slices: if override is non-nil and non-empty, replace entirely; else keep base
  - For maps: if override is non-nil and non-empty, replace entirely; else keep base
  - Do NOT concatenate arrays -- PRD explicitly states "arrays are replaced (not concatenated)"
- For nested structs like `RedactionConfig`: merge each sub-field individually
  - If `override.Redaction.Enabled` is set, use it; else use base
  - If `override.Redaction.ExcludePaths` is set, replace entirely
- Consider using `darccio/mergo` library for struct merging, but evaluate whether manual merging gives better control for the "arrays replace" semantics
- Track the inheritance chain as `[]string` for debugging/explain purposes
- Profile resolution should be called once during config resolution (T-017) and cached

## Files to Create/Modify

- `internal/config/profile.go` - Profile inheritance resolution and deep merge
- `internal/config/profile_test.go` - Comprehensive inheritance tests
- `internal/config/merge.go` - Deep merge utility functions
- `internal/config/merge_test.go` - Merge function tests
- `testdata/config/inheritance.toml` - Multi-profile config with extends chains
- `testdata/config/circular.toml` - Circular inheritance for error testing

## Testing Requirements

- Unit test: Profile with `extends = "default"` inherits all default values
- Unit test: Child overrides parent's `max_tokens` -- child value wins
- Unit test: Child overrides parent's `ignore` array -- entirely replaced, not concatenated
- Unit test: Child overrides parent's `relevance` tiers -- entirely replaced
- Unit test: Child adds `priority_files` not in parent -- new value used
- Unit test: Nested redaction merge: child sets `enabled = false`, parent's `exclude_paths` preserved
- Unit test: Nested redaction merge: child sets `exclude_paths`, parent's `enabled` preserved
- Unit test: Three-level inheritance: grandchild -> child -> default
- Unit test: Circular inheritance detected and returns error
- Unit test: Self-referential extends returns error
- Unit test: Missing parent profile returns error with helpful message
- Unit test: Depth > 3 emits warning (logged, not error)
- Unit test: Built-in `default` always available even if not in config file
- Unit test: Inheritance chain tracking is correct (e.g., `["finvault", "default"]`)
- Golden test: PRD example config (finvault extends default) resolves correctly

## References

- [darccio/mergo - Go struct merging](https://github.com/darccio/mergo)
- PRD Section 5.2 - "Profile inheritance uses deep merge: child profile values override parent, arrays are replaced (not concatenated)"
- PRD Section 5.2 - "Warn if profile inheritance exceeds 3 levels deep"
- PRD Section 5.2 - Example: `[profile.finvault]` with `extends = "default"`
