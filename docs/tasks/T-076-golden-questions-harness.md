# T-076: Golden Questions Harness and Quality Evaluation Framework

**Priority:** Should Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-070 (Brief Command), T-071 (Review Slice Command)
**Phase:** 5 - Workflows

---

## Description

Ship documentation, a template, and tooling for maintaining a "golden questions" evaluation harness -- a set of repo-specific questions with known answers (e.g., "Where is JWT validated?", "What is the retry default?") that can be used to compare LLM accuracy when given diff-only context versus diff + Harvx context. This provides a practical, repeatable way to measure whether Harvx is actually improving AI review quality for a specific project.

## User Story

As a team lead integrating Harvx into our review pipeline, I want a structured way to measure whether Harvx context actually improves AI review accuracy so that I can justify the integration to my team with data, not anecdotes.

## Acceptance Criteria

- [ ] `docs/guides/golden-questions.md` documents the evaluation methodology:
  - What golden questions are and why they matter
  - How to write good golden questions (specific, verifiable, architecture-dependent)
  - How to run evaluations (diff-only vs diff + Harvx context)
  - How to interpret results and track improvements over time
- [ ] `docs/templates/golden-questions.toml` provides a starter template:
  ```toml
  # Golden Questions Harness for [Project Name]
  # Each question tests whether the LLM can answer correctly
  # given the context provided by Harvx.
  
  [[questions]]
  id = "auth-jwt"
  question = "Where is JWT token validation performed?"
  expected_answer = "middleware.ts, verifyToken function"
  category = "architecture"
  critical_files = ["middleware.ts", "lib/auth/jwt.ts"]
  
  [[questions]]
  id = "retry-default"  
  question = "What is the default retry count for API calls?"
  expected_answer = "3 retries with exponential backoff"
  category = "configuration"
  critical_files = ["lib/config/defaults.ts"]
  ```
- [ ] `harvx quality` command (alias: `harvx qa`) is registered as a Cobra subcommand that:
  - Reads golden questions from `.harvx/golden-questions.toml` or `--questions <path>`
  - For each question, verifies that `critical_files` are included in Harvx output (via assert-include logic)
  - Reports coverage: how many golden questions have their critical files included
  - Outputs results in human-readable and `--json` format
- [ ] `harvx quality init` generates a starter golden questions file with example questions
- [ ] Results can be stored as CI artifacts for tracking over time (JSON output format)
- [ ] Coverage reporting: what percentage of golden question critical files are captured by the current profile
- [ ] Documentation includes example GitHub Actions workflow for automated quality tracking

## Technical Notes

- The golden questions harness is primarily a documentation and tooling effort -- the actual LLM evaluation is done externally
- The `harvx quality` command focuses on the coverage dimension: are the right files included?
- For the actual "ask LLM and compare answers" step, provide a shell script template that:
  1. Generates context with Harvx (`harvx brief + review-slice`)
  2. Sends golden questions to an LLM API with the context
  3. Compares responses to expected answers
  4. Logs results
- Golden questions TOML uses `[[questions]]` array of tables (standard TOML syntax, supported by BurntSushi/toml)
- Categories help organize questions: `architecture`, `configuration`, `security`, `conventions`, `integration`
- The `critical_files` field enables automated coverage checking without LLM calls
- Store evaluation results in `.harvx/quality/` directory (gitignored by default)
- Reference: PRD Section 5.11.4 (Golden questions harness)

## Files to Create/Modify

- `docs/guides/golden-questions.md` - Evaluation methodology documentation
- `docs/templates/golden-questions.toml` - Starter template with example questions
- `docs/templates/evaluate.sh` - Shell script template for LLM evaluation
- `internal/workflows/quality.go` - Golden questions loading and coverage analysis
- `internal/workflows/quality_test.go` - Unit tests
- `internal/cli/quality.go` - Cobra command registration (`quality`, `quality init`)
- `internal/cli/quality_test.go` - CLI tests
- `internal/config/quality.go` - Golden questions TOML types

## Testing Requirements

- Unit test: Golden questions TOML parses correctly with all fields
- Unit test: Coverage analysis correctly identifies included/missing critical files
- Unit test: `quality init` generates valid TOML with example questions
- Unit test: Coverage report includes per-question and aggregate stats
- Unit test: `--json` output contains structured results
- Unit test: Missing golden questions file produces helpful error
- Unit test: Questions with no critical_files are reported but not failures
- Edge case: Empty golden questions file produces meaningful output
- Edge case: Critical file patterns use glob syntax (e.g., `lib/auth/**`)