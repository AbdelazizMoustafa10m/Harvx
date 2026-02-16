# Harvx Review Checklist

## Correctness

- [ ] CLI behavior matches command help and docs
- [ ] Flag interactions are validated and deterministic
- [ ] Config/profile precedence behaves as intended
- [ ] Error paths return actionable wrapped errors

## Cobra and UX Contracts

- [ ] Commands remain script-friendly (stable stdout/stderr usage)
- [ ] Exit codes follow Harvx contract (0/1/2)
- [ ] `--json`/machine outputs stay parseable and stable

## Deterministic Behavior

- [ ] Output ordering is stable (files, findings, sections)
- [ ] No map-iteration nondeterminism leaks into output
- [ ] Hashing/token reporting behavior remains reproducible

## Security

- [ ] Secret redaction is preserved or improved
- [ ] No sensitive output in logs/errors
- [ ] Shell and filesystem interactions are safe
- [ ] Dependency/toolchain changes are justified and safe

## Testing and Reliability

- [ ] New logic has tests (unit/integration/golden where appropriate)
- [ ] Existing tests still validate changed behaviors
- [ ] Edge and failure paths are covered
- [ ] No flaky assumptions (time, order, shared mutable state)
