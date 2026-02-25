# Golden Questions Evaluation Guide

Measure whether Harvx context actually improves AI review accuracy using a structured, repeatable evaluation harness.

## What Are Golden Questions?

Golden questions are repo-specific questions with known correct answers. Each question targets knowledge that requires reading actual source code -- not just a diff -- to answer correctly.

```
Question:  "Where is JWT token validation performed?"
Expected:  "middleware.ts, verifyToken function"
Category:  architecture
```

When an LLM reviewer gets a question right with Harvx context but wrong with diff-only context, that is a measurable signal that Harvx is providing value. When it gets a question wrong in both cases, that reveals a coverage gap worth fixing.

Golden questions matter because they replace anecdotal "the reviews feel better" claims with data: X out of Y architecture questions answered correctly, trending up from last sprint.

## Writing Good Golden Questions

A useful golden question has three properties:

1. **Specific** -- It has one unambiguous correct answer, not a judgment call.
2. **Verifiable** -- You can check the answer against source code in under 30 seconds.
3. **Architecture-dependent** -- Answering it requires context beyond the changed files.

### Good questions

| Question | Why it works |
|----------|-------------|
| "What is the default retry count for API calls?" | Single value, lives in a config file |
| "Which middleware runs before every authenticated route?" | Requires understanding the request pipeline |
| "What database migration tool does this project use?" | Requires reading build config, not just code |

### Bad questions

| Question | Why it fails |
|----------|-------------|
| "Is the code well-structured?" | Subjective, no ground truth |
| "What does the main function do?" | Too obvious, any context is sufficient |
| "Explain the architecture" | Open-ended, hard to score |

### Writing tips

- Start with questions your team actually asks during reviews.
- Focus on the knowledge gaps that cause incorrect review feedback.
- Target files that are rarely in a diff but frequently needed for understanding.
- Aim for 15-30 questions per project. More than 50 becomes a maintenance burden.

## TOML Format

Golden questions live in `.harvx/golden-questions.toml` (or any path passed via `--questions`).

```toml
# Golden Questions Harness for MyProject
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

[[questions]]
id = "secret-rotation"
question = "How are database credentials rotated?"
expected_answer = "Vault agent sidecar, refreshed every 24h via config/vault.hcl"
category = "security"
critical_files = ["config/vault.hcl", "deploy/sidecar.yaml"]

[[questions]]
id = "error-convention"
question = "What is the standard error response format for API endpoints?"
expected_answer = "JSON object with code, message, and details fields, defined in lib/errors/response.ts"
category = "conventions"
critical_files = ["lib/errors/response.ts", "docs/api-conventions.md"]

[[questions]]
id = "event-bus"
question = "Which message broker does the notification service consume from?"
expected_answer = "RabbitMQ via amqplib, configured in services/notifications/consumer.ts"
category = "integration"
critical_files = ["services/notifications/consumer.ts", "docker-compose.yml"]
```

### Field reference

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier (kebab-case, used in reports) |
| `question` | Yes | The question to ask the LLM |
| `expected_answer` | Yes | The correct answer (used for manual or automated scoring) |
| `category` | Yes | One of: `architecture`, `configuration`, `security`, `conventions`, `integration` |
| `critical_files` | Yes | File paths (relative to repo root) that must be in context to answer correctly. Supports glob patterns like `lib/auth/**`. |

## Categories

Organize questions into categories to identify which knowledge dimensions are well-covered and which have gaps.

| Category | What it tests | Example |
|----------|--------------|---------|
| `architecture` | Structural decisions, control flow, module boundaries | "Which service handles payment webhooks?" |
| `configuration` | Defaults, feature flags, environment variables | "What is the rate limit for unauthenticated requests?" |
| `security` | Auth flows, secret management, access control | "How are API keys validated?" |
| `conventions` | Coding standards, naming, error handling patterns | "What logging library does the project use?" |
| `integration` | External services, APIs, message queues, databases | "Which S3 bucket stores user uploads?" |

## Checking Coverage with `harvx quality`

The `harvx quality` command (alias: `harvx qa`) checks whether your current Harvx configuration captures the files needed to answer your golden questions. It does not call an LLM -- it verifies file inclusion only.

### Basic usage

```bash
# Check coverage using default golden questions location
harvx quality

# Check coverage using a specific questions file
harvx quality --questions path/to/golden-questions.toml

# Machine-readable output for CI
harvx quality --json
```

### Sample output

```
Golden Questions Coverage Report
================================

Category        Covered  Total  Coverage
architecture    5        6      83%
configuration   3        3      100%
security        2        4      50%
conventions     4        4      100%
integration     1        2      50%

Overall: 15/19 (79%)

Uncovered questions:
  [auth-session]    "How are sessions invalidated on logout?"
    Missing: lib/auth/session.ts
  [vault-rotation]  "How are database credentials rotated?"
    Missing: config/vault.hcl
  [s3-uploads]      "Which S3 bucket stores user uploads?"
    Missing: services/uploads/s3.ts
  [cors-policy]     "What origins are allowed by the CORS policy?"
    Missing: config/cors.ts
```

