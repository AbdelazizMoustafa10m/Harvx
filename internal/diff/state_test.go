package diff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateSnapshot(t *testing.T) {
	t.Parallel()

	before := time.Now().UTC().Truncate(time.Second)
	snap := NewStateSnapshot("default", "/home/user/repo", "main", "abc123def456")
	after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

	assert.Equal(t, SchemaVersion, snap.Version)
	assert.Equal(t, "default", snap.ProfileName)
	assert.Equal(t, "/home/user/repo", snap.RootDir)
	assert.Equal(t, "main", snap.GitBranch)
	assert.Equal(t, "abc123def456", snap.GitHeadSHA)
	assert.NotNil(t, snap.Files)  // initialized, not nil
	assert.Len(t, snap.Files, 0) // but empty

	// Verify GeneratedAt is a valid RFC3339 timestamp within bounds.
	ts, err := time.Parse(time.RFC3339, snap.GeneratedAt)
	require.NoError(t, err)
	assert.False(t, ts.Before(before), "GeneratedAt should not be before test start")
	assert.False(t, ts.After(after), "GeneratedAt should not be after test end")
}

func TestNewStateSnapshot_EmptyGitFields(t *testing.T) {
	t.Parallel()

	snap := NewStateSnapshot("minimal", "/tmp/project", "", "")

	assert.Equal(t, "", snap.GitBranch)
	assert.Equal(t, "", snap.GitHeadSHA)
}

func TestStateSnapshot_AddFile(t *testing.T) {
	t.Parallel()

	snap := NewStateSnapshot("test", "/repo", "main", "sha1")
	snap.AddFile("src/main.go", FileState{
		Size:         1024,
		ContentHash:  0xdeadbeef,
		ModifiedTime: "2026-01-15T10:30:00Z",
	})
	snap.AddFile("README.md", FileState{
		Size:         512,
		ContentHash:  0xcafebabe,
		ModifiedTime: "2026-01-14T09:00:00Z",
	})

	assert.Len(t, snap.Files, 2)
	assert.Equal(t, int64(1024), snap.Files["src/main.go"].Size)
	assert.Equal(t, uint64(0xdeadbeef), snap.Files["src/main.go"].ContentHash)
	assert.Equal(t, int64(512), snap.Files["README.md"].Size)
}

func TestFileState_MarshalJSON_HexHash(t *testing.T) {
	t.Parallel()

	fs := FileState{
		Size:         2048,
		ContentHash:  0xa1b2c3d4e5f60718,
		ModifiedTime: "2026-02-20T14:30:00Z",
	}

	data, err := json.Marshal(fs)
	require.NoError(t, err)

	// ContentHash should be a hex string, not a number.
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"content_hash":"a1b2c3d4e5f60718"`)

	// Verify it is a string (quoted), not a bare number.
	var rawMsg map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &rawMsg))
	hashRaw := string(rawMsg["content_hash"])
	assert.True(t, strings.HasPrefix(hashRaw, `"`), "content_hash should be a JSON string, got: %s", hashRaw)

	// Verify full structure via generic unmarshal.
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, "a1b2c3d4e5f60718", raw["content_hash"])
	assert.Equal(t, float64(2048), raw["size"])
	assert.Equal(t, "2026-02-20T14:30:00Z", raw["modified_time"])
}

func TestFileState_MarshalJSON_ZeroHash(t *testing.T) {
	t.Parallel()

	fs := FileState{
		Size:         0,
		ContentHash:  0,
		ModifiedTime: "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(fs)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"content_hash":"0"`)
}

func TestFileState_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	input := `{"size":4096,"content_hash":"deadbeef","modified_time":"2026-03-01T12:00:00Z"}`

	var fs FileState
	err := json.Unmarshal([]byte(input), &fs)
	require.NoError(t, err)

	assert.Equal(t, int64(4096), fs.Size)
	assert.Equal(t, uint64(0xdeadbeef), fs.ContentHash)
	assert.Equal(t, "2026-03-01T12:00:00Z", fs.ModifiedTime)
}

