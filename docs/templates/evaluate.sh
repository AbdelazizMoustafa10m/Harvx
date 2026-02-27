#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# evaluate.sh -- Golden Questions Evaluation Script Template
# =============================================================================
#
# Purpose:
#   Run golden question evaluations against an LLM API to measure whether
#   Harvx-generated context improves AI accuracy compared to diff-only context.
#
# This is a TEMPLATE. You must fill in the TODO sections before running.
# The script performs two evaluation passes:
#   1. Diff-only:  sends the raw git diff + golden questions to the LLM
#   2. Harvx:      sends harvx brief + review-slice + golden questions to the LLM
#
# Results are written to .harvx/quality/ for tracking over time.
#
# Prerequisites:
#   - harvx binary on PATH (or adjust HARVX_BIN below)
#   - jq for JSON processing
#   - curl for API calls
#   - A golden questions file at .harvx/golden-questions.toml
#     (generate with: harvx quality init)
#
# Usage:
#   ./evaluate.sh --base origin/main --head HEAD
#   ./evaluate.sh --base v1.2.0 --head v1.3.0 --questions path/to/questions.toml
#
# =============================================================================

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

# TODO: Set your LLM API endpoint and key.
# Examples:
#   OpenAI:    https://api.openai.com/v1/chat/completions
#   Anthropic: https://api.anthropic.com/v1/messages
#   Local:     http://localhost:11434/v1/chat/completions
API_ENDPOINT="${LLM_API_ENDPOINT:-https://api.openai.com/v1/chat/completions}"
API_KEY="${LLM_API_KEY:-}"

# TODO: Set the model to use for evaluation.
MODEL="${LLM_MODEL:-gpt-4o}"

# TODO: Adjust the API request format if you are not using an OpenAI-compatible
# endpoint. The send_to_llm() function below constructs the request body.

# Harvx binary path.
HARVX_BIN="${HARVX_BIN:-harvx}"

# Default golden questions path.
QUESTIONS_FILE=".harvx/golden-questions.toml"

# Output directory for evaluation results.
QUALITY_DIR=".harvx/quality"

# Max tokens for LLM response.
MAX_RESPONSE_TOKENS=1024

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

BASE_REF=""
HEAD_REF=""

usage() {
    cat <<EOF
Usage: $(basename "$0") --base <ref> --head <ref> [OPTIONS]

Options:
  --base <ref>          Base git ref (e.g., origin/main, v1.0.0)
  --head <ref>          Head git ref (e.g., HEAD, feature-branch)
  --questions <path>    Path to golden questions TOML (default: .harvx/golden-questions.toml)
  --model <name>        LLM model name (default: gpt-4o)
  --help                Show this help message

Environment variables:
  LLM_API_ENDPOINT      API endpoint URL
  LLM_API_KEY           API key / bearer token
  LLM_MODEL             Model name override
  HARVX_BIN             Path to harvx binary
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --base)
            BASE_REF="$2"
            shift 2
            ;;
        --head)
            HEAD_REF="$2"
            shift 2
            ;;
        --questions)
            QUESTIONS_FILE="$2"
            shift 2
            ;;
        --model)
            MODEL="$2"
            shift 2
            ;;
        --help|-h)
            usage
            ;;
        *)
            echo "Error: unknown argument '$1'" >&2
            usage
            ;;
    esac
done

if [[ -z "$BASE_REF" || -z "$HEAD_REF" ]]; then
    echo "Error: --base and --head are required." >&2
    usage
fi

if [[ -z "$API_KEY" ]]; then
    echo "Error: LLM_API_KEY environment variable is not set." >&2
    echo "Set it with: export LLM_API_KEY='your-api-key'" >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Dependency checks
# ---------------------------------------------------------------------------

for cmd in "$HARVX_BIN" jq curl git; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Error: required command '$cmd' not found on PATH." >&2
        exit 1
    fi
done

if [[ ! -f "$QUESTIONS_FILE" ]]; then
    echo "Error: golden questions file not found at '$QUESTIONS_FILE'." >&2
    echo "Generate one with: harvx quality init" >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
