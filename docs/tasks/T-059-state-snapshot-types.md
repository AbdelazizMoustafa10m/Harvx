# T-059: State Snapshot Types and JSON Serialization

**Priority:** Should Have
**Effort:** Small (2-4hrs)
**Dependencies:** T-001 (Project Scaffolding)
**Phase:** 4 - State & Diff

---

## Description

Define the Go structs that represent a project state snapshot and implement JSON serialization/deserialization for persisting state between Harvx runs. This is the foundational data model for all state caching and differential output features. The state snapshot captures file paths, sizes, content hashes, generation timestamps, and git metadata (branch name, HEAD SHA).

## User Story

As a developer running Harvx iteratively, I want the tool to remember the state of my project from the last run so that it can later determine what changed.

## Acceptance Criteria

- [ ] `internal/diff/state.go` defines the `StateSnapshot` struct with fields: `Version` (schema version string), `ProfileName`, `GeneratedAt` (RFC3339 timestamp), `GitBranch`, `GitHeadSHA`, `RootDir`, and `Files` (map of relative path to `FileState`)
- [ ] `FileState` struct includes: `Size` (int64), `ContentHash` (uint64, XXH3), `ModifiedTime` (RFC3339 timestamp)
- [ ] All structs have `json` struct tags matching snake_case field names (e.g., `json:"content_hash"`)
- [ ] `NewStateSnapshot(profileName, rootDir, gitBranch, gitHeadSHA string) *StateSnapshot` constructor sets `Version` to `"1"` and `GeneratedAt` to current time
- [ ] `StateSnapshot.AddFile(relPath string, state FileState)` adds a file entry to the map
- [ ] `StateSnapshot.MarshalJSON()` produces valid, deterministic JSON (files map keys are sorted)
- [ ] `ParseStateSnapshot(data []byte) (*StateSnapshot, error)` deserializes JSON back into the struct
- [ ] Schema version `"1"` is a constant; parsing an unknown version returns a clear error
- [ ] Empty snapshot (no files) serializes and deserializes correctly
- [ ] Unit tests achieve 95%+ coverage

## Technical Notes

- Use Go's `encoding/json` with standard `json.Marshal` / `json.Unmarshal` -- no need for v2 experimental API yet
- For deterministic JSON output, implement a custom `MarshalJSON` on `StateSnapshot` that sorts the `Files` map keys before encoding. This ensures identical snapshots produce byte-identical JSON, which is important for comparing state files and for reproducibility
- The `ContentHash` field is `uint64` in Go but serialized as a hex string in JSON (e.g., `"a1b2c3d4e5f6"`) to avoid JavaScript integer precision issues and for human readability. Use `strconv.FormatUint(hash, 16)` for encoding and `strconv.ParseUint(s, 16, 64)` for decoding via custom JSON marshal/unmarshal methods on `FileState`
- Schema version allows future migration -- if we change the state format, we increment the version and add a migration path
- The `ModifiedTime` in `FileState` uses `os.FileInfo.ModTime()` format (RFC3339). This is an optimization hint -- if mod time and size match, we can skip re-hashing in future runs
- Git metadata fields (`GitBranch`, `GitHeadSHA`) can be empty strings when not in a git repo
- PRD Section 5.8 specifies the state structure; PRD Section 6.2 places this in `internal/diff/state.go`

## Files to Create/Modify

- `internal/diff/state.go` - StateSnapshot and FileState struct definitions, constructors, JSON serialization
- `internal/diff/state_test.go` - Unit tests for serialization, deserialization, edge cases
- `testdata/state/valid_snapshot.json` - Golden test fixture for a valid state snapshot
- `testdata/state/empty_snapshot.json` - Golden test fixture for an empty state snapshot

## Testing Requirements

- Unit test: Create a snapshot with 3 files, marshal to JSON, verify all fields present and correctly formatted
- Unit test: Unmarshal the golden fixture `valid_snapshot.json` and verify all fields round-trip correctly
- Unit test: Empty snapshot (zero files) round-trips correctly
- Unit test: ContentHash serializes as hex string, not integer
- Unit test: Files map is sorted alphabetically by key in JSON output
- Unit test: Unknown schema version returns descriptive error
- Unit test: Malformed JSON returns error (not panic)
- Unit test: Missing required fields return appropriate defaults or errors
- Golden test: Marshal a known snapshot and compare byte-for-byte against expected JSON fixture

## References

- [encoding/json package](https://pkg.go.dev/encoding/json)
- [zeebo/xxh3 package](https://pkg.go.dev/github.com/zeebo/xxh3) -- defines the uint64 hash type used in ContentHash
- PRD Section 5.8 (State Caching & Differential Output)
- PRD Section 6.5 (Central Data Types -- FileDescriptor.ContentHash)
