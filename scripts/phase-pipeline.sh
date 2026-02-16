#!/usr/bin/env bash

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

ALL_PHASE_IDS=(1 2a 2b 3a 3b 4a 4b 5a 5b 6)

DEFAULT_IMPL_AGENT="codex"
DEFAULT_REVIEW_MODE="all"
DEFAULT_REVIEW_AGENT="codex"
DEFAULT_REVIEW_CONCURRENCY=4
DEFAULT_MAX_REVIEW_CYCLES=2
DEFAULT_FIX_AGENT="codex"
DEFAULT_BASE_BRANCH="main"

PHASE_ID=""
PHASE_MODE="single"   # single | all | from
FROM_PHASE=""
PHASES_TO_RUN=()
CHAIN_BASE=""          # Tracks previous phase branch for multi-phase chaining
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
  ./scripts/phase-pipeline.sh --phase all [options]
  ./scripts/phase-pipeline.sh --from-phase <id> [options]
  ./scripts/phase-pipeline.sh --interactive

Required (one of):
  --phase <id>                 Single phase (1, 2a, 2b, 3a, 3b, 4a, 4b, 5a, 5b, 6)
  --phase all                  Run all phases sequentially (1 → 6)
  --from-phase <id>            Start from this phase, run sequentially through phase 6
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

Examples:
  ./scripts/phase-pipeline.sh --phase 1                  # Run phase 1 only
  ./scripts/phase-pipeline.sh --phase all                # Run all phases (1 → 6)
  ./scripts/phase-pipeline.sh --from-phase 3a            # Run phases 3a → 3b → ... → 6
  ./scripts/phase-pipeline.sh --phase all --skip-pr      # All phases, no PRs
  ./scripts/phase-pipeline.sh --from-phase 2a --dry-run  # Preview from phase 2a
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

validate_single_phase() {
    local phase="$1"
    case "$phase" in
        1|2a|2b|3a|3b|4a|4b|5a|5b|6) return 0 ;;
        *) return 1 ;;
    esac
}

