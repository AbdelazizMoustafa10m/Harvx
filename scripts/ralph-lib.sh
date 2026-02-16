#!/usr/bin/env bash
#
# ralph-lib.sh -- Shared library for Ralph Wiggum Loop scripts
#
# This file contains all shared logic used by both ralph_claude.sh and
# ralph_codex.sh. Each caller script must set the following variables before
# sourcing this file:
#
#   SCRIPT_DIR          - Directory containing the calling script
#   PROJECT_ROOT        - Repository root
#   PROMPT_TEMPLATE     - Path to the prompt template file
#   LOG_DIR             - Directory for log files
#   PROGRESS_FILE       - Path to PROGRESS.md
#   DEFAULT_MODEL       - Default model name (e.g., "claude-opus-4-6")
#   DEFAULT_EFFORT      - Default effort/reasoning level (e.g., "high")
#   EFFORT_LABEL        - Display label: "Effort" or "Reasoning"
#   LOG_PREFIX          - Log filename prefix: "ralph-claude" or "ralph-codex"
#   AGENT_LABEL         - Display label: "Claude Code" or "Codex CLI"
#   AGENT_CMD           - Agent command name: "claude" or "codex"
#
# Each caller must also define:
#   run_agent()         - Function that runs the agent; receives prompt_file,
#                         model, effort as args; outputs to stdout
#   usage()             - Function that prints script-specific usage
#   check_prerequisites() - Function that checks agent-specific prerequisites
#   pre_agent_setup()   - Function called before agent invocation (e.g., export env vars)
#   get_dry_run_command() - Function that returns the dry-run command display string
#

# Guard against double-sourcing
if [[ -n "${_RALPH_LIB_LOADED:-}" ]]; then
    return 0
fi
_RALPH_LIB_LOADED=1

# =============================================================================
# Shared Configuration Defaults
# =============================================================================

DEFAULT_MAX_ITERATIONS=20
SLEEP_BETWEEN_ITERATIONS=5
COOLDOWN_AFTER_ERROR=30

# Rate-limit handling
DEFAULT_MAX_LIMIT_WAITS=2
RATE_LIMIT_BUFFER_SECONDS=120  # 2 minute safety buffer after parsed reset time
BACKOFF_SCHEDULE=(120 300 900 1800)  # 2m, 5m, 15m, 30m
MAX_RATE_LIMIT_WAIT_SECONDS=21600  # 6 hours -- cap on parsed wait time

# =============================================================================
# Signal Handling
# =============================================================================

# Track temp files for cleanup
_RALPH_TEMP_FILES=()

# Register a temp file for cleanup on exit/signal.
ralph_register_temp_file() {
    _RALPH_TEMP_FILES+=("$1")
}

