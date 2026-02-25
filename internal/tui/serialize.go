package tui

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"github.com/BurntSushi/toml"
)

// serializeSelectionToTOML creates a TOML-encoded profile section containing
// the given include patterns. The profile name becomes the key under the
// [profile] table.
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
