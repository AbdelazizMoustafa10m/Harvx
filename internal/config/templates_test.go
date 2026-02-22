package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListTemplates_Count verifies that ListTemplates returns exactly 6 templates.
func TestListTemplates_Count(t *testing.T) {
	t.Parallel()

	templates := ListTemplates()
	assert.Len(t, templates, 6, "ListTemplates must return exactly 6 templates (base + 5 frameworks)")
}

// TestListTemplates_Names verifies that the returned templates include all expected names.
func TestListTemplates_Names(t *testing.T) {
	t.Parallel()

	templates := ListTemplates()

	names := make([]string, 0, len(templates))
	for _, tmpl := range templates {
		names = append(names, tmpl.Name)
	}

	expectedNames := []string{"base", "nextjs", "go-cli", "python-django", "rust-cargo", "monorepo"}
	for _, want := range expectedNames {
		assert.Contains(t, names, want, "ListTemplates must include template %q", want)
	}
}

// TestListTemplates_Descriptions verifies that every template has a non-empty description.
func TestListTemplates_Descriptions(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		assert.NotEmpty(t, tmpl.Description,
			"template %q must have a non-empty description", tmpl.Name)
	}
}

// TestListTemplates_ReturnsIndependentCopy verifies that mutating the returned
// slice does not affect subsequent calls (copy semantics).
func TestListTemplates_ReturnsIndependentCopy(t *testing.T) {
	t.Parallel()

	first := ListTemplates()
	first[0].Name = "mutated"

	second := ListTemplates()
	assert.NotEqual(t, "mutated", second[0].Name,
		"ListTemplates must return an independent copy")
}

// TestGetTemplate_KnownTemplates verifies that GetTemplate returns non-empty
// TOML content for every known template name.
func TestGetTemplate_KnownTemplates(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err, "GetTemplate(%q) must not return an error", tmpl.Name)
			assert.NotEmpty(t, content, "GetTemplate(%q) must return non-empty content", tmpl.Name)
		})
	}
}

// TestGetTemplate_ContainsTOMLSection verifies that each template contains
// a [profile.default] TOML section.
func TestGetTemplate_ContainsTOMLSection(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err)
			assert.Contains(t, content, "[profile.default]",
				"template %q must contain a [profile.default] TOML section", tmpl.Name)
		})
	}
}

// TestGetTemplate_ContainsProjectNamePlaceholder verifies that each template
// contains the {{project_name}} placeholder in the output field.
func TestGetTemplate_ContainsProjectNamePlaceholder(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err)
			assert.Contains(t, content, "{{project_name}}",
				"template %q must contain the {{project_name}} placeholder", tmpl.Name)
		})
	}
}

// TestGetTemplate_ContainsComments verifies that each template file contains
// at least one comment line (lines starting with '#') for user education.
func TestGetTemplate_ContainsComments(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err)

			hasComment := false
			for _, line := range strings.Split(content, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "#") {
					hasComment = true
					break
				}
			}
			assert.True(t, hasComment,
				"template %q must contain at least one comment line", tmpl.Name)
		})
	}
}

// TestGetTemplate_UnknownName verifies that GetTemplate returns an error for
// an unrecognised template name.
func TestGetTemplate_UnknownName(t *testing.T) {
	t.Parallel()

	_, err := GetTemplate("nonexistent")
	require.Error(t, err, "GetTemplate with unknown name must return an error")
	assert.Contains(t, err.Error(), "nonexistent",
		"error message must mention the unknown template name")
}

// TestGetTemplate_PathTraversalPrevented verifies that path traversal attempts
// are rejected.
func TestGetTemplate_PathTraversalPrevented(t *testing.T) {
	t.Parallel()

	traversalNames := []string{
		"../secrets",
		"../../etc/passwd",
		"../config/loader",
	}

	for _, name := range traversalNames {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := GetTemplate(name)
			require.Error(t, err,
				"GetTemplate(%q) must return an error (path traversal prevention)", name)
		})
	}
}

// TestRenderTemplate_ReplacesProjectName verifies that RenderTemplate substitutes
// {{project_name}} with the provided project name.
func TestRenderTemplate_ReplacesProjectName(t *testing.T) {
	t.Parallel()

	content, err := RenderTemplate("nextjs", "myproject")
	require.NoError(t, err)

	assert.Contains(t, content, "myproject",
		"rendered template must contain the project name")
	assert.NotContains(t, content, "{{project_name}}",
		"rendered template must not contain the placeholder after rendering")
}

// TestRenderTemplate_OutputFieldRendered verifies that the output field in
// the rendered template contains the project name.
func TestRenderTemplate_OutputFieldRendered(t *testing.T) {
	t.Parallel()

	content, err := RenderTemplate("go-cli", "mycli")
	require.NoError(t, err)

	assert.Contains(t, content, "mycli-context.md",
		"rendered output field must contain the project name")
}

