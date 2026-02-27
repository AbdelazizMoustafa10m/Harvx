package workflows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOutput_DetectsMarkdownFormat(t *testing.T) {
	t.Parallel()

	content := "# Harvx Context: TestProject\n\n## Files\n\n### `src/main.go`\n\n> **Size:** 1.0 KB | **Tokens:** 100 | **Tier:** critical | **Compressed:** no\n\n```go\npackage main\n```\n"

	result, err := ParseOutput(content)
	require.NoError(t, err)
	assert.Equal(t, "markdown", result.Format)
	assert.Len(t, result.Files, 1)
}

func TestParseOutput_DetectsXMLFormat(t *testing.T) {
	t.Parallel()

	content := `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="src/main.go" tokens="100" tier="critical" size="1024" language="go" compressed="false">
      <content><![CDATA[package main]]></content>
    </file>
  </files>
</repository>`

	result, err := ParseOutput(content)
	require.NoError(t, err)
	assert.Equal(t, "xml", result.Format)
	assert.Len(t, result.Files, 1)
}

func TestParseMarkdownOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		wantFiles  int
		wantPaths  []string
		wantConts  []string
		wantComp   []bool
		wantRedact []int
	}{
		{
			name: "single file",
			content: "# Harvx Context: MyProject\n\n## Files\n\n### `src/main.go`\n\n" +
				"> **Size:** 2.0 KB | **Tokens:** 1,200 | **Tier:** critical | **Compressed:** no\n\n" +
				"```go\npackage main\n\nfunc main() {\n}\n```\n",
			wantFiles: 1,
			wantPaths: []string{"src/main.go"},
			wantConts: []string{"package main\n\nfunc main() {\n}"},
			wantComp:  []bool{false},
			wantRedact: []int{0},
		},
		{
			name: "multiple files",
			content: "# Harvx Context: MyProject\n\n## Files\n\n" +
				"### `src/main.go`\n\n" +
				"> **Size:** 2.0 KB | **Tokens:** 1,200 | **Tier:** critical | **Compressed:** no\n\n" +
				"```go\npackage main\n```\n\n" +
				"### `src/util.go`\n\n" +
				"> **Size:** 1.5 KB | **Tokens:** 800 | **Tier:** primary | **Compressed:** yes\n\n" +
				"```go\npackage util\n```\n",
			wantFiles: 2,
			wantPaths: []string{"src/main.go", "src/util.go"},
			wantConts: []string{"package main", "package util"},
			wantComp:  []bool{false, true},
			wantRedact: []int{0, 0},
		},
		{
			name: "file with redactions",
			content: "# Harvx Context: MyProject\n\n## Files\n\n" +
				"### `config.yaml`\n\n" +
				"> **Size:** 512 B | **Tokens:** 50 | **Tier:** primary | **Compressed:** no\n\n" +
				"```yaml\ndb_host: localhost\ndb_password: [REDACTED:password]\napi_key: [REDACTED:api_key]\n```\n",
			wantFiles: 1,
			wantPaths: []string{"config.yaml"},
			wantConts: []string{"db_host: localhost\ndb_password: [REDACTED:password]\napi_key: [REDACTED:api_key]"},
			wantComp:  []bool{false},
			wantRedact: []int{2},
		},
		{
			name: "compressed file",
			content: "# Harvx Context: MyProject\n\n## Files\n\n" +
				"### `src/service.go`\n\n" +
				"> **Size:** 5.0 KB | **Tokens:** 3,000 | **Tier:** primary | **Compressed:** yes\n\n" +
				"```go\nfunc Process(ctx context.Context, input Input) (Output, error)\nfunc Validate(input Input) error\n```\n",
			wantFiles: 1,
			wantPaths: []string{"src/service.go"},
			wantComp:  []bool{true},
			wantRedact: []int{0},
		},
		{
			name: "escaped triple backticks in content",
			content: "# Harvx Context: MyProject\n\n## Files\n\n" +
				"### `docs/readme.md`\n\n" +
				"> **Size:** 1.0 KB | **Tokens:** 200 | **Tier:** docs | **Compressed:** no\n\n" +
				"```markdown\n# Example\n\n`` `go\nfmt.Println(\"hello\")\n`` `\n```\n",
			wantFiles: 1,
			wantPaths: []string{"docs/readme.md"},
			wantConts: []string{"# Example\n\n```go\nfmt.Println(\"hello\")\n```"},
			wantComp:  []bool{false},
			wantRedact: []int{0},
		},
		{
			name:      "no files section",
			content:   "# Harvx Context: EmptyProject\n\n## Summary\n\nNothing here.\n",
			wantFiles: 0,
		},
		{
			name:      "empty files section",
			content:   "# Harvx Context: EmptyProject\n\n## Files\n\n",
			wantFiles: 0,
		},
		{
			name: "file with summary sections before files",
			content: "# Harvx Context: MyProject\n\n" +
				"| Field | Value |\n|-------|-------|\n| Total Files | 1 |\n\n" +
				"## File Summary\n\n**Total Files:** 1 | **Total Tokens:** 100\n\n" +
				"## Directory Tree\n\n```\nsrc/\n  main.go\n```\n\n" +
				"## Files\n\n" +
				"### `src/main.go`\n\n" +
				"> **Size:** 1.0 KB | **Tokens:** 100 | **Tier:** critical | **Compressed:** no\n\n" +
				"```go\npackage main\n```\n",
			wantFiles: 1,
			wantPaths: []string{"src/main.go"},
			wantConts: []string{"package main"},
			wantComp:  []bool{false},
			wantRedact: []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseMarkdownOutput(tt.content)
			require.NoError(t, err)
			assert.Equal(t, "markdown", result.Format)
			assert.Len(t, result.Files, tt.wantFiles)

			for i := 0; i < len(tt.wantPaths) && i < len(result.Files); i++ {
				assert.Equal(t, tt.wantPaths[i], result.Files[i].Path, "file %d path", i)
			}
			for i := 0; i < len(tt.wantConts) && i < len(result.Files); i++ {
				assert.Equal(t, tt.wantConts[i], result.Files[i].Content, "file %d content", i)
			}
			for i := 0; i < len(tt.wantComp) && i < len(result.Files); i++ {
				assert.Equal(t, tt.wantComp[i], result.Files[i].IsCompressed, "file %d compressed", i)
			}
			for i := 0; i < len(tt.wantRedact) && i < len(result.Files); i++ {
				assert.Equal(t, tt.wantRedact[i], result.Files[i].Redactions, "file %d redactions", i)
			}
		})
	}
}

func TestParseXMLOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		wantFiles  int
		wantPaths  []string
		wantConts  []string
		wantComp   []bool
		wantRedact []int
		wantErr    bool
	}{
		{
			name: "single file",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <metadata><project_name>MyProject</project_name></metadata>
  <files>
    <file path="src/main.go" tokens="100" tier="critical" size="1024" language="go" compressed="false">
      <content><![CDATA[package main

func main() {
}]]></content>
    </file>
  </files>
</repository>`,
			wantFiles: 1,
			wantPaths: []string{"src/main.go"},
			wantConts: []string{"package main\n\nfunc main() {\n}"},
			wantComp:  []bool{false},
			wantRedact: []int{0},
		},
		{
			name: "multiple files",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="src/main.go" tokens="100" tier="critical" size="1024" language="go" compressed="false">
      <content><![CDATA[package main]]></content>
    </file>
    <file path="src/util.go" tokens="80" tier="primary" size="512" language="go" compressed="true">
      <content><![CDATA[package util]]></content>
    </file>
  </files>
</repository>`,
			wantFiles: 2,
			wantPaths: []string{"src/main.go", "src/util.go"},
			wantConts: []string{"package main", "package util"},
			wantComp:  []bool{false, true},
			wantRedact: []int{0, 0},
		},
		{
			name: "content with CDATA split sequences",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="test.xml" tokens="50" tier="primary" size="256" language="xml" compressed="false">
      <content><![CDATA[<data>contains ]]]]><![CDATA[> end marker</data>]]></content>
    </file>
  </files>
