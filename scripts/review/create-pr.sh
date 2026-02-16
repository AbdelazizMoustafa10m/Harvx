#!/usr/bin/env bash

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

PHASE_ID=""
BASE_BRANCH="main"
HEAD_BRANCH=""
REVIEW_VERDICT="UNKNOWN"
VERIFICATION_SUMMARY="not provided"
PR_TITLE=""
ARTIFACTS_DIR=""
DRY_RUN="false"

TEMP_BODY_FILE=""

usage() {
    cat <<'USAGE'
Create a GitHub PR for a phase branch.

Usage:
  ./scripts/review/create-pr.sh --phase <id> [options]

Required:
  --phase <id>                 Phase id included in PR metadata

Options:
  --base <branch>              Base branch (default: main)
  --head <branch>              Head branch (default: current branch)
  --review-verdict <value>     Review verdict (default: UNKNOWN)
  --verification-summary <txt> Verification summary line or paragraph
  --title <text>               Explicit PR title
  --artifacts-dir <path>       Optional dir to persist automation metadata
  --dry-run                    Print planned command and rendered body path
  -h, --help                   Show help
USAGE
}

log() {
    local msg="$1"
    local ts
    ts="$(date '+%Y-%m-%d %H:%M:%S')"
    printf '[%s] %s\n' "$ts" "$msg"
}

die() {
    log "ERROR: $1"
    exit 1
}

cleanup() {
    if [[ -n "$TEMP_BODY_FILE" && -f "$TEMP_BODY_FILE" ]]; then
        rm -f "$TEMP_BODY_FILE"
    fi
}
trap cleanup EXIT

resolve_base_ref() {
    if git show-ref --verify --quiet "refs/heads/$BASE_BRANCH"; then
        echo "$BASE_BRANCH"
        return 0
    fi
    if git show-ref --verify --quiet "refs/remotes/origin/$BASE_BRANCH"; then
        echo "origin/$BASE_BRANCH"
        return 0
    fi
    return 1
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --phase)
                PHASE_ID="$2"
                shift 2
                ;;
            --base)
                BASE_BRANCH="$2"
                shift 2
                ;;
            --head)
                HEAD_BRANCH="$2"
                shift 2
                ;;
            --review-verdict)
                REVIEW_VERDICT="$2"
                shift 2
                ;;
            --verification-summary)
                VERIFICATION_SUMMARY="$2"
                shift 2
                ;;
            --title)
                PR_TITLE="$2"
                shift 2
                ;;
            --artifacts-dir)
                ARTIFACTS_DIR="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                die "Unknown option: $1"
                ;;
        esac
    done

    if [[ -z "$PHASE_ID" ]]; then
        die "--phase is required"
    fi
}

preflight() {
    if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        die "Must run inside a git repository"
    fi

    if [[ -z "$HEAD_BRANCH" ]]; then
        HEAD_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
    fi

    if ! git show-ref --verify --quiet "refs/heads/$HEAD_BRANCH"; then
        die "Head branch '$HEAD_BRANCH' not found locally"
    fi

    if ! resolve_base_ref >/dev/null; then
        die "Could not resolve base branch '$BASE_BRANCH' locally or at origin/$BASE_BRANCH"
    fi

    if [[ "$DRY_RUN" != "true" ]]; then
        if ! command -v gh >/dev/null 2>&1; then
            die "GitHub CLI (gh) is required"
        fi
    fi
}

build_commit_list() {
    local base_ref="$1"
    git log --reverse --pretty='- %h %s (%an)' "${base_ref}..${HEAD_BRANCH}"
}

ensure_has_commits() {
    local base_ref="$1"
    local count
    count="$(git rev-list --count "${base_ref}..${HEAD_BRANCH}")"
    if [[ "$count" -eq 0 ]]; then
        die "No commits found between '$base_ref' and '$HEAD_BRANCH'"
    fi
}

render_pr_body() {
    local base_ref="$1"
    local template="$PROJECT_ROOT/.github/PULL_REQUEST_TEMPLATE.md"

    if [[ ! -f "$template" ]]; then
        die "PR template not found at .github/PULL_REQUEST_TEMPLATE.md"
    fi

    TEMP_BODY_FILE="$(mktemp "${TMPDIR:-/tmp}/harvx-pr-body-XXXXXX.md")"

    cp "$template" "$TEMP_BODY_FILE"

    local commits
    commits="$(build_commit_list "$base_ref")"

    cat >> "$TEMP_BODY_FILE" <<EOF_BODY

## Automation Metadata

- Phase ID: ${PHASE_ID}
- Review Verdict: ${REVIEW_VERDICT}
- Base Branch: ${BASE_BRANCH}
- Base Ref Used: ${base_ref}
- Head Branch: ${HEAD_BRANCH}
- Generated At (UTC): $(date -u +%Y-%m-%dT%H:%M:%SZ)

## Verification Summary

${VERIFICATION_SUMMARY}

## Commits in Scope

${commits}
EOF_BODY
}

default_title() {
    local latest
    latest="$(git log -1 --pretty='%s' "$HEAD_BRANCH")"
    echo "phase ${PHASE_ID}: ${latest}"
}

persist_artifact_metadata() {
    if [[ -z "$ARTIFACTS_DIR" ]]; then
        return 0
    fi

    mkdir -p "$ARTIFACTS_DIR"

    cat > "$ARTIFACTS_DIR/pr-create.env" <<EOF_META
phase=$PHASE_ID
base_branch=$BASE_BRANCH
head_branch=$HEAD_BRANCH
review_verdict=$REVIEW_VERDICT
dry_run=$DRY_RUN
body_file=$TEMP_BODY_FILE
generated_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF_META
}

create_pr() {
    local title="$PR_TITLE"
    if [[ -z "$title" ]]; then
        title="$(default_title)"
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        log "DRY-RUN: gh pr create --base $BASE_BRANCH --head $HEAD_BRANCH --title '$title' --body-file '$TEMP_BODY_FILE'"
        log "DRY-RUN: rendered PR body file: $TEMP_BODY_FILE"
        cat "$TEMP_BODY_FILE"
        return 0
    fi

    gh pr create \
        --base "$BASE_BRANCH" \
        --head "$HEAD_BRANCH" \
        --title "$title" \
        --body-file "$TEMP_BODY_FILE"
}

main() {
    parse_args "$@"
    preflight

    local base_ref
    base_ref="$(resolve_base_ref)"

    ensure_has_commits "$base_ref"
    render_pr_body "$base_ref"
    persist_artifact_metadata
    create_pr
}

main "$@"
