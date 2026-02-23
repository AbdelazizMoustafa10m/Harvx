package config

import "fmt"

// ApplyTargetPreset modifies p in-place based on the target name.
// Valid targets: "claude", "chatgpt", "generic".
// An empty target string is a no-op and returns nil.
// Unknown target names return an error.
func ApplyTargetPreset(p *Profile, target string) error {
	if target == "" {
		return nil
	}

	switch target {
	case "claude":
		p.Format = "xml"
		p.MaxTokens = 200000
	case "chatgpt":
		p.Format = "markdown"
		p.MaxTokens = 128000
	case "generic":
		p.Format = "markdown"
	default:
		return fmt.Errorf("unknown target %q (allowed: claude, chatgpt, generic)", target)
	}

	return nil
}
