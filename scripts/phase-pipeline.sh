#!/usr/bin/env bash

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DEFAULT_IMPL_AGENT="codex"
DEFAULT_REVIEW_MODE="all"
DEFAULT_REVIEW_AGENT="codex"
DEFAULT_REVIEW_CONCURRENCY=4
DEFAULT_MAX_REVIEW_CYCLES=2
DEFAULT_FIX_AGENT="codex"
DEFAULT_BASE_BRANCH="main"

PHASE_ID=""
IMPL_AGENT="$DEFAULT_IMPL_AGENT"
REVIEW_MODE="$DEFAULT_REVIEW_MODE"
REVIEW_AGENT="$DEFAULT_REVIEW_AGENT"
REVIEW_CONCURRENCY="$DEFAULT_REVIEW_CONCURRENCY"
MAX_REVIEW_CYCLES="$DEFAULT_MAX_REVIEW_CYCLES"
FIX_AGENT="$DEFAULT_FIX_AGENT"
BASE_BRANCH="$DEFAULT_BASE_BRANCH"
SYNC_BASE="false"
SKIP_IMPLEMENT="false"
SKIP_REVIEW="false"
SKIP_FIX="false"
SKIP_PR="false"
DRY_RUN="false"
INTERACTIVE="false"

EXPECTED_BRANCH=""
RUN_ID=""
RUN_DIR=""
REPORT_DIR=""
PIPELINE_LOG=""
METADATA_FILE=""

IMPLEMENT_STATUS="not-run"
REVIEW_VERDICT="NOT_RUN"
REVIEW_CYCLES=0
FIX_CYCLES=0
PR_STATUS="not-run"

usage() {
    cat <<'USAGE'
Harvx Phase Pipeline Orchestrator

Usage:
  ./scripts/phase-pipeline.sh --phase <id> [options]
  ./scripts/phase-pipeline.sh --interactive

Required:
  --phase <id>                 Phase id (1, 2a, 2b, 3a, 3b, 4a, 4b, 5a, 5b, 6)
                               If omitted in an interactive terminal, a wizard is shown.

Options:
  --interactive                Force interactive wizard prompts
  --impl-agent <claude|codex>
  --review <none|agent|all>
  --review-agent <claude|codex|gemini>
  --review-concurrency <n>
  --max-review-cycles <n>
  --fix-agent <claude|codex|gemini>
  --base <branch>              Base branch (default: main)
  --sync-base                  Fetch + fast-forward base from origin before bootstrap
  --skip-implement             Skip implementation phase
  --skip-review                Skip review phase
  --skip-fix                   Skip review fix cycles
  --skip-pr                    Skip PR creation
  --dry-run                    Print planned commands without executing
  -h, --help                   Show help
USAGE
}

log() {
    local msg="$1"
    local ts
    ts="$(date '+%Y-%m-%d %H:%M:%S')"
    if [[ -n "$PIPELINE_LOG" ]]; then
        printf '[%s] %s\n' "$ts" "$msg" | tee -a "$PIPELINE_LOG"
    else
        printf '[%s] %s\n' "$ts" "$msg"
    fi
}

die() {
    log "ERROR: $1"
    exit 1
}

run_cmd() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DRY-RUN: $*"
        return 0
    fi
    "$@"
}

capture_cmd() {
    local output_file="$1"
    shift

    if [[ "$DRY_RUN" == "true" ]]; then
        log "DRY-RUN: $* > $output_file"
        : > "$output_file"
        return 0
    fi

    set +e
    "$@" > >(tee "$output_file") 2>&1
    local rc=$?
    set -e
    return $rc
}

validate_phase() {
    case "$PHASE_ID" in
        1|2a|2b|3a|3b|4a|4b|5a|5b|6) return 0 ;;
        *) die "Invalid --phase '$PHASE_ID'. Expected one of: 1,2a,2b,3a,3b,4a,4b,5a,5b,6" ;;
    esac
}

phase_slug() {
    case "$1" in
        1) echo "foundation" ;;
        2a) echo "profiles" ;;
        2b) echo "relevance-tokens" ;;
        3a) echo "security" ;;
        3b) echo "compression" ;;
        4a) echo "output-rendering" ;;
        4b) echo "state-diff" ;;
        5a) echo "workflows" ;;
        5b) echo "interactive-tui" ;;
        6) echo "polish-distribution" ;;
        *) die "No slug mapping for phase '$1'" ;;
    esac
}

