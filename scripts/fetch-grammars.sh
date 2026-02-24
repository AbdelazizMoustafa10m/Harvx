#!/usr/bin/env bash
# fetch-grammars.sh -- Downloads prebuilt tree-sitter grammar .wasm files
# from the Sourcegraph tree-sitter-wasms npm package via CDN.
#
# Usage:
#   ./scripts/fetch-grammars.sh          # Download missing grammars
#   ./scripts/fetch-grammars.sh --force  # Re-download all grammars

set -euo pipefail

# ─── Configuration ──────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GRAMMAR_DIR="$PROJECT_ROOT/grammars"

# Primary CDN (unpkg.com via Sourcegraph's tree-sitter-wasms package)
PRIMARY_BASE="https://unpkg.com/tree-sitter-wasms@0.1.13/out"
# Fallback CDN (jsDelivr)
FALLBACK_BASE="https://cdn.jsdelivr.net/npm/tree-sitter-wasms@0.1.13/out"

LANGUAGES=(
    typescript
    javascript
    go
    python
    rust
    java
    c
    cpp
)

FORCE=false

# ─── Argument Parsing ──────────────────────────────────────────────────────

for arg in "$@"; do
    case "$arg" in
        --force)
            FORCE=true
            ;;
        --help|-h)
            echo "Usage: $0 [--force]"
            echo ""
            echo "Downloads tree-sitter grammar .wasm files to grammars/"
            echo ""
            echo "Options:"
            echo "  --force    Re-download all grammars even if they exist"
            echo "  --help     Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown argument: $arg"
            echo "Usage: $0 [--force]"
            exit 1
            ;;
    esac
done

# ─── Prerequisites ──────────────────────────────────────────────────────────

if ! command -v curl >/dev/null 2>&1; then
    echo "ERROR: curl is required but not found in PATH"
    exit 1
fi

# ─── Download ───────────────────────────────────────────────────────────────

mkdir -p "$GRAMMAR_DIR"

downloaded=0
skipped=0
failed=0

for lang in "${LANGUAGES[@]}"; do
    filename="tree-sitter-${lang}.wasm"
    filepath="$GRAMMAR_DIR/$filename"

    if [[ -f "$filepath" ]] && [[ "$FORCE" == "false" ]]; then
        echo "SKIP  $filename (already exists, use --force to re-download)"
        skipped=$((skipped + 1))
        continue
    fi

    echo -n "FETCH $filename ... "

    # Try primary CDN
    url="${PRIMARY_BASE}/${filename}"
    if curl -fsSL --retry 3 --retry-delay 2 -o "$filepath" "$url" 2>/dev/null; then
        size=$(wc -c < "$filepath" | tr -d ' ')
        echo "OK ($(numfmt --to=iec "$size" 2>/dev/null || echo "${size} bytes"))"
        downloaded=$((downloaded + 1))
        continue
    fi

    # Try fallback CDN
    echo -n "(fallback) "
    url="${FALLBACK_BASE}/${filename}"
    if curl -fsSL --retry 3 --retry-delay 2 -o "$filepath" "$url" 2>/dev/null; then
        size=$(wc -c < "$filepath" | tr -d ' ')
        echo "OK ($(numfmt --to=iec "$size" 2>/dev/null || echo "${size} bytes"))"
        downloaded=$((downloaded + 1))
        continue
    fi

    # Both failed
    echo "FAILED"
    rm -f "$filepath"
    failed=$((failed + 1))
done

# ─── Summary ────────────────────────────────────────────────────────────────

echo ""
echo "─── Summary ───────────────────────────────────────"
echo "  Downloaded: $downloaded"
echo "  Skipped:    $skipped"
echo "  Failed:     $failed"
echo ""

if [[ $downloaded -gt 0 ]] || [[ $skipped -gt 0 ]]; then
    echo "─── Grammar Files ─────────────────────────────────"
    for lang in "${LANGUAGES[@]}"; do
        filepath="$GRAMMAR_DIR/tree-sitter-${lang}.wasm"
        if [[ -f "$filepath" ]]; then
            size=$(wc -c < "$filepath" | tr -d ' ')
            printf "  %-35s %s\n" "tree-sitter-${lang}.wasm" "$(numfmt --to=iec "$size" 2>/dev/null || echo "${size} bytes")"
        else
            printf "  %-35s %s\n" "tree-sitter-${lang}.wasm" "MISSING"
        fi
    done
fi

if [[ $failed -gt 0 ]]; then
    echo ""
    echo "WARNING: $failed grammar(s) failed to download."
    echo "Check your internet connection and try again."
    exit 1
fi

echo ""
echo "All grammars ready in $GRAMMAR_DIR/"