// TestRenderTemplate_AllTemplates verifies that RenderTemplate works for all
// known template names.
func TestRenderTemplate_AllTemplates(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := RenderTemplate(tmpl.Name, "testproject")
			require.NoError(t, err, "RenderTemplate(%q, ...) must not return an error", tmpl.Name)
			assert.NotEmpty(t, content)
			assert.Contains(t, content, "testproject",
				"rendered template %q must contain the project name", tmpl.Name)
			assert.NotContains(t, content, "{{project_name}}",
				"rendered template %q must not contain the placeholder", tmpl.Name)
		})
	}
}

// TestRenderTemplate_UnknownName verifies that RenderTemplate propagates
// errors from GetTemplate for unrecognised names.
func TestRenderTemplate_UnknownName(t *testing.T) {
	t.Parallel()

	_, err := RenderTemplate("unknown", "myproject")
	require.Error(t, err)
}

// TestTemplates_ValidTOML verifies that each template can be decoded into
// a valid Config struct without TOML parse errors.
func TestTemplates_ValidTOML(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err)

			// Use a rendered version to avoid any placeholder interference.
			rendered := strings.ReplaceAll(content, "{{project_name}}", "testproject")

			cfg, err := LoadFromString(rendered, tmpl.Name+".toml")
			require.NoError(t, err,
				"template %q must decode into a valid Config struct", tmpl.Name)
			require.NotNil(t, cfg)
		})
	}
}

// TestTemplates_PassValidation verifies that each template produces zero
// validation errors and zero warnings when run through Validate().
func TestTemplates_PassValidation(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err)

			rendered := strings.ReplaceAll(content, "{{project_name}}", "testproject")

			cfg, err := LoadFromString(rendered, tmpl.Name+".toml")
			require.NoError(t, err)
			require.NotNil(t, cfg)

			results := Validate(cfg)

			var hardErrors []ValidationError
			var warnings []ValidationError
			for _, r := range results {
				switch r.Severity {
				case "error":
					hardErrors = append(hardErrors, r)
				case "warning":
					warnings = append(warnings, r)
				}
			}

			assert.Empty(t, hardErrors,
				"template %q must produce zero hard validation errors; got: %v",
				tmpl.Name, hardErrors)
			assert.Empty(t, warnings,
				"template %q must produce zero validation warnings; got: %v",
				tmpl.Name, warnings)
		})
	}
}

// TestTemplates_DefaultProfilePresent verifies that each template config
// contains a "default" profile entry.
func TestTemplates_DefaultProfilePresent(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err)

			rendered := strings.ReplaceAll(content, "{{project_name}}", "testproject")

			cfg, err := LoadFromString(rendered, tmpl.Name+".toml")
			require.NoError(t, err)
			require.NotNil(t, cfg)

			assert.Contains(t, cfg.Profile, "default",
				"template %q must contain a 'default' profile", tmpl.Name)
		})
	}
}

// TestTemplates_FormatSet verifies that each template sets a non-empty format.
func TestTemplates_FormatSet(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err)

			rendered := strings.ReplaceAll(content, "{{project_name}}", "testproject")

			cfg, err := LoadFromString(rendered, tmpl.Name+".toml")
			require.NoError(t, err)

			profile, ok := cfg.Profile["default"]
			require.True(t, ok)
			assert.NotEmpty(t, profile.Format,
				"template %q must set a non-empty format", tmpl.Name)
		})
	}
}

// TestTemplates_MaxTokensSet verifies that each template sets a positive max_tokens.
func TestTemplates_MaxTokensSet(t *testing.T) {
	t.Parallel()

	for _, tmpl := range ListTemplates() {
		tmpl := tmpl
		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			content, err := GetTemplate(tmpl.Name)
			require.NoError(t, err)

			rendered := strings.ReplaceAll(content, "{{project_name}}", "testproject")

			cfg, err := LoadFromString(rendered, tmpl.Name+".toml")
			require.NoError(t, err)

			profile, ok := cfg.Profile["default"]
			require.True(t, ok)
			assert.Greater(t, profile.MaxTokens, 0,
				"template %q must set a positive max_tokens", tmpl.Name)
		})
	}
}

// TestTemplates_EmbedFSAccessible verifies that all template files are
// accessible via the embedded filesystem.
func TestTemplates_EmbedFSAccessible(t *testing.T) {
	t.Parallel()

	expectedFiles := []string{
		"templates/base.toml",
		"templates/nextjs.toml",
		"templates/go-cli.toml",
		"templates/python-django.toml",
		"templates/rust-cargo.toml",
		"templates/monorepo.toml",
	}

	for _, path := range expectedFiles {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			data, err := templateFS.ReadFile(path)
			require.NoError(t, err,
				"embedded file %q must be accessible via embed.FS", path)
			assert.NotEmpty(t, data,
				"embedded file %q must not be empty", path)
		})
	}
}