phase_title() {
    case "$1" in
        1) echo "Foundation" ;;
        2a) echo "Intelligence: Config Profiles" ;;
        2b) echo "Intelligence: Relevance + Tokenization" ;;
        3a) echo "Security: Redaction + Entropy" ;;
        3b) echo "Compression: Tree-sitter WASM" ;;
        4a) echo "Output: Markdown/XML Rendering" ;;
        4b) echo "State: Snapshots + Diffing" ;;
        5a) echo "Workflows + MCP Server" ;;
        5b) echo "TUI: Interactive Selector" ;;
        6) echo "Polish + Distribution" ;;
        *) echo "Unknown" ;;
    esac
}

is_interactive_terminal() {
    [[ -t 0 && -t 1 ]]
}

read_required_input() {
    local prompt="$1"
    local input=""
    read -r -p "$prompt" input || die "Interactive input aborted"
    printf '%s' "$input"
}

prompt_choice() {
    local out_var="$1"
    local prompt="$2"
    local default_value="$3"
    shift 3
    local options=("$@")

    while true; do
        echo ""
        echo "$prompt"
        local i
        for i in "${!options[@]}"; do
            local value="${options[$i]}"
            local marker=""
            if [[ "$value" == "$default_value" ]]; then
                marker=" (default)"
            fi
            printf "  %d) %s%s\n" "$((i + 1))" "$value" "$marker"
        done

        local input
        input="$(read_required_input "Select option [${default_value}]: ")"
        if [[ -z "$input" ]]; then
            printf -v "$out_var" '%s' "$default_value"
            return 0
        fi

        if [[ "$input" =~ ^[0-9]+$ ]] && (( input >= 1 && input <= ${#options[@]} )); then
            printf -v "$out_var" '%s' "${options[$((input - 1))]}"
            return 0
        fi

        local value
        for value in "${options[@]}"; do
            if [[ "$input" == "$value" ]]; then
                printf -v "$out_var" '%s' "$value"
                return 0
            fi
        done

        echo "Invalid choice: $input"
    done
}

prompt_yes_no() {
    local out_var="$1"
    local prompt="$2"
    local default_bool="$3"
    local default_hint="y/N"
    if [[ "$default_bool" == "true" ]]; then
        default_hint="Y/n"
    fi

    while true; do
        local input
        input="$(read_required_input "$prompt [$default_hint]: ")"

        if [[ -z "$input" ]]; then
            printf -v "$out_var" '%s' "$default_bool"
            return 0
        fi

        local normalized
        normalized="$(printf '%s' "$input" | tr '[:upper:]' '[:lower:]')"

        case "$normalized" in
            y|yes|true|1)
                printf -v "$out_var" '%s' "true"
                return 0
                ;;
            n|no|false|0)
                printf -v "$out_var" '%s' "false"
                return 0
                ;;
            *)
                echo "Please answer yes or no."
                ;;
        esac
    done
}

prompt_number() {
    local out_var="$1"
    local prompt="$2"
    local default_number="$3"
    local pattern="$4"

    while true; do
        local input
        input="$(read_required_input "$prompt [$default_number]: ")"
        if [[ -z "$input" ]]; then
            input="$default_number"
        fi

        if [[ "$input" =~ $pattern ]]; then
            printf -v "$out_var" '%s' "$input"
            return 0
        fi

        echo "Invalid number: $input"
    done
}

prompt_text() {
    local out_var="$1"
    local prompt="$2"
    local default_value="$3"

    local input
    input="$(read_required_input "$prompt [$default_value]: ")"
    if [[ -z "$input" ]]; then
        input="$default_value"
    fi
    printf -v "$out_var" '%s' "$input"
}

