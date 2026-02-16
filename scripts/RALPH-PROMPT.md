# Harvx Ralph Loop -- Autonomous Task Runner

You are Claude Code working on **Harvx**, a Go CLI tool that packages codebases into LLM-optimized context documents.

## Your Mission

Pick the next uncompleted task from the specified phase, implement it, verify it, update progress, and commit.

## Phase Scope

**Phase:** {{PHASE_NAME}}
**Task Range:** {{TASK_RANGE}}
**Tasks in scope:**

{{TASK_LIST}}

## Instructions

### Step 1: Find Next Task

1. Read `docs/tasks/PROGRESS.md`
2. Find the next task in the range {{TASK_RANGE}} that is "Not Started"
3. Skip tasks whose dependencies are not yet completed
4. If ALL tasks in this phase are completed, output exactly: `PHASE_COMPLETE`

### Step 2: Read Task Specification

1. Read the task spec file: `docs/tasks/T-XXX-*.md` (glob to find it)
2. Understand acceptance criteria, dependencies, technical notes, files to create
3. If the task references PRD sections, read `docs/prd/PRD-Harvx.md` for context

### Step 3: Implement

Use `/implement-task T-XXX` to orchestrate the implementation.

This will:
1. Spawn a Go engineer subagent for implementation
2. Spawn a testing engineer subagent for tests
3. Run final verification

If `/implement-task` is not available, implement directly:
1. Read the task spec carefully
2. Create/modify files per the spec
3. Write tests alongside implementation
4. Follow Go conventions from CLAUDE.md

### Step 4: Verify

Run ALL of these -- every single one must pass:

```bash
go build ./cmd/harvx/    # Compilation
go vet ./...             # Static analysis
go test ./...            # All tests
go mod tidy              # Module hygiene
```

If any check fails, fix the issue and re-verify. Do NOT proceed with a failing build.

### Step 5: Update Progress

Update `docs/tasks/PROGRESS.md`:

1. **Summary table**: Increment "Completed", decrement "Not Started"
2. **Phase task table**: Change task status from "Not Started" to "Completed"
3. **Completed Tasks section**: Add entry with:
   - Task ID and title
   - Date (today)
   - What was built (bullet points)
   - Files created/modified
   - Verification status

### Step 6: Git Commit

Stage and commit the changes:

```bash
git add <specific-files>
git commit -m "feat(<scope>): implement T-XXX <task-name>

- <key change 1>
- <key change 2>

Task: T-XXX
Phase: {{PHASE_ID}}"
```

Use conventional commit format. Scope should match the primary package (e.g., `discovery`, `cli`, `config`).

### Step 7: Exit

After completing ONE task, exit cleanly. The loop will restart you with fresh context for the next task.

Do NOT try to implement multiple tasks in one iteration.

## Completion Signals

- If you completed a task successfully: just exit normally after committing
- If ALL tasks in this phase are done: output `PHASE_COMPLETE` and exit
- If a task is blocked by unfinished dependencies: output `TASK_BLOCKED: T-XXX requires T-YYY` and exit
- If you encounter an unrecoverable error: output `RALPH_ERROR: <description>` and exit

## Rules

1. ONE task per iteration. Quality over quantity.
2. Every task must have passing tests before committing.
3. Never commit with failing `go build`, `go vet`, or `go test`.
4. Always update PROGRESS.md before committing.
5. Use structured git commits with task references.
6. Read CLAUDE.md for project conventions.
7. When in doubt, read the PRD at `docs/prd/PRD-Harvx.md`.
