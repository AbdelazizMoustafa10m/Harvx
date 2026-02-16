# T-003: Central Data Types (FileDescriptor & Pipeline DTOs)

**Priority:** Must Have
**Effort:** Small (2-4hrs)
**Dependencies:** T-001
**Phase:** 1 - Foundation

---

## Description

Define the central `FileDescriptor` struct and related data types in `internal/pipeline/types.go` that serve as the shared DTO passed between all pipeline stages (discovery, filtering, relevance sorting, content loading, tokenization, rendering). This is the data backbone of the entire application.

## User Story

As a Harvx developer, I want well-defined shared data types so that all pipeline stages have a clear contract for passing file metadata and content between them.

## Acceptance Criteria

- [ ] `FileDescriptor` struct is defined in `internal/pipeline/types.go` matching PRD Section 6.5 with all specified fields
- [ ] `FileDescriptor` includes: `Path` (relative), `AbsPath` (absolute), `Size` (int64), `Tier` (int), `TokenCount` (int), `ContentHash` (uint64), `Content` (string), `IsCompressed` (bool), `Redactions` (int), `Language` (string)
- [ ] Additional helper fields are included: `IsSymlink` (bool), `Error` (error, for tracking per-file failures), `IsBinary` (bool)
- [ ] `ExitCode` constants are defined: `ExitSuccess = 0`, `ExitError = 1`, `ExitPartial = 2` per PRD Section 5.9
- [ ] `OutputFormat` type is defined as a string enum with constants: `FormatMarkdown`, `FormatXML`
- [ ] `LLMTarget` type is defined with constants: `TargetClaude`, `TargetChatGPT`, `TargetGeneric`
- [ ] `DiscoveryResult` struct is defined to hold the aggregate output of the discovery phase: a slice of `FileDescriptor` plus summary stats (total files found, total files skipped, skip reasons)
- [ ] All types have JSON struct tags for future serialization needs
- [ ] All types have comprehensive GoDoc comments explaining their purpose
- [ ] Unit tests validate struct initialization and any helper methods (e.g., `FileDescriptor.IsValid()`)
- [ ] `go vet ./internal/pipeline/...` passes

## Technical Notes

- This package has ZERO external dependencies -- only stdlib types.
- The `FileDescriptor` is intentionally a concrete struct, not an interface. Pipeline stages mutate it as it flows through (discovery sets path/size, relevance sets tier, content loading sets content/hash, etc.).
- The `Content` field stores the processed content (after redaction and optional compression). The original content is never stored -- we process file-by-file to keep memory usage bounded.
- `Tier` defaults to 2 (source code default) per PRD Section 5.3: "Unmatched files go to tier 2."
- `ContentHash` will use XXH3 (via `cespare/xxhash`) but the hash computation is done elsewhere -- this type just holds the result.
- Keep this file focused: only data types, no business logic. Helper methods should be limited to validation and formatting.
- Reference: PRD Sections 5.9 (exit codes), 6.5 (FileDescriptor), 5.7 (output formats)

## Files to Create/Modify

- `internal/pipeline/types.go` - Central data types
- `internal/pipeline/types_test.go` - Unit tests

## Testing Requirements

- Unit tests for struct initialization with zero values
- Unit tests for exit code constants (ensure they match expected values)
- Unit tests for format and target string constants
- Unit test that `FileDescriptor` with `Tier` zero-value defaults correctly
- Test that JSON marshaling/unmarshaling roundtrips correctly for `FileDescriptor`
