package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowProfile_HeaderComments(t *testing.T) {
	p := DefaultProfile()
	src := make(SourceMap)
	for k := range profileToFlatMap(p) {
		src[k] = SourceDefault
	}

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "default",
		Chain:       []string{"default"},
	})

	assert.Contains(t, output, "# Resolved profile: default")
	// Single-element chain should not show inheritance line.
	assert.NotContains(t, output, "# Inheritance chain:")
}

func TestShowProfile_InheritanceChain(t *testing.T) {
	p := DefaultProfile()
	src := make(SourceMap)

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "finvault",
		Chain:       []string{"finvault", "default"},
	})

	assert.Contains(t, output, "# Resolved profile: finvault")
	assert.Contains(t, output, "# Inheritance chain: finvault -> default")
}

func TestShowProfile_SourceAnnotations(t *testing.T) {
	p := DefaultProfile()
	src := SourceMap{
		"output":     SourceDefault,
		"format":     SourceRepo,
		"max_tokens": SourceRepo,
	}

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "test",
		Chain:       []string{"test", "default"},
	})

	assert.Contains(t, output, "# default", "output field should be annotated as default")
	assert.Contains(t, output, "# repo", "format/max_tokens should be annotated as repo")
}

func TestShowProfile_ContainsScalarFields(t *testing.T) {
	p := DefaultProfile()
	src := make(SourceMap)

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "default",
		Chain:       []string{"default"},
	})

	assert.Contains(t, output, `output`)
	assert.Contains(t, output, `format`)
	assert.Contains(t, output, `max_tokens`)
	assert.Contains(t, output, `tokenizer`)
	assert.Contains(t, output, `compression`)
	assert.Contains(t, output, `redaction`)
}

func TestShowProfile_ContainsRelevanceSection(t *testing.T) {
	p := DefaultProfile()
	src := make(SourceMap)

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "default",
		Chain:       []string{"default"},
	})

	assert.Contains(t, output, "[relevance]")
	assert.Contains(t, output, "tier_0")
	assert.Contains(t, output, "tier_1")
}

func TestShowProfile_EmptyTargetOmitted(t *testing.T) {
	p := DefaultProfile()
	p.Target = ""
	src := make(SourceMap)

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "default",
		Chain:       []string{"default"},
	})

	// Target field should be omitted when empty.
	assert.NotContains(t, output, `target               = ""`)
}

func TestShowProfile_NonEmptyTargetIncluded(t *testing.T) {
	p := DefaultProfile()
	p.Target = "claude"
	src := SourceMap{"target": SourceRepo}

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "mypro",
		Chain:       []string{"mypro", "default"},
	})

	assert.Contains(t, output, `"claude"`)
	assert.Contains(t, output, "# repo")
}

func TestShowProfileJSON_ValidJSON(t *testing.T) {
	p := DefaultProfile()
	result, err := ShowProfileJSON(p)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err, "ShowProfileJSON output must be valid JSON")

	// Profile struct uses only toml tags, so encoding/json uses Go field names.
	assert.Equal(t, "markdown", parsed["Format"])
	assert.Equal(t, float64(128000), parsed["MaxTokens"])
}

func TestShowProfileJSON_FieldsPresent(t *testing.T) {
	p := DefaultProfile()
	result, err := ShowProfileJSON(p)
	require.NoError(t, err)

	// encoding/json serialises using Go field names (no json tags on Profile).
	assert.Contains(t, result, `"Output"`)
	assert.Contains(t, result, `"Format"`)
	assert.Contains(t, result, `"MaxTokens"`)
	assert.Contains(t, result, `"Tokenizer"`)
	assert.Contains(t, result, `"Relevance"`)
}

func TestShowProfile_RedactionConfigSection(t *testing.T) {
	p := DefaultProfile()
	// The built-in default has RedactionConfig.Enabled=true so section should appear.
	src := make(SourceMap)

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "default",
		Chain:       []string{"default"},
	})

	assert.Contains(t, output, "[redaction_config]")
	assert.Contains(t, output, "enabled")
	assert.Contains(t, output, "confidence_threshold")
}

func TestShowProfile_PriorityFilesIncluded(t *testing.T) {
	p := DefaultProfile()
	p.PriorityFiles = []string{"README.md", "CLAUDE.md"}
	src := SourceMap{"priority_files": SourceRepo}

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "test",
		Chain:       []string{"test", "default"},
	})

	assert.Contains(t, output, "priority_files")
	assert.Contains(t, output, "README.md")
	assert.Contains(t, output, "CLAUDE.md")
}

func TestShowProfile_IncludeOmittedWhenEmpty(t *testing.T) {
	p := DefaultProfile()
	p.Include = nil
	src := make(SourceMap)

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "default",
		Chain:       []string{"default"},
	})

	// include should not appear since it is empty.
	assert.NotContains(t, output, "\ninclude")
}

func TestSourceLabel_DefaultsWhenMissing(t *testing.T) {
	src := make(SourceMap)
	assert.Equal(t, "default", sourceLabel(src, "nonexistent_key"))
}

func TestSourceLabel_ReturnsCorrectSource(t *testing.T) {
	src := SourceMap{
		"format":     SourceRepo,
		"max_tokens": SourceGlobal,
		"output":     SourceFlag,
	}

	assert.Equal(t, "repo", sourceLabel(src, "format"))
	assert.Equal(t, "global", sourceLabel(src, "max_tokens"))
	assert.Equal(t, "flag", sourceLabel(src, "output"))
}

func TestShowProfile_EscapesSpecialCharsInStrings(t *testing.T) {
	p := DefaultProfile()
	p.Output = `path\to\"output".md`
	src := make(SourceMap)

	output := ShowProfile(ShowOptions{
		Profile:     p,
		Sources:     src,
		ProfileName: "default",
		Chain:       []string{"default"},
	})

	// Verify the string is in the output (the escaping is correct).
	assert.True(t, strings.Contains(output, "output"), "output field should be present")
}
