# T-025: Profile System Integration Tests and Golden Tests

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-016 through T-024 (all prior profile tasks)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Create a comprehensive integration test suite for the entire profile system, ensuring all components work together correctly end-to-end. This includes golden tests that verify the full config resolution pipeline (discovery -> loading -> merging -> inheritance -> validation -> resolution) against known-good outputs. Also includes fuzz testing for config parsing robustness and performance benchmarks for config resolution latency.

## User Story

As a developer contributing to Harvx, I want a thorough integration test suite that catches regressions in the profile system's end-to-end behavior, so that changes to any config component don't silently break the overall configuration resolution.

## Acceptance Criteria

- [ ] Integration test fixtures in `testdata/integration/profiles/`:
  - `scenario-1-defaults-only/` - No config files, pure defaults
  - `scenario-2-repo-config/` - Only a `harvx.toml` at repo root
  - `scenario-3-global-plus-repo/` - Global config + repo config with merge
  - `scenario-4-inheritance/` - Multi-level profile inheritance
  - `scenario-5-env-overrides/` - Environment variable overrides
  - `scenario-6-cli-flags/` - CLI flag overrides on top of everything
  - `scenario-7-template-init/` - Init from template, then load
  - `scenario-8-complex-finvault/` - Full PRD example (finvault profile)
- [ ] Each scenario has:
  - Input files (TOML configs, env vars as JSON file, CLI flags as JSON file)
  - Expected output (`expected.toml` - resolved config as TOML)
  - Expected validation results (`expected_lint.json` - errors/warnings)
- [ ] Golden test runner compares actual output against expected output
- [ ] `go test -update` flag regenerates golden files (for intentional changes)
- [ ] Fuzz test: `FuzzConfigParse` feeds random/malformed TOML to the parser:
  - Must not panic
  - Must return a valid error for invalid input
  - Must not produce invalid Config structs for valid-looking TOML
- [ ] Performance benchmark: `BenchmarkConfigResolve`:
  - Full resolution (discovery + load + merge + inherit + validate) < 5ms
  - Config with 10 profiles resolves in < 10ms
  - Config validation (lint) completes in < 5ms
- [ ] Integration test: Full `profiles list` command output matches expected
- [ ] Integration test: Full `profiles show` command output matches expected
- [ ] Integration test: Full `profiles lint` command output matches expected
- [ ] Integration test: Full `profiles explain` command output matches expected for key file paths
- [ ] Integration test: `config debug` command output matches expected
- [ ] All integration tests pass in CI (GitHub Actions compatible)

## Technical Notes

- Use `testing.Short()` to skip integration tests in quick local runs (`go test -short`)
- Golden test pattern: `testutil.Golden(t, "testname", actualOutput)` that compares against `testdata/golden/testname.golden` and auto-updates with `-update` flag
- Implement a `testutil/golden.go` helper:
  ```go
  func Golden(t *testing.T, name string, actual []byte) {
      golden := filepath.Join("testdata", "golden", name+".golden")
      if *update {
          os.MkdirAll(filepath.Dir(golden), 0755)
          os.WriteFile(golden, actual, 0644)
          return
      }
      expected, _ := os.ReadFile(golden)
      if !bytes.Equal(actual, expected) {
          t.Errorf("golden mismatch for %s\n--- expected\n%s\n--- actual\n%s", name, expected, actual)
      }
  }
  ```
- For fuzz testing, use Go's built-in `testing.F` (available since Go 1.18):
  - Seed corpus: valid TOML configs, edge-case TOML, empty strings, binary data
  - The fuzz target parses input with `BurntSushi/toml` and validates with `Validate()`
- For environment variable testing, use `t.Setenv()` (Go 1.17+) which auto-cleans up
- Benchmark setup: pre-create config files in temp directories, measure only resolution time
- CLI command integration tests: use `cmd.Execute()` with captured stdout/stderr
- The PRD example config (Section 5.2) should be one of the golden test inputs -- exact reproduction
- Consider using `testscript` package (`rogpeppe/go-internal/testscript`) for CLI integration tests if Cobra command testing proves cumbersome

## Files to Create/Modify

- `internal/config/integration_test.go` - End-to-end config resolution tests
- `internal/config/fuzz_test.go` - Fuzz testing for config parsing
- `internal/config/benchmark_test.go` - Performance benchmarks
- `internal/cli/profiles_integration_test.go` - CLI command integration tests
- `internal/testutil/golden.go` - Golden test helper utility
- `testdata/integration/profiles/scenario-1-defaults-only/` - Test fixture
- `testdata/integration/profiles/scenario-2-repo-config/harvx.toml` - Test fixture
- `testdata/integration/profiles/scenario-3-global-plus-repo/global.toml` - Test fixture
- `testdata/integration/profiles/scenario-3-global-plus-repo/harvx.toml` - Test fixture
- `testdata/integration/profiles/scenario-4-inheritance/harvx.toml` - Multi-profile
- `testdata/integration/profiles/scenario-5-env-overrides/harvx.toml` - With env vars
- `testdata/integration/profiles/scenario-6-cli-flags/harvx.toml` - With flags
- `testdata/integration/profiles/scenario-7-template-init/` - Empty dir for init test
- `testdata/integration/profiles/scenario-8-complex-finvault/harvx.toml` - PRD example
- `testdata/golden/` - Golden output files (auto-generated)

## Testing Requirements

### Integration Tests
- Scenario 1: No config files -> resolved config equals built-in defaults
- Scenario 2: Repo config sets max_tokens=50000 -> resolved value is 50000
- Scenario 3: Global sets tokenizer=o200k, repo sets max_tokens=100000 -> both applied
- Scenario 4: Three-level inheritance resolves correctly (child -> parent -> default)
- Scenario 5: HARVX_MAX_TOKENS=75000 overrides repo config value
- Scenario 6: --max-tokens=60000 flag overrides env var
- Scenario 7: Init from nextjs template creates valid config that loads and validates
- Scenario 8: Full PRD finvault example resolves to expected config

### Fuzz Tests
- Fuzz config parser with random bytes (no panics)
- Fuzz config parser with mutated valid TOML (no panics)
- Fuzz validation with random Config structs (no panics)

### Benchmarks
- BenchmarkConfigResolve/defaults-only
- BenchmarkConfigResolve/single-file
- BenchmarkConfigResolve/multi-source
- BenchmarkConfigResolve/ten-profiles
- BenchmarkConfigValidate/clean-config
- BenchmarkConfigValidate/complex-config

### CLI Integration Tests
- `profiles list` output matches golden file
- `profiles show default` output matches golden file
- `profiles show finvault` (with fixture) output matches golden file
- `profiles lint` on clean config returns exit code 0
- `profiles lint` on broken config returns exit code 1
- `profiles explain src/main.ts` output matches golden file
- `config debug` output matches golden file

## References

- [Go testing package - Fuzz](https://pkg.go.dev/testing#F)
- [Go testing package - Benchmark](https://pkg.go.dev/testing#B)
- [rogpeppe/go-internal/testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)
- PRD Section 9.1 - "config: profile inheritance, merge behavior, validation, lint warnings"
- PRD Section 9.2 - Golden tests
- PRD Section 9.4 - "Fuzz config parsing with malformed TOML"