# Cleanup handler for signals and exit.
_ralph_cleanup() {
    local sig="${1:-EXIT}"

    # Prevent re-entrant cleanup (EXIT fires after signal handler calls exit)
    if [[ -n "${_RALPH_CLEANING_UP:-}" ]]; then
        return 0
    fi
    _RALPH_CLEANING_UP=1

    # Clean up temp prompt files
    if [[ ${#_RALPH_TEMP_FILES[@]} -gt 0 ]]; then
        for f in "${_RALPH_TEMP_FILES[@]}"; do
            rm -f "$f" 2>/dev/null
        done
    fi

    if [[ "$sig" != "EXIT" ]]; then
        # Log if LOG_FILE is set (logging has been initialized)
        if [[ -n "${LOG_FILE:-}" ]]; then
            local timestamp
            timestamp=$(date '+%Y-%m-%d %H:%M:%S')
            echo "[$timestamp] SIGNAL: Received $sig, shutting down..." >> "$LOG_FILE" 2>/dev/null
        fi
        echo ""
        echo "Signal $sig received, shutting down..."

        # Warn about dirty tree but do NOT auto-stash (let the user decide)
        if command -v git &>/dev/null && git rev-parse --is-inside-work-tree &>/dev/null 2>&1; then
            local status
            status=$(git status --porcelain 2>/dev/null)
            if [[ -n "$status" ]]; then
                echo "WARNING: Working tree has uncommitted changes. Review with 'git status'."
            fi
        fi

        exit 130
    fi
}

# Install signal traps. Called from main() after setup.
_ralph_install_traps() {
    trap '_ralph_cleanup SIGINT' SIGINT
    trap '_ralph_cleanup SIGTERM' SIGTERM
    trap '_ralph_cleanup EXIT' EXIT
}

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
# Task Selection
# =============================================================================

is_task_completed() {
    local task_id="$1"
    grep -q "${task_id}.*Completed" "$PROGRESS_FILE" 2>/dev/null
}

get_task_file() {
    local task_id="$1"
    find "$PROJECT_ROOT/docs/tasks" -name "${task_id}-*.md" -type f 2>/dev/null | head -1
}

get_task_name() {
    local task_id="$1"
    local task_file
    task_file=$(get_task_file "$task_id")
    if [[ -z "$task_file" ]]; then
        return 0
    fi
    head -1 "$task_file" | sed 's/^# //' | sed "s/^${task_id}: //"
}

get_missing_dependencies() {
    local task_id="$1"
    local task_file
    task_file=$(get_task_file "$task_id")
    if [[ -z "$task_file" ]]; then
        return 0
    fi

    local dep_line
    dep_line=$(grep -m1 '^\*\*Dependencies:\*\*' "$task_file" 2>/dev/null || true)
    if [[ -z "$dep_line" ]]; then
        return 0
    fi

    local missing=()
    while IFS= read -r dep; do
        [[ -z "$dep" ]] && continue
        if ! is_task_completed "$dep"; then
            missing+=("$dep")
        fi
    done < <(echo "$dep_line" | grep -oE 'T-[0-9]{3}' || true)

    echo "${missing[*]-}"
}

select_next_task() {
    local range="$1"
    local start="${range%%:*}"
    local end="${range##*:}"
    local start_num=$((10#${start#T-}))
    local end_num=$((10#${end#T-}))

    local first_blocked_task=""
    local first_blocked_missing=""

    for ((i = start_num; i <= end_num; i++)); do
        local task_id
        task_id=$(printf "T-%03d" "$i")

        if is_task_completed "$task_id"; then
            continue
        fi

        local missing
        missing=$(get_missing_dependencies "$task_id")
        if [[ -z "$missing" ]]; then
            echo "$task_id"
            return 0
        fi

        if [[ -z "$first_blocked_task" ]]; then
            first_blocked_task="$task_id"
            first_blocked_missing="$missing"
        fi
    done

    if [[ -n "$first_blocked_task" ]]; then
        echo "BLOCKED:${first_blocked_task}:${first_blocked_missing}"
    fi
}

get_task_list_for_single() {
    local task_id="$1"
    local task_file
    task_file=$(get_task_file "$task_id")
    if [[ -n "$task_file" ]]; then
        local task_name
        task_name=$(get_task_name "$task_id")
        echo "- [ ] ${task_id}: ${task_name} [Not Started]"
    fi
}

# Determine phase ID from task number.
get_phase_for_task() {
    local task_id="$1"
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
    echo "$phase_id"
}

# =============================================================================
# Prompt Generation
# =============================================================================

generate_prompt() {
    local phase_id="$1"
    local phase_name="$2"
    local task_range="$3"
    local task_list="$4"
    local task_id="$5"

    local prompt
    prompt=$(cat "$PROMPT_TEMPLATE")

    # Replace placeholders
    prompt="${prompt//\{\{PHASE_NAME\}\}/$phase_name}"
    prompt="${prompt//\{\{TASK_RANGE\}\}/$task_range}"
    prompt="${prompt//\{\{PHASE_ID\}\}/$phase_id}"
    prompt="${prompt//\{\{TASK_LIST\}\}/$task_list}"
    prompt="${prompt//\{\{TASK_ID\}\}/$task_id}"

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

# Delete log files older than 14 days.
rotate_logs() {
    find "$LOG_DIR" -name "ralph-*.log" -type f -mtime +14 -delete 2>/dev/null || true
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    rotate_logs
    LOG_FILE="$LOG_DIR/${LOG_PREFIX}-$(date +%Y%m%d-%H%M%S).log"
    echo "Logging to: $LOG_FILE"
}

log() {
    local msg="$1"
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $msg" | tee -a "$LOG_FILE"
}

# =============================================================================
# Visual Model Banner
# =============================================================================

# Print a configuration banner.
# Uses EFFORT_LABEL (set by each script) for the second row label.
print_model_banner() {
    local model="$1"
    local level="$2"
    local phase="$3"

    echo ""
    echo "┌──────────────────────────────────────────────────┐"
    echo "│            MODEL CONFIGURATION                   │"
    echo "├──────────────────────────────────────────────────┤"
    printf "│  %-10s %-38s│\n" "Model:" "$model"
    printf "│  %-10s %-38s│\n" "${EFFORT_LABEL}:" "$level"
    printf "│  %-10s %-38s│\n" "Phase:" "$phase"
    echo "└──────────────────────────────────────────────────┘"
    echo ""
}

# =============================================================================
# Rate-Limit Detection & Recovery
# =============================================================================

# Detect rate-limit messages in agent output.
# Returns 0 (true) if rate limit detected, 1 (false) otherwise.
is_rate_limited() {
    local output="$1"
    # Claude Code patterns
    if echo "$output" | grep -qi "you've hit your limit"; then
        return 0
    fi
    if echo "$output" | grep -qi "hit your usage limit"; then
        return 0
    fi
    if echo "$output" | grep -qi "usage limit.*resets"; then
        return 0
    fi
    # Tightened pattern: match specific rate-limit phrases to avoid false positives
    # from agent-generated code/comments that casually mention "rate limit"
    if echo "$output" | grep -qi "rate limit exceeded\|rate.limited\|rate_limit"; then
        return 0
    fi
    # Codex patterns
    if echo "$output" | grep -qiE "try again in.*(days|hours|minutes)"; then
        return 0
    fi
    if echo "$output" | grep -qi "upgrade to pro"; then
        return 0
    fi
    return 1
}

# Parse reset timestamp from Claude Code output.
# Prints seconds until reset, or empty string if unparseable.
# Matches patterns like "resets 7pm (Europe/Berlin)" or "resets 7pm".
#
# Timezone handling approach:
#   On macOS, `date -j -f` converts a time string to epoch, but always
#   interprets the time in the current system timezone. To handle an
#   explicit timezone (e.g., "Europe/Berlin"), we set TZ for the `date`
#   invocations so that both "now" and the target time are computed in
#   the same timezone. If TZ parsing fails, we fall back to local time.
#   If the entire parse fails, the caller falls back to backoff schedule.
parse_claude_reset_time() {
    local output="$1"

    # Try "resets Xpm (Timezone)" first, then bare "resets Xpm"
    local reset_match
    reset_match=$(echo "$output" | grep -oiE 'resets [0-9]{1,2}(:[0-9]{2})?(am|pm) \([^)]+\)' | head -1)
    if [[ -z "$reset_match" ]]; then
        reset_match=$(echo "$output" | grep -oiE 'resets [0-9]{1,2}(:[0-9]{2})?(am|pm)' | head -1)
    fi
    if [[ -z "$reset_match" ]]; then
        echo ""
        return 1
    fi

    # Extract time portion and optional timezone
    local time_str tz_str
    time_str=$(echo "$reset_match" | grep -oiE '[0-9]{1,2}(:[0-9]{2})?(am|pm)')
    tz_str=$(echo "$reset_match" | grep -oE '\([^)]+\)' | tr -d '()')

    # Normalize time_str: ensure HH:MM format
    local hour minute ampm
    ampm=$(echo "$time_str" | grep -oiE '(am|pm)' | tr '[:upper:]' '[:lower:]')
    local numeric_part
    numeric_part=$(echo "$time_str" | sed -E 's/(am|pm)//i')

    if echo "$numeric_part" | grep -q ':'; then
        hour=$(echo "$numeric_part" | cut -d: -f1)
        minute=$(echo "$numeric_part" | cut -d: -f2)
    else
        hour="$numeric_part"
        minute="00"
    fi

    # Convert to 24-hour
    hour=$((10#$hour))
    if [[ "$ampm" == "pm" && $hour -ne 12 ]]; then
        hour=$((hour + 12))
    elif [[ "$ampm" == "am" && $hour -eq 12 ]]; then
        hour=0
    fi

    # Build target time string
    local target_time
    target_time=$(printf "%02d:%02d:00" "$hour" "$minute")

    # Compute seconds until that time.
    # Strategy: Get today's date in the target timezone, combine with the
    # parsed time, and convert to epoch. Compare against "now" in the same TZ.
    local now_epoch target_epoch
    local effective_tz="${tz_str:-}"

    if [[ -n "$effective_tz" ]]; then
        # Verify the timezone is valid by attempting to use it
        if ! TZ="$effective_tz" date +%s &>/dev/null; then
            # Invalid timezone -- fall back to local time
            effective_tz=""
        fi
    fi

    if [[ -n "$effective_tz" ]]; then
        now_epoch=$(TZ="$effective_tz" date +%s)
        # Get today's date in the target timezone
        local today_date
        today_date=$(TZ="$effective_tz" date +%Y-%m-%d)
        local target_datetime="${today_date}T${target_time}"
        # macOS: use -j -f with full datetime format (date -j exists on macOS only)
        target_epoch=$(TZ="$effective_tz" date -j -f "%Y-%m-%dT%H:%M:%S" "$target_datetime" "+%s" 2>/dev/null || echo "")
        # GNU date fallback
        if [[ -z "$target_epoch" ]]; then
            target_epoch=$(TZ="$effective_tz" date -d "${today_date} ${target_time}" "+%s" 2>/dev/null || echo "")
        fi
    else
        now_epoch=$(date +%s)
        local today_date
        today_date=$(date +%Y-%m-%d)
        local target_datetime="${today_date}T${target_time}"
        # macOS: use -j -f with full datetime format
        target_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%S" "$target_datetime" "+%s" 2>/dev/null || echo "")
        # GNU date fallback
        if [[ -z "$target_epoch" ]]; then
            target_epoch=$(date -d "${today_date} ${target_time}" "+%s" 2>/dev/null || echo "")
        fi
    fi

    if [[ -z "$target_epoch" ]]; then
        echo ""
        return 1
    fi

    local diff=$((target_epoch - now_epoch))
    # If diff is negative, the reset is tomorrow
    if [[ $diff -lt 0 ]]; then
        diff=$((diff + 86400))
    fi

    echo "$diff"
    return 0
}

# Parse reset duration from Codex-style "try again in X days Y minutes" output.
# Prints total seconds to wait, or empty string if unparseable.
parse_codex_reset_time() {
    local output="$1"
    local total_seconds=0

    local days_match
    days_match=$(echo "$output" | grep -oE '[0-9]+ days?' | head -1 | grep -oE '[0-9]+')
    if [[ -n "$days_match" ]]; then
        total_seconds=$((total_seconds + days_match * 86400))
    fi

    local hours_match
    hours_match=$(echo "$output" | grep -oE '[0-9]+ hours?' | head -1 | grep -oE '[0-9]+')
    if [[ -n "$hours_match" ]]; then
        total_seconds=$((total_seconds + hours_match * 3600))
    fi

    local minutes_match
    minutes_match=$(echo "$output" | grep -oE '[0-9]+ minutes?' | head -1 | grep -oE '[0-9]+')
    if [[ -n "$minutes_match" ]]; then
        total_seconds=$((total_seconds + minutes_match * 60))
    fi

    if [[ $total_seconds -gt 0 ]]; then
        echo "$total_seconds"
        return 0
    fi
    echo ""
    return 1
}

# Bounded exponential backoff with jitter.
# Usage: get_backoff_seconds <attempt_index>
# Schedule: 2m, 5m, 15m, 30m (capped), plus 0-30s random jitter.
get_backoff_seconds() {
    local attempt="$1"
    local max_idx=$(( ${#BACKOFF_SCHEDULE[@]} - 1 ))
    local idx=$attempt
    if [[ $idx -gt $max_idx ]]; then
        idx=$max_idx
    fi
    # Add jitter: random 0-30s
    local jitter=$((RANDOM % 31))
    echo $(( ${BACKOFF_SCHEDULE[$idx]} + jitter ))
}

# Cap parsed wait time at MAX_RATE_LIMIT_WAIT_SECONDS to prevent sleeping
# excessively when the reset time was already in the past or mis-parsed.
# Arguments: wait_seconds
# Outputs: capped wait_seconds
cap_wait_seconds() {
    local wait_seconds="$1"
    if [[ "$wait_seconds" -gt "$MAX_RATE_LIMIT_WAIT_SECONDS" ]]; then
        log "RATE_LIMIT: Parsed wait time ${wait_seconds}s exceeds cap of ${MAX_RATE_LIMIT_WAIT_SECONDS}s. Using cap."
        echo "$MAX_RATE_LIMIT_WAIT_SECONDS"
    else
        echo "$wait_seconds"
    fi
}

# Wait for rate-limit reset with countdown display.
# Arguments: wait_seconds, wait_cycle (0-based), max_cycles
wait_for_rate_limit_reset() {
    local wait_seconds="$1"
    local wait_cycle="$2"
    local max_cycles="$3"

    if [[ $wait_cycle -ge $max_cycles ]]; then
        log "ABORT: Hit rate limit $((wait_cycle + 1)) times (max: $max_cycles). Stopping."
        exit 1
    fi

    local resume_time
    # macOS date uses -v, GNU date uses -d
    resume_time=$(date -v "+${wait_seconds}S" '+%H:%M:%S' 2>/dev/null \
               || date -d "+${wait_seconds} seconds" '+%H:%M:%S' 2>/dev/null \
               || echo "unknown")

    log "RATE_LIMIT: Waiting ${wait_seconds}s (resuming ~${resume_time}). Wait cycle $((wait_cycle + 1))/$max_cycles"

    # Countdown display every 60 seconds
    local remaining=$wait_seconds
    while [[ $remaining -gt 0 ]]; do
        local display_mins=$((remaining / 60))
        local display_secs=$((remaining % 60))
        printf "\r  Rate limit cooldown: %dm %ds remaining...  " "$display_mins" "$display_secs"

        local sleep_chunk=60
        if [[ $remaining -lt 60 ]]; then
            sleep_chunk=$remaining
        fi
        sleep "$sleep_chunk"
        remaining=$((remaining - sleep_chunk))
    done
    printf "\r  Rate limit cooldown: complete!                    \n"

    log "RATE_LIMIT: Cooldown finished. Resuming..."
}

# Compute the wait time from agent output during a rate limit event.
# Tries both Claude-style and Codex-style parsers, applies buffer and cap.
# Falls back to backoff schedule if parsing fails.
# Arguments: output, rate_limit_waits (attempt index)
# Outputs: wait_seconds
compute_rate_limit_wait() {
    local output="$1"
    local rate_limit_waits="$2"

    local wait_seconds=""
    # Try Claude-style reset time first, then Codex-style duration
    wait_seconds=$(parse_claude_reset_time "$output" 2>/dev/null) || true
    if [[ -z "$wait_seconds" ]]; then
        wait_seconds=$(parse_codex_reset_time "$output" 2>/dev/null) || true
    fi

    if [[ -n "$wait_seconds" ]] && [[ "$wait_seconds" -gt 0 ]] 2>/dev/null; then
        wait_seconds=$((wait_seconds + RATE_LIMIT_BUFFER_SECONDS))
        wait_seconds=$(cap_wait_seconds "$wait_seconds")
        log "RATE_LIMIT: Parsed reset time. Will wait ${wait_seconds}s (includes ${RATE_LIMIT_BUFFER_SECONDS}s buffer)."
    else
        wait_seconds=$(get_backoff_seconds "$rate_limit_waits")
        log "RATE_LIMIT: Could not parse reset time. Using backoff: ${wait_seconds}s."
    fi

    echo "$wait_seconds"
}

# =============================================================================
# Recovery Functions
# =============================================================================

# Check if the working tree has uncommitted changes.
is_tree_dirty() {
    local status
    status=$(git status --porcelain 2>/dev/null)
    [[ -n "$status" ]]
}

# Get a human-readable summary of dirty files for logging.
# Calls git status once and parses the output.
get_dirty_summary() {
    local status
    status=$(git status --porcelain 2>/dev/null)
    local modified added deleted
    modified=$(echo "$status" | grep -c '^ M\|^.M' || true)
    added=$(echo "$status" | grep -c '^A \|^??' || true)
    deleted=$(echo "$status" | grep -c '^ D\|^D ' || true)
    echo "${modified} modified, ${added} new/untracked, ${deleted} deleted"
}

# Attempt to auto-commit all pending changes as a recovery step.
# Use when agent updated PROGRESS.md but failed to create a commit.
# Arguments: task_id
# Returns 0 on success, 1 on failure.
run_commit_recovery() {
    local task_id="$1"

    if ! is_tree_dirty; then
        log "COMMIT_RECOVERY: Working tree is clean, nothing to commit."
        return 0
    fi

    local dirty_summary
    dirty_summary=$(get_dirty_summary)
    log "COMMIT_RECOVERY: Attempting auto-commit for $task_id ($dirty_summary)."

    # Stage all changes (respects .gitignore)
    if ! git add -A 2>/dev/null; then
        log "COMMIT_RECOVERY: git add -A failed."
        return 1
    fi

    # Check if anything is actually staged
    if git diff --cached --quiet 2>/dev/null; then
        log "COMMIT_RECOVERY: Nothing staged after git add (changes may be gitignored)."
        return 0
    fi

    # Create recovery commit
    if git commit -m "$(cat <<EOF
chore(recovery): auto-commit pending changes for ${task_id}

This commit was created by the Ralph loop recovery mechanism.
The agent completed ${task_id} but did not create a git commit
(likely due to permission restrictions in dontAsk mode).
EOF
    )" 2>/dev/null; then
        local new_head
        new_head=$(git rev-parse --short HEAD 2>/dev/null)
        log "COMMIT_RECOVERY: Successfully committed as $new_head."
        return 0
    else
        log "COMMIT_RECOVERY: git commit failed."
        return 1
    fi
}

# Stash dirty working tree changes, preserving them for later inspection.
# Use when agent was interrupted mid-task (e.g., rate limit hit).
# Arguments: context_msg (e.g., "rate-limit interrupt during T-014")
# Returns 0 on success, 1 if nothing to stash or stash failed.
stash_dirty_tree() {
    local context_msg="$1"

    if ! is_tree_dirty; then
        return 1
    fi

    local dirty_summary
    dirty_summary=$(get_dirty_summary)
    local stash_msg="ralph-recovery: ${context_msg} (${dirty_summary})"

    # Include untracked files in stash
    if git stash push -u -m "$stash_msg" 2>/dev/null; then
        local stash_ref
        stash_ref=$(git stash list | head -1 | cut -d: -f1)
        log "STASH_RECOVERY: Stashed partial work as $stash_ref"
        log "  Message: $stash_msg"
        log "  To inspect: git stash show -p $stash_ref"
        log "  To restore: git stash pop $stash_ref"
        return 0
    else
        log "STASH_RECOVERY: git stash failed."
        return 1
    fi
}

# Pre-iteration dirty tree check and recovery.
# If the tree is dirty at the start of an iteration, this means either:
#   (a) A prior iteration completed a task but didn't commit -> auto-commit
#   (b) A prior iteration was interrupted mid-task -> stash partial work
# Arguments: task_range, last_task_id (may be empty on first iteration)
recover_dirty_tree() {
    local task_range="$1"
    local last_task_id="$2"

    if ! is_tree_dirty; then
        return 0
    fi

    local dirty_summary
    dirty_summary=$(get_dirty_summary)
    log "DIRTY_TREE: Working tree is dirty at iteration start ($dirty_summary)."

    # Case (a): last task is marked complete but wasn't committed
    if [[ -n "$last_task_id" ]] && is_task_completed "$last_task_id"; then
        log "DIRTY_TREE: $last_task_id is marked complete in PROGRESS.md. Running commit recovery..."
        if run_commit_recovery "$last_task_id"; then
            return 0
        fi
        log "DIRTY_TREE: Commit recovery failed. Falling through to stash."
    fi

    # Case (b): interrupted mid-task or commit recovery failed -> stash
    stash_dirty_tree "dirty tree at iteration start${last_task_id:+ (last task: $last_task_id)}"
    return 0
}

# =============================================================================
# Status Display
# =============================================================================

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
    echo "  Default Model:  $DEFAULT_MODEL"
    printf "  Default %-8s %s\n" "${EFFORT_LABEL}:" "$DEFAULT_EFFORT"
    echo ""
}

# =============================================================================
# Main Loop
# =============================================================================

run_ralph_loop() {
    local phase_id="$1"
    local max_iterations="$2"
    local dry_run="$3"
    local model="$4"
    local effort="$5"

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
    log "Ralph Loop Starting ($AGENT_LABEL)"
    log "Phase: $phase_name ($phase_id)"
    log "Task Range: $task_range"
    log "Max Iterations: $max_iterations"
    if [[ -n "$model" ]]; then
        log "Model: $model"
    fi
    log "${EFFORT_LABEL}: $effort"
    log "=========================================="

    # Display model configuration banner
    print_model_banner "$model" "$effort" "$phase_name" | tee -a "$LOG_FILE"

    # Allow agent-specific setup (e.g., export CLAUDE_CODE_EFFORT_LEVEL)
    pre_agent_setup "$effort"

    local iteration=0
    local tasks_completed=0
    local consecutive_errors=0
    local rate_limit_waits=0
    local max_limit_waits=${MAX_LIMIT_WAITS:-$DEFAULT_MAX_LIMIT_WAITS}
    local last_completed_task=""

    while [[ $iteration -lt $max_iterations ]]; do
        iteration=$((iteration + 1))

        local remaining
        remaining=$(count_remaining_tasks "$task_range")
        log ""
        log "--- Iteration $iteration / $max_iterations (${remaining} tasks remaining) ---"

        # --- Pre-iteration dirty tree recovery ---
        recover_dirty_tree "$task_range" "$last_completed_task"

        if [[ "$remaining" -eq 0 ]]; then
            log "All tasks in $phase_name are complete!"
            break
        fi

        # Select the next unblocked task in this phase.
        local selection
        selection=$(select_next_task "$task_range")
        if [[ -z "$selection" ]]; then
            log "TASK_BLOCKED: No eligible tasks found in $task_range"
            break
        fi

        if [[ "$selection" == BLOCKED:* ]]; then
            local blocked_task="${selection#BLOCKED:}"
            blocked_task="${blocked_task%%:*}"
            local missing_deps="${selection#BLOCKED:${blocked_task}:}"
            log "TASK_BLOCKED: ${blocked_task} requires ${missing_deps}"
            break
        fi

        local selected_task="$selection"
        local selected_range="${selected_task}:${selected_task}"
        local task_list
        task_list=$(get_task_list_for_single "$selected_task")
        if [[ -z "$task_list" ]]; then
            log "RALPH_ERROR: Could not resolve spec file for $selected_task"
            exit 1
        fi
        log "Selected task for iteration: $selected_task"

        # Generate prompt scoped to exactly one task.
        local prompt
        prompt=$(generate_prompt "$phase_id" "$phase_name" "$selected_range" "$task_list" "$selected_task")

        if [[ "$dry_run" == "true" ]]; then
            log "DRY RUN -- Generated prompt:"
            echo "---"
            echo "$prompt"
            echo "---"
            echo ""
            echo "$(get_dry_run_command "$model" "$effort")"
            print_model_banner "$model" "$effort" "$phase_name"
            exit 0
        fi

        # Write prompt to temp file (avoids shell escaping issues)
        local prompt_file
        prompt_file=$(mktemp /tmp/${LOG_PREFIX}-prompt-XXXXXX.md)
        ralph_register_temp_file "$prompt_file"
        echo "$prompt" > "$prompt_file"

        # Record git HEAD before agent run (for commit-recovery detection)
        local start_head
        start_head=$(git rev-parse HEAD 2>/dev/null || echo "")

        # Run agent with the prompt
        log "Spawning $AGENT_LABEL (iteration $iteration)..."
        local start_time
        start_time=$(date +%s)

        local output=""
        local exit_code=0
        output=$(run_agent "$prompt_file" "$model" "$effort") || exit_code=$?

        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))

        # Clean up temp file
        rm -f "$prompt_file"

        # Log output to file
        echo "$output" >> "$LOG_FILE"

        log "$AGENT_LABEL exited (code=$exit_code, duration=${duration}s)"

        # --- Rate-limit detection (checked FIRST, before other signals) ---
        if is_rate_limited "$output"; then
            log "RATE_LIMIT: Rate limit detected in agent output."

            local wait_seconds
            wait_seconds=$(compute_rate_limit_wait "$output" "$rate_limit_waits")

            # Stash any partial work from the interrupted iteration so next attempt starts clean
            if is_tree_dirty; then
                stash_dirty_tree "rate-limit interrupt during $selected_task"
            fi

            wait_for_rate_limit_reset "$wait_seconds" "$rate_limit_waits" "$max_limit_waits"
            rate_limit_waits=$((rate_limit_waits + 1))
            # Do NOT increment consecutive_errors for rate limits
            continue
        fi

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

        # Check if a task was actually completed (PROGRESS.md update + git commit)
        local end_head
        end_head=$(git rev-parse HEAD 2>/dev/null || echo "")

        local new_remaining
        new_remaining=$(count_remaining_tasks "$task_range")
        local progress_made=false
        local commit_made=false

        if [[ "$new_remaining" -lt "$remaining" ]]; then
            progress_made=true
        fi
        if [[ -n "$start_head" && "$start_head" != "$end_head" ]]; then
            commit_made=true
        fi

        if [[ "$progress_made" == "true" && "$commit_made" == "true" ]]; then
            # Full completion: progress in PROGRESS.md and a git commit
            tasks_completed=$((tasks_completed + (remaining - new_remaining)))
            consecutive_errors=0
            rate_limit_waits=0
            last_completed_task="$selected_task"
            log "Task completed with commit! (total completed this session: $tasks_completed)"

        elif [[ "$progress_made" == "true" && "$commit_made" == "false" ]]; then
            # Progress but no commit -- attempt auto-commit recovery
            log "COMMIT_RECOVERY: $selected_task marked complete but no commit detected."
            if run_commit_recovery "$selected_task"; then
                tasks_completed=$((tasks_completed + (remaining - new_remaining)))
                consecutive_errors=0
                rate_limit_waits=0
                last_completed_task="$selected_task"
                log "Task completed with recovery commit! (total completed this session: $tasks_completed)"
            else
                # Auto-commit failed -- still count progress but warn loudly
                tasks_completed=$((tasks_completed + (remaining - new_remaining)))
                consecutive_errors=0
                rate_limit_waits=0
                last_completed_task="$selected_task"
                log "WARNING: Task progress counted but auto-commit failed. Uncommitted changes remain."
                log "  Next iteration will attempt dirty-tree recovery before proceeding."
            fi

        else
            # No progress at all
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
    log "Ralph Loop Complete ($AGENT_LABEL)"
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
    local effort="$4"

    log "=========================================="
    log "Ralph Single Task Mode ($AGENT_LABEL)"
    log "Task: $task_id"
    if [[ -n "$model" ]]; then
        log "Model: $model"
    fi
    log "${EFFORT_LABEL}: $effort"
    log "=========================================="

    # Display model configuration banner
    print_model_banner "$model" "$effort" "Single Task: $task_id" | tee -a "$LOG_FILE"

    # Allow agent-specific setup (e.g., export CLAUDE_CODE_EFFORT_LEVEL)
    pre_agent_setup "$effort"

    local task_list
    task_list=$(get_task_list_for_single "$task_id")

    if [[ -z "$task_list" ]]; then
        log "ERROR: Task $task_id not found in docs/tasks/"
        exit 1
    fi

    # Determine phase from task number
    local phase_id
    phase_id=$(get_phase_for_task "$task_id")

    local phase_name
    phase_name=$(get_phase_name "$phase_id")
    local task_range="${task_id}:${task_id}"

    local prompt
    prompt=$(generate_prompt "$phase_id" "$phase_name" "$task_range" "$task_list" "$task_id")

    if [[ "$dry_run" == "true" ]]; then
        log "DRY RUN -- Generated prompt:"
        echo "---"
        echo "$prompt"
        echo "---"
        echo ""
        echo "$(get_dry_run_command "$model" "$effort")"
        print_model_banner "$model" "$effort" "Single Task: $task_id"
        exit 0
    fi

    local prompt_file
    prompt_file=$(mktemp /tmp/${LOG_PREFIX}-prompt-XXXXXX.md)
    ralph_register_temp_file "$prompt_file"
    echo "$prompt" > "$prompt_file"

    # --- Pre-run dirty tree recovery ---
    recover_dirty_tree "$task_range" ""

    # Record git HEAD before agent run
    local start_head
    start_head=$(git rev-parse HEAD 2>/dev/null || echo "")

    log "Spawning $AGENT_LABEL for $task_id..."
    local start_time
    start_time=$(date +%s)

    local output=""
    local exit_code=0
    output=$(run_agent "$prompt_file" "$model" "$effort") || exit_code=$?

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Clean up temp file
    rm -f "$prompt_file"

    # Log output
    echo "$output" >> "$LOG_FILE"
    # Also display to terminal
    echo "$output"

    log "$AGENT_LABEL exited (code=$exit_code, duration=${duration}s)"

    # --- Rate-limit detection ---
    if is_rate_limited "$output"; then
        log "RATE_LIMIT: Rate limit detected during single task run."
        if is_tree_dirty; then
            stash_dirty_tree "rate-limit interrupt during $task_id"
        fi
        log "Single task aborted due to rate limit. Re-run when limit resets."
        exit 1
    fi

    # --- Post-run commit recovery ---
    local end_head
    end_head=$(git rev-parse HEAD 2>/dev/null || echo "")

    if is_task_completed "$task_id"; then
        if [[ -n "$start_head" && "$start_head" == "$end_head" ]] && is_tree_dirty; then
            log "COMMIT_RECOVERY: $task_id marked complete but no commit detected."
            run_commit_recovery "$task_id" || true
        fi
        log "Single task $task_id completed successfully."
    else
        if is_tree_dirty; then
            log "WARNING: Task $task_id not marked complete. Uncommitted changes remain."
            log "  Review with: git status && git diff"
        else
            log "WARNING: Task $task_id not marked complete and no changes detected."
        fi
    fi

    log "Single task run complete."
}

run_all_phases() {
    local max_iterations="$1"
    local dry_run="$2"
    local model="$3"
    local effort="$4"

    for phase in $ALL_PHASES; do
        local remaining
        remaining=$(count_remaining_tasks "$(get_phase_range "$phase")")

        if [[ "$remaining" -eq 0 ]]; then
            log "Skipping $(get_phase_name "$phase") -- all tasks complete"
            continue
        fi

        log "Starting $(get_phase_name "$phase") ($remaining tasks remaining)"
        run_ralph_loop "$phase" "$max_iterations" "$dry_run" "$model" "$effort"

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
# CLI Argument Parsing (shared logic)
# =============================================================================

# Main entry point. Called by each script after setting config and sourcing this lib.
# Each script must define: run_agent(), usage(), check_prerequisites(),
# pre_agent_setup(), get_dry_run_command()
main() {
    declare_phases

    local phase=""
    local task=""
    local max_iterations=$DEFAULT_MAX_ITERATIONS
    local model="$DEFAULT_MODEL"
    local effort="$DEFAULT_EFFORT"
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
            --max-limit-waits)
                MAX_LIMIT_WAITS="$2"
                shift 2
                ;;
            --model)
                model="$2"
                shift 2
                ;;
            --effort|--reasoning)
                effort="$2"
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

    # Check agent-specific prerequisites
    check_prerequisites

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

    # Install signal handlers after logging is ready
    _ralph_install_traps

    # Run
    if [[ -n "$task" ]]; then
        run_single_task "$task" "$dry_run" "$model" "$effort"
    elif [[ "$phase" == "all" ]]; then
        run_all_phases "$max_iterations" "$dry_run" "$model" "$effort"
    else
        run_ralph_loop "$phase" "$max_iterations" "$dry_run" "$model" "$effort"
    fi
}