prompt_phase_id() {
    local out_var="$1"
    local default_phase="${2:-1}"
    local phase_ids=(1 2a 2b 3a 3b 4a 4b 5a 5b 6)

    while true; do
        echo ""
        echo "Choose phase:"
        local i
        for i in "${!phase_ids[@]}"; do
            local phase_id="${phase_ids[$i]}"
            local marker=""
            if [[ "$phase_id" == "$default_phase" ]]; then
                marker=" (default)"
            fi
            printf "  %d) %s - %s%s\n" "$((i + 1))" "$phase_id" "$(phase_title "$phase_id")" "$marker"
        done

        local input
        input="$(read_required_input "Select phase [${default_phase}]: ")"
        if [[ -z "$input" ]]; then
            printf -v "$out_var" '%s' "$default_phase"
            return 0
        fi

        if [[ "$input" =~ ^[0-9]+$ ]] && (( input >= 1 && input <= ${#phase_ids[@]} )); then
            printf -v "$out_var" '%s' "${phase_ids[$((input - 1))]}"
            return 0
        fi

        local phase_id
        for phase_id in "${phase_ids[@]}"; do
            if [[ "$input" == "$phase_id" ]]; then
                printf -v "$out_var" '%s' "$phase_id"
                return 0
            fi
        done

        echo "Invalid phase: $input"
    done
}

run_interactive_wizard() {
    echo ""
    echo "Harvx Phase Pipeline Wizard"
    echo "==========================="

    prompt_phase_id PHASE_ID "${PHASE_ID:-1}"
    prompt_choice IMPL_AGENT "Select implementation agent:" "$IMPL_AGENT" "codex" "claude"
    prompt_choice REVIEW_MODE "Select review mode:" "$REVIEW_MODE" "all" "agent" "none"

    if [[ "$REVIEW_MODE" == "agent" ]]; then
        prompt_choice REVIEW_AGENT "Select review agent:" "$REVIEW_AGENT" "codex" "claude" "gemini"
    fi

    if [[ "$REVIEW_MODE" != "none" ]]; then
        prompt_number REVIEW_CONCURRENCY "Set review concurrency:" "$REVIEW_CONCURRENCY" '^[1-9][0-9]*$'
        prompt_number MAX_REVIEW_CYCLES "Set max review-fix cycles:" "$MAX_REVIEW_CYCLES" '^[0-9]+$'
        prompt_choice FIX_AGENT "Select fix agent for blocking review findings:" "$FIX_AGENT" "codex" "claude" "gemini"
    fi

    prompt_text BASE_BRANCH "Base branch:" "$BASE_BRANCH"
    prompt_yes_no SYNC_BASE "Sync base branch from origin before bootstrap?" "$SYNC_BASE"

    local execution_profile="full"
    prompt_choice execution_profile "Select execution profile:" "full" "full" "impl-only" "review-only" "custom"

    case "$execution_profile" in
        full)
            SKIP_IMPLEMENT="false"
            SKIP_REVIEW="false"
            SKIP_FIX="false"
            SKIP_PR="false"
            ;;
        impl-only)
            SKIP_IMPLEMENT="false"
            SKIP_REVIEW="true"
            SKIP_FIX="true"
            SKIP_PR="true"
            ;;
        review-only)
            SKIP_IMPLEMENT="true"
            SKIP_REVIEW="false"
            SKIP_FIX="false"
            SKIP_PR="false"
            ;;
        custom)
            prompt_yes_no SKIP_IMPLEMENT "Skip implementation stage?" "$SKIP_IMPLEMENT"
            prompt_yes_no SKIP_REVIEW "Skip review stage?" "$SKIP_REVIEW"
            prompt_yes_no SKIP_FIX "Skip review-fix cycles?" "$SKIP_FIX"
            prompt_yes_no SKIP_PR "Skip PR creation?" "$SKIP_PR"
            ;;
    esac

    prompt_yes_no DRY_RUN "Run in dry-run mode?" "$DRY_RUN"

    echo ""
    echo "Selected configuration:"
    echo "  phase=$PHASE_ID ($(phase_title "$PHASE_ID"))"
    echo "  impl_agent=$IMPL_AGENT"
    echo "  review_mode=$REVIEW_MODE"
    echo "  review_agent=$REVIEW_AGENT"
    echo "  review_concurrency=$REVIEW_CONCURRENCY"
    echo "  max_review_cycles=$MAX_REVIEW_CYCLES"
    echo "  fix_agent=$FIX_AGENT"
    echo "  base=$BASE_BRANCH"
    echo "  sync_base=$SYNC_BASE"
    echo "  skip_implement=$SKIP_IMPLEMENT"
    echo "  skip_review=$SKIP_REVIEW"
    echo "  skip_fix=$SKIP_FIX"
    echo "  skip_pr=$SKIP_PR"
    echo "  dry_run=$DRY_RUN"

    local proceed="false"
    prompt_yes_no proceed "Proceed with this configuration?" "true"
    if [[ "$proceed" != "true" ]]; then
        die "Pipeline cancelled from interactive wizard"
    fi
}

