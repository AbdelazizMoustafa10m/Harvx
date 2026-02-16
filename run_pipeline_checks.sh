#!/bin/bash

export FORCE_COLOR=1

# Track failures
FAILED=0

step() {
  echo ""
  echo "─────────────────────────────────────────"
  echo "  $1"
  echo "─────────────────────────────────────────"
}

pass() {
  echo "  ✅ $1 passed"
}

fail() {
  echo "  ❌ $1 failed"
  FAILED=1
}

# ─── Formatting ────────────────────────────────
formatting() {
  step "Formatting (gofmt)"
  unformatted=$(gofmt -l . 2>&1)
  if [[ -z "$unformatted" ]]; then
    pass "gofmt"
  else
    fail "gofmt"
    echo ""
    echo "  Unformatted files:"
    echo "$unformatted" | sed 's/^/    /'
    echo ""
    echo "  💡 Run 'gofmt -w .' to auto-fix"
  fi
}

# ─── Vetting ────────────────────────────────────
vetting() {
  step "Vetting (go vet)"
  if go vet ./... 2>&1; then
    pass "go vet"
  else
    fail "go vet"
  fi
}

# ─── Linting ───────────────────────────────────
linting() {
  step "Linting (golangci-lint)"

  # Check if golangci-lint is installed
  if ! command -v golangci-lint &> /dev/null; then
    echo "  ⚠️  golangci-lint not installed, skipping"
    echo ""
    echo "  💡 To install: https://golangci-lint.run/usage/install/"
    return
  fi

  if golangci-lint run 2>&1; then
    pass "golangci-lint"
  else
    fail "golangci-lint"
  fi
}

# ─── Module Hygiene ────────────────────────────
module_hygiene() {
  step "Module Hygiene (go mod tidy)"

  # Run go mod tidy
  go mod tidy 2>&1

  # Check if go.mod or go.sum changed
  if ! git diff --exit-code go.mod go.sum 2>/dev/null; then
    fail "go mod tidy"
    echo ""
    echo "  Modules were not in sync. Run 'go mod tidy' before committing."
  else
    pass "go mod tidy"
  fi
}

# ─── Testing ───────────────────────────────────
testing() {
  step "Testing (go test)"
  if go test -race -count=1 ./... 2>&1; then
    pass "go test"
  else
    fail "go test"
  fi
}

# ─── Build ─────────────────────────────────────
building() {
  step "Build (harvx)"
  if go build -o /dev/null ./cmd/harvx/ 2>&1; then
    pass "Build"
  else
    fail "Build"
  fi
}

# ─── Main ──────────────────────────────────────
echo "========================================="
echo "  🔍 Running Pipeline Checks"
echo "========================================="

formatting
if [[ $FAILED -ne 0 ]]; then
  echo ""
  echo "========================================="
  echo "  ❌ Formatting failed. Fix before continuing."
  echo "========================================="
  exit 1
fi

vetting
if [[ $FAILED -ne 0 ]]; then
  echo ""
  echo "========================================="
  echo "  ❌ Vetting failed. Fix before continuing."
  echo "========================================="
  exit 1
fi

linting
if [[ $FAILED -ne 0 ]]; then
  echo ""
  echo "========================================="
  echo "  ❌ Linting failed. Fix before continuing."
  echo "========================================="
  exit 1
fi

module_hygiene
if [[ $FAILED -ne 0 ]]; then
  echo ""
  echo "========================================="
  echo "  ❌ Module hygiene check failed. Fix before continuing."
  echo "========================================="
  exit 1
fi

testing
if [[ $FAILED -ne 0 ]]; then
  echo ""
  echo "========================================="
  echo "  ❌ Testing failed. Fix before continuing."
  echo "========================================="
  exit 1
fi

# Build is optional — pass --with-build to include
if [[ $FAILED -eq 0 && "$1" == "--with-build" ]]; then
  building
fi

echo ""
echo "========================================="
if [[ $FAILED -eq 0 ]]; then
  echo "  ✅ All checks passed! Ready to push."
else
  echo "  ❌ Some checks failed. Fix before pushing."
fi
echo "========================================="

exit $FAILED
