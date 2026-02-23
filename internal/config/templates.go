package config

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed templates/*.toml
var templateFS embed.FS

// TemplateInfo describes a single profile template.
type TemplateInfo struct {
	// Name is the short identifier used with --template (e.g. "nextjs", "go-cli").
	Name string

	// Description is a one-line human-readable description of the template.
	Description string
}

// templates is the registry of available profile templates in display order.
var templates = []TemplateInfo{
	{Name: "base", Description: "Minimal starter configuration for any project"},
	{Name: "nextjs", Description: "Next.js / React application"},
	{Name: "go-cli", Description: "Go CLI application"},
	{Name: "python-django", Description: "Python Django web application"},
	{Name: "rust-cargo", Description: "Rust Cargo project"},
	{Name: "monorepo", Description: "Monorepo with multiple packages"},
}

// ListTemplates returns metadata for all available profile templates.
// The returned slice is in display order (base first, then alphabetical by framework).
func ListTemplates() []TemplateInfo {
	result := make([]TemplateInfo, len(templates))
	copy(result, templates)
	return result
}

// GetTemplate returns the raw TOML content for the named template.
// The name must match one of the values returned by ListTemplates (e.g. "nextjs").
// It returns an error if the template name is not recognised.
func GetTemplate(name string) (string, error) {
	// Validate name against known templates to prevent path traversal.
	found := false
	for _, t := range templates {
		if t.Name == name {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("unknown template %q: use ListTemplates() to see available templates", name)
	}

	data, err := templateFS.ReadFile("templates/" + name + ".toml")
	if err != nil {
		return "", fmt.Errorf("read template %q: %w", name, err)
	}

	return string(data), nil
}

// RenderTemplate returns the TOML content for the named template with all
// occurrences of the placeholder {{project_name}} replaced by projectName.
// It returns an error if the template name is not recognised.
func RenderTemplate(name, projectName string) (string, error) {
	content, err := GetTemplate(name)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(content, "{{project_name}}", projectName), nil
}
