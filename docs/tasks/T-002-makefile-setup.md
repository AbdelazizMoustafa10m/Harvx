# T-002: Makefile Setup

**Priority:** Must Have
**Effort:** Small (2-4hrs)
**Dependencies:** T-001
**Phase:** 1 - Foundation

---

## Description

Create a comprehensive Makefile that provides standard targets for building, testing, linting, and development workflows. The Makefile is the primary developer interface for building and testing Harvx, and it must support injecting version/commit/date metadata via `-ldflags` at build time.

## User Story

As a developer working on Harvx, I want a single `make build` command to compile the binary with proper version metadata so that I can quickly iterate without remembering complex `go build` flags.

## Acceptance Criteria

- [ ] `make build` compiles `cmd/harvx/main.go` into `bin/harvx` with ldflags injecting `Version`, `Commit`, `Date`, and `GoVersion`
- [ ] `make run` builds and runs the binary
- [ ] `make test` runs `go test ./...` with race detection enabled
- [ ] `make test-verbose` runs tests with `-v` flag
- [ ] `make test-cover` runs tests with coverage and outputs an HTML report
- [ ] `make lint` runs `golangci-lint run` (assumes it is installed; prints a helpful message if not)
- [ ] `make fmt` runs `gofmt -w` and `goimports -w` on all Go files
- [ ] `make vet` runs `go vet ./...`
- [ ] `make clean` removes the `bin/` directory and any build artifacts
- [ ] `make install` installs the binary to `$GOPATH/bin/harvx`
- [ ] `make all` runs `fmt`, `vet`, `lint`, `test`, `build` in sequence
- [ ] `make help` lists all available targets with descriptions
- [ ] Version metadata variables are defined at the top of the Makefile and extracted from git (tag, short SHA, date)
- [ ] The Makefile uses `.PHONY` declarations for all non-file targets
- [ ] `make build` successfully produces a working binary (depends on T-001 entry point existing)

## Technical Notes

- Use `-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.goVersion=$(shell go version)"` pattern for build-time metadata injection. These variables will be moved to an `internal/buildinfo` package in T-006.
- The ldflags target package will be updated from `main` to `github.com/.../internal/buildinfo` once that package exists (T-006). For now, inject into `main`.
- `golangci-lint` is the standard Go linter aggregator. Do not add it as a Go dependency -- it should be installed separately (`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`).
- Create a `.golangci.yml` configuration file with sensible defaults (enable `errcheck`, `staticcheck`, `unused`, `govet`, `ineffassign`, `gosimple`).
- Binary output directory: `bin/` (add to `.gitignore` if not already).
- Reference: PRD Section 6.2 (Makefile is listed in project structure)

## Files to Create/Modify

- `Makefile` - Build system
- `.golangci.yml` - Linter configuration
- `.gitignore` - Add `bin/` if not present

## Testing Requirements

- `make build` produces a binary at `bin/harvx`
- `make test` exits 0 (requires at least one test file; create a trivial one if needed)
- `make clean` removes `bin/`
- `make help` lists all targets
- Version info is embedded: `bin/harvx` (once version command exists) or `go version -m bin/harvx` shows ldflags
