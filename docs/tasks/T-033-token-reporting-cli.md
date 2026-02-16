# T-033: Token Reporting CLI Flags and Heatmap

**Priority:** Should Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-029, T-030, T-031, T-032
**Phase:** 2 - Intelligence (Relevance & Tokens)

---

## Description

Implement the CLI-facing token reporting features: `--token-count` mode (report token counts without generating output), `--top-files <n>` flag (show N largest files by token count), and `--heatmap` flag on the preview command (show files sorted by token density -- tokens per line of code -- to identify context-bloat files). Also wire the `--tokenizer`, `--max-tokens`, and `--truncation-strategy` flags into the CLI framework and connect them to the tokenizer and budget modules from T-029/T-030/T-031.

## User Story

As a developer, I want to quickly see which files consume the most tokens and identify bloated files so that I can optimize my context configuration -- without having to generate the full output.

## Acceptance Criteria

- [ ] `--tokenizer <encoding>` flag accepts `cl100k_base`, `o200k_base`, `none` (default: `cl100k_base`)
- [ ] `--max-tokens <n>` flag sets token budget (also readable from profile `max_tokens`)
- [ ] `--truncation-strategy <strategy>` flag accepts `truncate` or `skip` (default: `skip`)
- [ ] `--token-count` flag: runs the full pipeline (discovery, relevance, content loading, token counting) but only outputs a token report to stderr, no context file generated
  ```
  Token Report (cl100k_base)
  ──────────────────────────
  Total files:    342
  Total tokens:   89,420
  Budget:         200,000 (45% used)
  
  By Tier:
    Tier 0 (Config):      5 files     2,100 tokens
    Tier 1 (Source):     48 files    45,000 tokens
    Tier 2 (Secondary): 180 files   35,000 tokens
    Tier 3 (Tests):      62 files    5,000 tokens
    Tier 4 (Docs):       30 files    1,500 tokens
    Tier 5 (CI/Lock):    17 files      820 tokens
  ```
- [ ] `--top-files <n>` flag (default n=10): shows the N largest files by token count
  ```
  Top 10 Files by Token Count:
  ─────────────────────────────
   1. prisma/schema.prisma          4,210 tokens  (Tier 0)
   2. lib/services/transaction.ts   3,890 tokens  (Tier 1)
   3. app/api/transactions/route.ts 2,340 tokens  (Tier 1)
   ...
  ```
- [ ] `--heatmap` flag on `harvx preview`: shows files sorted by token density (tokens per line)
  ```
  Token Heatmap (tokens per line):
  ────────────────────────────────
   1. data/fixtures.json           142.3 tok/line  (780 lines, 111,000 tokens)
   2. prisma/schema.prisma          12.1 tok/line  (348 lines,   4,210 tokens)
   3. package-lock.json              8.4 tok/line  (12K lines, 101,000 tokens)
  ```
  This helps identify files that are unusually token-dense (likely generated, minified, or data files)
- [ ] Per-file token counts are stored on `FileDescriptor.TokenCount` and available to the output renderer
- [ ] Tokenizer name is included in output metadata header (e.g., `Tokenizer: cl100k_base`)
- [ ] All output goes to stderr when in report-only mode (stdout remains clean for piping)
- [ ] Unit tests for report formatting; integration tests for flag wiring

## Technical Notes

- Modify `internal/cli/root.go` or `internal/cli/generate.go` to add the new flags
- Create `internal/cli/token_report.go` for the `--token-count` reporting logic
- Create `internal/cli/preview.go` additions for `--heatmap`
- The heatmap requires counting lines in each file: `lineCount = strings.Count(content, "\n") + 1`
- Token density = `float64(tokenCount) / float64(lineCount)`
- Use `charmbracelet/lipgloss` for styled terminal output (table formatting, colors for high-density items)
- Colors for heatmap: green (< 5 tok/line), yellow (5-20 tok/line), red (> 20 tok/line)
- The `--top-files` and `--heatmap` flags can be combined with normal generation (they appear as additional output on stderr)

### Flag Wiring

```go
// In cobra command setup
cmd.Flags().StringVar(&cfg.Tokenizer, "tokenizer", "cl100k_base", "Tokenizer encoding: cl100k_base, o200k_base, none")
cmd.Flags().IntVar(&cfg.MaxTokens, "max-tokens", 0, "Token budget (0 = unlimited)")
cmd.Flags().StringVar(&cfg.TruncationStrategy, "truncation-strategy", "skip", "Budget overflow: truncate or skip")
cmd.Flags().BoolVar(&cfg.TokenCountOnly, "token-count", false, "Report token counts only")
cmd.Flags().IntVar(&cfg.TopFiles, "top-files", 0, "Show N largest files by token count")
cmd.Flags().BoolVar(&cfg.Heatmap, "heatmap", false, "Show token density heatmap (preview only)")
```

### Dependencies & Versions

| Package/Library | Version | Purpose |
|-----------------|---------|---------|
| github.com/spf13/cobra | v1.8+ | CLI framework for flag registration |
| github.com/charmbracelet/lipgloss | v1.0+ | Terminal styling for reports |

## Files to Create/Modify

- `internal/cli/root.go` (or `generate.go`) - Add flag registration for --tokenizer, --max-tokens, --truncation-strategy
- `internal/cli/token_report.go` - Token report formatting and --token-count mode implementation
- `internal/cli/preview.go` - Add --heatmap flag and heatmap rendering
- `internal/tokenizer/report.go` - TokenReport, TopFilesReport, HeatmapReport data structures and formatters
- `internal/tokenizer/report_test.go` - Unit tests for report formatting

## Testing Requirements

- Unit test: TokenReport formatting with sample data produces expected output format
- Unit test: TopFilesReport sorts files correctly by token count descending
- Unit test: HeatmapReport calculates density correctly (tokens / lines)
- Unit test: HeatmapReport sorts by density descending
- Unit test: HeatmapReport handles files with 0 lines (guard against division by zero)
- Unit test: Flag parsing for --tokenizer with all valid values
- Unit test: Flag parsing for --truncation-strategy with valid and invalid values
- Unit test: --token-count mode skips output generation
- Unit test: --top-files 5 shows exactly 5 files (or fewer if total < 5)
- Golden test: fixed set of files -> verify exact report output
- Integration test: `harvx --token-count` runs successfully on a sample repo
- Integration test: `harvx preview --heatmap` runs successfully on a sample repo