RUN_DIR="${QUALITY_DIR}/run-${TIMESTAMP}"
mkdir -p "$RUN_DIR"

echo "=== Golden Questions Evaluation ==="
echo "Base:       $BASE_REF"
echo "Head:       $HEAD_REF"
echo "Model:      $MODEL"
echo "Questions:  $QUESTIONS_FILE"
echo "Output:     $RUN_DIR"
echo ""

# ---------------------------------------------------------------------------
# Step 1: Generate context artifacts
# ---------------------------------------------------------------------------

echo "--- Generating context artifacts ---"

# Generate the raw git diff (diff-only baseline).
echo "  Generating git diff..."
git diff "${BASE_REF}...${HEAD_REF}" > "${RUN_DIR}/diff-only.patch"
DIFF_CONTENT="$(cat "${RUN_DIR}/diff-only.patch")"

# Generate Harvx brief (stable project context).
echo "  Generating harvx brief..."
"$HARVX_BIN" brief --stdout > "${RUN_DIR}/brief.md" 2>/dev/null || {
    echo "  Warning: harvx brief failed; continuing with empty brief."
    touch "${RUN_DIR}/brief.md"
}
BRIEF_CONTENT="$(cat "${RUN_DIR}/brief.md")"

# Generate Harvx review-slice (PR-specific context).
echo "  Generating harvx review-slice..."
"$HARVX_BIN" review-slice --base "$BASE_REF" --head "$HEAD_REF" --stdout \
    > "${RUN_DIR}/review-slice.md" 2>/dev/null || {
    echo "  Warning: harvx review-slice failed; continuing with empty slice."
    touch "${RUN_DIR}/review-slice.md"
}
SLICE_CONTENT="$(cat "${RUN_DIR}/review-slice.md")"

echo ""

# ---------------------------------------------------------------------------
# Step 2: Parse golden questions from TOML
# ---------------------------------------------------------------------------

# Parse the TOML file into a JSON array for easier shell processing.
# This uses a simple awk-based parser. For production use, consider a proper
# TOML parser or use `harvx quality --json` to get structured output.
#
# TODO: Replace this parser with a more robust solution if your questions
# contain multi-line values or special characters.

parse_questions() {
    local file="$1"
    local in_question=0
    local id="" question="" expected="" category=""
    local first=1

    echo "["
    while IFS= read -r line; do
        # Strip leading/trailing whitespace.
        line="$(echo "$line" | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')"

        # Skip comments and blank lines.
        [[ -z "$line" || "$line" == \#* ]] && continue

        if [[ "$line" == "[[questions]]" ]]; then
            # Emit previous question if we have one.
            if [[ $in_question -eq 1 && -n "$id" ]]; then
                [[ $first -eq 0 ]] && echo ","
                first=0
                printf '  {"id": %s, "question": %s, "expected_answer": %s, "category": %s}' \
                    "$(echo "$id" | jq -R .)" \
                    "$(echo "$question" | jq -R .)" \
                    "$(echo "$expected" | jq -R .)" \
                    "$(echo "$category" | jq -R .)"
            fi
            in_question=1
            id="" question="" expected="" category=""
            continue
        fi

        if [[ $in_question -eq 1 ]]; then
            case "$line" in
                id\ =\ *|id=*)
                    id="$(echo "$line" | sed 's/^id[[:space:]]*=[[:space:]]*//' | tr -d '"')"
                    ;;
                question\ =\ *|question=*)
                    question="$(echo "$line" | sed 's/^question[[:space:]]*=[[:space:]]*//' | tr -d '"')"
                    ;;
                expected_answer\ =\ *|expected_answer=*)
                    expected="$(echo "$line" | sed 's/^expected_answer[[:space:]]*=[[:space:]]*//' | tr -d '"')"
                    ;;
                category\ =\ *|category=*)
                    category="$(echo "$line" | sed 's/^category[[:space:]]*=[[:space:]]*//' | tr -d '"')"
                    ;;
            esac
        fi
    done < "$file"

    # Emit the last question.
    if [[ $in_question -eq 1 && -n "$id" ]]; then
        [[ $first -eq 0 ]] && echo ","
        printf '  {"id": %s, "question": %s, "expected_answer": %s, "category": %s}' \
            "$(echo "$id" | jq -R .)" \
            "$(echo "$question" | jq -R .)" \
            "$(echo "$expected" | jq -R .)" \
            "$(echo "$category" | jq -R .)"
    fi
    echo ""
    echo "]"
}