resolve_phases_to_run() {
    if [[ -n "$FROM_PHASE" && -n "$PHASE_ID" ]]; then
        die "Cannot use both --phase and --from-phase"
    fi

    if [[ -n "$FROM_PHASE" ]]; then
        if ! validate_single_phase "$FROM_PHASE"; then
            die "Invalid --from-phase '$FROM_PHASE'. Expected one of: ${ALL_PHASE_IDS[*]}"
        fi
        PHASE_MODE="from"
        local collecting=false
        for p in "${ALL_PHASE_IDS[@]}"; do
            if [[ "$p" == "$FROM_PHASE" ]]; then collecting=true; fi
            if [[ "$collecting" == "true" ]]; then
                PHASES_TO_RUN+=("$p")
            fi
        done
    elif [[ "$PHASE_ID" == "all" ]]; then
        PHASE_MODE="all"
        PHASES_TO_RUN=("${ALL_PHASE_IDS[@]}")
    else
        if ! validate_single_phase "$PHASE_ID"; then
            die "Invalid --phase '$PHASE_ID'. Expected one of: ${ALL_PHASE_IDS[*]}, all"
        fi
        PHASE_MODE="single"
        PHASES_TO_RUN=("$PHASE_ID")
    fi

    if [[ ${#PHASES_TO_RUN[@]} -eq 0 ]]; then
        die "No phases resolved to run"
    fi
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

    while true; do
        echo ""
        echo "Choose phase:"
        local i
        for i in "${!ALL_PHASE_IDS[@]}"; do
            local phase_id="${ALL_PHASE_IDS[$i]}"
            local marker=""
            if [[ "$phase_id" == "$default_phase" ]]; then
                marker=" (default)"
            fi
            printf "  %2d) %-3s - %s%s\n" "$((i + 1))" "$phase_id" "$(phase_title "$phase_id")" "$marker"
        done
        echo "  ──────────────────────────────────────────"
        printf "  %2d) all  - Run all phases sequentially (1 → 6)\n" "$((${#ALL_PHASE_IDS[@]} + 1))"
        printf "  %2d) from - Start from a specific phase through 6\n" "$((${#ALL_PHASE_IDS[@]} + 2))"

        local input
        input="$(read_required_input "Select phase [${default_phase}]: ")"
        if [[ -z "$input" ]]; then
            printf -v "$out_var" '%s' "$default_phase"
            return 0
        fi

        # "all" by name or number
        if [[ "$input" == "all" || "$input" == "$((${#ALL_PHASE_IDS[@]} + 1))" ]]; then
            printf -v "$out_var" '%s' "all"
            return 0
        fi

        # "from" by name or number
        if [[ "$input" == "from" || "$input" == "$((${#ALL_PHASE_IDS[@]} + 2))" ]]; then
            printf -v "$out_var" '%s' "from"
            return 0
        fi

        # Number selection for individual phase
        if [[ "$input" =~ ^[0-9]+$ ]] && (( input >= 1 && input <= ${#ALL_PHASE_IDS[@]} )); then
            printf -v "$out_var" '%s' "${ALL_PHASE_IDS[$((input - 1))]}"
            return 0
        fi

        # Direct phase ID
        local phase_id
        for phase_id in "${ALL_PHASE_IDS[@]}"; do
            if [[ "$input" == "$phase_id" ]]; then
                printf -v "$out_var" '%s' "$phase_id"
                return 0
            fi
        done

        echo "Invalid phase: $input"
    done
}

prompt_from_phase() {
    local out_var="$1"
    local default_phase="${2:-1}"

    while true; do
        echo ""
        echo "Choose starting phase (will run sequentially through phase 6):"
        local i
        for i in "${!ALL_PHASE_IDS[@]}"; do
            local phase_id="${ALL_PHASE_IDS[$i]}"
            # Count remaining phases from this one
            local count=0
            local collecting=false
            for p in "${ALL_PHASE_IDS[@]}"; do
                if [[ "$p" == "$phase_id" ]]; then collecting=true; fi
                if [[ "$collecting" == "true" ]]; then count=$((count + 1)); fi
            done
            local marker=""
            if [[ "$phase_id" == "$default_phase" ]]; then
                marker=" (default)"
            fi
            printf "  %2d) %-3s - %s (%d phase%s)%s\n" \
                "$((i + 1))" "$phase_id" "$(phase_title "$phase_id")" \
                "$count" "$(if [[ $count -gt 1 ]]; then echo "s"; fi)" "$marker"
        done

        local input
        input="$(read_required_input "Select starting phase [${default_phase}]: ")"
        if [[ -z "$input" ]]; then
            printf -v "$out_var" '%s' "$default_phase"
            return 0
        fi

        if [[ "$input" =~ ^[0-9]+$ ]] && (( input >= 1 && input <= ${#ALL_PHASE_IDS[@]} )); then
            printf -v "$out_var" '%s' "${ALL_PHASE_IDS[$((input - 1))]}"
            return 0
        fi

        local phase_id
        for phase_id in "${ALL_PHASE_IDS[@]}"; do
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

    # Handle "from" selection: prompt for starting phase
    if [[ "$PHASE_ID" == "from" ]]; then
        prompt_from_phase FROM_PHASE "${FROM_PHASE:-1}"
        PHASE_ID=""  # Clear so resolve_phases_to_run uses FROM_PHASE
    fi

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

    # Resolve phases early so we can display them in the summary
    resolve_phases_to_run

    echo ""
    echo "Selected configuration:"
    if [[ ${#PHASES_TO_RUN[@]} -gt 1 ]]; then
        local phases_display
        phases_display="$(printf '%s → ' "${PHASES_TO_RUN[@]}")"
        phases_display="${phases_display% → }"
        echo "  phases=$phases_display (${#PHASES_TO_RUN[@]} phases)"
    else
        echo "  phase=${PHASES_TO_RUN[0]} ($(phase_title "${PHASES_TO_RUN[0]}"))"
    fi
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
    for phase in "${PHASES_TO_RUN[@]}"; do
        if ! validate_single_phase "$phase"; then
            die "Invalid phase '$phase' in run list. Expected one of: ${ALL_PHASE_IDS[*]}"
        fi
    done

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

    # In multi-phase mode, chain from the previous phase's branch
    local base_ref
    if [[ -n "$CHAIN_BASE" ]]; then
        base_ref="$CHAIN_BASE"
    else
        base_ref="$(resolve_base_ref)"
    fi

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
            --from-phase)
                FROM_PHASE="$2"
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

    if [[ -z "$PHASE_ID" && -z "$FROM_PHASE" ]] && is_interactive_terminal; then
        INTERACTIVE="true"
        run_interactive_wizard
        return 0
    fi

    if [[ -z "$PHASE_ID" && -z "$FROM_PHASE" ]]; then
        die "--phase or --from-phase is required (or run with --interactive)"
    fi
}

run_single_phase() {
    local phase="$1"

    # Reset per-phase state
    PHASE_ID="$phase"
    IMPLEMENT_STATUS="not-run"
    REVIEW_VERDICT="NOT_RUN"
    REVIEW_CYCLES=0
    FIX_CYCLES=0
    PR_STATUS="not-run"
    EXPECTED_BRANCH=""

    init_artifacts

    log "Starting phase $PHASE_ID: $(phase_title "$PHASE_ID")"

    ensure_clean_tree_before_bootstrap
    bootstrap_branch
    run_implementation
    run_review_and_fix_cycles
    run_pr_creation

    log "Phase $PHASE_ID complete: impl=$IMPLEMENT_STATUS review=$REVIEW_VERDICT pr=$PR_STATUS"

    if is_blocking_verdict "$REVIEW_VERDICT"; then
        log "WARNING: Phase $PHASE_ID ended with blocking review verdict: $REVIEW_VERDICT"
    fi

    # Update chain base so next phase branches from this phase's branch
    CHAIN_BASE="$EXPECTED_BRANCH"
}

main() {
    parse_args "$@"
    resolve_interactive_mode

    # resolve_phases_to_run is called in the wizard for interactive mode;
    # for non-interactive, call it here
    if [[ "$INTERACTIVE" != "true" ]]; then
        resolve_phases_to_run
    fi

    preflight

    local total=${#PHASES_TO_RUN[@]}
    if [[ $total -gt 1 ]]; then
        local phases_display
        phases_display="$(printf '%s → ' "${PHASES_TO_RUN[@]}")"
        phases_display="${phases_display% → }"
        log "Starting multi-phase pipeline: $phases_display ($total phases) base=$BASE_BRANCH dry_run=$DRY_RUN"
    else
        log "Starting phase pipeline: phase=${PHASES_TO_RUN[0]} base=$BASE_BRANCH dry_run=$DRY_RUN"
    fi

    sync_base_branch

    local idx=0
    CHAIN_BASE=""

    for phase in "${PHASES_TO_RUN[@]}"; do
        idx=$((idx + 1))

        if [[ $total -gt 1 ]]; then
            log "═══════════════════════════════════════════════════"
            log "Phase $idx/$total: $phase - $(phase_title "$phase")"
            log "═══════════════════════════════════════════════════"
        fi

        run_single_phase "$phase"
    done

    if [[ $total -gt 1 ]]; then
        log "All $total phases complete."
    else
        log "Pipeline complete. Metadata: $METADATA_FILE"
        log "Artifacts: $RUN_DIR"
    fi
}

main "$@"