is_blocking_verdict() {
    local verdict="$1"
    [[ "$verdict" == "REQUEST_CHANGES" || "$verdict" == "NEEDS_FIXES" ]]
}

assert_expected_branch() {
    if [[ "$DRY_RUN" == "true" ]]; then
        return 0
    fi

    if [[ -z "$EXPECTED_BRANCH" ]]; then
        return 0
    fi

    local current
    current="$(git rev-parse --abbrev-ref HEAD)"
    if [[ "$current" != "$EXPECTED_BRANCH" ]]; then
        die "Expected branch '$EXPECTED_BRANCH' but found '$current'"
    fi
}

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

persist_metadata() {
    cat > "$METADATA_FILE" <<EOF_META
run_id=$RUN_ID
phase=$PHASE_ID
branch=$EXPECTED_BRANCH
base_branch=$BASE_BRANCH
impl_agent=$IMPL_AGENT
review_mode=$REVIEW_MODE
review_agent=$REVIEW_AGENT
review_concurrency=$REVIEW_CONCURRENCY
max_review_cycles=$MAX_REVIEW_CYCLES
fix_agent=$FIX_AGENT
sync_base=$SYNC_BASE
dry_run=$DRY_RUN
skip_implement=$SKIP_IMPLEMENT
skip_review=$SKIP_REVIEW
skip_fix=$SKIP_FIX
skip_pr=$SKIP_PR
run_dir=$RUN_DIR
report_dir=$REPORT_DIR
pipeline_log=$PIPELINE_LOG
implementation_status=$IMPLEMENT_STATUS
review_verdict=$REVIEW_VERDICT
review_cycles=$REVIEW_CYCLES
fix_cycles=$FIX_CYCLES
pr_status=$PR_STATUS
updated_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF_META
}

init_artifacts() {
    RUN_ID="phase-${PHASE_ID}-$(date -u +%Y%m%dT%H%M%SZ)"
    RUN_DIR="$PROJECT_ROOT/.review-workspace/phase-pipeline/$RUN_ID"
    REPORT_DIR="$PROJECT_ROOT/reports/review/$RUN_ID"
    mkdir -p "$RUN_DIR" "$REPORT_DIR"

    PIPELINE_LOG="$RUN_DIR/pipeline.log"
    METADATA_FILE="$RUN_DIR/metadata.env"
    : > "$PIPELINE_LOG"

    persist_metadata
}

ensure_clean_tree_before_bootstrap() {
    if [[ "$DRY_RUN" == "true" ]]; then
        return 0
    fi

    local dirty
    dirty="$(git status --porcelain)"
    if [[ -n "$dirty" ]]; then
        die "Working tree is dirty before branch bootstrap. Commit/stash first."
    fi
}

preflight() {
    validate_phase

    if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        die "Must run inside a git repository"
    fi

    if [[ "$SKIP_IMPLEMENT" != "true" ]]; then
        if [[ "$IMPL_AGENT" != "claude" && "$IMPL_AGENT" != "codex" ]]; then
            die "--impl-agent must be claude or codex"
        fi
        if [[ ! -f "$PROJECT_ROOT/scripts/ralph_${IMPL_AGENT}.sh" ]]; then
            die "Implementation script missing: scripts/ralph_${IMPL_AGENT}.sh"
        fi
    fi

    if [[ "$REVIEW_MODE" != "none" && "$REVIEW_MODE" != "agent" && "$REVIEW_MODE" != "all" ]]; then
        die "--review must be one of: none, agent, all"
    fi

    if [[ "$REVIEW_AGENT" != "claude" && "$REVIEW_AGENT" != "codex" && "$REVIEW_AGENT" != "gemini" ]]; then
        die "--review-agent must be one of: claude, codex, gemini"
    fi

    if [[ "$FIX_AGENT" != "claude" && "$FIX_AGENT" != "codex" && "$FIX_AGENT" != "gemini" ]]; then
        die "--fix-agent must be one of: claude, codex, gemini"
    fi

    if ! [[ "$REVIEW_CONCURRENCY" =~ ^[1-9][0-9]*$ ]]; then
        die "--review-concurrency must be a positive integer"
    fi

    if ! [[ "$MAX_REVIEW_CYCLES" =~ ^[0-9]+$ ]]; then
        die "--max-review-cycles must be a non-negative integer"
    fi

    if [[ "$SKIP_PR" != "true" ]] && [[ "$DRY_RUN" != "true" ]]; then
        if ! command -v gh >/dev/null 2>&1; then
            die "GitHub CLI (gh) is required for PR creation"
        fi
    fi

    if ! resolve_base_ref >/dev/null; then
        die "Could not resolve base branch '$BASE_BRANCH' locally or at origin/$BASE_BRANCH"
    fi

    ensure_clean_tree_before_bootstrap
}