QUESTIONS_JSON="$(parse_questions "$QUESTIONS_FILE")"
QUESTION_COUNT="$(echo "$QUESTIONS_JSON" | jq 'length')"

echo "Parsed $QUESTION_COUNT golden questions."
echo ""

if [[ "$QUESTION_COUNT" -eq 0 ]]; then
    echo "No questions found. Exiting."
    exit 0
fi

# ---------------------------------------------------------------------------
# Step 3: LLM interaction helper
# ---------------------------------------------------------------------------

# send_to_llm sends a prompt to the configured LLM API and returns the response text.
#
# TODO: Adapt this function to match your LLM provider's API format.
# The default implementation targets OpenAI-compatible chat completions.
# For Anthropic's Messages API, you would change the request body structure
# and the response parsing (jq path).
send_to_llm() {
    local system_prompt="$1"
    local user_prompt="$2"

    # Build request body for OpenAI-compatible API.
    local request_body
    request_body="$(jq -n \
        --arg model "$MODEL" \
        --arg system "$system_prompt" \
        --arg user "$user_prompt" \
        --argjson max_tokens "$MAX_RESPONSE_TOKENS" \
        '{
            model: $model,
            max_tokens: $max_tokens,
            messages: [
                { role: "system", content: $system },
                { role: "user", content: $user }
            ]
        }'
    )"

    # TODO: If using Anthropic's API, replace the above with:
    # request_body="$(jq -n \
    #     --arg model "$MODEL" \
    #     --arg system "$system_prompt" \
    #     --arg user "$user_prompt" \
    #     --argjson max_tokens "$MAX_RESPONSE_TOKENS" \
    #     '{
    #         model: $model,
    #         max_tokens: $max_tokens,
    #         system: $system,
    #         messages: [
    #             { role: "user", content: $user }
    #         ]
    #     }'
    # )"
    # And change the Authorization header to: "x-api-key: $API_KEY"
    # And change the jq response path to: .content[0].text

    local response
    response="$(curl -s -w "\n%{http_code}" \
        -X POST "$API_ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $API_KEY" \
        -d "$request_body"
    )"

    local http_code
    http_code="$(echo "$response" | tail -n1)"
    local body
    body="$(echo "$response" | sed '$d')"

    if [[ "$http_code" -ne 200 ]]; then
        echo "ERROR: API returned HTTP $http_code: $(echo "$body" | head -c 200)" >&2
        echo "[API Error]"
        return 1
    fi

    # Extract the response text.
    # TODO: Adjust the jq path for your API provider.
    echo "$body" | jq -r '.choices[0].message.content // "[No response]"'
}

# ---------------------------------------------------------------------------
# Step 4: Scoring function
# ---------------------------------------------------------------------------