### Initializing a starter file

```bash
# Generate .harvx/golden-questions.toml with example questions
harvx quality init
```

This creates a commented template you can customize for your project.

### JSON output

The `--json` flag produces structured output suitable for CI artifacts:

```json
{
  "total_questions": 19,
  "covered_questions": 15,
  "coverage_percent": 78.9,
  "categories": {
    "architecture": {"covered": 5, "total": 6},
    "configuration": {"covered": 3, "total": 3},
    "security": {"covered": 2, "total": 4},
    "conventions": {"covered": 4, "total": 4},
    "integration": {"covered": 1, "total": 2}
  },
  "uncovered": [
    {
      "id": "auth-session",
      "question": "How are sessions invalidated on logout?",
      "missing_files": ["lib/auth/session.ts"]
    }
  ]
}
```

## Running Evaluations

Coverage checks tell you whether the right files are included. To measure whether Harvx context actually improves LLM accuracy, run an A/B evaluation: diff-only versus diff + Harvx context.

### Shell script template

Save this as `scripts/evaluate-golden.sh` and adapt it to your LLM API:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Configuration
QUESTIONS_FILE="${QUESTIONS_FILE:-.harvx/golden-questions.toml}"
BASE_REF="${BASE_REF:-origin/main}"
HEAD_REF="${HEAD_REF:-HEAD}"
RESULTS_DIR="${RESULTS_DIR:-.harvx/quality}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

mkdir -p "$RESULTS_DIR"

# Step 1: Generate the diff-only context
echo "Generating diff-only context..."
git diff "$BASE_REF"..."$HEAD_REF" > "$RESULTS_DIR/diff-only.txt"

# Step 2: Generate Harvx context (brief + review slice)
echo "Generating Harvx context..."
harvx brief --stdout > "$RESULTS_DIR/brief.md"
harvx review-slice \
  --base "$BASE_REF" --head "$HEAD_REF" \
  --stdout > "$RESULTS_DIR/review-slice.md"
cat "$RESULTS_DIR/brief.md" "$RESULTS_DIR/review-slice.md" > "$RESULTS_DIR/harvx-context.md"

# Step 3: Extract questions from TOML (requires yq or a TOML parser)
# This example uses a simple grep approach; adapt to your tooling.
echo "Running evaluations..."

RESULTS_FILE="$RESULTS_DIR/eval-$TIMESTAMP.json"
echo '{"timestamp":"'"$TIMESTAMP"'","results":[' > "$RESULTS_FILE"

FIRST=true
while IFS='|' read -r ID QUESTION EXPECTED; do
  [ -z "$ID" ] && continue

  # --- Condition A: diff-only ---
  PROMPT_A="Given this diff, answer the question.\n\nDiff:\n$(cat "$RESULTS_DIR/diff-only.txt")\n\nQuestion: $QUESTION"
  # Replace this with your LLM API call:
  # ANSWER_A=$(call_llm "$PROMPT_A")
  ANSWER_A="<replace with LLM API call>"

  # --- Condition B: diff + Harvx context ---
  PROMPT_B="Given this context and diff, answer the question.\n\nContext:\n$(cat "$RESULTS_DIR/harvx-context.md")\n\nDiff:\n$(cat "$RESULTS_DIR/diff-only.txt")\n\nQuestion: $QUESTION"
  # ANSWER_B=$(call_llm "$PROMPT_B")
  ANSWER_B="<replace with LLM API call>"

  # Record results (scoring is manual or via a separate grading step)
  if [ "$FIRST" = true ]; then FIRST=false; else echo ',' >> "$RESULTS_FILE"; fi
  cat >> "$RESULTS_FILE" <<EOF
  {"id":"$ID","question":"$QUESTION","expected":"$EXPECTED","answer_diff_only":"$ANSWER_A","answer_harvx":"$ANSWER_B"}
EOF

done < <(grep -E '^(id|question|expected_answer)' "$QUESTIONS_FILE" \
  | paste - - - \
  | sed 's/id = "//;s/question = "//;s/expected_answer = "//;s/"//g' \
  | awk -F'\t' '{print $1"|"$2"|"$3}')

echo ']}' >> "$RESULTS_FILE"
echo "Results written to $RESULTS_FILE"
```

### Scoring

After collecting LLM responses, score each answer:

| Score | Meaning |
|-------|---------|
| **Correct** | Answer matches expected answer in substance |
| **Partial** | Answer is directionally right but missing key details |
| **Wrong** | Answer is incorrect or contradicts expected answer |
| **Abstain** | LLM says it cannot answer with the given context |

For automated scoring, use a grading LLM with a structured prompt:

```
Given the expected answer and the LLM's response, score as:
correct, partial, wrong, or abstain.

