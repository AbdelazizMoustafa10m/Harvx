---
description: Orchestrate task implementation with specialized subagents (Go engineer + testing engineer)
argument-hint: <task-number e.g. T-001>
---

# Task Implementation Orchestrator

You are the **Orchestrator** for implementing task `$ARGUMENTS` in Harvx. You coordinate specialized subagents to complete the full implementation cycle.

## Workflow Overview

```
+-------------------------------------------------------------+
|  1. PRE-FLIGHT CHECKS (Orchestrator)                         |
|     - Check PROGRESS.md for task status                      |
|     - Verify dependencies are complete                       |
|     - Read task specification                                |
+-------------------------------------------------------------+
|  2. IMPLEMENTATION (spawn backend-engineer subagent)         |
|     - Implement Go packages, CLI commands, core logic        |
|     - Write initial tests alongside implementation           |
|     - Run go build and go vet                                |
+-------------------------------------------------------------+
|  3. TESTING (spawn testing-engineer subagent)                |
|     - Write comprehensive tests for new functionality        |
|     - Verify acceptance criteria                             |
|     - Run full test suite with race detector                 |
+-------------------------------------------------------------+
|  4. FINAL VERIFICATION (Orchestrator)                        |
|     - go build ./cmd/harvx/                                  |
|     - go vet ./...                                           |
|     - go test ./...                                          |
|     - go mod tidy (verify no drift)                          |
+-------------------------------------------------------------+
|  5. UPDATE PROGRESS (Orchestrator)                           |
|     - Update PROGRESS.md with completion status              |
+-------------------------------------------------------------+
```

---

## Step 1: Pre-Flight Checks

### 1.1 Check Progress Status

Read `docs/tasks/PROGRESS.md` to check:

- Is task `$ARGUMENTS` already completed? -> **STOP and report**
- Which dependency tasks are completed?
- Current project status

### 1.2 Read Task Specification

Read `docs/tasks/$ARGUMENTS-*.md` (use glob pattern) to understand:

- Full requirements and acceptance criteria
- Dependencies on other tasks
- Technical specifications
- Files to create/modify
- Testing requirements

### 1.3 Verify Dependencies

Cross-reference with PROGRESS.md:

- All blocking tasks must be marked complete
- If dependencies incomplete -> **STOP and report blockers**

### 1.4 Review PRD Context (if referenced)

If task references PRD sections, read `docs/prd/PRD-Harvx.md` for context.

---

## Step 2: Spawn Go Engineer Subagent

**Use the Task tool to spawn the backend-engineer subagent:**

```
Task(
  subagent_type: "backend-engineer",
  description: "Implement $ARGUMENTS",
  prompt: """
  Implement task $ARGUMENTS for Harvx.

  TASK SPECIFICATION:
  [Include full task spec from docs/tasks/$ARGUMENTS-*.md]

  REQUIREMENTS:
  1. Follow Go idioms (error wrapping, interface-driven, no global state)
  2. Use slog for structured logging
  3. Write *_test.go alongside implementation
  4. CGO_ENABLED=0 -- pure Go only

  After implementation, run:
  - go build ./cmd/harvx/
  - go vet ./...
  - go test ./...

  Report back with:
  - Summary of what was implemented
  - Files created/modified
  - Key design decisions
  - Any issues or concerns
  """
)
```

**Wait for backend-engineer to complete before proceeding.**

---

## Step 3: Spawn Testing Engineer Subagent

**Use the Task tool to spawn the testing-engineer subagent:**

```
Task(
  subagent_type: "testing-engineer",
  description: "Test $ARGUMENTS implementation",
  prompt: """
  Write comprehensive tests for task $ARGUMENTS in Harvx.

  TASK SPECIFICATION:
  [Include full task spec from docs/tasks/$ARGUMENTS-*.md]

  IMPLEMENTATION SUMMARY:
  [Include summary from backend-engineer]

  REQUIREMENTS:
  1. Table-driven tests with testify
  2. Cover all acceptance criteria
  3. Test edge cases: empty inputs, errors, boundaries
  4. Use t.TempDir() for filesystem tests
  5. Use t.Helper() in test helpers

  After writing tests, run:
  - go test ./...
  - go test -race ./...
  - go vet ./...

  Report back with:
  - Test files created
  - Coverage areas
  - Acceptance criteria verified
  - Any issues found
  """
)
```

**Wait for testing-engineer to complete before proceeding.**

---

## Step 4: Final Verification

Run the complete verification suite:

```bash
go build ./cmd/harvx/    # Must compile successfully
go vet ./...             # Must pass with zero warnings
go test ./...            # All tests must pass
go mod tidy              # No module drift
```

**If any verification fails:**

1. Identify the issue
2. Spawn appropriate subagent to fix
3. Re-run verification

---

## Step 5: Update PROGRESS.md

After all verification passes, update `docs/tasks/PROGRESS.md`:

### 5.1 Update Summary Table

Increment "Completed" count, decrement "Not Started" or "In Progress"

### 5.2 Update Task Status in Phase Table

Change the task status from "Not Started" to "Completed" in the phase task table.

### 5.3 Add Completion Entry

Add to the "Completed Tasks" section:

```markdown
### $ARGUMENTS: <Task Title>

- **Status:** Completed
- **Date:** <YYYY-MM-DD>

**What was built:**

- <Key deliverable 1>
- <Key deliverable 2>

**Files created/modified:**

- `internal/pkg/file.go` - Description

**Verification:**

- `go build` pass
- `go vet` pass
- `go test` pass
```

---

## Output Format

Provide final summary:

```markdown
## Task $ARGUMENTS Implementation Complete

### Summary

[Brief description of what was accomplished]

### Go Implementation

[Summary from backend-engineer]

### Testing

[Summary from testing-engineer]

### Files Changed

- Created: [list]
- Modified: [list]

### Verification Results

- Build: pass/fail
- Vet: pass/fail
- Tests: pass/fail
- Mod tidy: pass/fail

### PROGRESS.md Updated

- Status: Completed
- Date: YYYY-MM-DD

### Follow-up Items

- [Any discovered issues or future improvements]
```

---

## Error Handling

### If Dependencies Not Met

```
BLOCKED: Task $ARGUMENTS requires completion of:
- T-XXX: [status]
- T-YYY: [status]

Please complete blocking tasks first.
```

### If Already Completed

```
SKIPPED: Task $ARGUMENTS is already marked as completed in PROGRESS.md
```

### If Verification Fails

1. Identify specific failure
2. Spawn appropriate subagent with fix instructions
3. Re-run verification
4. Repeat until all checks pass

---

## Subagent Coordination

- **backend-engineer**: Go packages, CLI commands, core logic, initial tests
- **testing-engineer**: Comprehensive tests, acceptance criteria verification, edge cases

**Both subagents share:**
- Same Go conventions (error wrapping, slog, testify)
- Same project structure (`internal/` packages)
- Same verification commands
