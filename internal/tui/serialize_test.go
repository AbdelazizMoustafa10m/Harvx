package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/tui/filetree"
)

func TestSerializeSelectionToTOML_Basic(t *testing.T) {
	t.Parallel()

	result, err := serializeSelectionToTOML("myprofile", []string{
		"internal/config/types.go",
		"cmd/harvx/main.go",
		"go.mod",
	})
	require.NoError(t, err)

	// Should contain the profile name.
	assert.Contains(t, result, "myprofile")
	// Should contain include key.
	assert.Contains(t, result, "include")
	// Paths should be sorted.
	assert.Contains(t, result, "cmd/harvx/main.go")
	assert.Contains(t, result, "go.mod")
	assert.Contains(t, result, "internal/config/types.go")
}

func TestSerializeSelectionToTOML_EmptyName(t *testing.T) {
	t.Parallel()

	_, err := serializeSelectionToTOML("", []string{"a.go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile name must not be empty")
}

func TestSerializeSelectionToTOML_EmptyPaths(t *testing.T) {
	t.Parallel()

	result, err := serializeSelectionToTOML("empty", nil)
	require.NoError(t, err)
	assert.Contains(t, result, "empty")
}

func TestSerializeSelectionToTOML_SortsDeterministically(t *testing.T) {
	t.Parallel()

	paths := []string{"z.go", "a.go", "m.go"}
	result1, err := serializeSelectionToTOML("test", paths)
	require.NoError(t, err)

	result2, err := serializeSelectionToTOML("test", paths)
	require.NoError(t, err)

	assert.Equal(t, result1, result2, "output should be deterministic")
}

func TestAppendProfileToFile_CreatesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "harvx.toml")

	err := appendProfileToFile(path, "newprofile", []string{"main.go", "lib/util.go"})
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Contains(t, string(content), "newprofile")
	assert.Contains(t, string(content), "main.go")
	assert.Contains(t, string(content), "lib/util.go")
}

func TestAppendProfileToFile_AppendsToExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "harvx.toml")

	// Write initial content.
	err := os.WriteFile(path, []byte("[profile.default]\nformat = \"markdown\"\n"), 0o644)
	require.NoError(t, err)

	err = appendProfileToFile(path, "custom", []string{"app.go"})
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	// Original content should be preserved.
	assert.Contains(t, string(content), "[profile.default]")
	assert.Contains(t, string(content), "format = \"markdown\"")
	// New profile should be appended.
	assert.Contains(t, string(content), "custom")
	assert.Contains(t, string(content), "app.go")
}

// --- SerializeToProfile tests ---

func TestSerializeToProfile_EmptyName(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	_, err := SerializeToProfile("", root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile name must not be empty")
}

func TestSerializeToProfile_NilRoot(t *testing.T) {
	t.Parallel()

	_, err := SerializeToProfile("test", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file tree root must not be nil")
}

func TestSerializeToProfile_EmptyTree(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	data, err := SerializeToProfile("empty", root, nil)
	require.NoError(t, err)

	// Should produce valid TOML.
	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Should have profile section with extends = "default".
	profile, ok := parsed["profile"].(map[string]any)
	require.True(t, ok)
	section, ok := profile["empty"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "default", section["extends"])
}

func TestSerializeToProfile_WithIncludedFiles(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	f1 := filetree.NewNode("main.go", "main.go", false)
	f1.Included = filetree.Included
	f1.Tier = 1
	root.AddChild(f1)

	f2 := filetree.NewNode("README.md", "README.md", false)
	f2.Included = filetree.Included
	f2.Tier = 4
	root.AddChild(f2)

	data, err := SerializeToProfile("myselection", root, nil)
	require.NoError(t, err)

	s := string(data)
	assert.Contains(t, s, "myselection")
	assert.Contains(t, s, "main.go")
	assert.Contains(t, s, "README.md")
	assert.Contains(t, s, `extends = "default"`)

	// Verify valid TOML.
	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)
}

func TestSerializeToProfile_WithPriorityFiles(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	// Tier 0 file should appear in priority_files.
	f1 := filetree.NewNode("go.mod", "go.mod", false)
	f1.Included = filetree.Included
	f1.Tier = 0
	root.AddChild(f1)

	f2 := filetree.NewNode("main.go", "main.go", false)
	f2.Included = filetree.Included
	f2.Tier = 1
	root.AddChild(f2)

	data, err := SerializeToProfile("prio", root, nil)
	require.NoError(t, err)

	s := string(data)
	assert.Contains(t, s, "priority_files")
	assert.Contains(t, s, "go.mod")

	// Verify tier info is present.
	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	profile := parsed["profile"].(map[string]any)
	section := profile["prio"].(map[string]any)

	pf, ok := section["priority_files"].([]any)
	require.True(t, ok)
	assert.Len(t, pf, 1)
	assert.Equal(t, "go.mod", pf[0])
}

func TestSerializeToProfile_DirectoryGlobMinimization(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	// Create a directory with all children included.
	dir := filetree.NewNode("internal", "internal", true)
	dir.Included = filetree.Included
	root.AddChild(dir)

	f1 := filetree.NewNode("internal/a.go", "a.go", false)
	f1.Included = filetree.Included
	f1.Tier = 1
	dir.AddChild(f1)

	f2 := filetree.NewNode("internal/b.go", "b.go", false)
	f2.Included = filetree.Included
	f2.Tier = 1
	dir.AddChild(f2)

	data, err := SerializeToProfile("dirglob", root, nil)
	require.NoError(t, err)

	s := string(data)
	// Should use directory glob instead of individual files.
	assert.Contains(t, s, "internal/**")

	// Verify valid TOML.
	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)
}

