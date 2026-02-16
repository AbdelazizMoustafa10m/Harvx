# T-056: Output Splitter (Multi-Part File Generation)

**Priority:** Should Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-052, T-055
**Phase:** 4 - Output & Rendering

---

## Description

Implement the `--split <tokens-per-part>` feature that splits the output into multiple numbered part files (e.g., `harvx-output.part-001.md`, `harvx-output.part-002.md`). The splitter respects tier boundaries -- no individual file is ever split across parts. Files are grouped by top-level directory when possible to maintain context coherence. Each part gets its own header with part number, total parts, and a content hash. The split feature is essential for working with LLMs that have smaller context windows or for breaking up very large codebases.

## User Story

As a developer working with a token-limited LLM, I want to split the output into multiple parts that each fit within the context window so that I can feed my codebase to the LLM in manageable chunks.

## Acceptance Criteria

- [ ] `--split <tokens-per-part>` flag triggers multi-part output generation
- [ ] Output files are named: `<base>.part-001.<ext>`, `<base>.part-002.<ext>`, etc.
- [ ] Each part contains: a header with part number (`Part 1 of N`), total parts, part-specific content hash, and the global content hash
- [ ] No single file is ever split across parts -- files are atomic units
- [ ] Files are assigned to parts greedily: add files in tier/path order until adding the next file would exceed the per-part token budget
- [ ] Tier boundaries are respected: all files from a tier go into the same part when possible; if a tier is too large, it can span parts but individual files remain intact
- [ ] The directory tree is included in Part 1 only (with a note that it covers the entire project)
- [ ] File summary section is included in Part 1 only
- [ ] Each part after Part 1 includes a minimal header referencing Part 1 for full summary
- [ ] `Splitter` in `internal/output/splitter.go` takes the full `RenderData` and returns `[]PartData` (one per part)
- [ ] Works with both Markdown and XML formats
- [ ] Returns metadata about parts: `[]PartResult{PartNumber int, Path string, TokenCount int, FileCount int, Hash uint64}`
- [ ] Unit tests achieve >= 85% coverage

## Technical Notes

- **Algorithm**: 
  1. Calculate the header/summary overhead in tokens (this comes out of each part's budget).
  2. Iterate through files in tier-sorted order. Accumulate token counts.
  3. When adding the next file would exceed `tokensPerPart - overhead`, start a new part.
  4. If a single file exceeds `tokensPerPart`, it gets its own part (with a warning logged).
- **Token counting**: The splitter needs token counts per file (already available in `FileDescriptor.TokenCount`). It also needs to estimate the overhead tokens for headers/separators. Use a rough estimate (e.g., 200 tokens per file for header/fence overhead).
- **Naming**: Parse the base output path, insert `.part-NNN` before the extension. Use zero-padded 3-digit part numbers.
- **Context coherence**: Within a tier, try to keep files from the same top-level directory together. This is a soft preference, not a hard constraint.
- **Renderer integration**: Each part is rendered independently using the same renderer. The `OutputWriter` from T-055 calls the splitter when `--split` is provided, then renders each part sequentially.
- Reference: PRD Section 5.7 (--split, respects tier boundaries)

## Files to Create/Modify

- `internal/output/splitter.go` - `Splitter`, `PartData`, `SplitOpts`, `Split` method
- `internal/output/splitter_test.go` - Unit tests
- `internal/output/writer.go` - Modify `Write` to handle split mode (calls splitter, renders each part)

## Testing Requirements

- Unit test: single-part output when total tokens < split threshold (no splitting occurs)
- Unit test: two files totaling 150K tokens split at 100K produces 2 parts
- Unit test: file ordering is preserved across parts (tier order, then alphabetical)
- Unit test: no file is split across parts
- Unit test: part file names follow `<base>.part-NNN.<ext>` pattern
- Unit test: Part 1 includes directory tree and file summary; Part 2+ does not
- Unit test: each part header includes correct part number and total
- Unit test: single oversized file gets its own part (with warning)
- Unit test: empty file list produces no parts (or one empty part)
- Unit test: works with both `.md` and `.xml` extensions
- Edge case: all files fit in one part -- no `.part-001` suffix, just the regular output
- Edge case: split threshold of 0 or negative is rejected with error
- Edge case: 100+ parts produces correctly zero-padded names