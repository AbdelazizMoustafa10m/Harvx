package workflows

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// ExtractMakefileTargets extracts target names from a Makefile and returns
// them as a formatted list. Only user-defined targets (not starting with .)
// are included.
func ExtractMakefileTargets(content string) string {
	// Match lines that look like Makefile targets: "target-name:" at the start of a line.
	re := regexp.MustCompile(`(?m)^([a-zA-Z_][a-zA-Z0-9_\-]*)\s*:`)
	matches := re.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return ""
	}

	seen := make(map[string]bool, len(matches))
	var targets []string
	for _, m := range matches {
		target := m[1]
		if seen[target] {
			continue
		}
		seen[target] = true
		targets = append(targets, target)
	}

	var b strings.Builder
	b.WriteString("Available targets:\n")
	for _, t := range targets {
		b.WriteString("- `")
		b.WriteString(t)
		b.WriteString("`\n")
	}
	return b.String()
}

// ExtractPackageJSONScripts extracts the "scripts" section from a package.json
// file and returns it as a formatted list of script names and commands.
func ExtractPackageJSONScripts(content string) string {
	var pkg struct {
		Name    string            `json:"name"`
		Scripts map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return ""
	}

	if len(pkg.Scripts) == 0 {
		return ""
	}

	var b strings.Builder
	if pkg.Name != "" {
		b.WriteString("Package: ")
		b.WriteString(pkg.Name)
		b.WriteString("\n")
	}
	b.WriteString("Scripts:\n")

	// Sort script names for deterministic output.
	sorted := sortedKeys(pkg.Scripts)
	for _, name := range sorted {
		b.WriteString("- `")
		b.WriteString(name)
		b.WriteString("`: `")
		b.WriteString(pkg.Scripts[name])
		b.WriteString("`\n")
	}
	return b.String()
}

// ExtractGoModInfo extracts the module name and Go version from a go.mod file.
func ExtractGoModInfo(content string) string {
	var b strings.Builder

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			b.WriteString("Module: ")
			b.WriteString(strings.TrimPrefix(line, "module "))
			b.WriteString("\n")
		}
		if strings.HasPrefix(line, "go ") {
			b.WriteString("Go version: ")
			b.WriteString(strings.TrimPrefix(line, "go "))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// ExtractCargoTomlInfo extracts the [package] name and version from Cargo.toml.
func ExtractCargoTomlInfo(content string) string {
	var b strings.Builder
	inPackage := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[package]" {
			inPackage = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") && trimmed != "[package]" {
			inPackage = false
			continue
		}

		if !inPackage {
			continue
		}

		if strings.HasPrefix(trimmed, "name") {
			b.WriteString("Crate: ")
			b.WriteString(extractTomlValue(trimmed))
			b.WriteString("\n")
		}
		if strings.HasPrefix(trimmed, "version") {
			b.WriteString("Version: ")
			b.WriteString(extractTomlValue(trimmed))
			b.WriteString("\n")
		}
		if strings.HasPrefix(trimmed, "edition") {
			b.WriteString("Edition: ")
			b.WriteString(extractTomlValue(trimmed))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// ExtractPyprojectInfo extracts the [project] name and version from pyproject.toml.
func ExtractPyprojectInfo(content string) string {
	var b strings.Builder
	inProject := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[project]" {
			inProject = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") && trimmed != "[project]" {
			inProject = false
			continue
		}

		if !inProject {
			continue
		}

		if strings.HasPrefix(trimmed, "name") {
			b.WriteString("Project: ")
			b.WriteString(extractTomlValue(trimmed))
			b.WriteString("\n")
		}
		if strings.HasPrefix(trimmed, "version") {
			b.WriteString("Version: ")
			b.WriteString(extractTomlValue(trimmed))
			b.WriteString("\n")
		}
		if strings.HasPrefix(trimmed, "requires-python") {
			b.WriteString("Python: ")
			b.WriteString(extractTomlValue(trimmed))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// extractTomlValue extracts the value from a simple TOML key = "value" line.
func extractTomlValue(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	val := strings.TrimSpace(parts[1])
	// Remove surrounding quotes.
	val = strings.Trim(val, `"'`)
	return val
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
