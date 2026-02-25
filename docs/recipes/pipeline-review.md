# Recipe: Pipeline Review (Zizo Persona)

Set up Harvx as part of an automated review pipeline for PR-driven development.

## Use Case

You want to automate code review context generation so that every PR gets a review slice with the changed files, their tests, and project context—ready for an LLM reviewer.

## Recipe 1: Manual PR Review

```bash
# Generate context for the current branch vs main
harvx brief --target claude --stdout > /tmp/brief.md
harvx review-slice --base origin/main --head HEAD --target claude --stdout > /tmp/slice.md

# Combine and review
cat /tmp/brief.md /tmp/slice.md | pbcopy
```

## Recipe 2: Review Script

Create `scripts/review.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

BASE="${1:-origin/main}"
HEAD="${2:-HEAD}"

echo "=== Repo Brief ===" >&2
harvx brief --target claude --stdout
echo ""
echo "=== Review Slice: $BASE..$HEAD ===" >&2
harvx review-slice --base "$BASE" --head "$HEAD" --target claude --stdout
```

Usage:

```bash
# Review current branch
./scripts/review.sh | pbcopy

# Review specific refs
./scripts/review.sh origin/develop feature/auth | pbcopy
```

## Recipe 3: Session Hook with Review Context

For branches with active PRs, include the review slice in session bootstrap:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "command": "harvx brief --target claude --stdout && harvx review-slice --base origin/main --head HEAD --target claude --stdout 2>/dev/null || true",
        "timeout": 8000
      }
    ]
  }
}
```

The `|| true` ensures the hook doesn't fail if you're on main (no diff).

## Recipe 4: Pre-Push Review Check

```bash
# Add to .git/hooks/pre-push or use as a manual check
harvx review-slice \
  --base origin/main \
  --head HEAD \
  --assert-include "**/*_test.go" \
  --json | jq '.total_files'
```

The `--assert-include` check confirms test files are present for the changed code.

## Recipe 5: Module-Focused Review

When reviewing changes in a specific module:

```bash
# Combine module slice with review slice
harvx slice --path internal/auth --target claude --stdout > /tmp/module.md
harvx review-slice --base origin/main --head HEAD --target claude --stdout > /tmp/changes.md
cat /tmp/module.md /tmp/changes.md
```

## Tips

- Use `--json` output with `jq` for scripting conditional pipelines
- Set `HARVX_QUIET=1` in scripts to suppress progress output
- The review slice prioritizes changed files—neighbors fill remaining budget
- Configure `slice_max_tokens` and `slice_depth` in your `.harvx.toml` profile for fine-tuning