sync_base_branch() {
    if [[ "$SYNC_BASE" != "true" ]]; then
        return 0
    fi

    log "Syncing base branch '$BASE_BRANCH' from origin"
    run_cmd git fetch origin "$BASE_BRANCH"

    local base_ref
    if git show-ref --verify --quiet "refs/remotes/origin/$BASE_BRANCH"; then
        if git show-ref --verify --quiet "refs/heads/$BASE_BRANCH"; then
            run_cmd git checkout "$BASE_BRANCH"
            run_cmd git merge --ff-only "origin/$BASE_BRANCH"
        else
            run_cmd git checkout -b "$BASE_BRANCH" "origin/$BASE_BRANCH"
        fi
    else
        die "origin/$BASE_BRANCH not found after fetch"
    fi
}

bootstrap_branch() {
    local slug
    slug="$(phase_slug "$PHASE_ID")"
    local branch
    branch="phase/${PHASE_ID}-${slug}"

    local base_ref
    base_ref="$(resolve_base_ref)"

    log "Bootstrapping branch '$branch' from '$base_ref'"

    if git show-ref --verify --quiet "refs/heads/$branch"; then
        run_cmd git checkout "$branch"
    else
        run_cmd git checkout -b "$branch" "$base_ref"
    fi

    EXPECTED_BRANCH="$branch"
    assert_expected_branch
}

run_implementation() {
    if [[ "$SKIP_IMPLEMENT" == "true" ]]; then
        IMPLEMENT_STATUS="skipped"
        log "Implementation skipped"
        persist_metadata
        return 0
    fi

    assert_expected_branch

    local impl_script="$PROJECT_ROOT/scripts/ralph_${IMPL_AGENT}.sh"
    log "Running implementation with $impl_script --phase $PHASE_ID"

    run_cmd "$impl_script" --phase "$PHASE_ID"

    IMPLEMENT_STATUS="completed"
    assert_expected_branch
    persist_metadata
}

extract_verdict() {
    local log_file="$1"

    if [[ ! -f "$log_file" ]]; then
        echo "UNKNOWN"
        return 0
    fi

    local token
    token="$(grep -Eo '\b(REQUEST_CHANGES|NEEDS_FIXES|APPROVE|APPROVED|COMMENT|COMMENTS_ONLY|LGTM|PASS|PASSED|BLOCKING|FAIL)\b' "$log_file" | tail -1 || true)"

    case "$token" in
        REQUEST_CHANGES) echo "REQUEST_CHANGES" ;;
        NEEDS_FIXES|BLOCKING|FAIL) echo "NEEDS_FIXES" ;;
        APPROVE|APPROVED|COMMENTS_ONLY|LGTM|PASS|PASSED) echo "APPROVED" ;;
        COMMENT) echo "COMMENT" ;;
        *) echo "UNKNOWN" ;;
    esac
}

extract_verdict_from_consolidated() {
    local consolidated_file="$PROJECT_ROOT/reports/review/latest/consolidated.json"

    if [[ ! -f "$consolidated_file" ]]; then
        echo "UNKNOWN"
        return 0
    fi

    local verdict
    verdict="$(jq -r '.verdict // "UNKNOWN"' "$consolidated_file" 2>/dev/null || echo "UNKNOWN")"

    case "$verdict" in
        REQUEST_CHANGES) echo "REQUEST_CHANGES" ;;
        NEEDS_FIXES) echo "NEEDS_FIXES" ;;
        APPROVE|APPROVED) echo "APPROVED" ;;
        COMMENT) echo "COMMENT" ;;
        *) echo "UNKNOWN" ;;
    esac
}

