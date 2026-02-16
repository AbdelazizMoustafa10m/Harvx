#!/usr/bin/env bash
#
# ralph_codex.sh -- Ralph Wiggum Loop for Harvx (Codex CLI)
#
# Runs OpenAI Codex CLI in an autonomous loop, implementing one task per iteration.
# Each iteration gets a fresh context window via `codex exec`. Memory persists via
# files on disk and git history.
#
# Usage:
#   ./scripts/ralph_codex.sh --phase 1                    # Run Phase 1 tasks
#   ./scripts/ralph_codex.sh --phase 2a --max-iterations 5
#   ./scripts/ralph_codex.sh --task T-003                  # Run single task
#   ./scripts/ralph_codex.sh --phase 1 --dry-run           # Preview prompt only
#   ./scripts/ralph_codex.sh --phase all                   # Run all phases sequentially
#   ./scripts/ralph_codex.sh --model o3                    # Use specific model
#
# Prerequisites:
#   - Codex CLI installed (`codex` command available)
#   - CODEX_API_KEY or authenticated via `codex login`
#   - Git repository initialized
#   - docs/tasks/ populated with task specs and PROGRESS.md
#

set -euo pipefail

# =============================================================================
# Configuration
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROMPT_TEMPLATE="$SCRIPT_DIR/RALPH-PROMPT-CODEX.md"
LOG_DIR="$PROJECT_ROOT/scripts/logs"
PROGRESS_FILE="$PROJECT_ROOT/docs/tasks/PROGRESS.md"

DEFAULT_MAX_ITERATIONS=20
DEFAULT_MODEL=""  # empty = use codex default
SLEEP_BETWEEN_ITERATIONS=5
COOLDOWN_AFTER_ERROR=30

# =============================================================================
# Phase Definitions
# =============================================================================

declare_phases() {
    # Phase ID -> Task Range
    PHASE_RANGES_1="T-001:T-015"
    PHASE_RANGES_2a="T-016:T-025"
    PHASE_RANGES_2b="T-026:T-033"
    PHASE_RANGES_3a="T-034:T-041"
    PHASE_RANGES_3b="T-042:T-050"
    PHASE_RANGES_4a="T-051:T-058"
    PHASE_RANGES_4b="T-059:T-065"
    PHASE_RANGES_5a="T-066:T-078"
    PHASE_RANGES_5b="T-079:T-087"
    PHASE_RANGES_6="T-088:T-095"

    # Phase ID -> Display Name
    PHASE_NAMES_1="Phase 1: Foundation"
    PHASE_NAMES_2a="Phase 2a: Profiles"
    PHASE_NAMES_2b="Phase 2b: Relevance & Tokens"
    PHASE_NAMES_3a="Phase 3a: Security"
    PHASE_NAMES_3b="Phase 3b: Compression"
    PHASE_NAMES_4a="Phase 4a: Output & Rendering"
    PHASE_NAMES_4b="Phase 4b: State & Diff"
    PHASE_NAMES_5a="Phase 5a: Workflows"
    PHASE_NAMES_5b="Phase 5b: Interactive TUI"
    PHASE_NAMES_6="Phase 6: Polish & Distribution"

    ALL_PHASES="1 2a 2b 3a 3b 4a 4b 5a 5b 6"
}

get_phase_range() {
    local phase="$1"
    local var="PHASE_RANGES_${phase}"
    echo "${!var:-}"
}

get_phase_name() {
    local phase="$1"
    local var="PHASE_NAMES_${phase}"
    echo "${!var:-}"
}

# =============================================================================
# Task List Generation
# =============================================================================

