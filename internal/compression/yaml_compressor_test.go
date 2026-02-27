package compression

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// YAMLCompressor metadata tests
// ---------------------------------------------------------------------------

func TestYAMLCompressor_Language(t *testing.T) {
	c := NewYAMLCompressor()
	assert.Equal(t, "yaml", c.Language())
}

func TestYAMLCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewYAMLCompressor()
	types := c.SupportedNodeTypes()
	assert.Equal(t, []string{"mapping", "sequence", "comment"}, types)
}

// ---------------------------------------------------------------------------
// Empty input
// ---------------------------------------------------------------------------

func TestYAMLCompressor_EmptyInput(t *testing.T) {
	c := NewYAMLCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "yaml", output.Language)
}

// ---------------------------------------------------------------------------
// Comment preservation
// ---------------------------------------------------------------------------

func TestYAMLCompressor_CommentPreservation(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantComments    []string
		notWantComments []string
	}{
		{
			name: "top-level comment preserved",
			input: `# Top-level comment
name: my-app`,
			wantComments: []string{"# Top-level comment"},
		},
		{
			name: "depth-0 comment preserved",
			input: `# Root comment
key: value`,
			wantComments: []string{"# Root comment"},
		},
		{
			name: "depth-1 comment preserved",
			input: `services:
  # Service comment
  web:
    image: nginx`,
			wantComments: []string{"# Service comment"},
		},
		{
			name: "depth-2 comment preserved",
			input: `services:
  web:
    # Build comment
    build: .`,
			wantComments: []string{"# Build comment"},
		},
		{
			name: "depth-3 comment filtered out",
			input: `services:
  web:
    build:
      # Deep comment
      context: .`,
			notWantComments: []string{"# Deep comment"},
		},
		{
			name: "multiple comments at different depths",
			input: `# Root comment
services:
  # Depth 1 comment
  web:
    # Depth 2 comment
    build:
      # Depth 3 comment - too deep
      context: .`,
			wantComments:    []string{"# Root comment", "# Depth 1 comment", "# Depth 2 comment"},
			notWantComments: []string{"# Depth 3 comment"},
		},
	}

	c := NewYAMLCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, output.Signatures)

			compressed := output.Signatures[0].Source
			for _, want := range tt.wantComments {
				assert.Contains(t, compressed, want, "expected comment to be preserved")
			}
			for _, notWant := range tt.notWantComments {
				assert.NotContains(t, compressed, notWant, "expected comment to be filtered out")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Depth filtering
// ---------------------------------------------------------------------------

func TestYAMLCompressor_DepthFiltering(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeys []string
		skipKeys []string
	}{
		{
			name: "depth 0 keys preserved",
			input: `name: my-app
version: "1.0"`,
			wantKeys: []string{"name:", "version:"},
		},
		{
			name: "depth 0 and 1 keys preserved, depth 2 preserved too",
			input: `services:
  web:
    image: nginx
    build:
      context: .`,
			wantKeys: []string{"services:", "web:", "image:", "build:"},
			skipKeys: []string{"context:"},
		},
		{
			name: "depth 2 keys preserved, depth 3 skipped",
			input: `services:
  web:
    build:
      context: .
      dockerfile: Dockerfile`,
			wantKeys: []string{"services:", "web:", "build:"},
			skipKeys: []string{"context:", "dockerfile:"},
		},
		{
			name: "deeply nested content skipped",
			input: `level0:
  level1:
    level2:
      level3:
        level4: deep-value`,
			wantKeys: []string{"level0:", "level1:", "level2:"},
			skipKeys: []string{"level3:", "level4:"},
		},
	}

	c := NewYAMLCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, output.Signatures)

			compressed := output.Signatures[0].Source
			for _, key := range tt.wantKeys {
				assert.Contains(t, compressed, key, "expected key %q to be preserved", key)
			}
			for _, key := range tt.skipKeys {
				assert.NotContains(t, compressed, key, "expected key %q to be filtered", key)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// List collapsing
// ---------------------------------------------------------------------------

func TestYAMLCompressor_ListCollapsing(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantItems    []string
		wantSummary  string
		notWantItems []string
	}{
		{
			name: "list with 3 items - no collapsing",
			input: `items:
  - one
  - two
  - three`,
			wantItems: []string{"- one", "- two", "- three"},
		},
		{
			name: "list with exactly 5 items - no collapsing",
			input: `items:
  - one
  - two
  - three
  - four
  - five`,
			wantItems: []string{"- one", "- two", "- three", "- four", "- five"},
		},
		{
			name: "list with 8 items - collapsed after 5",
			input: `items:
  - one
  - two
  - three
  - four
  - five
  - six
  - seven
  - eight`,
			wantItems:    []string{"- one", "- two", "- three", "- four", "- five"},
			wantSummary:  "# ... (3 more items)",
			notWantItems: []string{"- six", "- seven", "- eight"},
		},
		{
			name: "list with 10 items - collapsed after 5",
			input: `env:
  - A=1
  - B=2
  - C=3
  - D=4
  - E=5
  - F=6
  - G=7
  - H=8
  - I=9
  - J=10`,
			wantItems:    []string{"- A=1", "- B=2", "- C=3", "- D=4", "- E=5"},
			wantSummary:  "# ... (5 more items)",
			notWantItems: []string{"- F=6", "- G=7", "- H=8", "- I=9", "- J=10"},
		},
		{
			name: "list with 6 items - collapsed after 5",
			input: `ports:
  - "80:80"
  - "443:443"
  - "8080:8080"
  - "8443:8443"
  - "3000:3000"
  - "5000:5000"`,
			wantItems:    []string{`- "80:80"`, `- "443:443"`, `- "8080:8080"`, `- "8443:8443"`, `- "3000:3000"`},
			wantSummary:  "# ... (1 more items)",
			notWantItems: []string{`- "5000:5000"`},
		},
	}

	c := NewYAMLCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, output.Signatures)

			compressed := output.Signatures[0].Source
			for _, item := range tt.wantItems {
				assert.Contains(t, compressed, item, "expected item %q to be present", item)
			}
			if tt.wantSummary != "" {
				assert.Contains(t, compressed, tt.wantSummary, "expected summary comment")
			}
			for _, item := range tt.notWantItems {
				assert.NotContains(t, compressed, item, "expected item %q to be collapsed", item)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Multi-line string truncation
// ---------------------------------------------------------------------------

func TestYAMLCompressor_MultiLineStringTruncation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantLines    []string
		wantTrunc    bool
		notWantLines []string
	}{
		{
			name: "block scalar with pipe - 2 lines no truncation",
			input: `description: |
  Line one
  Line two`,
			wantLines: []string{"Line one", "Line two"},
			wantTrunc: false,
		},
		{
			name: "block scalar with pipe - 3 lines no truncation",
			input: `description: |
  Line one
  Line two
  Line three`,
			wantLines: []string{"Line one", "Line two", "Line three"},
			wantTrunc: false,
		},
		{
			name: "block scalar with pipe - 5 lines truncated after 3",
			input: `description: |
  Line one
  Line two
  Line three
  Line four
  Line five`,
			wantLines:    []string{"Line one", "Line two", "Line three"},
			wantTrunc:    true,
			notWantLines: []string{"Line four", "Line five"},
		},
		{
			name: "block scalar with > indicator",
			input: `description: >
  Folded line one
  Folded line two
  Folded line three
  Folded line four
  Folded line five`,
			wantLines:    []string{"Folded line one", "Folded line two", "Folded line three"},
			wantTrunc:    true,
			notWantLines: []string{"Folded line four", "Folded line five"},
		},
		{
			name: "block scalar with |- indicator",
			input: `script: |-
  echo "step 1"
  echo "step 2"
  echo "step 3"
  echo "step 4"
  echo "step 5"`,
			wantLines:    []string{`echo "step 1"`, `echo "step 2"`, `echo "step 3"`},
			wantTrunc:    true,
			notWantLines: []string{`echo "step 4"`, `echo "step 5"`},
		},
	}

	c := NewYAMLCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, output.Signatures)

			compressed := output.Signatures[0].Source
			for _, line := range tt.wantLines {
				assert.Contains(t, compressed, line, "expected line %q to be present", line)
			}
			if tt.wantTrunc {
				assert.Contains(t, compressed, "# ... (truncated)", "expected truncation marker")
			}
			for _, line := range tt.notWantLines {
				assert.NotContains(t, compressed, line, "expected line %q to be truncated", line)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Blank line collapsing
// ---------------------------------------------------------------------------

func TestYAMLCompressor_BlankLineCollapsing(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		maxConsecBlank int
	}{
		{
			name: "two consecutive blank lines collapsed",
			input: `name: app


version: "1.0"`,
			maxConsecBlank: 1,
		},
		{
			name: "three consecutive blank lines collapsed",
			input: `name: app



version: "1.0"`,
			maxConsecBlank: 1,
		},
		{
			name: "five consecutive blank lines collapsed",
			input: `name: app





version: "1.0"`,
			maxConsecBlank: 1,
		},
	}

	c := NewYAMLCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, output.Signatures)

			compressed := output.Signatures[0].Source
			lines := strings.Split(compressed, "\n")
			consecutiveBlank := 0
			maxBlank := 0
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					consecutiveBlank++
					if consecutiveBlank > maxBlank {
						maxBlank = consecutiveBlank
					}
				} else {
					consecutiveBlank = 0
				}
			}
			assert.LessOrEqual(t, maxBlank, tt.maxConsecBlank,
				"expected at most %d consecutive blank lines, got %d", tt.maxConsecBlank, maxBlank)
		})
	}
}

