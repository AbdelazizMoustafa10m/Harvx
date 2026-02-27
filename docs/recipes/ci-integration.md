# Recipe: CI Integration (Jordan Persona)

Set up Harvx in CI/CD pipelines for automated review context generation and quality gates.

## Use Case

You want CI to automatically generate review context for every PR, enforce secret scanning, and publish context artifacts for downstream tools or reviewers.

## Recipe 1: GitHub Actions — Basic Review Context

```yaml
# .github/workflows/harvx-review.yml
name: Harvx Review Context
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review-context:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install Harvx
        run: go install github.com/your-org/harvx/cmd/harvx@latest

      - name: Generate review context
        env:
          HARVX_TARGET: claude
          HARVX_QUIET: "1"
        run: |
          harvx brief --output brief.md
          harvx review-slice \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --output review-slice.md

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: review-context
          path: |
            brief.md
            review-slice.md
```

## Recipe 2: Secret Scanning Gate

```yaml
      - name: Generate context with secret scanning
        env:
          HARVX_FAIL_ON_REDACTION: "1"
          HARVX_LOG_FORMAT: json
        run: |
          harvx review-slice \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --redaction-report redaction-report.json \
            --output review-slice.md

      - name: Upload redaction report
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: redaction-report
          path: redaction-report.json
```

`HARVX_FAIL_ON_REDACTION=1` exits with code 1 if any secrets are found, failing the CI step.

## Recipe 3: Test Coverage Assertion

```yaml
      - name: Generate context with test coverage check
        run: |
          harvx review-slice \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --assert-include "**/*_test.go" \
            --assert-include "**/*.test.ts" \
            --output review-slice.md
```

`--assert-include` ensures that test files matching the patterns are included in the review slice. The command fails if no files match any pattern.

## Recipe 4: JSON Metadata for Downstream Tools

```yaml
      - name: Generate review metadata
        id: review
        run: |
          harvx review-slice \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.sha }} \
            --json > review-meta.json
          echo "files=$(jq '.total_files' review-meta.json)" >> "$GITHUB_OUTPUT"
          echo "tokens=$(jq '.token_count' review-meta.json)" >> "$GITHUB_OUTPUT"

      - name: Comment PR with review stats
        if: steps.review.outputs.files != '0'
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `Review context: ${{ steps.review.outputs.files }} files, ${{ steps.review.outputs.tokens }} tokens`
            })
```

## Recipe 5: CI Profile

Create a CI-specific profile in `.harvx.toml`:

```toml
[profiles.ci]
brief_max_tokens = 3000
slice_max_tokens = 30000
exclude = ["testdata/**", "vendor/**", "node_modules/**"]
```

Use it in CI:

```yaml
        env:
          HARVX_PROFILE: ci
```

## Environment Variable Reference

All `HARVX_*` variables work in CI without flags:

| Variable | Value | Purpose |
|----------|-------|---------|
| `HARVX_TARGET` | `claude` | XML output format |
| `HARVX_QUIET` | `1` | No progress output |
| `HARVX_PROFILE` | `ci` | CI-optimized profile |
| `HARVX_FAIL_ON_REDACTION` | `1` | Fail on secrets |
| `HARVX_LOG_FORMAT` | `json` | Structured CI logs |
| `HARVX_STDOUT` | `true` | Force stdout output |
| `HARVX_COMPRESS` | `true` | Enable compression |
| `HARVX_MAX_TOKENS` | `50000` | Token budget |
| `HARVX_TOKENIZER` | `cl100k_base` | Tokenizer engine |
| `HARVX_NO_REDACT` | `1` | Skip redaction (trusted envs) |

Boolean variables accept: `true`, `1`, `yes` (case-insensitive).

## Tips

- Always use `fetch-depth: 0` in checkout for full git history (needed for `review-slice`)
- Use `HARVX_LOG_FORMAT=json` for structured CI log output
- Combine `--json` metadata with GitHub Actions outputs for conditional workflows
- Store the CI profile in `.harvx.toml` (checked into repo) for reproducibility
