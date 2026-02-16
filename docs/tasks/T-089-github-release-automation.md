# T-089: GitHub Release Automation Workflow

**Priority:** Must Have
**Effort:** Small (4-6hrs)
**Dependencies:** T-088 (GoReleaser configuration must exist)
**Phase:** 6 - Polish & Distribution

---

## Description

Create the GitHub Actions workflow that automates the entire release process: on pushing a semantic version tag (e.g., `v1.0.0`), the workflow checks out the code, sets up Go, installs Cosign and Syft, runs GoReleaser to produce cross-platform binaries, signs artifacts, generates SBOMs, and publishes everything as a GitHub Release with auto-generated release notes.

## User Story

As a maintainer of Harvx, I want to trigger a complete release by pushing a version tag so that the entire build, sign, and publish process is automated and reproducible.

## Acceptance Criteria

- [ ] `.github/workflows/release.yml` workflow triggers on `push` of tags matching `v*.*.*`
- [ ] Workflow uses `actions/checkout@v4` with `fetch-depth: 0` (required by GoReleaser for changelog)
- [ ] Sets up Go using `actions/setup-go@v5` with Go version from `go.mod`
- [ ] Installs Cosign via `sigstore/cosign-installer@v3`
- [ ] Installs Syft via `anchore/sbom-action/download-syft@v0`
- [ ] Runs GoReleaser via `goreleaser/goreleaser-action@v6` with version `~> v2`
- [ ] `GITHUB_TOKEN` is passed for GitHub Release creation
- [ ] Release includes: platform archives, checksums, Cosign signatures (.sig + .pem), SBOM files
- [ ] Release notes are auto-generated from conventional commits between tags
- [ ] Workflow also runs on PRs (with `goreleaser build --snapshot` only, no publish) for CI validation
- [ ] Separate CI workflow `.github/workflows/ci.yml` runs tests, lint, vet on every push/PR
- [ ] CI workflow includes: `go test ./...`, `go vet ./...`, `golangci-lint run`, `goreleaser check`
- [ ] Workflow permissions set correctly: `contents: write` (for release), `id-token: write` (for Cosign OIDC)
- [ ] README includes badges for CI status and latest release

## Technical Notes

- GoReleaser Action v6 with `version: "~> v2"` ensures GoReleaser v2 is used.
- `fetch-depth: 0` is critical -- GoReleaser needs the full git history to generate changelogs.
- Cosign keyless signing requires `id-token: write` permission to request an OIDC token from GitHub Actions.
- The workflow should cache Go modules: use `actions/setup-go@v5` which has built-in caching, or explicitly cache `~/go/pkg/mod`.
- For PR validation, run `goreleaser build --snapshot --clean` without publishing. This catches configuration errors before merge.
- CI workflow should be separate from release workflow. CI runs on every push and PR; release only on tags.
- Consider adding a `release-please` or manual dispatch trigger for creating release tags systematically.
- Go version should be read from `go.mod` dynamically using `go-version-file: 'go.mod'` in the setup-go action.
- Reference: PRD Section 10 Phase 6 (GitHub Release automation), GoReleaser Actions docs (https://goreleaser.com/ci/actions/)

## Files to Create/Modify

- `.github/workflows/release.yml` - Release automation workflow
- `.github/workflows/ci.yml` - CI workflow (test, lint, vet, goreleaser check)
- `README.md` - Add CI and release badges (modify)

## Testing Requirements

- Workflow YAML passes `yamllint` validation
- `act` (local GitHub Actions runner) can execute the workflow in dry-run mode
- GoReleaser check passes within the CI workflow
- Release workflow produces expected artifacts on a test tag (can be verified with `--snapshot`)
- CI workflow correctly runs all test suites
- Permissions are correctly scoped (no overly broad permissions)