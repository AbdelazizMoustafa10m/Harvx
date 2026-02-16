# T-094: Golden Test Infrastructure

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** Core pipeline complete, T-001 (testdata directories)
**Phase:** 6 - Polish & Distribution

---

## Description

Build the golden test infrastructure: curated sample repositories under `testdata/` with known file structures, and expected output files that serve as the "golden" reference. Tests run the pipeline against sample repos and compare output byte-for-byte against expected output. When output intentionally changes (due to format improvements), a `--update` flag regenerates the golden files. This catches unintended output regressions across the entire pipeline.

## User Story

As a developer making changes to the output renderer, I want golden tests to immediately tell me if I've accidentally changed the output format so that I can ensure backward compatibility and intentional changes are reviewed.

## Acceptance Criteria

- [ ] `testdata/sample-repo/` contains a curated TypeScript/Go/Python project with:
  - 20-30 files across 5+ directories
  - Known file sizes and content (deterministic)
  - Config files (package.json, go.mod, tsconfig.json)
  - Source files in multiple languages
  - Test files (\_test.go, .test.ts)
  - Documentation (README.md, docs/)
  - CI config (.github/)
  - A `.gitignore` with patterns to test ignore behavior
- [ ] `testdata/secrets/` contains mock secrets for redaction testing:
  - Files with realistic AWS keys (AKIA prefixed, correct format but fake)
  - Files with GitHub tokens (ghp\_ prefix, correct length but fake)
  - Files with Stripe keys (sk\_live\_ prefix)
  - Files with private key blocks (-----BEGIN RSA PRIVATE KEY-----)
  - Files with connection strings (postgres://, mongodb://)
  - Files with JWT tokens (eyJ prefix)
  - `.env` files with various `KEY=VALUE` patterns
  - False positive examples: test fixtures, documentation containing key format descriptions
- [ ] `testdata/monorepo/` contains a multi-package structure with:
  - 3+ packages/apps in subdirectories
  - Shared configuration at root
  - Nested `.gitignore` files
  - Build artifacts to verify ignore behavior
- [ ] `testdata/expected-output/` contains golden files for each test scenario:
  - `default-profile-markdown.md` -- default profile, Markdown format
  - `default-profile-xml.xml` -- default profile, XML format
  - `compressed-markdown.md` -- compression enabled
  - `redacted-markdown.md` -- redaction with known secret replacements
  - `finvault-profile-markdown.md` -- custom profile with tier assignments
  - `git-tracked-only-markdown.md` -- `--git-tracked-only` mode
  - `token-budget-10k.md` -- max_tokens=10000 (files omitted)
  - `preview-output.json` -- `harvx preview --json` output
- [ ] Golden test runner: `internal/golden/golden_test.go` with a `TestGolden` table-driven test
- [ ] `go test -run TestGolden -update` regenerates all golden files (writes new expected output)
- [ ] Golden comparison ignores timestamps and content hashes (these change between runs) using a normalizer function
- [ ] Clear diff output on failure: show unified diff between expected and actual output
- [ ] Golden tests run as part of `go test ./...` (not behind a build tag -- they should catch regressions on every PR)
- [ ] `make golden-update` target regenerates golden files
- [ ] Each golden file has a header comment documenting the command/profile used to generate it

## Technical Notes

- Golden test pattern: read expected file, run pipeline, compare. Use a `testutil.GoldenFile(t, name, actual)` helper:
  ```go
  func GoldenFile(t *testing.T, name string, actual []byte) {
      t.Helper()
      golden := filepath.Join("testdata", "expected-output", name)
      if *update {
          os.WriteFile(golden, actual, 0644)
          return
      }
      expected, err := os.ReadFile(golden)
      require.NoError(t, err)
      if !bytes.Equal(normalize(expected), normalize(actual)) {
          t.Errorf("golden mismatch for %s:\n%s", name, diff(expected, actual))
      }
  }
  ```
- Use `-update` flag via `flag.Bool("update", false, "update golden files")` in the test file.
- Normalization: replace timestamps (ISO 8601 patterns), content hashes (hex strings after "Hash:"), and absolute paths with placeholders.
- For unified diff output, use `sergi/go-diff` or `pmezard/go-difflib` to produce readable diffs.
- The `testdata/secrets/` directory should contain ONLY fake secrets that match the real patterns but are not actual credentials. Add a `README.md` in that directory explaining they are test fixtures.
- For git-tracked-only tests, the sample-repo needs to be a git repository. Initialize it with `git init` and `git add .` during test setup.
- Reference: PRD Section 9.2 (Golden Tests)

## Files to Create/Modify

- `testdata/sample-repo/` - Complete sample repository (20-30 files)
- `testdata/secrets/` - Mock secret files for redaction testing
- `testdata/secrets/README.md` - Explanation that these are fake test secrets
- `testdata/monorepo/` - Multi-package test repository
- `testdata/expected-output/` - Golden output files (8+ files)
- `internal/golden/golden_test.go` - Golden test runner
- `internal/golden/normalize.go` - Output normalization (timestamps, hashes, paths)
- `internal/golden/helpers.go` - Test helper functions (GoldenFile, diff)
- `Makefile` - Add `golden-update` target (modify)

## Testing Requirements

- All golden tests pass with current expected output
- `go test -run TestGolden -update` regenerates all golden files successfully
- Regenerated files match previous files when no pipeline changes were made
- Normalization correctly replaces timestamps, hashes, and paths
- Diff output is readable and shows exact location of differences
- Secret test fixtures contain realistic patterns but no real credentials
- Sample repo covers all file types mentioned in relevance tier defaults
- Monorepo test exercises nested .gitignore handling