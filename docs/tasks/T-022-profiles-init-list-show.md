# T-022: Profile CLI Subcommands -- init, list, show

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-002 (Cobra CLI), T-017 (Multi-Source Config Merging), T-019 (Profile Inheritance), T-021 (Templates)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Implement the first three profile management subcommands under `harvx profiles`: `init` (generate a starter config), `list` (show available profiles), and `show` (display resolved configuration for a named profile). These are the primary user-facing commands for profile discovery and setup, registered as Cobra subcommands under a `profiles` parent command.

## User Story

As a developer new to Harvx, I want to run `harvx profiles init --template nextjs` to generate a starter configuration, then `harvx profiles list` to see my profiles, and `harvx profiles show finvault` to verify the resolved settings, so that I can understand and manage my configuration.

## Acceptance Criteria

### `harvx profiles list`
- [ ] Lists all available profiles from all config sources (global + repo + built-in)
- [ ] Shows profile name, source (built-in / global / repo), and whether it extends another
- [ ] Highlights the active/default profile
- [ ] Uses tabular output with proper alignment via `text/tabwriter`
- [ ] Shows template names as available starting points
- [ ] Example output:
  ```
  Available Profiles:

    NAME       SOURCE    EXTENDS    DESCRIPTION
    default    built-in  -          Built-in defaults for any repository
    finvault   repo      default    Loaded from ./harvx.toml
    work       repo      default    Loaded from ./harvx.toml
    session    repo      default    Loaded from ./harvx.toml

  Templates (use with `harvx profiles init --template <name>`):
    nextjs, go-cli, python-django, rust-cargo, monorepo
  ```

### `harvx profiles init`
- [ ] Without `--template`: generates a minimal `harvx.toml` with the base template
- [ ] With `--template <name>`: generates `harvx.toml` using the specified framework template
- [ ] Writes to `./harvx.toml` in the current directory
- [ ] If `harvx.toml` already exists, prompts for confirmation (unless `--yes` is passed)
- [ ] With `--output <path>`: writes to the specified path instead
- [ ] Shows a success message with next steps
- [ ] Example output:
  ```
  Created harvx.toml (template: nextjs)

  Next steps:
    1. Review and customize the profile settings
    2. Run `harvx profiles lint` to validate
    3. Run `harvx preview` to see what would be included
  ```

### `harvx profiles show <name>`
- [ ] Resolves the named profile (including inheritance) and displays the fully merged config
- [ ] Output is valid TOML that could be used as a standalone profile
- [ ] Shows source annotations: which values came from which layer (default/global/repo/flag)
- [ ] With `--json` flag: outputs as JSON instead of TOML
- [ ] Without `<name>` argument: shows the active/default profile
- [ ] If profile not found, lists available profiles in error message
- [ ] Example output:
  ```toml
  # Resolved profile: finvault
  # Inheritance chain: finvault -> default
  # Sources: repo (./harvx.toml) + built-in defaults

  output = ".harvx/finvault-context.md"     # repo
  format = "markdown"                        # default
  max_tokens = 200000                        # repo
  tokenizer = "o200k_base"                   # repo
  compression = true                         # repo
  redaction = true                           # default
  target = "claude"                          # repo

  priority_files = [                         # repo
    "CLAUDE.md",
    "prisma/schema.prisma",
  ]

  [relevance]                                # repo
  tier_0 = ["CLAUDE.md", "prisma/schema.prisma"]
  tier_1 = ["app/api/**", "lib/services/**"]
  # ...
  ```

### General
- [ ] All three subcommands are registered under `harvx profiles` parent command
- [ ] `harvx profiles` with no subcommand shows help text listing available subcommands
- [ ] Shell completions work for `harvx profiles <TAB>` showing subcommand names
- [ ] Shell completions work for `harvx profiles show <TAB>` listing profile names
- [ ] Shell completions work for `harvx profiles init --template <TAB>` listing template names

## Technical Notes

- Use `spf13/cobra` to register subcommands: `profilesCmd` parent with `listCmd`, `initCmd`, `showCmd` children
- For `profiles list`: use `text/tabwriter` from stdlib for aligned table output
- For `profiles show`: use `BurntSushi/toml` encoder (`toml.NewEncoder().Encode()`) to serialize the resolved config back to TOML
- For `profiles init`: read template via T-021's `GetTemplate()` or `RenderTemplate()`, then `os.WriteFile()`
- Source annotations (from T-017) should be rendered as inline TOML comments (`# source`)
- Profile name completion: use Cobra's `RegisterFlagCompletionFunc` and `ValidArgsFunction`
- Template name completion: hardcoded list via `cobra.FixedCompletions`
- The `--yes` flag should be a persistent flag on the root command (shared across subcommands)
- Use `charmbracelet/lipgloss` for styled terminal output if available (from Phase 1), but degrade gracefully to plain text if not yet integrated
- All user-facing output goes to stderr (stdout reserved for piped content)

## Files to Create/Modify

- `internal/cli/profiles.go` - Parent `profiles` command and `list`, `init`, `show` subcommands
- `internal/cli/profiles_test.go` - Command execution tests
- `internal/config/show.go` - Profile serialization with source annotations
- `internal/config/show_test.go` - Serialization tests

## Testing Requirements

- Unit test: `profiles list` shows built-in default profile
- Unit test: `profiles list` shows profiles from loaded config
- Unit test: `profiles list` shows available templates
- Unit test: `profiles init` creates `harvx.toml` in current directory
- Unit test: `profiles init --template nextjs` uses Next.js template
- Unit test: `profiles init` with existing file prompts (returns error without `--yes`)
- Unit test: `profiles init --yes` overwrites without prompting
- Unit test: `profiles show default` shows built-in defaults
- Unit test: `profiles show finvault` shows resolved config with inheritance
- Unit test: `profiles show --json` outputs valid JSON
- Unit test: `profiles show nonexistent` returns error with available names
- Unit test: Shell completion for subcommand names works
- Unit test: Shell completion for profile names works
- Integration test: Full flow -- init, list, show in sequence

## References

- [Cobra subcommands](https://pkg.go.dev/github.com/spf13/cobra)
- [text/tabwriter](https://pkg.go.dev/text/tabwriter)
- [BurntSushi/toml encoder](https://pkg.go.dev/github.com/BurntSushi/toml#Encoder)
- PRD Section 5.2 - `harvx profiles list`, `harvx profiles init`, `harvx profiles show <name>`
- PRD Section 5.9 - Profile management subcommands
