#!/usr/bin/env bash
# build.sh - Build all packages in the Acme Platform monorepo.
#
# Usage:
#   ./scripts/build.sh          # Build everything
#   ./scripts/build.sh --only worker  # Build only the worker service

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== Building Acme Platform ==="
echo "Repository root: $REPO_ROOT"

# Build TypeScript packages
build_ts() {
    echo ""
    echo "--- Building TypeScript packages ---"

    for pkg in core api web; do
        echo "Building @acme/$pkg..."
        cd "$REPO_ROOT/packages/$pkg"
        npm run build 2>&1 || { echo "FAILED: @acme/$pkg"; exit 1; }
        echo "OK: @acme/$pkg"
    done
}

# Build Go services
build_go() {
    echo ""
    echo "--- Building Go services ---"

    echo "Building worker..."
    cd "$REPO_ROOT/services/worker"
    CGO_ENABLED=0 go build -ldflags="-s -w" -o "$REPO_ROOT/bin/worker" .
    echo "OK: worker ($(du -h "$REPO_ROOT/bin/worker" | cut -f1))"
}

# Parse arguments
ONLY=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --only)
            ONLY="$2"
            shift 2
            ;;
        *)
            echo "Unknown argument: $1"
            exit 1
            ;;
    esac
done

# Execute builds
mkdir -p "$REPO_ROOT/bin"

case "$ONLY" in
    worker)
        build_go
        ;;
    ts|typescript)
        build_ts
        ;;
    "")
        build_ts
        build_go
        ;;
    *)
        echo "Unknown target: $ONLY"
        exit 1
        ;;
esac

echo ""
echo "=== Build complete ==="