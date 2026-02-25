package tui

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"github.com/BurntSushi/toml"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/tui/filetree"
)

// SerializeToProfile converts the current TUI file selection state into a valid
// TOML profile section. It uses pattern minimization to produce compact globs
// and preserves relevant settings from the active profile.
func SerializeToProfile(name string, root *filetree.Node, baseCfg *config.ResolvedConfig) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("profile name must not be empty")
	}
	if root == nil {
		return nil, fmt.Errorf("file tree root must not be nil")
	}

	// Minimize patterns from tree state.
	patterns := MinimizePatterns(root)

	// Build the profile data structure.
	profileData := buildProfileData(name, patterns, baseCfg)

	// Encode to TOML.
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	if err := enc.Encode(profileData); err != nil {
		return nil, fmt.Errorf("encoding profile TOML: %w", err)
	}

	return buf.Bytes(), nil
}

// buildProfileData constructs the nested map structure for TOML encoding.
// The output has the form:
//
//	[profile.<name>]
//	extends = "default"
//	format = "markdown"
//	...
func buildProfileData(name string, patterns MinimizedPatterns, baseCfg *config.ResolvedConfig) map[string]any {
	prof := make(map[string]any)

	// Always extend from default for inheritance.
	prof["extends"] = "default"

	// Preserve settings from the active profile.
	if baseCfg != nil && baseCfg.Profile != nil {
		p := baseCfg.Profile
		if p.Format != "" {
			prof["format"] = p.Format
		}
		if p.MaxTokens > 0 {
			prof["max_tokens"] = p.MaxTokens
		}
		if p.Tokenizer != "" {
			prof["tokenizer"] = p.Tokenizer
		}
		if p.Target != "" {
			prof["target"] = p.Target
		}
		if p.Compression {
			prof["compression"] = true
		}
		if p.Redaction {
			prof["redaction"] = true
		}
	}

	// Priority files from tier 0.
	if len(patterns.PriorityFiles) > 0 {
		prof["priority_files"] = patterns.PriorityFiles
	}

	// Include patterns.
	if len(patterns.Include) > 0 {
		prof["include"] = patterns.Include
	}

	// Ignore patterns.
	if len(patterns.Ignore) > 0 {
		prof["ignore"] = patterns.Ignore
	}

	// Build relevance tiers from collected tier info.
	relevance := buildRelevanceTiers(patterns.TierFiles)
	if len(relevance) > 0 {
		prof["relevance"] = relevance
	}

	return map[string]any{
		"profile": map[string]any{
			name: prof,
		},
	}
}

// buildRelevanceTiers converts the tier-to-files mapping into the TOML
// relevance section format with tier_0 through tier_5 arrays.
func buildRelevanceTiers(tierFiles map[int][]string) map[string]any {
	if len(tierFiles) == 0 {
		return nil
	}

	relevance := make(map[string]any)
	tierKeys := []string{"tier_0", "tier_1", "tier_2", "tier_3", "tier_4", "tier_5"}

	for tier := 0; tier <= 5; tier++ {
		files, ok := tierFiles[tier]
		if !ok || len(files) == 0 {
			continue
		}
		relevance[tierKeys[tier]] = files
	}

	if len(relevance) == 0 {
		return nil
	}
	return relevance
}

// SaveProfileToFile serializes the TUI selection state and writes it to the
// given TOML config file. If the file exists, the profile section is appended.
// If it doesn't exist, a new file is created.
func SaveProfileToFile(path, name string, root *filetree.Node, baseCfg *config.ResolvedConfig) error {
	data, err := SerializeToProfile(name, root, baseCfg)
	if err != nil {
		return fmt.Errorf("serializing profile: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening config file %s: %w", path, err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n%s", data); err != nil {
		return fmt.Errorf("writing profile to %s: %w", path, err)
	}

	return nil
}

// serializeSelectionToTOML creates a TOML-encoded profile section containing
// the given include patterns. The profile name becomes the key under the
// [profile] table. This is the legacy serialization used by the basic save flow.
func serializeSelectionToTOML(profileName string, includePaths []string) (string, error) {
	if profileName == "" {
		return "", fmt.Errorf("profile name must not be empty")
	}

	// Sort paths for deterministic output.
	sorted := make([]string, len(includePaths))
	copy(sorted, includePaths)
	sort.Strings(sorted)

	// Build the nested TOML structure.
	data := map[string]any{
		"profile": map[string]any{
			profileName: map[string]any{
				"include": sorted,
			},
		},
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	if err := enc.Encode(data); err != nil {
		return "", fmt.Errorf("encoding profile TOML: %w", err)
	}

	return buf.String(), nil
}

// appendProfileToFile appends a serialized profile section to the given TOML
// config file. If the file does not exist, it is created.
func appendProfileToFile(path, profileName string, includePaths []string) error {
	section, err := serializeSelectionToTOML(profileName, includePaths)
	if err != nil {
		return fmt.Errorf("serializing profile: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening config file %s: %w", path, err)
	}
	defer f.Close()

	// Add a newline separator before the new section.
	if _, err := fmt.Fprintf(f, "\n%s", section); err != nil {
		return fmt.Errorf("writing profile to %s: %w", path, err)
	}

	return nil
}