// ---------------------------------------------------------------------------
// Document separator preservation
// ---------------------------------------------------------------------------

func TestYAMLCompressor_DocumentSeparator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSeps []string
	}{
		{
			name: "document start marker ---",
			input: `---
name: my-app
version: "1.0"`,
			wantSeps: []string{"---"},
		},
		{
			name: "document end marker ...",
			input: `name: my-app
...`,
			wantSeps: []string{"..."},
		},
		{
			name: "multiple documents",
			input: `---
name: doc1
...
---
name: doc2
...`,
			wantSeps: []string{"---", "..."},
		},
		{
			name: "document start only",
			input: `---
key: value`,
			wantSeps: []string{"---"},
		},
	}

	c := NewYAMLCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, output.Signatures)

			compressed := output.Signatures[0].Source
			for _, sep := range tt.wantSeps {
				assert.Contains(t, compressed, sep, "expected document separator %q to be preserved", sep)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// String truncation (long inline values)
// ---------------------------------------------------------------------------

func TestYAMLCompressor_StringTruncation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantTrunc    bool
		wantContains string
	}{
		{
			name:         "short value not truncated",
			input:        `key: short value`,
			wantTrunc:    false,
			wantContains: "short value",
		},
		{
			name:         "exactly 80 chars not truncated",
			input:        fmt.Sprintf("key: %s", strings.Repeat("a", 80)),
			wantTrunc:    false,
			wantContains: strings.Repeat("a", 80),
		},
		{
			name:         "81 chars truncated",
			input:        fmt.Sprintf("key: %s", strings.Repeat("b", 81)),
			wantTrunc:    true,
			wantContains: strings.Repeat("b", 80) + "...",
		},
		{
			name:         "very long value truncated",
			input:        fmt.Sprintf("key: %s", strings.Repeat("x", 200)),
			wantTrunc:    true,
			wantContains: strings.Repeat("x", 80) + "...",
		},
		{
			name:      "block scalar indicator not truncated",
			input:     `key: |`,
			wantTrunc: false,
		},
		{
			name:      "folded scalar indicator not truncated",
			input:     `key: >`,
			wantTrunc: false,
		},
		{
			name:      "empty value not truncated",
			input:     `key:`,
			wantTrunc: false,
		},
	}

	c := NewYAMLCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, output.Signatures)

			compressed := output.Signatures[0].Source
			if tt.wantTrunc {
				assert.Contains(t, compressed, "...", "expected truncation ellipsis")
			}
			if tt.wantContains != "" {
				assert.Contains(t, compressed, tt.wantContains)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestYAMLCompressor_ContextCancellation(t *testing.T) {
	t.Run("immediate cancellation before processing", func(t *testing.T) {
		c := NewYAMLCompressor()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		_, err := c.Compress(ctx, []byte("key: value"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("cancellation during large input processing", func(t *testing.T) {
		c := NewYAMLCompressor()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		// Generate a large YAML source to ensure cancellation is checked
		// during the parse loop (every 1000 lines).
		var b strings.Builder
		for i := 0; i < 5000; i++ {
			fmt.Fprintf(&b, "key_%d: value_%d\n", i, i)
		}

		_, err := c.Compress(ctx, []byte(b.String()))
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

// ---------------------------------------------------------------------------
// Golden test: Docker Compose fixture
// ---------------------------------------------------------------------------

func TestYAMLCompressor_GoldenDockerCompose(t *testing.T) {
	path := filepath.Join(testdataDir(), "yaml", "docker-compose.yml")
	source, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read docker-compose fixture")

	c := NewYAMLCompressor()
	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source

	// Top-level comment should be preserved.
	assert.Contains(t, compressed, "# Docker Compose configuration")

	// Top-level keys should be preserved.
	assert.Contains(t, compressed, "version:")
	assert.Contains(t, compressed, "services:")
	assert.Contains(t, compressed, "volumes:")
	assert.Contains(t, compressed, "networks:")

	// Depth-1 service names should be preserved.
	assert.Contains(t, compressed, "web:")
	assert.Contains(t, compressed, "db:")
	assert.Contains(t, compressed, "cache:")

	// Depth-2 keys should be preserved.
	assert.Contains(t, compressed, "build:")
	assert.Contains(t, compressed, "ports:")
	assert.Contains(t, compressed, "environment:")
	assert.Contains(t, compressed, "image:")

	// Depth-3 keys (e.g., context under build) should be filtered.
	assert.NotContains(t, compressed, "context: .")
	assert.NotContains(t, compressed, "dockerfile: Dockerfile")

	// Environment list items are at depth 3 (indent 6), so they are all
	// filtered by depth. No list collapsing message for deep lists.
	assert.NotContains(t, compressed, "NODE_ENV=production",
		"environment list items at depth 3 should be filtered")

	// Output metadata should be correct.
	assert.Equal(t, "yaml", output.Language)
	assert.Equal(t, len(source), output.OriginalSize)
	assert.Equal(t, 1, output.NodeCount)
}

// ---------------------------------------------------------------------------
// Golden test: GitHub Workflow fixture
// ---------------------------------------------------------------------------

func TestYAMLCompressor_GoldenGithubWorkflow(t *testing.T) {
	path := filepath.Join(testdataDir(), "yaml", "github-workflow.yml")
	source, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read github-workflow fixture")

	c := NewYAMLCompressor()
	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source

	// Top-level comment should be preserved.
	assert.Contains(t, compressed, "# CI/CD Pipeline")

	// Top-level keys should be preserved.
	assert.Contains(t, compressed, "name:")
	assert.Contains(t, compressed, "on:")
	assert.Contains(t, compressed, "permissions:")
	assert.Contains(t, compressed, "jobs:")

	// Depth-1 keys should be preserved.
	assert.Contains(t, compressed, "push:")
	assert.Contains(t, compressed, "pull_request:")
	assert.Contains(t, compressed, "contents:")
	assert.Contains(t, compressed, "test:")
	assert.Contains(t, compressed, "deploy:")

	// Depth-2 keys should be preserved.
	assert.Contains(t, compressed, "runs-on:")
	assert.Contains(t, compressed, "strategy:")
	assert.Contains(t, compressed, "steps:")
	assert.Contains(t, compressed, "needs:")
	assert.Contains(t, compressed, "if:")

	// Depth-3+ content should be filtered (matrix is at depth 3).
	assert.NotContains(t, compressed, "node-version: ${{ matrix.node-version }}")
	assert.NotContains(t, compressed, "matrix:")

	// Steps list items are at depth 3 and are filtered by depth,
	// so individual step items should not appear.
	assert.NotContains(t, compressed, "actions/checkout@v4",
		"step items at depth 3 should be filtered")

	// Output size should be smaller than input.
	assert.Less(t, output.OutputSize, output.OriginalSize,
		"compressed output should be smaller than original")
}

// ---------------------------------------------------------------------------
// Compression ratio
// ---------------------------------------------------------------------------

func TestYAMLCompressor_CompressionRatio(t *testing.T) {
	// Use the docker-compose fixture which has enough depth/content to compress.
	path := filepath.Join(testdataDir(), "yaml", "docker-compose.yml")
	source, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read docker-compose fixture")

	c := NewYAMLCompressor()
	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	ratio := output.CompressionRatio()
	assert.Greater(t, ratio, 0.05, "ratio should be > 0.05 (not fully empty)")
	assert.Less(t, ratio, 0.95, "ratio should be < 0.95 (meaningful compression)")

	// The compressed output should be smaller than original.
	assert.Less(t, output.OutputSize, output.OriginalSize,
		"compressed output should be smaller than original")
}

// ---------------------------------------------------------------------------
// Output structure validation
// ---------------------------------------------------------------------------

func TestYAMLCompressor_OutputStructure(t *testing.T) {
	c := NewYAMLCompressor()
	input := `# Config file
name: my-app
version: "1.0"
`
	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	// Should have exactly one signature.
	require.Len(t, output.Signatures, 1)

	sig := output.Signatures[0]
	assert.Equal(t, KindDocComment, sig.Kind)
	assert.Equal(t, "yaml-structure", sig.Name)
	assert.Equal(t, 1, sig.StartLine)
	assert.Greater(t, sig.EndLine, 0)

	// Language and size fields.
	assert.Equal(t, "yaml", output.Language)
	assert.Equal(t, len(input), output.OriginalSize)
	assert.Equal(t, 1, output.NodeCount)
	assert.Greater(t, output.OutputSize, 0)
}

// ---------------------------------------------------------------------------
// Simple fixture test
// ---------------------------------------------------------------------------

func TestYAMLCompressor_SimpleFixture(t *testing.T) {
	path := filepath.Join(testdataDir(), "yaml", "simple.yml")
	source, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read simple.yml fixture")

	c := NewYAMLCompressor()
	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source

	// Comment preserved.
	assert.Contains(t, compressed, "# Simple config")

	// All top-level keys preserved.
	assert.Contains(t, compressed, "name:")
	assert.Contains(t, compressed, "version:")
	assert.Contains(t, compressed, "debug:")
	assert.Contains(t, compressed, "items:")

	// List items preserved (only 3, under maxListItems).
	assert.Contains(t, compressed, "- one")
	assert.Contains(t, compressed, "- two")
	assert.Contains(t, compressed, "- three")

	// No collapsing for short list.
	assert.NotContains(t, compressed, "# ... (")
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestYAMLCompressor_OnlyComments(t *testing.T) {
	c := NewYAMLCompressor()
	input := `# Comment line 1
# Comment line 2
# Comment line 3`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source
	assert.Contains(t, compressed, "# Comment line 1")
	assert.Contains(t, compressed, "# Comment line 2")
	assert.Contains(t, compressed, "# Comment line 3")
}

func TestYAMLCompressor_OnlyDocumentSeparators(t *testing.T) {
	c := NewYAMLCompressor()
	input := `---
...`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source
	assert.Contains(t, compressed, "---")
	assert.Contains(t, compressed, "...")
}

func TestYAMLCompressor_SingleKeyValue(t *testing.T) {
	c := NewYAMLCompressor()
	input := `key: value`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source
	assert.Equal(t, "key: value", strings.TrimSpace(compressed))
}

func TestYAMLCompressor_TrailingNewlines(t *testing.T) {
	c := NewYAMLCompressor()
	input := "key: value\n\n\n\n"

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source
	// Trailing blank lines should be removed.
	assert.Equal(t, "key: value", strings.TrimSpace(compressed))
}

func TestYAMLCompressor_MixedListsAndMappings(t *testing.T) {
	c := NewYAMLCompressor()
	input := `mappings:
  key1: value1
  key2: value2
lists:
  - item1
  - item2
  - item3`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source
	assert.Contains(t, compressed, "mappings:")
	assert.Contains(t, compressed, "key1:")
	assert.Contains(t, compressed, "key2:")
	assert.Contains(t, compressed, "lists:")
	assert.Contains(t, compressed, "- item1")
	assert.Contains(t, compressed, "- item2")
	assert.Contains(t, compressed, "- item3")
}

func TestYAMLCompressor_AnchorAndAlias(t *testing.T) {
	// Values starting with & or * should not be truncated (anchor/alias).
	c := NewYAMLCompressor()
	input := `defaults: &defaults
  adapter: postgres
  host: localhost

production:
  <<: *defaults
  database: myapp_production`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source
	assert.Contains(t, compressed, "&defaults")
	assert.Contains(t, compressed, "*defaults")
}

func TestYAMLCompressor_FlowMappingAndSequence(t *testing.T) {
	// Values starting with { or [ should not be truncated.
	c := NewYAMLCompressor()
	input := `flow_map: {key: value, key2: value2}
flow_seq: [1, 2, 3, 4, 5]`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source
	assert.Contains(t, compressed, "{key: value, key2: value2}")
	assert.Contains(t, compressed, "[1, 2, 3, 4, 5]")
}

func TestYAMLCompressor_ListItemsWithKeys(t *testing.T) {
	// List items that are key-value mappings.
	c := NewYAMLCompressor()
	input := `steps:
  - name: Checkout
  - name: Build
  - name: Test
  - name: Deploy
  - name: Notify
  - name: Cleanup`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source
	// First 5 items preserved.
	assert.Contains(t, compressed, "- name: Checkout")
	assert.Contains(t, compressed, "- name: Build")
	assert.Contains(t, compressed, "- name: Test")
	assert.Contains(t, compressed, "- name: Deploy")
	assert.Contains(t, compressed, "- name: Notify")
	// 6th item collapsed.
	assert.Contains(t, compressed, "# ... (1 more items)")
	assert.NotContains(t, compressed, "- name: Cleanup")
}

func TestYAMLCompressor_MultipleListsInDocument(t *testing.T) {
	c := NewYAMLCompressor()
	input := `list_a:
  - a1
  - a2
  - a3
list_b:
  - b1
  - b2
  - b3
  - b4
  - b5
  - b6
  - b7`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, output.Signatures)

	compressed := output.Signatures[0].Source

	// list_a: all 3 items preserved.
	assert.Contains(t, compressed, "- a1")
	assert.Contains(t, compressed, "- a2")
	assert.Contains(t, compressed, "- a3")

	// list_b: 5 of 7 items preserved, 2 collapsed.
	assert.Contains(t, compressed, "- b1")
	assert.Contains(t, compressed, "- b5")
	assert.Contains(t, compressed, "# ... (2 more items)")
	assert.NotContains(t, compressed, "- b6")
	assert.NotContains(t, compressed, "- b7")
}

// ---------------------------------------------------------------------------
// Helper function unit tests
// ---------------------------------------------------------------------------

func TestYamlIndentLevel(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{name: "no indent", line: "key: value", expected: 0},
		{name: "2 spaces", line: "  key: value", expected: 2},
		{name: "4 spaces", line: "    key: value", expected: 4},
		{name: "tab", line: "\tkey: value", expected: 1},
		{name: "empty line", line: "", expected: 0},
		{name: "only spaces", line: "    ", expected: 4},
		{name: "mixed tabs and spaces", line: "\t  key", expected: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, yamlIndentLevel(tt.line))
		})
	}
}

func TestYamlIsComment(t *testing.T) {
	tests := []struct {
		name     string
		trimmed  string
		expected bool
	}{
		{name: "hash comment", trimmed: "# This is a comment", expected: true},
		{name: "hash only", trimmed: "#", expected: true},
		{name: "not a comment", trimmed: "key: value", expected: false},
		{name: "hash in value", trimmed: "key: value # inline", expected: false},
		{name: "empty", trimmed: "", expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, yamlIsComment(tt.trimmed))
		})
	}
}

func TestYamlIsListItem(t *testing.T) {
	tests := []struct {
		name     string
		trimmed  string
		expected bool
	}{
		{name: "simple list item", trimmed: "- item", expected: true},
		{name: "bare dash", trimmed: "-", expected: true},
		{name: "not a list item", trimmed: "key: value", expected: false},
		{name: "dash in value", trimmed: "key: some-value", expected: false},
		{name: "empty", trimmed: "", expected: false},
		{name: "list item with key", trimmed: "- name: Foo", expected: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, yamlIsListItem(tt.trimmed))
		})
	}
}

func TestYamlHasBlockScalarIndicator(t *testing.T) {
	tests := []struct {
		name     string
		trimmed  string
		expected bool
	}{
		{name: "pipe", trimmed: "key: |", expected: true},
		{name: "folded", trimmed: "key: >", expected: true},
		{name: "pipe strip", trimmed: "key: |-", expected: true},
		{name: "folded strip", trimmed: "key: >-", expected: true},
		{name: "pipe keep", trimmed: "key: |+", expected: true},
		{name: "folded keep", trimmed: "key: >+", expected: true},
		{name: "regular value", trimmed: "key: value", expected: false},
		{name: "no colon", trimmed: "just a line", expected: false},
		{name: "empty after colon", trimmed: "key:", expected: false},
		{name: "pipe in value", trimmed: "key: a | b", expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, yamlHasBlockScalarIndicator(tt.trimmed))
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkYAMLCompress(b *testing.B) {
	path := filepath.Join(testdataDir(), "yaml", "docker-compose.yml")
	source, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("failed to read fixture: %v", err)
	}

	c := NewYAMLCompressor()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkYAMLCompress_LargeInput(b *testing.B) {
	// Generate a large YAML document for benchmarking.
	var buf strings.Builder
	buf.WriteString("# Large YAML document\n")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&buf, "service_%d:\n", i)
		fmt.Fprintf(&buf, "  image: nginx:%d\n", i)
		fmt.Fprintf(&buf, "  ports:\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&buf, "    - \"%d:%d\"\n", 8000+j, 80+j)
		}
		fmt.Fprintf(&buf, "  environment:\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&buf, "    - ENV_%d=value_%d\n", j, j)
		}
		buf.WriteString("\n")
	}
	source := []byte(buf.String())

	c := NewYAMLCompressor()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}