func TestSerializeToProfile_IgnorePatternsForExcluded(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	// Create a directory with all children excluded.
	dir := filetree.NewNode("vendor", "vendor", true)
	dir.Included = filetree.Excluded
	root.AddChild(dir)

	f1 := filetree.NewNode("vendor/dep.go", "dep.go", false)
	f1.Included = filetree.Excluded
	dir.AddChild(f1)

	data, err := SerializeToProfile("withignore", root, nil)
	require.NoError(t, err)

	s := string(data)
	// Should have ignore patterns.
	assert.Contains(t, s, "ignore")
	assert.Contains(t, s, "vendor/**")
}

func TestSerializeToProfile_PreservesBaseConfig(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	f1 := filetree.NewNode("main.go", "main.go", false)
	f1.Included = filetree.Included
	f1.Tier = 1
	root.AddChild(f1)

	baseCfg := &config.ResolvedConfig{
		Profile: &config.Profile{
			Format:      "xml",
			MaxTokens:   50000,
			Tokenizer:   "o200k_base",
			Target:      "claude",
			Compression: true,
			Redaction:   true,
		},
		ProfileName: "custom",
	}

	data, err := SerializeToProfile("saved", root, baseCfg)
	require.NoError(t, err)

	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	profile := parsed["profile"].(map[string]any)
	section := profile["saved"].(map[string]any)

	assert.Equal(t, "default", section["extends"])
	assert.Equal(t, "xml", section["format"])
	assert.Equal(t, int64(50000), section["max_tokens"])
	assert.Equal(t, "o200k_base", section["tokenizer"])
	assert.Equal(t, "claude", section["target"])
	assert.Equal(t, true, section["compression"])
	assert.Equal(t, true, section["redaction"])
}

func TestSerializeToProfile_NilBaseCfg(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	f1 := filetree.NewNode("main.go", "main.go", false)
	f1.Included = filetree.Included
	root.AddChild(f1)

	// Should not panic with nil config.
	data, err := SerializeToProfile("noconfig", root, nil)
	require.NoError(t, err)

	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	profile := parsed["profile"].(map[string]any)
	section := profile["noconfig"].(map[string]any)

	// Should still have extends but no format/tokens etc.
	assert.Equal(t, "default", section["extends"])
	assert.Nil(t, section["format"])
	assert.Nil(t, section["max_tokens"])
}

func TestSerializeToProfile_NilProfile(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	f1 := filetree.NewNode("main.go", "main.go", false)
	f1.Included = filetree.Included
	root.AddChild(f1)

	baseCfg := &config.ResolvedConfig{
		Profile: nil,
	}

	// Should not panic with nil profile.
	data, err := SerializeToProfile("nilprofile", root, baseCfg)
	require.NoError(t, err)

	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)
}

