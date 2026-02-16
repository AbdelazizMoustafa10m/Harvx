# T-018: Configuration File Auto-Detection and Discovery

**Priority:** Must Have
**Effort:** Small (3-5hrs)
**Dependencies:** T-016 (Config Types & Defaults), T-017 (Multi-Source Config Merging)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Implement auto-detection of `harvx.toml` configuration files by walking up the directory tree from the current working directory (or `--dir` target). Also implement discovery of the global config at `~/.config/harvx/config.toml` using XDG-compatible paths. This ensures Harvx automatically loads project-specific configuration without requiring explicit `--profile-file` flags.

## User Story

As a developer, I want Harvx to automatically find and load my project's `harvx.toml` file when I run the command from any subdirectory of my project, so that I don't have to specify the config path every time.

## Acceptance Criteria

- [ ] `internal/config/discover.go` implements `DiscoverRepoConfig(startDir string) (string, error)`:
  - Walks from `startDir` up to filesystem root looking for `harvx.toml`
  - Returns the absolute path of the first `harvx.toml` found, or empty string if none
  - Stops at filesystem root (or a configurable max depth of 20 levels to prevent runaway)
  - Stops early if a `.git` directory is found at the same level (repo root boundary)
- [ ] `internal/config/discover.go` implements `DiscoverGlobalConfig() (string, error)`:
  - Returns `~/.config/harvx/config.toml` on Linux/macOS
  - Respects `XDG_CONFIG_HOME` if set (returns `$XDG_CONFIG_HOME/harvx/config.toml`)
  - Returns `%APPDATA%\harvx\config.toml` on Windows
  - Returns empty string if the file does not exist (no error)
- [ ] Discovery is integrated into the resolver from T-017:
  - If no `--profile-file` is given, auto-discover repo config
  - Always attempt to load global config
- [ ] The `--dir` flag is respected as the starting directory for auto-detection
- [ ] If both `--profile-file` and auto-detection find configs, `--profile-file` wins (repo auto-detect is skipped)
- [ ] Symlinks in the directory chain are resolved before walking
- [ ] Unit tests cover Linux, macOS, and Windows path scenarios
- [ ] Discovery does not follow symlinks that point outside the directory tree (security)

## Technical Notes

- Use `os.UserConfigDir()` (Go stdlib, available since Go 1.13) for XDG/AppData detection -- it handles Linux (`$XDG_CONFIG_HOME` or `~/.config`), macOS (`~/Library/Application Support`), and Windows (`%AppData%`) automatically. However, for Harvx, prefer the XDG convention on macOS too (`~/.config/harvx/`) since CLI tools conventionally use `~/.config` even on macOS. Override `os.UserConfigDir()` on macOS to use `~/.config` instead of `~/Library/Application Support`.
- Use `filepath.Abs()` and `filepath.Dir()` for directory traversal
- Use `os.Stat()` to check file existence (not `os.Open` -- just check, don't read)
- The `.git` boundary check uses `os.Stat(filepath.Join(dir, ".git"))` -- if it exists, this is the repo root; check for `harvx.toml` here and stop
- Edge case: monorepo with `harvx.toml` in a subdirectory -- the first one found (closest to CWD) wins
- This is a pure filesystem operation -- no TOML parsing here; just path discovery

## Files to Create/Modify

- `internal/config/discover.go` - Config file discovery logic
- `internal/config/discover_test.go` - Unit tests with temp directory fixtures
- `internal/config/resolver.go` - Integrate discovery into Resolve() (modify from T-017)

## Testing Requirements

- Unit test: `harvx.toml` in current directory is found
- Unit test: `harvx.toml` in parent directory (2 levels up) is found
- Unit test: No `harvx.toml` anywhere returns empty string
- Unit test: `.git` boundary stops the search (won't look above repo root)
- Unit test: Max depth (20 levels) prevents infinite traversal
- Unit test: Global config at `~/.config/harvx/config.toml` is discovered
- Unit test: `XDG_CONFIG_HOME` override is respected
- Unit test: Missing global config returns empty string (not error)
- Unit test: `--profile-file` bypasses auto-detection
- Edge case: Symlink loop in directory tree does not cause infinite loop
- Edge case: Permission denied on a parent directory is handled gracefully

## References

- [os.UserConfigDir() documentation](https://pkg.go.dev/os#UserConfigDir)
- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/)
- PRD Section 5.2 - Auto-detection: "if a `harvx.toml` exists in the current directory or any parent directory, it is automatically loaded"
- PRD Section 5.2 - Global config: `~/.config/harvx/config.toml`