get_task_list_for_range() {
    local range="$1"
    local start="${range%%:*}"
    local end="${range##*:}"

    # Extract numeric parts
    local start_num="${start#T-}"
    local end_num="${end#T-}"

    # Remove leading zeros for arithmetic
    start_num=$((10#$start_num))
    end_num=$((10#$end_num))

    local task_list=""
    for ((i = start_num; i <= end_num; i++)); do
        local task_id=$(printf "T-%03d" "$i")
        # Find the task file
        local task_file=$(find "$PROJECT_ROOT/docs/tasks" -name "${task_id}-*.md" -type f 2>/dev/null | head -1)
        if [[ -n "$task_file" ]]; then
            local task_name=$(head -1 "$task_file" | sed 's/^# //' | sed "s/^${task_id}: //")
            # Check status in PROGRESS.md
            local status="Not Started"
            if grep -q "${task_id}.*Completed" "$PROGRESS_FILE" 2>/dev/null; then
                status="Completed"
            fi
            task_list+="- [ ] ${task_id}: ${task_name} [${status}]"$'\n'
        fi
    done
    echo "$task_list"
}

get_task_list_for_single() {
    local task_id="$1"
    local task_file=$(find "$PROJECT_ROOT/docs/tasks" -name "${task_id}-*.md" -type f 2>/dev/null | head -1)
    if [[ -n "$task_file" ]]; then
        local task_name=$(head -1 "$task_file" | sed 's/^# //' | sed "s/^${task_id}: //")
        echo "- [ ] ${task_id}: ${task_name}"
    fi
}

# =============================================================================
# Prompt Generation
# =============================================================================

generate_prompt() {
    local phase_id="$1"
    local phase_name="$2"
    local task_range="$3"
    local task_list="$4"

    local prompt
    prompt=$(cat "$PROMPT_TEMPLATE")

    # Replace placeholders
    prompt="${prompt//\{\{PHASE_NAME\}\}/$phase_name}"
    prompt="${prompt//\{\{TASK_RANGE\}\}/$task_range}"
    prompt="${prompt//\{\{PHASE_ID\}\}/$phase_id}"
    prompt="${prompt//\{\{TASK_LIST\}\}/$task_list}"

    echo "$prompt"
}

# =============================================================================
# Progress Checking
# =============================================================================

count_remaining_tasks() {
    local range="$1"
    local start="${range%%:*}"
    local end="${range##*:}"
    local start_num=$((10#${start#T-}))
    local end_num=$((10#${end#T-}))

    local remaining=0
    for ((i = start_num; i <= end_num; i++)); do
        local task_id=$(printf "T-%03d" "$i")
        if ! grep -q "${task_id}.*Completed" "$PROGRESS_FILE" 2>/dev/null; then
            remaining=$((remaining + 1))
        fi
    done
    echo "$remaining"
}

# =============================================================================
# Logging
# =============================================================================

setup_logging() {
    mkdir -p "$LOG_DIR"
    LOG_FILE="$LOG_DIR/ralph-codex-$(date +%Y%m%d-%H%M%S).log"
    echo "Logging to: $LOG_FILE"
}

log() {
    local msg="$1"
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $msg" | tee -a "$LOG_FILE"
}

# =============================================================================
# Codex Execution
# =============================================================================

build_codex_cmd() {
    local prompt_file="$1"
    local model="$2"

    # Base command: codex exec with full-auto and ephemeral session
    local cmd="codex exec"
    cmd+=" --full-auto"
    cmd+=" --ephemeral"

    # Model override
    if [[ -n "$model" ]]; then
        cmd+=" -m $model"
    fi

    # The prompt is passed as a positional argument via command substitution
    # codex exec doesn't support stdin piping natively
    cmd+=" \"\$(cat '$prompt_file')\""

    echo "$cmd"
}

run_codex() {
    local prompt_file="$1"
    local model="$2"

    local args=()
    args+=(exec)
    args+=(--full-auto)
    args+=(--ephemeral)

    if [[ -n "$model" ]]; then
        args+=(-m "$model")
    fi

    # Read prompt from file and pass as positional argument
    local prompt_content
    prompt_content=$(cat "$prompt_file")

    codex "${args[@]}" "$prompt_content" 2>&1
}

# =============================================================================
# Main Loop
# =============================================================================

run_ralph_loop() {
    local phase_id="$1"
    local max_iterations="$2"
    local dry_run="$3"
    local model="$4"

    local phase_name
    phase_name=$(get_phase_name "$phase_id")
    local task_range
    task_range=$(get_phase_range "$phase_id")

    if [[ -z "$phase_name" || -z "$task_range" ]]; then
        echo "ERROR: Unknown phase '$phase_id'"
        echo "Valid phases: $ALL_PHASES"
        exit 1
    fi

    log "=========================================="
    log "Ralph Loop Starting (Codex CLI)"
    log "Phase: $phase_name ($phase_id)"
    log "Task Range: $task_range"
    log "Max Iterations: $max_iterations"
    if [[ -n "$model" ]]; then
        log "Model: $model"
    fi
    log "=========================================="

    local iteration=0
    local tasks_completed=0
    local consecutive_errors=0

    while [[ $iteration -lt $max_iterations ]]; do
        iteration=$((iteration + 1))

        local remaining
        remaining=$(count_remaining_tasks "$task_range")
        log ""
        log "--- Iteration $iteration / $max_iterations (${remaining} tasks remaining) ---"

        if [[ "$remaining" -eq 0 ]]; then
            log "All tasks in $phase_name are complete!"
            break
        fi

        # Generate fresh task list each iteration
        local task_list
        task_list=$(get_task_list_for_range "$task_range")

        # Generate prompt
        local prompt
        prompt=$(generate_prompt "$phase_id" "$phase_name" "$task_range" "$task_list")

        if [[ "$dry_run" == "true" ]]; then
            log "DRY RUN -- Generated prompt:"
            echo "---"
            echo "$prompt"
            echo "---"
            echo ""
            echo "Codex command: codex exec --full-auto --ephemeral${model:+ -m $model} \"<prompt>\""
            exit 0
        fi

        # Write prompt to temp file (avoids shell escaping issues)
        local prompt_file
        prompt_file=$(mktemp /tmp/ralph-codex-prompt-XXXXXX.md)
        echo "$prompt" > "$prompt_file"

        # Run Codex CLI with the prompt
        log "Spawning Codex CLI (iteration $iteration)..."
        local start_time
        start_time=$(date +%s)

        local output=""
        local exit_code=0
        output=$(run_codex "$prompt_file" "$model") || exit_code=$?

        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))

        # Clean up temp file
        rm -f "$prompt_file"

        # Log output to file
        echo "$output" >> "$LOG_FILE"

        log "Codex CLI exited (code=$exit_code, duration=${duration}s)"

        # Check for completion signals
        if echo "$output" | grep -q "PHASE_COMPLETE"; then
            log "PHASE_COMPLETE signal received!"
            log "All tasks in $phase_name are done."
            break
        fi

        if echo "$output" | grep -q "RALPH_ERROR"; then
            local error_msg
            error_msg=$(echo "$output" | grep "RALPH_ERROR" | head -1)
            log "ERROR: $error_msg"
            consecutive_errors=$((consecutive_errors + 1))

            if [[ $consecutive_errors -ge 3 ]]; then
                log "ABORT: 3 consecutive errors. Stopping loop."
                exit 1
            fi

            log "Cooling down for ${COOLDOWN_AFTER_ERROR}s before retry..."
            sleep "$COOLDOWN_AFTER_ERROR"
            continue
        fi

        if echo "$output" | grep -q "TASK_BLOCKED"; then
            local blocked_msg
            blocked_msg=$(echo "$output" | grep "TASK_BLOCKED" | head -1)
            log "BLOCKED: $blocked_msg"
            log "No more tasks available in this phase. Stopping."
            break
        fi

        # Check if a task was actually completed (look for PROGRESS.md update)
        local new_remaining
        new_remaining=$(count_remaining_tasks "$task_range")
        if [[ "$new_remaining" -lt "$remaining" ]]; then
            tasks_completed=$((tasks_completed + (remaining - new_remaining)))
            consecutive_errors=0
            log "Task completed! (total completed this session: $tasks_completed)"
        else
            log "WARNING: No task completed in this iteration."
            consecutive_errors=$((consecutive_errors + 1))

            if [[ $consecutive_errors -ge 3 ]]; then
                log "ABORT: 3 iterations without progress. Stopping."
                exit 1
            fi
        fi

        if [[ $iteration -lt $max_iterations ]]; then
            log "Sleeping ${SLEEP_BETWEEN_ITERATIONS}s before next iteration..."
            sleep "$SLEEP_BETWEEN_ITERATIONS"
        fi
    done

    log ""
    log "=========================================="
    log "Ralph Loop Complete (Codex CLI)"
    log "Phase: $phase_name"
    log "Iterations: $iteration"
    log "Tasks Completed: $tasks_completed"
    local final_remaining
    final_remaining=$(count_remaining_tasks "$task_range")
    log "Tasks Remaining: $final_remaining"
    log "=========================================="
}

run_single_task() {
    local task_id="$1"
    local dry_run="$2"
    local model="$3"

    log "=========================================="
    log "Ralph Single Task Mode (Codex CLI)"
    log "Task: $task_id"
    if [[ -n "$model" ]]; then
        log "Model: $model"
    fi
    log "=========================================="

    local task_list
    task_list=$(get_task_list_for_single "$task_id")

    if [[ -z "$task_list" ]]; then
        log "ERROR: Task $task_id not found in docs/tasks/"
        exit 1
    fi

    # Determine phase from task number
    local task_num=$((10#${task_id#T-}))
    local phase_id=""
    if   [[ $task_num -le 15 ]];  then phase_id="1"
    elif [[ $task_num -le 25 ]];  then phase_id="2a"
    elif [[ $task_num -le 33 ]];  then phase_id="2b"
    elif [[ $task_num -le 41 ]];  then phase_id="3a"
    elif [[ $task_num -le 50 ]];  then phase_id="3b"
    elif [[ $task_num -le 58 ]];  then phase_id="4a"
    elif [[ $task_num -le 65 ]];  then phase_id="4b"
    elif [[ $task_num -le 78 ]];  then phase_id="5a"
    elif [[ $task_num -le 87 ]];  then phase_id="5b"
    elif [[ $task_num -le 95 ]];  then phase_id="6"
    fi

    local phase_name
    phase_name=$(get_phase_name "$phase_id")
    local task_range="${task_id}:${task_id}"

    local prompt
    prompt=$(generate_prompt "$phase_id" "$phase_name" "$task_range" "$task_list")

    if [[ "$dry_run" == "true" ]]; then
        log "DRY RUN -- Generated prompt:"
        echo "---"
        echo "$prompt"
        echo "---"
        echo ""
        echo "Codex command: codex exec --full-auto --ephemeral${model:+ -m $model} \"<prompt>\""
        exit 0
    fi

    local prompt_file
    prompt_file=$(mktemp /tmp/ralph-codex-prompt-XXXXXX.md)
    echo "$prompt" > "$prompt_file"

    log "Spawning Codex CLI for $task_id..."
    run_codex "$prompt_file" "$model" | tee -a "$LOG_FILE"

    rm -f "$prompt_file"
    log "Single task run complete."
}

run_all_phases() {
    local max_iterations="$1"
    local dry_run="$2"
    local model="$3"

    for phase in $ALL_PHASES; do
        local remaining
        remaining=$(count_remaining_tasks "$(get_phase_range "$phase")")

        if [[ "$remaining" -eq 0 ]]; then
            log "Skipping $(get_phase_name "$phase") -- all tasks complete"
            continue
        fi

        log "Starting $(get_phase_name "$phase") ($remaining tasks remaining)"
        run_ralph_loop "$phase" "$max_iterations" "$dry_run" "$model"

        # Check if phase completed
        remaining=$(count_remaining_tasks "$(get_phase_range "$phase")")
        if [[ "$remaining" -gt 0 ]]; then
            log "Phase incomplete ($remaining tasks remaining). Stopping sequential run."
            exit 1
        fi
    done

    log "ALL PHASES COMPLETE!"
}

# =============================================================================
# CLI Argument Parsing
# =============================================================================

usage() {
    cat <<'EOF'
Ralph Wiggum Loop for Harvx (Codex CLI)

Usage:
  ./scripts/ralph_codex.sh --phase <phase>  [options]
  ./scripts/ralph_codex.sh --task <task-id> [options]
  ./scripts/ralph_codex.sh --status

Phases:
  1    Foundation (T-001 to T-015)
  2a   Profiles (T-016 to T-025)
  2b   Relevance & Tokens (T-026 to T-033)
  3a   Security (T-034 to T-041)
  3b   Compression (T-042 to T-050)
  4a   Output & Rendering (T-051 to T-058)
  4b   State & Diff (T-059 to T-065)
  5a   Workflows (T-066 to T-078)
  5b   Interactive TUI (T-079 to T-087)
  6    Polish & Distribution (T-088 to T-095)
  all  Run all phases sequentially

Options:
  --phase <id>         Phase to run (required unless --task)
  --task <T-XXX>       Run a single specific task
  --max-iterations <n> Max loop iterations (default: 20)
  --model <name>       Model to use (e.g., o3, o4-mini, gpt-4.1)
  --dry-run            Print generated prompt without running
  --status             Show task completion status and exit
  -h, --help           Show this help

Examples:
  ./scripts/ralph_codex.sh --phase 1                        # Run all Phase 1 tasks
  ./scripts/ralph_codex.sh --phase 1 --model o3             # Use o3 model
  ./scripts/ralph_codex.sh --phase 1 --max-iterations 5     # Cap at 5 iterations
  ./scripts/ralph_codex.sh --task T-003                      # Run single task T-003
  ./scripts/ralph_codex.sh --phase 1 --dry-run               # Preview the prompt
  ./scripts/ralph_codex.sh --phase all                       # Run all phases sequentially
  ./scripts/ralph_codex.sh --status                          # Show completion status
EOF
}

show_status() {
    echo ""
    echo "Harvx Task Status"
    echo "=================="
    echo ""

    for phase in $ALL_PHASES; do
        local range
        range=$(get_phase_range "$phase")
        local name
        name=$(get_phase_name "$phase")
        local start="${range%%:*}"
        local end="${range##*:}"
        local start_num=$((10#${start#T-}))
        local end_num=$((10#${end#T-}))
        local total=$((end_num - start_num + 1))
        local remaining
        remaining=$(count_remaining_tasks "$range")
        local completed=$((total - remaining))

        local pct=0
        if [[ $total -gt 0 ]]; then
            pct=$((completed * 100 / total))
        fi

        # Simple progress bar (20 chars wide)
        local filled=$((pct / 5))
        local empty=$((20 - filled))
        local bar=""
        local i
        for ((i = 0; i < filled; i++)); do bar+="#"; done
        for ((i = 0; i < empty; i++)); do bar+="-"; done

        printf "  %-35s [%-20s] %3d%% (%d/%d)\n" "$name" "$bar" "$pct" "$completed" "$total"
    done

    echo ""

    # Total
    local total_tasks=95
    local total_completed=0
    for p in $ALL_PHASES; do
        local r
        r=$(get_phase_range "$p")
        local s="${r%%:*}"
        local e="${r##*:}"
        local sn=$((10#${s#T-}))
        local en=$((10#${e#T-}))
        local t=$((en - sn + 1))
        local rem
        rem=$(count_remaining_tasks "$r")
        total_completed=$((total_completed + t - rem))
    done
    local total_pct=$((total_completed * 100 / total_tasks))

    echo "  Total: $total_completed/$total_tasks tasks completed ($total_pct%)"
    echo ""
}

main() {
    declare_phases

    local phase=""
    local task=""
    local max_iterations=$DEFAULT_MAX_ITERATIONS
    local model="$DEFAULT_MODEL"
    local dry_run="false"
    local show_status_flag="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --phase)
                phase="$2"
                shift 2
                ;;
            --task)
                task="$2"
                shift 2
                ;;
            --max-iterations)
                max_iterations="$2"
                shift 2
                ;;
            --model)
                model="$2"
                shift 2
                ;;
            --dry-run)
                dry_run="true"
                shift
                ;;
            --status)
                show_status_flag="true"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done

    # Status mode
    if [[ "$show_status_flag" == "true" ]]; then
        show_status
        exit 0
    fi

    # Validate arguments
    if [[ -z "$phase" && -z "$task" ]]; then
        echo "ERROR: Must specify --phase or --task"
        echo ""
        usage
        exit 1
    fi

    # Check prerequisites
    if ! command -v codex &>/dev/null; then
        echo "ERROR: 'codex' CLI not found. Install Codex CLI first."
        echo ""
        echo "  npm install -g @openai/codex"
        echo ""
        echo "  Or see: https://developers.openai.com/codex/cli/"
        exit 1
    fi

    if [[ ! -f "$PROGRESS_FILE" ]]; then
        echo "ERROR: PROGRESS.md not found at $PROGRESS_FILE"
        exit 1
    fi

    if [[ ! -f "$PROMPT_TEMPLATE" ]]; then
        echo "ERROR: Prompt template not found at $PROMPT_TEMPLATE"
        exit 1
    fi

    # Change to project root
    cd "$PROJECT_ROOT"

    setup_logging

    # Run
    if [[ -n "$task" ]]; then
        run_single_task "$task" "$dry_run" "$model"
    elif [[ "$phase" == "all" ]]; then
        run_all_phases "$max_iterations" "$dry_run" "$model"
    else
        run_ralph_loop "$phase" "$max_iterations" "$dry_run" "$model"
    fi
}

main "$@"
