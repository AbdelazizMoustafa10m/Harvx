# T-053: XML Output Renderer for Claude Target Preset

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-052
**Phase:** 4 - Output & Rendering

---

## Description

Implement the XML output renderer that produces context documents optimized for Claude, following Anthropic's XML tag best practices and inspired by Repomix's proven XML structure. This renderer is activated by `--target claude` or `--format xml` and generates well-structured XML with semantic tags like `<repository>`, `<file_summary>`, `<directory_structure>`, `<files>`, and `<file>` elements. The XML format helps Claude parse the codebase context more accurately, leading to higher-quality responses.

## User Story

As a developer using Claude for code review, I want the output formatted in Claude-optimized XML so that Claude can more accurately parse and reason about my codebase structure.

## Acceptance Criteria

- [ ] `XMLRenderer` struct implements the `Renderer` interface from T-052
- [ ] XML output follows this structure:
  ```xml
  <?xml version="1.0" encoding="UTF-8"?>
  <repository>
    <metadata>...</metadata>
    <file_summary>...</file_summary>
    <directory_structure>...</directory_structure>
    <files>
      <file path="..." tokens="..." tier="..." size="...">
        <content>...</content>
      </file>
    </files>
    <statistics>...</statistics>
  </repository>
  ```
- [ ] `<metadata>` includes: project name, generation timestamp, content hash, profile, tokenizer, total tokens
- [ ] `<file_summary>` includes: total files, total tokens, per-tier counts, top N files, redaction summary
- [ ] `<directory_structure>` contains the same tree visualization as Markdown (plain text inside CDATA)
- [ ] `<files>` contains one `<file>` element per included file, with `path`, `tokens`, `tier`, `size` attributes
- [ ] File content is wrapped in CDATA sections to handle special XML characters: `<![CDATA[...]]>`
- [ ] Content with `]]>` sequences inside is properly split across CDATA sections
- [ ] `--line-numbers` support works in XML format (line numbers prepended to each line within CDATA)
- [ ] `--target claude` automatically sets format to XML and default budget to 200,000 tokens
- [ ] Output is valid, well-formed XML (parseable by any XML parser)
- [ ] Templates are defined as Go `const` strings (same pattern as Markdown renderer)
- [ ] Output is deterministic
- [ ] Unit tests achieve >= 85% coverage
- [ ] Golden test comparing rendered XML against expected output

## Technical Notes

- **Anthropic XML best practices** (ref: https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/use-xml-tags): Use meaningful tag names, nest tags for hierarchical content, maintain consistent naming.
- **Repomix XML structure** (ref: https://repomix.com/guide/output): Follow the proven structure with `<repository>` root, `<file_summary>`, `<directory_structure>`, `<files>`, `<statistics>`.
- **CDATA handling**: File content must be in CDATA sections to avoid XML escaping issues with `<`, `>`, `&` in source code. If content contains `]]>`, split it: `<![CDATA[content before]]]]><![CDATA[>content after]]>`.
- **Template approach**: Use `text/template` (not `encoding/xml`) for the XML structure since we need fine-grained control over CDATA sections and formatting. Define template functions for CDATA wrapping.
- **No dependency on encoding/xml Marshal**: The output is a document, not a serialized Go struct. Templates give better control.
- **Target presets**: The `--target` flag is handled at the CLI/config layer. The XML renderer receives `RenderData` and does not need to know about presets directly.
- Reference: PRD Section 5.7 (XML format, Claude target preset)

## Files to Create/Modify

- `internal/output/xml.go` - `XMLRenderer` implementation
- `internal/output/xml_test.go` - Unit tests
- `internal/output/templates.go` - Add XML template constants (append to existing file from T-052)
- `testdata/expected-output/xml-basic.xml` - Golden test expected output
- `testdata/expected-output/xml-cdata-edge.xml` - Golden test with CDATA edge cases

## Testing Requirements

- Unit test: output is valid well-formed XML (parse with `encoding/xml` decoder)
- Unit test: metadata section contains all required fields
- Unit test: file content is wrapped in CDATA sections
- Unit test: content containing `]]>` is correctly split across CDATA sections
- Unit test: content containing `<`, `>`, `&` inside CDATA renders without XML escaping
- Unit test: file attributes (path, tokens, tier, size) are correct
- Unit test: line numbers work within CDATA content
- Unit test: empty file list produces valid XML with zero counts
- Golden test: full render against expected XML output
- Edge case: file content that is valid XML itself (e.g., `.xml`, `.html` files)
- Edge case: file paths with special XML characters in names (e.g., `&` in dir name)
- Edge case: very large file content does not break CDATA sections