run_review_once() {
    local cycle="$1"

    assert_expected_branch

    local review_script="$PROJECT_ROOT/scripts/review/review.sh"
    if [[ ! -f "$review_script" ]]; then
        die "Review script not found: scripts/review/review.sh"
    fi

    local review_log="$RUN_DIR/review-cycle-${cycle}.log"
    local review_args=()

    review_args+=(--base "$BASE_BRANCH")
    review_args+=(--concurrency "$REVIEW_CONCURRENCY")
    if [[ "$REVIEW_MODE" == "agent" ]]; then
        review_args+=(--agent "$REVIEW_AGENT")
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        review_args+=(--dry-run)
    fi

    log "Running review cycle $cycle using $(basename "$review_script") mode=$REVIEW_MODE"

    export HARVX_PHASE_ID="$PHASE_ID"
    export HARVX_REVIEW_MODE="$REVIEW_MODE"
    export HARVX_REVIEW_AGENT="$REVIEW_AGENT"
    export HARVX_REVIEW_CONCURRENCY="$REVIEW_CONCURRENCY"
    export HARVX_REVIEW_REPORT_DIR="$REPORT_DIR"
    export HARVX_REVIEW_RUN_DIR="$RUN_DIR"

    if [[ "$DRY_RUN" == "true" ]]; then
        if ! capture_cmd "$review_log" "$review_script" "${review_args[@]}"; then
            log "Review command returned non-zero in dry-run"
        fi
        REVIEW_VERDICT="UNKNOWN"
        log "Review verdict: $REVIEW_VERDICT (dry-run)"
        persist_metadata
        return 0
    else
        if capture_cmd "$review_log" "$review_script" "${review_args[@]}"; then
            :
        else
            log "Review command returned non-zero; evaluating verdict from output"
        fi
    fi

    REVIEW_VERDICT="$(extract_verdict_from_consolidated)"
    if [[ "$REVIEW_VERDICT" == "UNKNOWN" ]]; then
        REVIEW_VERDICT="$(extract_verdict "$review_log")"
    fi
    log "Review verdict: $REVIEW_VERDICT"
    persist_metadata
}

run_fix_once() {
    local cycle="$1"

    assert_expected_branch

    local fix_script="$PROJECT_ROOT/scripts/review/review-fix.sh"
    if [[ ! -f "$fix_script" ]]; then
        die "Fix script not found: scripts/review/review-fix.sh"
    fi

    local fix_log="$RUN_DIR/fix-cycle-${cycle}.log"
    log "Running fix cycle $cycle using $(basename "$fix_script")"

    export HARVX_PHASE_ID="$PHASE_ID"
    export HARVX_FIX_AGENT="$FIX_AGENT"
    export HARVX_REVIEW_VERDICT="$REVIEW_VERDICT"
    export HARVX_REVIEW_REPORT_DIR="$REPORT_DIR"
    export HARVX_REVIEW_RUN_DIR="$RUN_DIR"

    local fix_args=()
    fix_args+=(--agent "$FIX_AGENT")
    if [[ "$DRY_RUN" == "true" ]]; then
        fix_args+=(--dry-run)
    fi

    if capture_cmd "$fix_log" "$fix_script" "${fix_args[@]}"; then
        log "Fix cycle $cycle completed"
    else
        die "Fix command failed in cycle $cycle (see $fix_log)"
    fi

    FIX_CYCLES=$((FIX_CYCLES + 1))
    persist_metadata
}