# score_response compares an LLM response to the expected answer.
# Returns a score between 0 and 100.
#
# This is a simple keyword-based scorer. For production use, consider:
#   - Using an LLM-as-judge approach for semantic comparison
#   - Embedding similarity (cosine distance)
#   - Exact match for factual answers
#
# TODO: Customize the scoring logic for your project's needs.
score_response() {
    local response="$1"
    local expected="$2"

    # Normalize both strings: lowercase, collapse whitespace.
    local norm_response norm_expected
    norm_response="$(echo "$response" | tr '[:upper:]' '[:lower:]' | tr -s '[:space:]' ' ')"
    norm_expected="$(echo "$expected" | tr '[:upper:]' '[:lower:]' | tr -s '[:space:]' ' ')"

    # Split expected answer into keywords (comma or space separated).
    local keywords
    keywords="$(echo "$norm_expected" | tr ',' '\n' | tr ' ' '\n' | sort -u | grep -v '^$')"

    local total=0
    local matched=0

    while IFS= read -r keyword; do
        [[ -z "$keyword" ]] && continue
        # Skip very short words (articles, prepositions).
        [[ ${#keyword} -lt 3 ]] && continue
        total=$((total + 1))
        if echo "$norm_response" | grep -qi "$keyword"; then
            matched=$((matched + 1))
        fi
    done <<< "$keywords"

    if [[ $total -eq 0 ]]; then
        echo 0
        return
    fi

    # Score as percentage of keywords found.
    echo $(( (matched * 100) / total ))
}

# ---------------------------------------------------------------------------
# Step 5: Run evaluations
# ---------------------------------------------------------------------------

SYSTEM_PROMPT="You are a senior software engineer reviewing code. Answer the following question about this codebase accurately and concisely. Base your answer only on the context provided."

# --- Pass 1: Diff-only context ---

echo "=== Pass 1: Diff-Only Context ==="
echo ""

DIFF_RESULTS="[]"

for i in $(seq 0 $((QUESTION_COUNT - 1))); do
    q_id="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].id")"
    q_text="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].question")"
    q_expected="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].expected_answer")"
    q_category="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].category")"

    echo "  [$((i + 1))/$QUESTION_COUNT] $q_id: $q_text"

    user_prompt="$(printf "## Git Diff\n\n%s\n\n## Question\n\n%s" "$DIFF_CONTENT" "$q_text")"

    llm_response="$(send_to_llm "$SYSTEM_PROMPT" "$user_prompt" 2>"${RUN_DIR}/error.log")" || llm_response="[Error]"
    score="$(score_response "$llm_response" "$q_expected")"

    echo "    Score: ${score}/100"

    # Append to results array.
    DIFF_RESULTS="$(echo "$DIFF_RESULTS" | jq \
        --arg id "$q_id" \
        --arg question "$q_text" \
        --arg expected "$q_expected" \
        --arg response "$llm_response" \
        --arg category "$q_category" \
        --argjson score "$score" \
        '. + [{
            id: $id,
            question: $question,
            expected_answer: $expected,
            llm_response: $response,
            category: $category,
            score: $score
        }]'
    )"
done

echo ""

# --- Pass 2: Harvx context (brief + review-slice) ---

echo "=== Pass 2: Harvx Context (Brief + Review Slice) ==="
echo ""

HARVX_RESULTS="[]"

for i in $(seq 0 $((QUESTION_COUNT - 1))); do
    q_id="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].id")"
    q_text="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].question")"
    q_expected="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].expected_answer")"
    q_category="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].category")"

    echo "  [$((i + 1))/$QUESTION_COUNT] $q_id: $q_text"

    user_prompt="$(printf "## Repo Brief\n\n%s\n\n## Review Slice\n\n%s\n\n## Git Diff\n\n%s\n\n## Question\n\n%s" \
        "$BRIEF_CONTENT" "$SLICE_CONTENT" "$DIFF_CONTENT" "$q_text")"

    llm_response="$(send_to_llm "$SYSTEM_PROMPT" "$user_prompt" 2>"${RUN_DIR}/error.log")" || llm_response="[Error]"
    score="$(score_response "$llm_response" "$q_expected")"

    echo "    Score: ${score}/100"

    # Append to results array.
    HARVX_RESULTS="$(echo "$HARVX_RESULTS" | jq \
        --arg id "$q_id" \
        --arg question "$q_text" \
        --arg expected "$q_expected" \
        --arg response "$llm_response" \
        --arg category "$q_category" \
        --argjson score "$score" \
        '. + [{
            id: $id,
            question: $question,
            expected_answer: $expected,
            llm_response: $response,
            category: $category,
            score: $score
        }]'
    )"