</repository>`,
			wantFiles: 1,
			wantPaths: []string{"test.xml"},
			wantConts: []string{"<data>contains ]]> end marker</data>"},
			wantComp:  []bool{false},
			wantRedact: []int{0},
		},
		{
			name: "file with redactions",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="config.yaml" tokens="50" tier="primary" size="256" language="yaml" compressed="false">
      <content><![CDATA[db_host: localhost
db_password: [REDACTED:password]
api_key: [REDACTED:api_key]]]></content>
    </file>
  </files>
</repository>`,
			wantFiles: 1,
			wantPaths: []string{"config.yaml"},
			wantComp:  []bool{false},
			wantRedact: []int{2},
		},
		{
			name: "XML-escaped path attribute",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="src/a&amp;b.go" tokens="50" tier="primary" size="256" language="go" compressed="false">
      <content><![CDATA[package ab]]></content>
    </file>
  </files>
</repository>`,
			wantFiles: 1,
			wantPaths: []string{"src/a&b.go"},
			wantConts: []string{"package ab"},
			wantComp:  []bool{false},
			wantRedact: []int{0},
		},
		{
			name: "no files section",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <metadata><project_name>Empty</project_name></metadata>
</repository>`,
			wantFiles: 0,
		},
		{
			name: "missing closing files tag",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="main.go" tokens="50" tier="primary" size="256" language="go" compressed="false">
      <content><![CDATA[package main]]></content>
    </file>
</repository>`,
			wantErr: true,
		},
		{
			name: "compressed file",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="src/service.go" tokens="300" tier="primary" size="5120" language="go" compressed="true">
      <content><![CDATA[func Process(ctx context.Context, input Input) (Output, error)
func Validate(input Input) error]]></content>
    </file>
  </files>
</repository>`,
			wantFiles: 1,
			wantPaths: []string{"src/service.go"},
			wantComp:  []bool{true},
			wantRedact: []int{0},
		},
		{
			name: "empty CDATA content",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="empty.txt" tokens="0" tier="low" size="0" language="" compressed="false">
      <content><![CDATA[]]></content>
    </file>
  </files>
</repository>`,
			wantFiles: 1,
			wantPaths: []string{"empty.txt"},
			wantConts: []string{""},
			wantComp:  []bool{false},
			wantRedact: []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseXMLOutput(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "xml", result.Format)
			assert.Len(t, result.Files, tt.wantFiles)

			for i := 0; i < len(tt.wantPaths) && i < len(result.Files); i++ {
				assert.Equal(t, tt.wantPaths[i], result.Files[i].Path, "file %d path", i)
			}
			for i := 0; i < len(tt.wantConts) && i < len(result.Files); i++ {
				assert.Equal(t, tt.wantConts[i], result.Files[i].Content, "file %d content", i)
			}
			for i := 0; i < len(tt.wantComp) && i < len(result.Files); i++ {
				assert.Equal(t, tt.wantComp[i], result.Files[i].IsCompressed, "file %d compressed", i)
			}
			for i := 0; i < len(tt.wantRedact) && i < len(result.Files); i++ {
				assert.Equal(t, tt.wantRedact[i], result.Files[i].Redactions, "file %d redactions", i)
			}
		})
	}
}

func TestExtractFencedContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		block   string
		want    string
	}{
		{
			name:  "simple fenced block",
			block: "\n```go\npackage main\n```\n",
			want:  "package main",
		},
		{
			name:  "no language identifier",
			block: "\n```\nhello world\n```\n",
			want:  "hello world",
		},
		{
			name:  "multiline content",
			block: "\n```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```\n",
			want:  "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hi\")\n}",
		},
		{
			name:  "no fenced block",
			block: "\nsome text without code\n",
			want:  "",
		},
		{
			name:  "empty fenced block",
			block: "\n```go\n```\n",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractFencedContent(tt.block)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnwrapCDATA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		input string
		want  string
	}{
		{
			name:  "simple CDATA",
			input: "<![CDATA[hello world]]>",
			want:  "hello world",
		},
		{
			name:  "CDATA with split sequence",
			input: "<![CDATA[contains ]]]]><![CDATA[> end marker]]>",
			want:  "contains ]]> end marker",
		},
		{
			name:  "empty CDATA",
			input: "<![CDATA[]]>",
			want:  "",
		},
		{
			name:  "no CDATA wrapper",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "multiple split sequences",
			input: "<![CDATA[a]]]]><![CDATA[>b]]]]><![CDATA[>c]]>",
			want:  "a]]>b]]>c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unwrapCDATA(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnescapeXMLAttr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escapes",
			input: "simple/path.go",
			want:  "simple/path.go",
		},
		{
			name:  "ampersand",
			input: "a&amp;b",
			want:  "a&b",
		},
		{
			name:  "all entities",
			input: "&lt;tag&gt; &amp; &quot;quoted&quot; &apos;apos&apos;",
			want:  "<tag> & \"quoted\" 'apos'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unescapeXMLAttr(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCountRedactions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "no redactions",
			content: "normal content here",
			want:    0,
		},
		{
			name:    "single redaction",
			content: "api_key: [REDACTED:api_key]",
			want:    1,
		},
		{
			name:    "multiple redactions",
			content: "password: [REDACTED:password]\ntoken: [REDACTED:token]\nkey: [REDACTED:api_key]",
			want:    3,
		},
		{
			name:    "not a redaction pattern",
			content: "[NOT_REDACTED] and [SOMETHING:else but no closing",
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := countRedactions(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseOutput_WhitespaceBeforeXMLDeclaration(t *testing.T) {
	t.Parallel()

	content := `  <?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="main.go" tokens="50" tier="critical" size="256" language="go" compressed="false">
      <content><![CDATA[package main]]></content>
    </file>
  </files>
</repository>`

	result, err := ParseOutput(content)
	require.NoError(t, err)
	assert.Equal(t, "xml", result.Format)
	assert.Len(t, result.Files, 1)
}

func TestParseMarkdownOutput_FileWithErrorBlock(t *testing.T) {
	t.Parallel()

	// Files with errors have no code block, just an error message.
	// The parser should still handle them gracefully (empty content).
	content := "# Harvx Context: MyProject\n\n## Files\n\n" +
		"### `broken/file.bin`\n\n" +
		"> **Size:** 10.0 KB | **Tokens:** 0 | **Tier:** low | **Compressed:** no\n\n" +
		"**Error:** binary file detected\n\n" +
		"### `src/main.go`\n\n" +
		"> **Size:** 1.0 KB | **Tokens:** 100 | **Tier:** critical | **Compressed:** no\n\n" +
		"```go\npackage main\n```\n"

	result, err := ParseMarkdownOutput(content)
	require.NoError(t, err)
	assert.Len(t, result.Files, 2)
	assert.Equal(t, "broken/file.bin", result.Files[0].Path)
	assert.Equal(t, "", result.Files[0].Content)
	assert.Equal(t, "src/main.go", result.Files[1].Path)
	assert.Equal(t, "package main", result.Files[1].Content)
}

func TestParseXMLOutput_FileWithErrorElement(t *testing.T) {
	t.Parallel()

	content := `<?xml version="1.0" encoding="UTF-8"?>
<repository>
  <files>
    <file path="broken/file.bin" tokens="0" tier="low" size="10240" language="" compressed="false">
      <error>binary file detected</error>
    </file>
    <file path="src/main.go" tokens="100" tier="critical" size="1024" language="go" compressed="false">
      <content><![CDATA[package main]]></content>
    </file>
  </files>
</repository>`

	result, err := ParseXMLOutput(content)
	require.NoError(t, err)
	assert.Len(t, result.Files, 2)
	assert.Equal(t, "broken/file.bin", result.Files[0].Path)
	assert.Equal(t, "", result.Files[0].Content)
	assert.Equal(t, "src/main.go", result.Files[1].Path)
	assert.Equal(t, "package main", result.Files[1].Content)
}

func TestParseIntAttr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		attrs    string
		attrName string
		want     int
	}{
		{
			name:     "valid integer",
			attrs:    `tokens="1200" tier="critical"`,
			attrName: "tokens",
			want:     1200,
		},
		{
			name:     "missing attribute",
			attrs:    `tier="critical"`,
			attrName: "tokens",
			want:     0,
		},
		{
			name:     "non-integer value",
			attrs:    `tokens="abc"`,
			attrName: "tokens",
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseIntAttr(tt.attrs, tt.attrName)
			assert.Equal(t, tt.want, got)
		})
	}
}
