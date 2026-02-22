package config

import (
	"fmt"
	"log/slog"
	"strings"
)

// maxInheritanceDepth is the maximum chain length before a warning is emitted.
// Chains deeper than this are still resolved; only a warning is logged.
const maxInheritanceDepth = 3

// ProfileResolution is the result of resolving a profile with full inheritance
// chain flattened into a single Profile value.
type ProfileResolution struct {
	// Profile is the fully merged profile with all parent values applied.
	// The Extends field is always nil (cleared after resolution).
	Profile *Profile

	// Chain is the ordered list of profile names in the inheritance chain,
	// from the requested profile to the ultimate ancestor. For example,
	// ["finvault", "base", "default"]. Useful for debugging.
	Chain []string
}

// ResolveProfile resolves the named profile by following its inheritance chain
// and deep-merging parent values beneath child values.
//
// The built-in "default" profile is always available as a base even if it is
// not explicitly defined in the profiles map. It is synthesized from
// DefaultProfile() when needed.
//
// Merge rules (child always wins):
//   - String scalars: child overrides if non-empty.
//   - Int scalars: child overrides if non-zero.
//   - Bool scalars: child always overrides (false is a valid explicit value).
//   - Slices (Ignore, PriorityFiles, Include, tier globs): child replaces
//     parent entirely when non-nil and non-empty.
//   - RedactionConfig: merged field-by-field with the same rules.
//
// Error conditions:
//   - Profile not found (and is not "default"): returns descriptive error.
//   - Circular inheritance detected: returns the full cycle path in the error.
//   - Self-referential extends: detected as circular.
//
// The returned ProfileResolution.Profile always has Extends == nil.
func ResolveProfile(name string, profiles map[string]*Profile) (*ProfileResolution, error) {
	resolution, err := resolveChain(name, profiles, nil)
	if err != nil {
		return nil, err
	}

	depth := len(resolution.Chain)
	if depth > maxInheritanceDepth {
		slog.Warn("deep profile inheritance; consider flattening",
			"profile", name,
			"depth", depth,
			"chain", strings.Join(resolution.Chain, " -> "),
		)
	}

	slog.Debug("profile resolved",
		"profile", name,
		"chain", strings.Join(resolution.Chain, " -> "),
	)

	return resolution, nil
}

// resolveChain is the recursive helper that builds the inheritance chain and
// merges profiles from ancestor to descendant. visited tracks the names
// already seen in the current call path for circular dependency detection.
func resolveChain(name string, profiles map[string]*Profile, visited []string) (*ProfileResolution, error) {
	// Detect circular inheritance before doing any work.
	for _, v := range visited {
		if v == name {
			cycle := append(visited, name)
			return nil, fmt.Errorf("circular profile inheritance: %s", strings.Join(cycle, " -> "))
		}
	}

	visited = append(visited, name)

	// Look up the profile. The built-in "default" profile is synthesized from
	// DefaultProfile() if it is not explicitly defined in the map.
	profile := lookupProfile(name, profiles)
	if profile == nil {
		return nil, fmt.Errorf("profile %q is not defined", name)
	}

	// Base case: no parent profile. Start the chain with the current profile.
	if profile.Extends == nil || *profile.Extends == "" {
		// If this is not the built-in "default" and has no extends, merge it
		// on top of the resolved "default" profile so all unset fields get
		// sensible values. We start a fresh visited set (nil) for this
		// implicit resolution to avoid false circular detection when "default"
		// happens to appear elsewhere in the ancestor chain.
		if name != "default" {
			defaultResolution, err := resolveChain("default", profiles, nil)
			if err != nil {
				return nil, fmt.Errorf("resolving default base for %q: %w", name, err)
			}
			merged := mergeProfile(defaultResolution.Profile, profile)
			chain := append([]string{name}, defaultResolution.Chain...)
			return &ProfileResolution{Profile: merged, Chain: chain}, nil
		}

		// This is the default profile with no parent: return a copy merged on
		// top of the built-in DefaultProfile() to fill any gaps.
		builtin := DefaultProfile()
		merged := mergeProfile(builtin, profile)
		return &ProfileResolution{Profile: merged, Chain: []string{name}}, nil
	}

	parentName := *profile.Extends

	// Resolve the parent first (depth-first recursion).
	parentResolution, err := resolveChain(parentName, profiles, visited)
	if err != nil {
		return nil, fmt.Errorf("resolving parent %q for profile %q: %w", parentName, name, err)
	}

	// Merge: apply child on top of resolved parent.
	merged := mergeProfile(parentResolution.Profile, profile)

	// Chain: current profile first, then parent chain.
	chain := append([]string{name}, parentResolution.Chain...)

	return &ProfileResolution{Profile: merged, Chain: chain}, nil
}

// lookupProfile returns the named profile from the map, or the synthesized
// DefaultProfile() for the special name "default" when it is absent from the
// map, or nil when the profile is truly unknown.
func lookupProfile(name string, profiles map[string]*Profile) *Profile {
	if p, ok := profiles[name]; ok {
		return p
	}
	if name == "default" {
		return DefaultProfile()
	}
	return nil
}
