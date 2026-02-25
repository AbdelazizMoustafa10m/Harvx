# Review Pipeline Guide

Set up an end-to-end review pipeline combining `harvx brief` and `harvx review-slice` for automated code review with LLM agents.

## Pipeline Overview

```
┌──────────┐    ┌───────────────┐    ┌──────────────┐    ┌───────────┐
│ git diff │───▶│ review-slice  │───▶│ brief + slice│───▶│ LLM agent │
│ (refs)   │    │ (changed code)│    │ (context)    │    │ (review)  │
└──────────┘    └───────────────┘    └──────────────┘    └───────────┘
```

The pipeline produces two context artifacts:
1. **Repo Brief** — stable project-wide invariants (README, conventions, module map)
2. **Review Slice** — changed files plus bounded neighborhood (tests, importers)

Together they give an LLM reviewer both the "what changed" and "how it fits."

## Shell Script Example

```bash
#!/usr/bin/env bash
set -euo pipefail

# Configuration
BASE_REF="${BASE_REF:-origin/main}"
HEAD_REF="${HEAD_REF:-HEAD}"
TARGET="${TARGET:-claude}"
OUTPUT_DIR="${OUTPUT_DIR:-.harvx/review}"

mkdir -p "$OUTPUT_DIR"

# Step 1: Generate repo brief for project context
echo "Generating repo brief..."
harvx brief \
  --target "$TARGET" \
  --output "$OUTPUT_DIR/brief.md" \
  --quiet

# Step 2: Generate review slice with changed files and neighbors
echo "Generating review slice..."
harvx review-slice \
  --base "$BASE_REF" \
  --head "$HEAD_REF" \
  --target "$TARGET" \
  --output "$OUTPUT_DIR/review-slice.md" \
  --quiet

# Step 3: Combine for a complete review context
cat "$OUTPUT_DIR/brief.md" "$OUTPUT_DIR/review-slice.md" > "$OUTPUT_DIR/review-context.md"

echo "Review context written to $OUTPUT_DIR/review-context.md"
echo "  Brief: $(wc -l < "$OUTPUT_DIR/brief.md") lines"
echo "  Slice: $(wc -l < "$OUTPUT_DIR/review-slice.md") lines"
```

Save this as `scripts/harvx-review.sh` and run:

```bash
chmod +x scripts/harvx-review.sh
./scripts/harvx-review.sh
```

## CI Integration: GitHub Actions

### Basic Workflow

```yaml
# .github/workflows/harvx-review.yml
name: Harvx Code Review Context
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review-context:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history needed for diff

      - name: Install Harvx
        run: go install github.com/your-org/harvx/cmd/harvx@latest

      - name: Generate review context
        env:
          HARVX_TARGET: claude
          HARVX_QUIET: "1"
        run: |
          harvx brief --stdout > brief.md
          harvx review-slice \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --stdout > review-slice.md

      - name: Upload review artifacts
        uses: actions/upload-artifact@v4
        with:
          name: review-context
          path: |
            brief.md
            review-slice.md
```

### With Assert-Include Safety

```yaml
      - name: Generate review context with coverage checks
        run: |
          harvx review-slice \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --assert-include "**/*_test.go" \
            --target claude \
            --stdout > review-slice.md
```

The `--assert-include` flag ensures the review slice contains test files for the changed code. If no test files match the pattern, the command exits with a non-zero status.

### Redaction Enforcement in CI

```yaml
      - name: Generate context with secret scanning
        env:
          HARVX_FAIL_ON_REDACTION: "1"
        run: |
          harvx review-slice \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --redaction-report redaction-report.json \
            --stdout > review-slice.md
```

Setting `HARVX_FAIL_ON_REDACTION=1` causes the pipeline to fail if any secrets are detected in the output, preventing accidental secret exposure.

## Environment Variables for CI

| Variable | Purpose | Example |
|----------|---------|---------|
| `HARVX_TARGET` | Set LLM target format | `claude` |
| `HARVX_QUIET` | Suppress progress output | `1` |
| `HARVX_STDOUT` | Force stdout mode | `true` |
| `HARVX_NO_REDACT` | Disable redaction (trusted CI only) | `1` |
| `HARVX_FAIL_ON_REDACTION` | Fail on secrets found | `1` |
| `HARVX_COMPRESS` | Enable compression | `true` |
| `HARVX_TOKENIZER` | Override tokenizer | `cl100k_base` |
| `HARVX_MAX_TOKENS` | Set token budget | `50000` |
| `HARVX_LOG_FORMAT` | Structured logs for CI | `json` |
| `HARVX_PROFILE` | Use a named profile | `ci` |

## `review-slice` Command Reference

```bash
harvx review-slice --base <ref> --head <ref> [flags]
```

**Required flags:**

| Flag | Description | Example |
|------|-------------|---------|
| `--base` | Base git ref | `origin/main`, `abc1234` |
| `--head` | Head git ref | `HEAD`, `feature-branch` |

**Optional flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output machine-readable JSON metadata |
| `--target` | `generic` | LLM target: `claude`, `chatgpt`, `generic` |
| `--output` | `harvx-review-slice.md` | Output file path |
| `--stdout` | `false` | Output to stdout |
| `--max-tokens` | `20000` | Token budget |
| `--profile` | `default` | Named profile |
| `--assert-include` | — | Assert glob patterns (repeatable) |

**Neighborhood discovery:** The review slice automatically includes related files:
- Test files for changed code
- Files that import the changed modules
- Same-directory files for unsupported languages

**Token budget:** Changed files are always included first. Neighbors fill remaining budget. Configure via `slice_max_tokens` in profile or `--max-tokens` flag.

## JSON Output for Automation

Use `--json` to get machine-readable metadata for scripting:

```bash
harvx review-slice --base origin/main --head HEAD --json
```

```json
{
  "token_count": 8500,
  "content_hash": "a1b2c3d4e5f6a7b8",
  "changed_files": ["internal/auth/auth.go", "internal/auth/handler.go"],
  "neighbor_files": ["internal/auth/auth_test.go"],
  "deleted_files": [],
  "total_files": 3,
  "max_tokens": 20000,
  "base_ref": "origin/main",
  "head_ref": "HEAD"
}
```

Use this to build conditional pipelines:

```bash
# Only run review if there are changed files
REVIEW_JSON=$(harvx review-slice --base origin/main --head HEAD --json)
TOTAL=$(echo "$REVIEW_JSON" | jq '.total_files')

if [ "$TOTAL" -gt 0 ]; then
  harvx review-slice --base origin/main --head HEAD --target claude --stdout
fi
```
