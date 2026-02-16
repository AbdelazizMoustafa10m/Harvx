# T-075: Verify Command and Faithfulness Checking (`harvx verify`)

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-066 (Pipeline Library API), T-070 (Brief Command)
**Phase:** 5 - Workflows

---

## Description

Implement the `harvx verify [--sample <n>]` command that compares packed output to original source files for a sample of N files (or specific paths), reporting any differences beyond expected compression and redaction transformations. This provides a practical, automatable way to verify that Harvx output is faithful to the source -- that compression has not distorted meaning and redaction has not removed non-sensitive content.

## User Story

As a developer integrating Harvx into my review pipeline, I want to verify that the packed output faithfully represents my source code so that I can trust AI agents are reviewing accurate code, not artifacts of incorrect compression or over-aggressive redaction.

## Acceptance Criteria

- [ ] `harvx verify` command is registered as a Cobra subcommand
- [ ] Without arguments, verifies the most recently generated output file (from default or profile output path)
- [ ] `--sample <n>` flag randomly samples N files from the output for verification (default: 10)
- [ ] `--path <file>` flag verifies specific files (repeatable)
- [ ] For each verified file, the command:
  1. Reads the original source file from disk
  2. Reads the packed version from the output
  3. Applies expected transformations (compression/redaction) to the original
  4. Compares the result to the packed version
  5. Reports: MATCH, REDACTION_DIFF (expected), COMPRESSION_DIFF (expected), UNEXPECTED_DIFF (problem)
- [ ] Verification report output:
  ```
  Verifying harvx-output.md (10 sampled files)

  [PASS] src/auth/middleware.ts           - Match
  [PASS] prisma/schema.prisma             - Match
  [PASS] lib/services/transaction.ts      - Match (2 redactions applied)
  [PASS] lib/config/defaults.ts           - Match (compressed: signatures only)
  [WARN] lib/utils/helpers.ts             - Unexpected difference at line 42
  
  Result: 9/10 passed, 1 warning
  ```
- [ ] Exit code 0 if all files pass, exit code 2 if any have unexpected differences
- [ ] `--json` flag outputs verification results as structured JSON
- [ ] Supports `--profile <name>` to use the correct output path and settings
- [ ] Budget reporting is included in verification output: tokenizer used, total tokens, budget utilization %, truncated/omitted files

## Technical Notes

- Implement in `internal/workflows/verify.go` and `internal/cli/verify.go`
- Verification algorithm:
  1. Parse the output file to extract individual file blocks (path + content)
  2. For each selected file, read the original from disk
  3. If compression was enabled: apply the same compression to the original and compare
  4. If redaction was enabled: apply redaction to the original and compare
  5. Diff the expected output against the actual packed output
  6. Any remaining differences are flagged as UNEXPECTED_DIFF
- Output file parsing: need to handle both Markdown and XML format output files
  - Markdown: file blocks are delimited by `## File:` or `### File:` headers with path
  - XML: file blocks are `<file path="...">` elements
- Random sampling uses `math/rand` with a seed derived from the content hash (reproducible sampling)
- For compressed files, use the same tree-sitter compression pipeline to generate the expected compressed output
- UNEXPECTED_DIFF should show a unified diff snippet (first 10 lines) to help debug
- Budget reporting reads from the output file's header metadata or the `.meta.json` sidecar if available
- Reference: PRD Section 5.11.4 (Faithfulness verification, Budget reporting)

## Files to Create/Modify

- `internal/workflows/verify.go` - Verification logic
- `internal/workflows/verify_test.go` - Unit tests
- `internal/workflows/output_parser.go` - Parse Harvx output files to extract file blocks
- `internal/workflows/output_parser_test.go` - Output parser tests
- `internal/cli/verify.go` - Cobra command registration
- `internal/cli/verify_test.go` - CLI integration tests
- `testdata/expected-output/verify-pass.md` - Test fixture with matching output
- `testdata/expected-output/verify-fail.md` - Test fixture with mismatching output

## Testing Requirements

- Unit test: Verification passes for output that exactly matches source
- Unit test: Verification passes for output with expected redactions
- Unit test: Verification passes for output with expected compression
- Unit test: Verification detects unexpected differences
- Unit test: `--sample 5` verifies exactly 5 files
- Unit test: `--path` verifies specific named files
- Unit test: Sampling is reproducible (same content hash -> same sample set)
- Unit test: Output file parsing extracts correct file blocks (Markdown format)
- Unit test: Output file parsing extracts correct file blocks (XML format)
- Unit test: `--json` returns structured verification results
- Unit test: Budget reporting includes tokenizer, total tokens, utilization %
- Edge case: File on disk has changed since output was generated (report as changed, not failure)
- Edge case: Output file not found returns clear error with path suggestion
- Edge case: Sample size larger than file count verifies all files