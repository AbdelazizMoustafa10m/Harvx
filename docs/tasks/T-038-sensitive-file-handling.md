# T-038: Sensitive File Default Exclusions and Heightened Scanning

**Priority:** Must Have
**Effort:** Small (4-6hrs)
**Dependencies:** T-034, T-037
**Phase:** 3 - Security

---

## Description

Implement the sensitive file handling layer specified in PRD Section 7.3. This includes default exclusions for inherently sensitive file types (`.env`, `.pem`, `.key`, `.p12`, `.pfx`, files matching `*secret*`, `*credential*`, `*password*`), integration with the file discovery system to exclude these by default, and a warning system that emits a log message when a profile explicitly overrides these exclusions. Additionally, files like `.env` and `*.pem` that survive into the pipeline (e.g., because a profile overrides the default exclusion) trigger heightened scanning mode in the redactor.

## User Story

As a developer, I want Harvx to automatically exclude sensitive files like `.env` and private key files from the output by default, and warn me if I override this behavior, so that I am protected from accidental exposure even before the redaction engine runs.

## Acceptance Criteria

- [ ] Default sensitive file exclusion patterns defined as a constant list:
  - `.env`, `.env.*`, `.env.local`, `.env.production`, `.env.development`
  - `*.pem`, `*.key`, `*.p12`, `*.pfx`, `*.jks`, `*.keystore`
  - `*secret*`, `*credential*`, `*password*`
  - `id_rsa`, `id_dsa`, `id_ecdsa`, `id_ed25519`
  - `.htpasswd`, `.netrc`, `.npmrc` (when containing auth tokens)
  - `*.gpg`, `*.asc` (encrypted/signed files)
- [ ] These patterns are injected into the file discovery system's default ignore list (alongside `node_modules/`, `.git/`, etc.)
- [ ] When a profile explicitly includes a file matching a sensitive pattern (via `include` or by overriding the ignore), a warning is emitted via `slog.Warn`: `"Sensitive file included by profile override"` with file path and pattern matched
- [ ] The warning does NOT block processing -- it is informational only
- [ ] `IsSensitiveFile(path string) bool` function exported for use by the redactor's heightened scanning mode
- [ ] `SensitiveFilePatterns() []string` returns the full list of default exclusion patterns
- [ ] The patterns are configurable: profiles can add to the list via `redaction.sensitive_patterns` but cannot silently remove from the defaults (removal requires explicit `redaction.override_sensitive_defaults = true` which triggers the warning)
- [ ] Unit tests verify all default patterns match expected file paths
- [ ] Integration with discovery: `internal/discovery/walker.go` applies sensitive file patterns as part of the default ignore set

## Technical Notes

- **Two distinct behaviors**: (1) File discovery exclusion -- prevents sensitive files from being discovered at all. (2) Heightened scanning -- for files that ARE discovered (because of explicit include), the redactor applies stricter scanning. These are separate mechanisms with a shared pattern list.
- **Pattern matching**: Use `bmatcuk/doublestar/v4` `Match()` for glob pattern evaluation, consistent with the rest of the codebase.
- **Discovery integration**: The default ignore patterns list (defined in `internal/config/defaults.go` or equivalent) should include the sensitive file patterns. This task extends that list, NOT replaces it. The sensitive file patterns are additive to the existing defaults (`.git/`, `node_modules/`, etc.).
- **Warning mechanism**: Use `log/slog` with `slog.Warn("sensitive file included by profile override", "path", filePath, "matched_pattern", pattern)`. This goes to stderr and is visible even in default log level.
- **Override semantics**: The profile can set `redaction.override_sensitive_defaults = true` to suppress warnings. This is a conscious opt-in for advanced users who know what they are doing (e.g., they want to include `.env.example` files that contain placeholder values).
- Reference: PRD Section 7.3 "Sensitive File Handling"

## Files to Create/Modify

- `internal/security/sensitive.go` - `IsSensitiveFile`, `SensitiveFilePatterns`, pattern constants (may already be partially created in T-037; this task completes and formalizes it)
- `internal/security/sensitive_test.go` - Comprehensive pattern matching tests
- `internal/config/defaults.go` - Add sensitive patterns to default ignore list (create or modify)
- `internal/discovery/walker.go` - Integration point: apply sensitive file exclusions during walk (modify existing)

## Testing Requirements

- **Pattern matching tests** (table-driven):
  - `.env` -> sensitive
  - `.env.local` -> sensitive
  - `.env.production` -> sensitive
  - `config/.env` -> sensitive
  - `secrets/api.key` -> sensitive
  - `certs/server.pem` -> sensitive
  - `id_rsa` -> sensitive
  - `~/.ssh/id_ed25519` -> sensitive
  - `credentials.json` -> sensitive (matches `*credential*`)
  - `password_store.txt` -> sensitive (matches `*password*`)
- **Negative tests**:
  - `environment.go` -> NOT sensitive (does not match `.env` pattern)
  - `pkg/keys/handler.go` -> NOT sensitive (`.go` is not `.key`)
  - `docs/security-overview.md` -> NOT sensitive
  - `.env.example` -> IS sensitive by default (contains `.env` prefix)
- **Warning emission tests**: Mock slog handler, verify warning is emitted when sensitive file is explicitly included
- **Override tests**: Verify `override_sensitive_defaults = true` suppresses the warning
- **Discovery integration tests**: Verify that sensitive files are skipped during file walk with default config
