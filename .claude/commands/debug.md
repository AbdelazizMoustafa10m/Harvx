---
description: Debug and fix an issue in Harvx with systematic root cause analysis
argument-hint: <description of the issue>
---

You are a **Senior Go Engineer** debugging: `$ARGUMENTS`

## Debugging Process

### 1. Reproduce & Understand

- What is the exact error message or unexpected behavior?
- What are the steps to reproduce?
- Check test output and slog debug logs

### 2. Locate the Problem

- Trace to source file and line
- Review recent changes to affected files
- Use compiler errors and `go vet` output as hints

### 3. Analyze Root Cause

**Compilation Errors:**
- Missing imports or dependencies
- Type mismatches
- Undefined symbols

**Runtime Errors:**
- Nil pointer dereference
- Index out of range
- Deadlock or goroutine leak

**Logic Errors:**
- Incorrect file filtering
- Wrong sort order
- Off-by-one in token counting
- Config merge precedence wrong

**Test Failures:**
- Stale golden files (run with `-update`)
- Race conditions (`go test -race`)
- Environment-dependent paths

### 4. Implement Fix

- Make minimal change needed
- Match existing code style and patterns
- Add defensive checks if appropriate
- Write a test that would have caught this bug

### 5. Verify

```bash
go build ./cmd/harvx/
go vet ./...
go test ./...
go test -race ./...
```

- Confirm issue is resolved
- Test related functionality still works

### 6. Prevent Recurrence

- Would a test catch this? Write one.
- Should types be tighter? Update them.
- Document if non-obvious gotcha.
