## Description

<!-- What does this PR do? Link any related issues. -->

## Type of Change

- [ ] Bug fix (non-breaking change fixing an issue)
- [ ] New feature (non-breaking change adding functionality)
- [ ] Breaking change (fix or feature causing existing functionality to change)
- [ ] Refactor (code change that neither fixes a bug nor adds a feature)
- [ ] Documentation / config update

## Self-Review Checklist

### Go / Architecture

- [ ] Exported functions/types have doc comments
- [ ] Errors are wrapped with context (`fmt.Errorf("...: %w", err)`)
- [ ] No unintended global mutable state introduced
- [ ] Public APIs stay consistent (or breaking changes are documented)

### Security / Safety

- [ ] No secrets committed in source, fixtures, or docs
- [ ] Redaction-sensitive changes include tests for false positives/negatives
- [ ] Regex/glob additions are safe for untrusted input (no catastrophic behavior)

### Verification

- [ ] `go build ./cmd/harvx/` passes
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes
- [ ] `go mod tidy` produces no diff
- [ ] Tested locally (describe below)

### Task Tracking

- [ ] Relevant `docs/tasks/T-XXX-*.md` acceptance criteria are satisfied
- [ ] `docs/tasks/PROGRESS.md` updated if task completion status changed

## Testing Done

<!-- How did you verify this works? -->