func TestSerializeToProfile_RelevanceTiers(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	f0 := filetree.NewNode("go.mod", "go.mod", false)
	f0.Included = filetree.Included
	f0.Tier = 0
	root.AddChild(f0)

	f1 := filetree.NewNode("main.go", "main.go", false)
	f1.Included = filetree.Included
	f1.Tier = 1
	root.AddChild(f1)

	f3 := filetree.NewNode("main_test.go", "main_test.go", false)
	f3.Included = filetree.Included
	f3.Tier = 3
	root.AddChild(f3)

	data, err := SerializeToProfile("tiered", root, nil)
	require.NoError(t, err)

	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	profile := parsed["profile"].(map[string]any)
	section := profile["tiered"].(map[string]any)

	rel, ok := section["relevance"].(map[string]any)
	require.True(t, ok, "should have relevance section")

	tier0, ok := rel["tier_0"].([]any)
	require.True(t, ok)
	assert.Contains(t, tier0, "go.mod")

	tier1, ok := rel["tier_1"].([]any)
	require.True(t, ok)
	assert.Contains(t, tier1, "main.go")

	tier3, ok := rel["tier_3"].([]any)
	require.True(t, ok)
	assert.Contains(t, tier3, "main_test.go")

	// Tiers 2, 4, 5 should not be present.
	assert.Nil(t, rel["tier_2"])
	assert.Nil(t, rel["tier_4"])
	assert.Nil(t, rel["tier_5"])
}

func TestSerializeToProfile_SkipsZeroValueSettings(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	f1 := filetree.NewNode("main.go", "main.go", false)
	f1.Included = filetree.Included
	root.AddChild(f1)

	// Profile with mostly zero values.
	baseCfg := &config.ResolvedConfig{
		Profile: &config.Profile{
			Format: "markdown",
			// MaxTokens: 0 (zero, should be skipped)
			// Compression: false (should be skipped)
			// Redaction: false (should be skipped)
		},
	}

	data, err := SerializeToProfile("minimal", root, baseCfg)
	require.NoError(t, err)

	var parsed map[string]any
	err = toml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	profile := parsed["profile"].(map[string]any)
	section := profile["minimal"].(map[string]any)

	assert.Equal(t, "markdown", section["format"])
	assert.Nil(t, section["max_tokens"], "zero max_tokens should be omitted")
	assert.Nil(t, section["compression"], "false compression should be omitted")
	assert.Nil(t, section["redaction"], "false redaction should be omitted")
	assert.Nil(t, section["tokenizer"], "empty tokenizer should be omitted")
	assert.Nil(t, section["target"], "empty target should be omitted")
}

func TestSerializeToProfile_DeterministicOutput(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	for _, name := range []string{"z.go", "a.go", "m.go"} {
		f := filetree.NewNode(name, name, false)
		f.Included = filetree.Included
		f.Tier = 1
		root.AddChild(f)
	}

	data1, err := SerializeToProfile("det", root, nil)
	require.NoError(t, err)

	data2, err := SerializeToProfile("det", root, nil)
	require.NoError(t, err)

	assert.Equal(t, data1, data2, "output should be deterministic across calls")
}

func TestSerializeToProfile_ValidTOMLParseable(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	// Build a mixed tree.
	dir := filetree.NewNode("src", "src", true)
	dir.Included = filetree.Partial
	root.AddChild(dir)

	f1 := filetree.NewNode("src/app.go", "app.go", false)
	f1.Included = filetree.Included
	f1.Tier = 1
	dir.AddChild(f1)

	f2 := filetree.NewNode("src/test.go", "test.go", false)
	f2.Included = filetree.Excluded
	f2.Tier = 3
	dir.AddChild(f2)

	baseCfg := &config.ResolvedConfig{
		Profile: &config.Profile{
			Format:    "markdown",
			MaxTokens: 120000,
		},
	}

	data, err := SerializeToProfile("roundtrip", root, baseCfg)
	require.NoError(t, err)

	// Must be parseable as valid TOML config.
	var cfg config.Config
	err = toml.Unmarshal(data, &cfg)
	require.NoError(t, err)

	p, ok := cfg.Profile["roundtrip"]
	require.True(t, ok, "profile 'roundtrip' should exist")
	require.NotNil(t, p.Extends)
	assert.Equal(t, "default", *p.Extends)
	assert.Equal(t, "markdown", p.Format)
	assert.Equal(t, 120000, p.MaxTokens)
	assert.Contains(t, p.Include, "src/app.go")
}

