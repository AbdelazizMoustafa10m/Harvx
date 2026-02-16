# T-088: GoReleaser Configuration with Cosign Signing & Syft SBOM

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-001 (project initialized), T-002 (Makefile exists)
**Phase:** 6 - Polish & Distribution

---

## Description

Configure GoReleaser for cross-platform binary builds targeting macOS (amd64/arm64), Linux (amd64/arm64), and Windows (amd64). Integrate Cosign keyless signing via Sigstore for binary authenticity verification and Syft for Software Bill of Materials (SBOM) generation. Every release artifact will include SHA-256 checksums, a signed SBOM, and signed binaries, fulfilling PRD Section 7.4 (Supply Chain Security).

## User Story

As a developer distributing Harvx, I want automated cross-platform builds with signed binaries and SBOM so that users can verify the authenticity and integrity of every release, and security-conscious organizations can audit our dependencies.

## Acceptance Criteria

- [ ] `.goreleaser.yaml` configures builds for 5 targets: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`
- [ ] `CGO_ENABLED=0` set for all builds (pure Go, no CGO dependencies -- required for cross-compilation)
- [ ] Build ldflags inject version, commit hash, and build date: `-X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}`
- [ ] Binary name is `harvx` (or `harvx.exe` on Windows)
- [ ] Archives: `.tar.gz` for macOS/Linux, `.zip` for Windows
- [ ] Archives include `LICENSE`, `README.md`, and shell completion files
- [ ] Checksum file (`harvx_checksums.txt`) with SHA-256 hashes for all archives
- [ ] Cosign keyless signing enabled: signs the checksum file using Sigstore OIDC (GitHub Actions identity)
- [ ] Syft SBOM generation: produces `harvx_sbom.spdx.json` (SPDX format) for each archive
- [ ] SBOM is attached to the GitHub Release as an additional artifact
- [ ] `goreleaser check` validates the configuration without errors
- [ ] `goreleaser build --snapshot --clean` produces all 5 binaries locally
- [ ] Makefile targets: `make release-snapshot` (local test), `make release-check` (validate config)
- [ ] Version info embedded correctly: `./harvx version` displays version, commit, date

## Technical Notes

- Use GoReleaser v2 (latest stable). Configuration file is `.goreleaser.yaml` (YAML, not YML -- GoReleaser v2 convention).
- Cosign keyless signing via Sigstore: uses OIDC token from GitHub Actions to sign artifacts without managing private keys. The signing identity is the GitHub Actions workflow, verifiable with `cosign verify-blob`.
- Syft must be available in the CI environment. Install via `anchore/sbom-action` or `goreleaser/goreleaser-action` which handles Syft setup.
- GoReleaser SBOM configuration:
  ```yaml
  sboms:
    - artifacts: archive
      cmd: syft
      args: ["${artifact}", "--output", "spdx-json=${document}"]
  ```
- GoReleaser signing configuration:
  ```yaml
  signs:
    - cmd: cosign
      artifacts: checksum
      output: true
      args:
        - "sign-blob"
        - "--yes"
        - "--output-signature=${signature}"
        - "--output-certificate=${certificate}"
        - "${artifact}"
  ```
- The `env` section should set `CGO_ENABLED=0` globally.
- The `builds` section should set `main: ./cmd/harvx`.
- Version injection: define `var (version, commit, date string)` in `cmd/harvx/main.go` and use ldflags to set them at build time.
- Reference: PRD Section 7.4 (Supply Chain Security), Section 10 Phase 6, GoReleaser docs (https://goreleaser.com/), Cosign docs (https://docs.sigstore.dev/)

## Files to Create/Modify

- `.goreleaser.yaml` - Complete GoReleaser configuration
- `cmd/harvx/main.go` - Add version/commit/date vars with ldflags (modify)
- `internal/cli/version.go` - Version command implementation using build vars
- `Makefile` - Add `release-snapshot`, `release-check` targets (modify)
- `docs/SECURITY.md` - Document verification instructions (cosign verify-blob commands)

## Testing Requirements

- `goreleaser check` passes with no warnings or errors
- `goreleaser build --snapshot --clean` produces all 5 platform binaries
- Built binary reports correct version info via `./harvx version`
- SBOM file is valid SPDX JSON (validate with `spdx-tools` or online validator)
- Checksum file contains SHA-256 hashes matching actual binary checksums
- Makefile targets `release-snapshot` and `release-check` work correctly
- Cross-compiled binaries have correct GOOS/GOARCH (verify with `file` command)