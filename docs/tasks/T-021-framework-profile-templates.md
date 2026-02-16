# T-021: Framework-Specific Profile Templates

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-016 (Config Types & Defaults), T-020 (Config Validation)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Create five framework-specific profile templates (Next.js, Go CLI, Python Django, Rust Cargo, Monorepo) that ship embedded in the Harvx binary. These templates are used by `harvx profiles init --template <name>` and provide optimized relevance tiers, ignore patterns, and priority files tailored to each framework's conventions. Templates are embedded TOML files using Go's `//go:embed` directive.

## User Story

As a developer starting to use Harvx on a Next.js project, I want to quickly generate an optimized configuration with `harvx profiles init --template nextjs`, so that I don't have to manually figure out which files are important and which should be ignored.

## Acceptance Criteria

- [ ] Five template files embedded in the binary via `//go:embed`:
  - `templates/nextjs.toml` - Next.js / React projects
  - `templates/go-cli.toml` - Go CLI applications
  - `templates/python-django.toml` - Python Django projects
  - `templates/rust-cargo.toml` - Rust Cargo projects
  - `templates/monorepo.toml` - Monorepo with multiple packages
- [ ] `internal/config/templates.go` implements:
  - `ListTemplates() []TemplateInfo` - returns name + description for each template
  - `GetTemplate(name string) (string, error)` - returns raw TOML content
  - `RenderTemplate(name string, projectName string) (string, error)` - returns TOML with project name filled in
- [ ] Each template includes a header comment explaining the configuration
- [ ] Each template defines:
  - Sensible `ignore` patterns for the framework (e.g., `.next/` for Next.js, `target/` for Rust)
  - Framework-appropriate `priority_files` (e.g., `next.config.js`, `package.json` for Next.js)
  - Relevance tiers tuned for the framework's directory structure
  - Appropriate `max_tokens`, `format`, and `tokenizer` defaults
- [ ] A `base.toml` template also exists as a minimal starter (used when no `--template` is specified)
- [ ] All templates pass validation (T-020) without errors or warnings
- [ ] Templates include comments explaining each section for user education
- [ ] Unit tests verify all templates are valid TOML and pass config validation

## Technical Notes

- Use `//go:embed templates/*.toml` in `internal/config/templates.go` to embed all template files
- Use `embed.FS` for the embedded filesystem
- Template rendering: simple `strings.Replace` for `{{project_name}}` placeholder -- no need for full `text/template` since these are static TOML with one variable
- Each template file should be a complete, standalone `harvx.toml` that can be dropped into a project
- Template quality matters -- these are the first experience for many users

### Template Content Guidelines

**nextjs.toml:**
- Tier 0: `next.config.*`, `package.json`, `tsconfig.json`, `.env.example`
- Tier 1: `app/**/*.ts`, `app/**/*.tsx`, `pages/**`, `lib/**`, `middleware.ts`
- Tier 2: `components/**`, `hooks/**`, `utils/**`, `styles/**`
- Tier 3: `__tests__/**`, `*.test.*`, `*.spec.*`, `cypress/**`
- Tier 4: `docs/**`, `*.md`, `README*`
- Tier 5: `.github/**`, `*.lock`, `.eslintrc*`, `.prettierrc*`
- Ignore: `.next/`, `node_modules/`, `out/`, `coverage/`, `.vercel/`

**go-cli.toml:**
- Tier 0: `go.mod`, `go.sum`, `Makefile`, `main.go`, `cmd/*/main.go`
- Tier 1: `cmd/**`, `internal/**`, `pkg/**`
- Tier 2: `api/**`, `proto/**`, `configs/**`
- Tier 3: `*_test.go`, `testdata/**`
- Tier 4: `docs/**`, `*.md`
- Tier 5: `.github/**`, `.goreleaser*`, `vendor/**`
- Ignore: `vendor/`, `dist/`, `bin/`, `coverage/`

**python-django.toml:**
- Tier 0: `manage.py`, `requirements*.txt`, `pyproject.toml`, `settings.py`, `urls.py`
- Tier 1: `*/views.py`, `*/models.py`, `*/serializers.py`, `*/urls.py`
- Tier 2: `*/forms.py`, `*/admin.py`, `*/signals.py`, `*/middleware.py`, `templates/**`
- Tier 3: `*/tests/**`, `tests/**`, `conftest.py`
- Tier 4: `docs/**`, `*.md`
- Tier 5: `.github/**`, `*.lock`, `static/**`
- Ignore: `__pycache__/`, `*.pyc`, `.venv/`, `venv/`, `media/`, `staticfiles/`, `*.egg-info/`

**rust-cargo.toml:**
- Tier 0: `Cargo.toml`, `Cargo.lock`, `build.rs`, `rust-toolchain.toml`
- Tier 1: `src/main.rs`, `src/lib.rs`, `src/**/*.rs`
- Tier 2: `benches/**`, `examples/**`
- Tier 3: `tests/**`
- Tier 4: `docs/**`, `*.md`
- Tier 5: `.github/**`, `.cargo/**`
- Ignore: `target/`

**monorepo.toml:**
- Tier 0: Root config files, workspace definitions
- Tier 1: `packages/*/src/**`, `apps/*/src/**`
- Tier 2: Shared libraries, common utilities
- Tier 3: Test files across all packages
- Tier 4: Package-level docs
- Tier 5: CI/CD, lock files
- Ignore: `node_modules/`, `dist/`, `build/`, `.turbo/`, `.nx/`

## Files to Create/Modify

- `templates/base.toml` - Minimal starter template
- `templates/nextjs.toml` - Next.js profile template
- `templates/go-cli.toml` - Go CLI profile template
- `templates/python-django.toml` - Django profile template
- `templates/rust-cargo.toml` - Rust Cargo profile template
- `templates/monorepo.toml` - Monorepo profile template
- `internal/config/templates.go` - Template embedding and retrieval API
- `internal/config/templates_test.go` - Template validation tests

## Testing Requirements

- Unit test: `ListTemplates()` returns exactly 6 templates (base + 5 frameworks)
- Unit test: `GetTemplate("nextjs")` returns valid TOML content
- Unit test: `GetTemplate("nonexistent")` returns error
- Unit test: Each template decodes into valid `Config` struct
- Unit test: Each template passes `Validate()` with no errors
- Unit test: `RenderTemplate("nextjs", "myproject")` replaces project name
- Unit test: Template TOML comments are preserved (check raw string)
- Unit test: All template files are accessible via embed.FS
- Golden test: Each template round-trips through decode/encode correctly

## References

- [Go embed directive](https://pkg.go.dev/embed)
- PRD Section 5.2 - "Ships with framework-specific profile templates"
- PRD Section 5.2 - Template list: nextjs, go-cli, python-django, rust-cargo, monorepo
- PRD Section 5.9 - `harvx profiles init [--template <framework>]`
