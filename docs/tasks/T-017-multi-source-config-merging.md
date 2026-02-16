# T-017: Multi-Source Configuration Merging and Resolution

**Priority:** Must Have
**Effort:** Large (14-20hrs)
**Dependencies:** T-016 (Config Types & Defaults), T-002 (Cobra CLI Setup)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Implement the multi-source configuration resolution engine that merges settings from four layers in order of precedence: (1) built-in defaults, (2) global config at `~/.config/harvx/config.toml`, (3) repository config at `harvx.toml`, and (4) CLI flags. This also includes environment variable overrides with the `HARVX_` prefix. The task uses `knadh/koanf/v2` as the merging engine (chosen over Viper for smaller binary size and cleaner abstractions per PRD guidance).

## User Story

As a developer, I want my CLI flags to override my project config, which overrides my global config, which overrides sensible defaults, so that I have full control over Harvx behavior at every level.

## Acceptance Criteria

- [ ] Configuration is resolved from four sources in order (lowest to highest precedence):
  1. Built-in defaults (from T-016)
  2. Global config: `~/.config/harvx/config.toml`
  3. Repository config: `harvx.toml` (auto-detected, see T-018)
  4. CLI flags and environment variables
- [ ] `internal/config/resolver.go` implements `Resolve(opts ResolveOptions) (*ResolvedConfig, error)`:
  - `ResolveOptions` includes: CLI flag values, profile name, profile file path, target directory
  - `ResolvedConfig` includes: the merged profile, plus source annotations (which value came from which layer)
- [ ] Environment variables with `HARVX_` prefix override config file values:
  - `HARVX_PROFILE` - profile name
  - `HARVX_MAX_TOKENS` - token budget
  - `HARVX_FORMAT` - output format
  - `HARVX_TOKENIZER` - tokenizer encoding
  - `HARVX_OUTPUT` - output file path
  - `HARVX_TARGET` - LLM target preset
  - `HARVX_LOG_FORMAT` - log format (text/json)
  - `HARVX_COMPRESS` - enable compression (bool)
  - `HARVX_REDACT` - enable redaction (bool)
- [ ] The `--profile <name>` flag selects a named profile from the loaded config
- [ ] The `--profile-file <path>` flag loads a standalone profile file (overrides repo config)
- [ ] LLM target presets (`--target`) apply sensible defaults:
  - `claude`: XML format, 200K token budget
  - `chatgpt`: Markdown format, 128K token budget
  - `generic`: Markdown format, no budget preset
- [ ] Source annotations track where each config value originated (for `config debug` command)
- [ ] CLI flags always win -- even over env vars
- [ ] If no profile is specified, use the `default` profile
- [ ] If a named profile does not exist, return a clear error listing available profiles
- [ ] Unit tests achieve 90%+ coverage
- [ ] Integration test: full 4-layer merge produces expected resolved config

## Technical Notes

- Use `github.com/knadh/koanf/v2` as the configuration merging library (lighter than Viper, 313% smaller binary per koanf wiki)
- Install providers separately: `koanf/providers/file`, `koanf/providers/env`, `koanf/providers/confmap`, `koanf/parsers/toml`
- Koanf merge order: load defaults via `confmap` provider, then load global file, then repo file, then env vars -- each `Load()` call merges on top of previous
- For CLI flags, use `koanf/providers/posflag` to bridge Cobra pflags into koanf
- The `koanf.New(".")` delimiter allows nested key access like `profile.finvault.relevance.tier_0`
- Source annotations: maintain a parallel map of `key -> source` (default/global/repo/env/flag) by tracking which keys change after each `Load()` call
- Do NOT implement profile inheritance in this task -- that is T-019
- The resolver should be usable as a library function (not coupled to Cobra command struct)
- Consider thread safety: resolver returns a new `ResolvedConfig` each time, no shared mutable state

## Files to Create/Modify

- `internal/config/resolver.go` - Multi-source config resolution engine
- `internal/config/resolver_test.go` - Comprehensive merge tests
- `internal/config/sources.go` - Source annotation tracking
- `internal/config/env.go` - Environment variable mapping and parsing
- `internal/config/env_test.go` - Env var override tests
- `internal/config/target.go` - LLM target preset definitions
- `internal/config/target_test.go` - Target preset tests
- `internal/cli/flags.go` - Cobra flag registration (if not already in T-002) with koanf bridge
- `testdata/config/global.toml` - Test global config
- `testdata/config/repo.toml` - Test repo config

## Testing Requirements

- Unit test: Defaults-only resolution (no files, no flags) returns built-in defaults
- Unit test: Global config overrides defaults
- Unit test: Repo config overrides global config
- Unit test: CLI flags override everything
- Unit test: Env vars override config files but not CLI flags
- Unit test: `HARVX_MAX_TOKENS=50000` correctly parsed as int
- Unit test: `HARVX_COMPRESS=true` correctly parsed as bool
- Unit test: `--target claude` sets format=xml and max_tokens=200000
- Unit test: `--profile nonexistent` returns error with available profile names
- Unit test: `--profile-file` loads standalone file and uses it
- Unit test: Source annotations correctly track origin of each value
- Integration test: Load global.toml + repo.toml + env vars + flags and verify final merged config
- Edge case: Empty global config does not break merge
- Edge case: Missing global config file (file not found) is silently ignored

## References

- [knadh/koanf v2](https://github.com/knadh/koanf)
- [koanf comparison with Viper](https://github.com/knadh/koanf/wiki/Comparison-with-spf13-viper)
- [koanf environment provider](https://github.com/knadh/koanf/blob/master/providers/env/env.go)
- [koanf TOML parser](https://pkg.go.dev/github.com/knadh/koanf/parsers/toml)
- PRD Section 5.2 - Three configuration scopes
- PRD Section 5.9 - Environment variable overrides (HARVX_ prefix)