run_review_and_fix_cycles() {
    if [[ "$SKIP_REVIEW" == "true" || "$REVIEW_MODE" == "none" ]]; then
        REVIEW_VERDICT="SKIPPED"
        log "Review skipped"
        persist_metadata
        return 0
    fi

    run_review_once 0
    REVIEW_CYCLES=0

    while is_blocking_verdict "$REVIEW_VERDICT"; do
        if [[ "$SKIP_FIX" == "true" ]]; then
            log "Blocking review verdict but fix cycles are skipped"
            break
        fi

        if (( REVIEW_CYCLES >= MAX_REVIEW_CYCLES )); then
            log "Max review cycles reached ($MAX_REVIEW_CYCLES) with verdict $REVIEW_VERDICT"
            break
        fi

        REVIEW_CYCLES=$((REVIEW_CYCLES + 1))
        run_fix_once "$REVIEW_CYCLES"
        run_review_once "$REVIEW_CYCLES"
    done

    if is_blocking_verdict "$REVIEW_VERDICT" && [[ "$SKIP_FIX" != "true" ]]; then
        die "Blocking review verdict remains after $REVIEW_CYCLES fix cycle(s): $REVIEW_VERDICT"
    fi

    persist_metadata
}

run_pr_creation() {
    if [[ "$SKIP_PR" == "true" ]]; then
        PR_STATUS="skipped"
        log "PR creation skipped"
        persist_metadata
        return 0
    fi

    assert_expected_branch

    local verification_summary
    verification_summary="implementation=${IMPLEMENT_STATUS}; review_verdict=${REVIEW_VERDICT}; review_cycles=${REVIEW_CYCLES}; fix_cycles=${FIX_CYCLES}; artifacts=${RUN_DIR}"

    local pr_script="$PROJECT_ROOT/scripts/review/create-pr.sh"
    if [[ ! -f "$pr_script" ]]; then
        die "PR script not found: scripts/review/create-pr.sh"
    fi

    log "Creating PR via scripts/review/create-pr.sh"

    local pr_args=()
    pr_args+=(--phase "$PHASE_ID")
    pr_args+=(--base "$BASE_BRANCH")
    pr_args+=(--review-verdict "$REVIEW_VERDICT")
    pr_args+=(--verification-summary "$verification_summary")
    pr_args+=(--artifacts-dir "$RUN_DIR")
    if [[ "$DRY_RUN" == "true" ]]; then
        pr_args+=(--dry-run)
    fi

    "$pr_script" "${pr_args[@]}"

    PR_STATUS="completed"
    persist_metadata
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --phase)
                PHASE_ID="$2"
                shift 2
                ;;
            --impl-agent)
                IMPL_AGENT="$2"
                shift 2
                ;;
            --review)
                REVIEW_MODE="$2"
                shift 2
                ;;
            --review-agent)
                REVIEW_AGENT="$2"
                shift 2
                ;;
            --review-concurrency)
                REVIEW_CONCURRENCY="$2"
                shift 2
                ;;
            --max-review-cycles)
                MAX_REVIEW_CYCLES="$2"
                shift 2
                ;;
            --fix-agent)
                FIX_AGENT="$2"
                shift 2
                ;;
            --base)
                BASE_BRANCH="$2"
                shift 2
                ;;
            --sync-base)
                SYNC_BASE="true"
                shift
                ;;
            --skip-implement)
                SKIP_IMPLEMENT="true"
                shift
                ;;
            --skip-review)
                SKIP_REVIEW="true"
                shift
                ;;
            --skip-fix)
                SKIP_FIX="true"
                shift
                ;;
            --skip-pr)
                SKIP_PR="true"
                shift
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            --interactive)
                INTERACTIVE="true"
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

}

resolve_interactive_mode() {
    if [[ "$INTERACTIVE" == "true" ]]; then
        run_interactive_wizard
        return 0
    fi

    if [[ -z "$PHASE_ID" ]] && is_interactive_terminal; then
        INTERACTIVE="true"
        run_interactive_wizard
        return 0
    fi

    if [[ -z "$PHASE_ID" ]]; then
        die "--phase is required (or run with --interactive)"
    fi
}

main() {
    parse_args "$@"
    resolve_interactive_mode

    preflight
    init_artifacts

    log "Starting phase pipeline: phase=$PHASE_ID base=$BASE_BRANCH dry_run=$DRY_RUN"

    sync_base_branch
    bootstrap_branch
    run_implementation
    run_review_and_fix_cycles
    run_pr_creation

    log "Pipeline complete. Metadata: $METADATA_FILE"
    log "Artifacts: $RUN_DIR"

    if is_blocking_verdict "$REVIEW_VERDICT"; then
        log "Pipeline ended with blocking review verdict: $REVIEW_VERDICT"
    fi
}

main "$@"