// --- buildRelevanceTiers tests ---

func TestBuildRelevanceTiers_Nil(t *testing.T) {
	t.Parallel()

	result := buildRelevanceTiers(nil)
	assert.Nil(t, result)
}

func TestBuildRelevanceTiers_EmptyMap(t *testing.T) {
	t.Parallel()

	result := buildRelevanceTiers(map[int][]string{})
	assert.Nil(t, result)
}

func TestBuildRelevanceTiers_EmptySlices(t *testing.T) {
	t.Parallel()

	result := buildRelevanceTiers(map[int][]string{
		0: {},
		1: {},
	})
	assert.Nil(t, result)
}

func TestBuildRelevanceTiers_MultipleTiers(t *testing.T) {
	t.Parallel()

	result := buildRelevanceTiers(map[int][]string{
		0: {"go.mod"},
		2: {"src/util.go"},
		5: {"ci.yml"},
	})

	require.NotNil(t, result)
	assert.Equal(t, []string{"go.mod"}, result["tier_0"])
	assert.Equal(t, []string{"src/util.go"}, result["tier_2"])
	assert.Equal(t, []string{"ci.yml"}, result["tier_5"])
	assert.Nil(t, result["tier_1"])
	assert.Nil(t, result["tier_3"])
	assert.Nil(t, result["tier_4"])
}

func TestBuildRelevanceTiers_IgnoresOutOfRange(t *testing.T) {
	t.Parallel()

	// Tier 7 is out of range (0-5), should be ignored.
	result := buildRelevanceTiers(map[int][]string{
		1: {"main.go"},
		7: {"weird.go"},
	})

	require.NotNil(t, result)
	assert.Equal(t, []string{"main.go"}, result["tier_1"])
	assert.Nil(t, result["tier_7"])
}

// --- SaveProfileToFile tests ---

func TestSaveProfileToFile_CreatesNewFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "harvx.toml")

	root := filetree.NewNode("", "root", true)
	f1 := filetree.NewNode("main.go", "main.go", false)
	f1.Included = filetree.Included
	f1.Tier = 1
	root.AddChild(f1)

	err := SaveProfileToFile(path, "saved", root, nil)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	s := string(content)
	assert.Contains(t, s, "saved")
	assert.Contains(t, s, "main.go")
	assert.Contains(t, s, `extends = "default"`)

	// Should be valid TOML.
	var parsed map[string]any
	err = toml.Unmarshal(content, &parsed)
	require.NoError(t, err)
}

func TestSaveProfileToFile_AppendsToExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "harvx.toml")

	// Pre-populate the file.
	err := os.WriteFile(path, []byte("[profile.default]\nformat = \"markdown\"\n"), 0o644)
	require.NoError(t, err)

	root := filetree.NewNode("", "root", true)
	f1 := filetree.NewNode("app.go", "app.go", false)
	f1.Included = filetree.Included
	f1.Tier = 2
	root.AddChild(f1)

	err = SaveProfileToFile(path, "appended", root, nil)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	s := string(content)
	// Original content should be preserved.
	assert.Contains(t, s, "[profile.default]")
	assert.Contains(t, s, `format = "markdown"`)
	// New profile should be present.
	assert.Contains(t, s, "appended")
	assert.Contains(t, s, "app.go")
}

func TestSaveProfileToFile_ValidationErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "harvx.toml")

	root := filetree.NewNode("", "root", true)

	err := SaveProfileToFile(path, "", root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile name must not be empty")

	err = SaveProfileToFile(path, "test", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file tree root must not be nil")
}

func TestSaveProfileToFile_InvalidPath(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)

	err := SaveProfileToFile("/nonexistent/deeply/nested/path/harvx.toml", "test", root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opening config file")
}
