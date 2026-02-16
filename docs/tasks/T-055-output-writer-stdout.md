# T-055: Output Writer, File Path Resolution, and Stdout Support

**Priority:** Must Have
**Effort:** Medium (6-8hrs)
**Dependencies:** T-052, T-054
**Phase:** 4 - Output & Rendering

---

## Description

Implement the output writing orchestration layer that ties together the renderer, content hasher, and output destination. This module handles: resolving the output file path (from `--output`/`-o` flag, profile config, or default), writing to a file or stdout (`--stdout`), coordinating the streaming write pipeline, and ensuring atomic file writes (write to temp file, then rename). It also manages the `--target` presets that set format and default budget.

## User Story

As a developer, I want to control where the output goes -- to a named file, the default location, or stdout for piping -- so that harvx integrates into my workflow regardless of how I consume the output.

## Acceptance Criteria

- [ ] `OutputWriter` in `internal/output/writer.go` orchestrates rendering to the final destination
- [ ] `Write(ctx context.Context, data *RenderData, opts OutputOpts) error` is the primary method
- [ ] `OutputOpts` includes: `OutputPath string`, `UseStdout bool`, `Format string` (markdown/xml), `Target string` (claude/chatgpt/generic)
- [ ] Default output file: `harvx-output.md` in the current directory (or `.xml` for XML format)
- [ ] `--output` / `-o` flag overrides the default path
- [ ] Profile-level `output` setting is respected (lower priority than CLI flag)
- [ ] `--stdout` writes to `os.Stdout`; no file is created
- [ ] When writing to file: write to a temporary file in the same directory first, then `os.Rename` for atomicity
- [ ] **LLM target presets**:
  - `claude`: sets format to `xml`, default max_tokens to 200,000
  - `chatgpt`: sets format to `markdown`, default max_tokens to 128,000
  - `generic`: sets format to `markdown`, no default max_tokens
- [ ] Target preset values are defaults that can be overridden by explicit `--format` or `--max-tokens` flags
- [ ] Format dispatch: selects `MarkdownRenderer` or `XMLRenderer` based on resolved format
- [ ] Content hash is computed during or before rendering and included in the output
- [ ] File extension matches format: `.md` for markdown, `.xml` for xml
- [ ] If output path has no extension, the correct one is appended
- [ ] Returns structured result: `OutputResult{Path string, Hash uint64, TotalTokens int, BytesWritten int64}`
- [ ] Unit tests achieve >= 85% coverage

## Technical Notes

- **Atomic writes**: Use `os.CreateTemp` in the same directory as the output, write content, `f.Sync()`, `f.Close()`, then `os.Rename(temp, final)`. This prevents partial output files if the process is interrupted.
- **Streaming to file**: The renderer writes directly to the file's `io.Writer`. For hash computation during streaming, use `io.MultiWriter(file, hasher)` to compute the hash as bytes are written.
- **Stdout mode**: When `--stdout`, the renderer writes to `os.Stdout`. Content hash is still computed (via `io.MultiWriter(os.Stdout, hasher)`) and can be logged to stderr for pipeline use.
- **Target presets**: These are resolved at config time (in `internal/config/`), not in the output writer. The writer receives already-resolved values. However, the writer should validate that format and renderer are consistent.
- **Format dispatch**: Use a factory pattern or simple switch to create the appropriate renderer.
- Reference: PRD Section 5.7 (output file path, stdout, target presets)

## Files to Create/Modify

- `internal/output/writer.go` - `OutputWriter`, `OutputOpts`, `OutputResult`, `Write` method
- `internal/output/writer_test.go` - Unit tests
- `internal/output/format.go` - Format constants, renderer factory/dispatch
- `internal/output/format_test.go` - Format dispatch tests

## Testing Requirements

- Unit test: default output path is `harvx-output.md` for markdown format
- Unit test: default output path is `harvx-output.xml` for xml format
- Unit test: `--output` flag overrides default path
- Unit test: profile output setting is used when no CLI flag
- Unit test: CLI flag takes precedence over profile setting
- Unit test: stdout mode writes to provided writer (not to a file)
- Unit test: atomic write creates temp file and renames (verify no partial files)
- Unit test: content hash is computed and returned in result
- Unit test: target `claude` resolves to XML format
- Unit test: target `chatgpt` resolves to markdown format
- Unit test: explicit `--format markdown` overrides target `claude` format
- Unit test: file extension is appended when missing from output path
- Unit test: format dispatch returns correct renderer type
- Edge case: output directory does not exist (return clear error)
- Edge case: output path is read-only (return permission error)
- Edge case: very large output (> 100MB) streams without excessive memory