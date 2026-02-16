# T-052: Markdown Output Renderer with Go Templates

**Priority:** Must Have
**Effort:** Large (14-20hrs)
**Dependencies:** T-051
**Phase:** 4 - Output & Rendering

---

## Description

Implement the default Markdown output renderer that produces the complete harvx context document. This renderer uses Go's `text/template` with embedded template strings to generate a well-structured Markdown file containing: header block, file summary, directory tree, file contents (with optional line numbers), and change summary section. The renderer accepts the fully processed pipeline data (sorted `FileDescriptor` slice, config, tree string) and writes to an `io.Writer` for streaming output.

## User Story

As a developer, I want to run `harvx` and get a clean, well-organized Markdown file that LLMs can easily parse so that I get better responses when I share my codebase context.

## Acceptance Criteria

- [ ] `MarkdownRenderer` struct implements a `Renderer` interface: `Render(ctx context.Context, w io.Writer, data *RenderData) error`
- [ ] `Renderer` interface is defined in `internal/output/renderer.go` for format dispatch
- [ ] `RenderData` struct holds all data needed for rendering: project name, timestamp, content hash, profile name, tokenizer name, total tokens, files (with per-file metadata), tree string, diff summary, config details
- [ ] **Header block** includes: project name, generation timestamp (RFC 3339), content hash (hex), profile used, tokenizer, total token count
- [ ] **File summary section** includes: total files, total tokens, files by tier (count per tier 0-5), top N largest files by token count, redaction summary (count by type)
- [ ] **Directory tree section** includes the rendered tree from T-051
- [ ] **File contents section**: each file has a heading with path, size, token count, tier label, followed by a fenced code block with language identifier inferred from extension
- [ ] `--line-numbers` support: when enabled, each line inside code blocks is prefixed with its line number (e.g., `  1 | func main() {`)
- [ ] **Change summary section** (conditional): if diff data is present, shows added/modified/deleted file counts and paths
- [ ] Templates are defined as Go `const` strings in `internal/output/templates.go` (not external files)
- [ ] Template uses `template.FuncMap` for helper functions: `formatBytes`, `formatNumber`, `languageFromExt`, `addLineNumbers`, `repeatString`
- [ ] Output is deterministic: given the same input data, produces byte-identical output
- [ ] Streaming: template executes directly to the `io.Writer`, not buffering the entire output in memory
- [ ] Unit tests achieve >= 85% coverage
- [ ] Golden test comparing full rendered output against expected Markdown for a sample dataset

## Technical Notes

- **Go `text/template`**: Use `template.Must(template.New("markdown").Funcs(funcMap).Parse(tmpl))` pattern. Parse once at package init, execute per render call. See https://pkg.go.dev/text/template
- **Template structure**: Break the template into named sub-templates (header, summary, tree, files, file-entry, change-summary) and compose them via `{{template "name" .}}`.
- **Language detection**: Map file extensions to Markdown code fence language identifiers. E.g., `.go` -> `go`, `.ts` -> `typescript`, `.py` -> `python`, `.rs` -> `rust`, `.md` -> `markdown`. Maintain a map in a helper function.
- **Line numbers**: Implemented as a template function that takes content string, splits by newlines, prefixes each with right-aligned line number and ` | ` separator. Applied inside the file-entry template when `ShowLineNumbers` is true.
- **Content hash**: The hash is pre-computed and passed in `RenderData`; the renderer just formats it as hex.
- **Streaming**: `template.Execute(w, data)` writes directly to the `io.Writer`. For large repos, this avoids holding the entire output in a `bytes.Buffer`.
- **Deterministic output**: The renderer must not introduce any non-deterministic elements. Timestamps are passed in, not generated during rendering. File order is pre-sorted.
- Reference: PRD Section 5.7

## Files to Create/Modify

- `internal/output/renderer.go` - `Renderer` interface, `RenderData` struct, `FileRenderEntry` struct
- `internal/output/templates.go` - Markdown template constants and `template.FuncMap` helpers
- `internal/output/markdown.go` - `MarkdownRenderer` implementation
- `internal/output/markdown_test.go` - Unit tests
- `internal/output/helpers.go` - Shared helper functions (language detection, formatting)
- `internal/output/helpers_test.go` - Helper function tests
- `testdata/expected-output/markdown-basic.md` - Golden test expected output
- `testdata/expected-output/markdown-line-numbers.md` - Golden test with line numbers

## Testing Requirements

- Unit test: header block contains all required fields in correct format
- Unit test: file summary counts match input data
- Unit test: file contents use correct language identifier in code fences
- Unit test: line numbers are correctly prefixed when enabled
- Unit test: line numbers are absent when disabled
- Unit test: change summary appears only when diff data is provided
- Unit test: empty file list produces valid output with zero-file summary
- Unit test: `formatBytes` correctly formats B, KB, MB, GB
- Unit test: `languageFromExt` maps all common extensions correctly
- Golden test: full render against expected Markdown output
- Edge case: files with triple-backtick content are escaped properly
- Edge case: extremely long file paths do not break formatting
- Edge case: files with zero tokens are rendered correctly