func TestFileState_UnmarshalJSON_InvalidHex(t *testing.T) {
	t.Parallel()

	input := `{"size":100,"content_hash":"not_hex!","modified_time":"2026-01-01T00:00:00Z"}`

	var fs FileState
	err := json.Unmarshal([]byte(input), &fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing content hash")
}

func TestFileState_UnmarshalJSON_MalformedJSON(t *testing.T) {
	t.Parallel()

	var fs FileState
	err := json.Unmarshal([]byte(`{broken`), &fs)
	assert.Error(t, err)
}

func TestFileState_RoundTrip(t *testing.T) {
	t.Parallel()

	original := FileState{
		Size:         999999,
		ContentHash:  0xffffffffffffffff,
		ModifiedTime: "2026-06-15T23:59:59Z",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded FileState
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original, decoded)
}

func TestStateSnapshot_MarshalJSON_Deterministic(t *testing.T) {
	t.Parallel()

	snap := &StateSnapshot{
		Version:     "1",
		ProfileName: "test",
		GeneratedAt: "2026-02-25T10:00:00Z",
		GitBranch:   "main",
		GitHeadSHA:  "abc123",
		RootDir:     "/repo",
		Files: map[string]FileState{
			"z/last.go": {
				Size:         100,
				ContentHash:  0xaaa,
				ModifiedTime: "2026-02-25T10:00:00Z",
			},
			"a/first.go": {
				Size:         200,
				ContentHash:  0xbbb,
				ModifiedTime: "2026-02-25T10:00:00Z",
			},
			"m/middle.go": {
				Size:         300,
				ContentHash:  0xccc,
				ModifiedTime: "2026-02-25T10:00:00Z",
			},
		},
	}

	data1, err := json.Marshal(snap)
	require.NoError(t, err)

	data2, err := json.Marshal(snap)
	require.NoError(t, err)

	// Byte-identical output.
	assert.Equal(t, data1, data2)

	// Verify keys are sorted: a/first.go < m/middle.go < z/last.go.
	jsonStr := string(data1)
	idxA := strings.Index(jsonStr, "a/first.go")
	idxM := strings.Index(jsonStr, "m/middle.go")
	idxZ := strings.Index(jsonStr, "z/last.go")

	assert.Greater(t, idxA, 0, "a/first.go should appear in JSON")
	assert.Greater(t, idxM, idxA, "m/middle.go should appear after a/first.go")
	assert.Greater(t, idxZ, idxM, "z/last.go should appear after m/middle.go")
}

func TestStateSnapshot_MarshalJSON_FieldOrder(t *testing.T) {
	t.Parallel()

	snap := &StateSnapshot{
		Version:     "1",
		ProfileName: "myprofile",
		GeneratedAt: "2026-02-25T10:00:00Z",
		GitBranch:   "develop",
		GitHeadSHA:  "def456",
		RootDir:     "/home/user/project",
		Files:       map[string]FileState{},
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)

	jsonStr := string(data)

	// Verify all expected JSON field names are present.
	assert.Contains(t, jsonStr, `"version"`)
	assert.Contains(t, jsonStr, `"profile_name"`)
	assert.Contains(t, jsonStr, `"generated_at"`)
	assert.Contains(t, jsonStr, `"git_branch"`)
	assert.Contains(t, jsonStr, `"git_head_sha"`)
	assert.Contains(t, jsonStr, `"root_dir"`)
	assert.Contains(t, jsonStr, `"files"`)
}

func TestStateSnapshot_RoundTrip_WithFiles(t *testing.T) {
	t.Parallel()

	original := &StateSnapshot{
		Version:     "1",
		ProfileName: "go-cli",
		GeneratedAt: "2026-02-25T10:30:00Z",
		GitBranch:   "feature/state",
		GitHeadSHA:  "1234567890abcdef",
		RootDir:     "/home/dev/harvx",
		Files: map[string]FileState{
			"cmd/main.go": {
				Size:         512,
				ContentHash:  0x123456789abcdef0,
				ModifiedTime: "2026-02-24T08:00:00Z",
			},
			"internal/diff/state.go": {
				Size:         2048,
				ContentHash:  0xfedcba9876543210,
				ModifiedTime: "2026-02-25T09:00:00Z",
			},
			"README.md": {
				Size:         256,
				ContentHash:  0x1111111111111111,
				ModifiedTime: "2026-02-20T12:00:00Z",
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	parsed, err := ParseStateSnapshot(data)
	require.NoError(t, err)

	assert.Equal(t, original.Version, parsed.Version)
	assert.Equal(t, original.ProfileName, parsed.ProfileName)
	assert.Equal(t, original.GeneratedAt, parsed.GeneratedAt)
	assert.Equal(t, original.GitBranch, parsed.GitBranch)
	assert.Equal(t, original.GitHeadSHA, parsed.GitHeadSHA)
	assert.Equal(t, original.RootDir, parsed.RootDir)
	assert.Len(t, parsed.Files, 3)

	for path, origState := range original.Files {
		parsedState, ok := parsed.Files[path]
		require.True(t, ok, "missing file: %s", path)
		assert.Equal(t, origState, parsedState, "mismatch for file: %s", path)
	}
}

func TestStateSnapshot_RoundTrip_Empty(t *testing.T) {
	t.Parallel()

	original := &StateSnapshot{
		Version:     "1",
		ProfileName: "empty",
		GeneratedAt: "2026-01-01T00:00:00Z",
		GitBranch:   "",
		GitHeadSHA:  "",
		RootDir:     "/tmp/empty",
		Files:       map[string]FileState{},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	parsed, err := ParseStateSnapshot(data)
	require.NoError(t, err)

	assert.Equal(t, original.Version, parsed.Version)
	assert.Equal(t, original.ProfileName, parsed.ProfileName)
	assert.NotNil(t, parsed.Files)
	assert.Len(t, parsed.Files, 0)
}

func TestParseStateSnapshot_UnsupportedVersion(t *testing.T) {
	t.Parallel()

	input := `{"version":"99","profile_name":"test","generated_at":"2026-01-01T00:00:00Z","git_branch":"","git_head_sha":"","root_dir":"/tmp","files":{}}`

	_, err := ParseStateSnapshot([]byte(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported state snapshot version "99"`)
	assert.Contains(t, err.Error(), `expected "1"`)
	assert.True(t, errors.Is(err, ErrUnsupportedVersion))
}

func TestParseStateSnapshot_EmptyVersion(t *testing.T) {
	t.Parallel()

	input := `{"version":"","profile_name":"test","generated_at":"2026-01-01T00:00:00Z","git_branch":"","git_head_sha":"","root_dir":"/tmp","files":{}}`

	_, err := ParseStateSnapshot([]byte(input))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedVersion))
}

func TestParseStateSnapshot_MalformedJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseStateSnapshot([]byte(`{not valid json`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing state snapshot")
}

func TestParseStateSnapshot_NilFilesInitialized(t *testing.T) {
	t.Parallel()

	// JSON with files explicitly null.
	input := `{"version":"1","profile_name":"test","generated_at":"2026-01-01T00:00:00Z","git_branch":"","git_head_sha":"","root_dir":"/tmp","files":null}`

	snap, err := ParseStateSnapshot([]byte(input))
	require.NoError(t, err)
	assert.NotNil(t, snap.Files)
	assert.Len(t, snap.Files, 0)
}

func TestParseStateSnapshot_MissingFilesField(t *testing.T) {
	t.Parallel()

	// JSON without files field at all.
	input := `{"version":"1","profile_name":"test","generated_at":"2026-01-01T00:00:00Z","git_branch":"","git_head_sha":"","root_dir":"/tmp"}`

	snap, err := ParseStateSnapshot([]byte(input))
	require.NoError(t, err)
	assert.NotNil(t, snap.Files)
	assert.Len(t, snap.Files, 0)
}

func TestParseStateSnapshot_MissingOptionalFields(t *testing.T) {
	t.Parallel()

	// Minimal valid snapshot: only version and files.
	input := `{"version":"1","files":{}}`

	snap, err := ParseStateSnapshot([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, "", snap.ProfileName)
	assert.Equal(t, "", snap.GeneratedAt)
	assert.Equal(t, "", snap.GitBranch)
	assert.Equal(t, "", snap.GitHeadSHA)
	assert.Equal(t, "", snap.RootDir)
}

func TestStateSnapshot_MarshalJSON_EmptyFiles(t *testing.T) {
	t.Parallel()

	snap := &StateSnapshot{
		Version:     "1",
		ProfileName: "empty",
		GeneratedAt: "2026-01-01T00:00:00Z",
		RootDir:     "/tmp",
		Files:       map[string]FileState{},
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"files":{}`)
}

func TestStateSnapshot_GoldenJSON(t *testing.T) {
	t.Parallel()

	snap := &StateSnapshot{
		Version:     "1",
		ProfileName: "default",
		GeneratedAt: "2026-02-25T10:00:00Z",
		GitBranch:   "main",
		GitHeadSHA:  "abc123",
		RootDir:     "/home/user/project",
		Files: map[string]FileState{
			"README.md": {
				Size:         256,
				ContentHash:  0xff,
				ModifiedTime: "2026-02-20T12:00:00Z",
			},
			"src/main.go": {
				Size:         1024,
				ContentHash:  0xdeadbeef,
				ModifiedTime: "2026-02-25T09:00:00Z",
			},
		},
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)

	// Pretty-print for readability in golden comparison.
	var pretty bytes.Buffer
	require.NoError(t, json.Indent(&pretty, data, "", "  "))

	expected := `{
  "version": "1",
  "profile_name": "default",
  "generated_at": "2026-02-25T10:00:00Z",
  "git_branch": "main",
  "git_head_sha": "abc123",
  "root_dir": "/home/user/project",
  "files": {
    "README.md": {
      "size": 256,
      "content_hash": "ff",
      "modified_time": "2026-02-20T12:00:00Z"
    },
    "src/main.go": {
      "size": 1024,
      "content_hash": "deadbeef",
      "modified_time": "2026-02-25T09:00:00Z"
    }
  }
}`

	assert.Equal(t, expected, pretty.String())
}

func TestSchemaVersion_Constant(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "1", SchemaVersion)
}

func TestErrUnsupportedVersion_Sentinel(t *testing.T) {
	t.Parallel()

	// Verify the sentinel error can be used with errors.Is through wrapping.
	wrapped := fmt.Errorf("some context: %w", ErrUnsupportedVersion)
	assert.True(t, errors.Is(wrapped, ErrUnsupportedVersion))
}

func TestFileState_LargeHash(t *testing.T) {
	t.Parallel()

	// Test max uint64 value round-trips correctly.
	fs := FileState{
		Size:         1,
		ContentHash:  0xffffffffffffffff,
		ModifiedTime: "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(fs)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"content_hash":"ffffffffffffffff"`)

	var decoded FileState
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, uint64(0xffffffffffffffff), decoded.ContentHash)
}

// testdataDir returns the absolute path to the testdata/state directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	// Walk up from internal/diff to the project root.
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "state"))
	require.NoError(t, err)
	return dir
}

func TestGoldenFixture_ValidSnapshot(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(testdataDir(t), "valid_snapshot.json"))
	require.NoError(t, err)

	snap, err := ParseStateSnapshot(data)
	require.NoError(t, err)

	assert.Equal(t, "1", snap.Version)
	assert.Equal(t, "default", snap.ProfileName)
	assert.Equal(t, "2026-02-25T10:00:00Z", snap.GeneratedAt)
	assert.Equal(t, "main", snap.GitBranch)
	assert.Equal(t, "abc123", snap.GitHeadSHA)
	assert.Equal(t, "/home/user/project", snap.RootDir)
	assert.Len(t, snap.Files, 2)

	readme, ok := snap.Files["README.md"]
	require.True(t, ok)
	assert.Equal(t, int64(256), readme.Size)
	assert.Equal(t, uint64(0xff), readme.ContentHash)
	assert.Equal(t, "2026-02-20T12:00:00Z", readme.ModifiedTime)

	mainGo, ok := snap.Files["src/main.go"]
	require.True(t, ok)
	assert.Equal(t, int64(1024), mainGo.Size)
	assert.Equal(t, uint64(0xdeadbeef), mainGo.ContentHash)
	assert.Equal(t, "2026-02-25T09:00:00Z", mainGo.ModifiedTime)

	// Re-marshal and verify it matches the golden fixture byte-for-byte.
	remarshaled, err := json.Marshal(snap)
	require.NoError(t, err)

	var prettyRemarshaled bytes.Buffer
	require.NoError(t, json.Indent(&prettyRemarshaled, remarshaled, "", "  "))

	// Normalize the fixture (trim trailing whitespace/newlines).
	expected := strings.TrimSpace(string(data))
	actual := strings.TrimSpace(prettyRemarshaled.String())
	assert.Equal(t, expected, actual)
}

func TestGoldenFixture_EmptySnapshot(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(testdataDir(t), "empty_snapshot.json"))
	require.NoError(t, err)

	snap, err := ParseStateSnapshot(data)
	require.NoError(t, err)

	assert.Equal(t, "1", snap.Version)
	assert.Equal(t, "empty", snap.ProfileName)
	assert.Equal(t, "2026-01-01T00:00:00Z", snap.GeneratedAt)
	assert.Equal(t, "", snap.GitBranch)
	assert.Equal(t, "", snap.GitHeadSHA)
	assert.Equal(t, "/tmp/empty", snap.RootDir)
	assert.NotNil(t, snap.Files)
	assert.Len(t, snap.Files, 0)

	// Re-marshal and verify round-trip matches.
	remarshaled, err := json.Marshal(snap)
	require.NoError(t, err)

	var prettyRemarshaled bytes.Buffer
	require.NoError(t, json.Indent(&prettyRemarshaled, remarshaled, "", "  "))

	expected := strings.TrimSpace(string(data))
	actual := strings.TrimSpace(prettyRemarshaled.String())
	assert.Equal(t, expected, actual)
}