Expected: {expected_answer}
Response: {llm_response}

Score:
```

## Interpreting Results

### Key metrics

| Metric | Formula | Target |
|--------|---------|--------|
| **Accuracy lift** | (correct_harvx - correct_diff) / total | > 0 means Harvx is helping |
| **Abstain reduction** | (abstain_diff - abstain_harvx) / abstain_diff | Lower abstains = more confidence |
| **Coverage** | covered_questions / total_questions | > 80% for critical categories |

### What the numbers tell you

- **High accuracy lift (>20%):** Harvx is providing significant value. The context it includes is directly enabling better answers.
- **Low accuracy lift (<5%):** Either the questions are too easy (answerable from diff alone) or the profile needs tuning to include more relevant files.
- **Coverage < 80%:** Your profile is too restrictive. Add the missing critical files to your include patterns or adjust tier weights.
- **Security category scores low:** Check that security-related files are not being excluded by overly broad ignore patterns.

### Tracking over time

Store evaluation results as timestamped JSON files in `.harvx/quality/` and track trends:

```
.harvx/quality/
  eval-20260201-143022.json
  eval-20260215-091500.json
  eval-20260301-160045.json
```

Plot coverage and accuracy lift over time. Expect improvements as you:
- Refine your golden questions based on real review failures
- Tune profiles to capture consistently missed files
- Add questions for new architectural decisions

## CI Quality Tracking

### GitHub Actions workflow

```yaml
# .github/workflows/harvx-quality.yml
name: Harvx Quality Gate
on:
  pull_request:
    types: [opened, synchronize]
  schedule:
    - cron: '0 6 * * 1'  # Weekly Monday check

jobs:
  quality-coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install Harvx
        run: go install github.com/your-org/harvx/cmd/harvx@latest

      - name: Run coverage check
        run: |
          harvx quality --json > quality-report.json
          COVERAGE=$(jq '.coverage_percent' quality-report.json)
          echo "Coverage: ${COVERAGE}%"

          # Fail if coverage drops below threshold
          if (( $(echo "$COVERAGE < 75" | bc -l) )); then
            echo "::error::Golden questions coverage dropped below 75%"
            exit 1
          fi

      - name: Upload quality report
        uses: actions/upload-artifact@v4
        with:
          name: quality-report-${{ github.sha }}
          path: quality-report.json
          retention-days: 90

      - name: Comment on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const report = JSON.parse(fs.readFileSync('quality-report.json', 'utf8'));
            const body = `### Harvx Quality Report

            | Category | Coverage |
            |----------|----------|
            | Architecture | ${report.categories.architecture.covered}/${report.categories.architecture.total} |
            | Configuration | ${report.categories.configuration.covered}/${report.categories.configuration.total} |
            | Security | ${report.categories.security.covered}/${report.categories.security.total} |
            | Conventions | ${report.categories.conventions.covered}/${report.categories.conventions.total} |
            | Integration | ${report.categories.integration.covered}/${report.categories.integration.total} |
            | **Overall** | **${report.covered_questions}/${report.total_questions} (${report.coverage_percent.toFixed(1)}%)** |

            ${report.uncovered.length > 0
              ? '**Uncovered:** ' + report.uncovered.map(q => q.id).join(', ')
              : 'All golden questions covered.'}`;

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: body
            });
```

### Storing historical data

Use the weekly scheduled run to build a coverage history. Download artifacts periodically or push results to a tracking spreadsheet or dashboard.

## Maintaining the Question Corpus

### When to add questions

- A reviewer catches a bug that an LLM reviewer missed due to missing context.
- A new architectural pattern is introduced (new auth flow, new message queue, new config source).
- An integration point is added or changed (new external API, new database).

### When to update questions

- A refactor moves the answer to a different file. Update `critical_files` and `expected_answer`.
- A configuration default changes. Update `expected_answer`.
- A file is renamed or deleted. Update `critical_files`.

### When to remove questions

- The feature is decommissioned. Dead questions inflate coverage numbers without providing signal.
- The question is consistently answered correctly by both conditions (diff-only and Harvx). It is no longer discriminating.

### Review cadence

Review the golden questions file once per quarter:

1. Run `harvx quality` and check for stale `critical_files` paths (files that no longer exist).
2. Remove questions where the expected answer is outdated.
3. Add 3-5 new questions based on recent review failures.
4. Verify the total stays in the 15-30 range per project.

### Ownership

Assign golden questions maintenance to whoever owns the review pipeline configuration. This is typically a team lead or a developer experience engineer. Include `.harvx/golden-questions.toml` in code review -- changes to expected answers should be deliberate and documented in the PR description.