done

echo ""

# ---------------------------------------------------------------------------
# Step 6: Compute aggregate scores and write results
# ---------------------------------------------------------------------------

compute_average() {
    local results_json="$1"
    echo "$results_json" | jq '[.[].score] | if length == 0 then 0 else (add / length | floor) end'
}

DIFF_AVG="$(compute_average "$DIFF_RESULTS")"
HARVX_AVG="$(compute_average "$HARVX_RESULTS")"
IMPROVEMENT=$((HARVX_AVG - DIFF_AVG))

# Build the final results JSON.
FINAL_RESULTS="$(jq -n \
    --arg timestamp "$TIMESTAMP" \
    --arg base "$BASE_REF" \
    --arg head "$HEAD_REF" \
    --arg model "$MODEL" \
    --arg questions_file "$QUESTIONS_FILE" \
    --argjson question_count "$QUESTION_COUNT" \
    --argjson diff_avg "$DIFF_AVG" \
    --argjson harvx_avg "$HARVX_AVG" \
    --argjson improvement "$IMPROVEMENT" \
    --argjson diff_results "$DIFF_RESULTS" \
    --argjson harvx_results "$HARVX_RESULTS" \
    '{
        metadata: {
            timestamp: $timestamp,
            base_ref: $base,
            head_ref: $head,
            model: $model,
            questions_file: $questions_file,
            question_count: $question_count
        },
        summary: {
            diff_only_average: $diff_avg,
            harvx_average: $harvx_avg,
            improvement: $improvement
        },
        diff_only: $diff_results,
        harvx: $harvx_results
    }'
)"

# Write results.
echo "$FINAL_RESULTS" | jq . > "${RUN_DIR}/results.json"
echo "$DIFF_RESULTS" | jq . > "${RUN_DIR}/diff-only-results.json"
echo "$HARVX_RESULTS" | jq . > "${RUN_DIR}/harvx-results.json"

# ---------------------------------------------------------------------------
# Step 7: Print summary
# ---------------------------------------------------------------------------

echo "==========================================="
echo "  Golden Questions Evaluation Summary"
echo "==========================================="
echo ""
echo "  Questions evaluated:   $QUESTION_COUNT"
echo "  Model:                 $MODEL"
echo "  Base ref:              $BASE_REF"
echo "  Head ref:              $HEAD_REF"
echo ""
echo "  Diff-only average:     ${DIFF_AVG}/100"
echo "  Harvx average:         ${HARVX_AVG}/100"
echo "  Improvement:           ${IMPROVEMENT} points"
echo ""

if [[ $IMPROVEMENT -gt 0 ]]; then
    echo "  Result: Harvx context IMPROVED accuracy by $IMPROVEMENT points."
elif [[ $IMPROVEMENT -eq 0 ]]; then
    echo "  Result: No measurable difference between diff-only and Harvx context."
else
    echo "  Result: Diff-only scored higher by $((-IMPROVEMENT)) points. Review your profile configuration."
fi

echo ""
echo "  Per-question breakdown:"
echo ""
printf "  %-20s %-12s %-12s %-12s\n" "QUESTION" "DIFF-ONLY" "HARVX" "DELTA"
printf "  %-20s %-12s %-12s %-12s\n" "--------" "---------" "-----" "-----"

for i in $(seq 0 $((QUESTION_COUNT - 1))); do
    q_id="$(echo "$QUESTIONS_JSON" | jq -r ".[$i].id")"
    d_score="$(echo "$DIFF_RESULTS" | jq -r ".[$i].score")"
    h_score="$(echo "$HARVX_RESULTS" | jq -r ".[$i].score")"
    delta=$((h_score - d_score))
    sign=""
    [[ $delta -gt 0 ]] && sign="+"
    printf "  %-20s %-12s %-12s %-12s\n" "$q_id" "${d_score}/100" "${h_score}/100" "${sign}${delta}"
done

echo ""
echo "  Full results: ${RUN_DIR}/results.json"
echo "==========================================="