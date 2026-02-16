# T-012: Default Ignore Patterns & .harvxignore Support

**Priority:** Must Have
**Effort:** Medium (6-8hrs)
**Dependencies:** T-011
**Phase:** 1 - Foundation

---

## Description

Implement the built-in default ignore patterns that Harvx always applies (e.g., `node_modules/`, `.git/`, `dist/`), and add support for `.harvxignore` files that provide tool-specific ignore patterns separate from `.gitignore`. Also implement the sensitive file exclusion patterns from PRD Section 7.3. The ignore chain is: defaults + `.gitignore` + `.harvxignore` + CLI `--exclude` flags.

## User Story

As a developer, I want Harvx to automatically skip common junk directories like `node_modules` and `.git` without configuration, and I want a `.harvxignore` file for patterns specific to context generation that I don't want in my `.gitignore`.

## Acceptance Criteria

- [ ] `internal/discovery/defaults.go` defines a `DefaultIgnorePatterns` string slice with all default patterns from the PRD:
  - `.git/`, `node_modules/`, `dist/`, `build/`, `coverage/`, `__pycache__/`, `.next/`, `target/`, `vendor/`, `.harvx/`
  - `.env`, `.env.*` (environment files)
  - `*.pem`, `*.key`, `*.p12`, `*.pfx` (certificate/key files)
  - `*secret*`, `*credential*`, `*password*` (sensitive naming patterns)
  - `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `Gemfile.lock`, `Cargo.lock`, `go.sum`, `poetry.lock` (lock files)
  - `*.pyc`, `*.pyo`, `*.class`, `*.o`, `*.obj`, `*.exe`, `*.dll`, `*.so`, `*.dylib` (compiled artifacts)
  - `.DS_Store`, `Thumbs.db`, `.idea/`, `.vscode/`, `*.swp`, `*.swo` (OS/editor files)
- [ ] Default patterns are compiled into a matcher at startup (same interface as `GitignoreMatcher`)
- [ ] `.harvxignore` files are parsed using the same gitignore pattern syntax (via `sabhiram/go-gitignore`)
- [ ] `.harvxignore` supports the same features as `.gitignore`: negation, directory patterns, wildcards, nested files
- [ ] `internal/discovery/ignore.go` defines a `CompositeIgnorer` that chains multiple ignore sources:
  1. Default patterns (always active)
  2. `.gitignore` patterns (from T-011)
  3. `.harvxignore` patterns
  4. CLI `--exclude` patterns (from T-014)
- [ ] `CompositeIgnorer.IsIgnored(path string, isDir bool) bool` returns true if ANY source matches
- [ ] When sensitive file defaults are overridden (e.g., user explicitly includes `*.pem`), a warning is logged to stderr
- [ ] Default patterns are exported for documentation/inspection purposes
- [ ] Unit tests cover all default patterns
- [ ] Unit tests verify `.harvxignore` loading and pattern matching
- [ ] Unit tests verify composite ignorer chaining behavior

## Technical Notes

- The `CompositeIgnorer` pattern:
  ```go
  type Ignorer interface {
      IsIgnored(path string, isDir bool) bool
  }

  type CompositeIgnorer struct {
      ignorers []Ignorer
  }

  func (c *CompositeIgnorer) IsIgnored(path string, isDir bool) bool {
      for _, ig := range c.ignorers {
          if ig.IsIgnored(path, isDir) {
              return true
          }
      }
      return false
  }
  ```
- `.harvxignore` uses the same `sabhiram/go-gitignore` parser as `.gitignore` -- the format is identical.
- Per PRD Section 5.1: "Respects `.harvxignore` for tool-specific ignore patterns" and "Applies a default set of ignore patterns."
- Per PRD Section 7.3: `.env` files, `.pem`/`.key`/`.p12`/`.pfx` files, and `*secret*`/`*credential*`/`*password*` patterns are excluded by default. "These defaults can be overridden in profile configuration, but a warning is emitted."
- Per PRD Section 5.8: "State files are gitignored by default (Harvx adds `.harvx/` to its default ignore list)."
- The `bmatcuk/doublestar` package (v4) will be used for `--include`/`--exclude` glob matching in T-014, but default patterns can use the simpler `go-gitignore` syntax.
- Reference: PRD Sections 5.1, 7.3, 5.8

## Files to Create/Modify

- `internal/discovery/defaults.go` - Default ignore patterns list
- `internal/discovery/ignore.go` - CompositeIgnorer and Ignorer interface
- `internal/discovery/harvxignore.go` - .harvxignore file loading
- `internal/discovery/defaults_test.go` - Unit tests for default patterns
- `internal/discovery/ignore_test.go` - Unit tests for composite ignorer
- `internal/discovery/harvxignore_test.go` - Unit tests for .harvxignore
- `testdata/harvxignore/` - Test fixtures with sample .harvxignore files

## Testing Requirements

- Unit test: `node_modules/` is ignored by default
- Unit test: `.git/` is ignored by default
- Unit test: `.env` and `.env.local` are ignored by default
- Unit test: `*.pem` and `*.key` are ignored by default
- Unit test: `*secret*` pattern matches `my-secret-config.yml`
- Unit test: lock files (`package-lock.json`, `go.sum`, etc.) are ignored by default
- Unit test: compiled artifacts (`*.pyc`, `*.class`, etc.) are ignored by default
- Unit test: `.harvxignore` patterns are loaded and applied
- Unit test: `.harvxignore` negation patterns work
- Unit test: composite ignorer returns true if any source matches
- Unit test: composite ignorer returns false if no source matches
- Unit test: missing `.harvxignore` is handled gracefully (no error)
- Unit test: sensitive file override triggers warning log
