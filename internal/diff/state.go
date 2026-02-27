// Package diff provides state snapshot types and JSON serialization for
// persisting project state between Harvx runs. The state snapshot captures
// file paths, sizes, content hashes, generation timestamps, and git metadata,
// enabling differential output and change detection across runs.
package diff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// SchemaVersion is the current state snapshot schema version. Incrementing this
// value signals a breaking change in the serialization format and triggers a
// migration path in future readers.
const SchemaVersion = "1"

// ErrUnsupportedVersion is returned when parsing a state snapshot with an
// unrecognized schema version. Callers can check this with errors.Is.
var ErrUnsupportedVersion = errors.New("unsupported state snapshot version")

// FileState captures the filesystem metadata and content hash for a single file.
// Together with the relative path (stored as the map key in StateSnapshot.Files),
// this provides enough information to detect changes between runs.
type FileState struct {
	// Size is the file size in bytes as reported by the filesystem.
	Size int64 `json:"size"`

	// ContentHash is the XXH3 64-bit hash of the file content. It is serialized
	// as a hexadecimal string in JSON for human readability and to avoid
	// JavaScript integer precision issues.
	ContentHash uint64 `json:"content_hash"`

	// ModifiedTime is the file modification time in RFC3339 format. Used as an
	// optimization hint: if mod time and size both match, re-hashing can be skipped.
	ModifiedTime string `json:"modified_time"`
}

// fileStateJSON is an alias used for custom JSON marshaling of FileState. The
// ContentHash field is represented as a hex string instead of a numeric value.
type fileStateJSON struct {
	Size         int64  `json:"size"`
	ContentHash  string `json:"content_hash"`
	ModifiedTime string `json:"modified_time"`
}

// MarshalJSON implements json.Marshaler for FileState. The ContentHash field
// is encoded as a lowercase hexadecimal string.
func (fs FileState) MarshalJSON() ([]byte, error) {
	return json.Marshal(fileStateJSON{
		Size:         fs.Size,
		ContentHash:  strconv.FormatUint(fs.ContentHash, 16),
		ModifiedTime: fs.ModifiedTime,
	})
}

// UnmarshalJSON implements json.Unmarshaler for FileState. The ContentHash
// field is decoded from a hexadecimal string back to uint64.
func (fs *FileState) UnmarshalJSON(data []byte) error {
	var raw fileStateJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshaling file state: %w", err)
	}

	hash, err := strconv.ParseUint(raw.ContentHash, 16, 64)
	if err != nil {
		return fmt.Errorf("parsing content hash %q: %w", raw.ContentHash, err)
	}

	fs.Size = raw.Size
	fs.ContentHash = hash
	fs.ModifiedTime = raw.ModifiedTime
	return nil
}

// StateSnapshot captures the complete project state at a point in time. It
// records metadata about the generation run (profile, timestamp, git info) and
// a map of every discovered file's state keyed by relative path.
type StateSnapshot struct {
	// Version is the schema version string, always set to SchemaVersion.
	Version string `json:"version"`

	// ProfileName is the name of the profile used for this generation run.
	ProfileName string `json:"profile_name"`

	// GeneratedAt is the RFC3339 timestamp of when the snapshot was created.
	GeneratedAt string `json:"generated_at"`

	// GitBranch is the current git branch name. May be empty when not in a
	// git repository.
	GitBranch string `json:"git_branch"`

	// GitHeadSHA is the HEAD commit SHA. May be empty when not in a git
	// repository.
	GitHeadSHA string `json:"git_head_sha"`

	// RootDir is the absolute path to the repository root directory.
	RootDir string `json:"root_dir"`

	// Files maps relative file paths to their FileState. Keys are sorted
	// alphabetically during JSON serialization for deterministic output.
	Files map[string]FileState `json:"files"`
}

// NewStateSnapshot creates a new StateSnapshot with the given metadata. The
// Version is set to SchemaVersion, GeneratedAt is set to the current UTC time,
// and Files is initialized to an empty map.
func NewStateSnapshot(profileName, rootDir, gitBranch, gitHeadSHA string) *StateSnapshot {
	return &StateSnapshot{
		Version:     SchemaVersion,
		ProfileName: profileName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		GitBranch:   gitBranch,
		GitHeadSHA:  gitHeadSHA,
		RootDir:     rootDir,
		Files:       map[string]FileState{},
	}
}

// AddFile adds a file entry to the snapshot's Files map.
func (s *StateSnapshot) AddFile(relPath string, state FileState) {
	s.Files[relPath] = state
}

// MarshalJSON implements json.Marshaler for StateSnapshot. It produces
// deterministic JSON by sorting the Files map keys alphabetically before
// encoding. This ensures identical snapshots produce byte-identical JSON.
func (s StateSnapshot) MarshalJSON() ([]byte, error) {
	// Sort file keys for deterministic output.
	keys := make([]string, 0, len(s.Files))
	for k := range s.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build ordered files JSON manually to preserve key ordering.
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, fmt.Errorf("marshaling file key %s: %w", k, err)
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		valJSON, err := json.Marshal(s.Files[k])
		if err != nil {
			return nil, fmt.Errorf("marshaling file state for %s: %w", k, err)
		}
		buf.Write(valJSON)
	}
	buf.WriteByte('}')

	// Use an alias struct to avoid infinite recursion when marshaling the
	// outer fields. Files is embedded as pre-built json.RawMessage.
	type alias struct {
		Version     string          `json:"version"`
		ProfileName string          `json:"profile_name"`
		GeneratedAt string          `json:"generated_at"`
		GitBranch   string          `json:"git_branch"`
		GitHeadSHA  string          `json:"git_head_sha"`
		RootDir     string          `json:"root_dir"`
		Files       json.RawMessage `json:"files"`
	}

	return json.Marshal(alias{
		Version:     s.Version,
		ProfileName: s.ProfileName,
		GeneratedAt: s.GeneratedAt,
		GitBranch:   s.GitBranch,
		GitHeadSHA:  s.GitHeadSHA,
		RootDir:     s.RootDir,
		Files:       buf.Bytes(),
	})
}

// ParseStateSnapshot deserializes JSON bytes into a StateSnapshot. It validates
// that the schema version matches SchemaVersion and initializes the Files map
// if it is nil after unmarshaling.
func ParseStateSnapshot(data []byte) (*StateSnapshot, error) {
	var snap StateSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parsing state snapshot: %w", err)
	}

	if snap.Version != SchemaVersion {
		return nil, fmt.Errorf("unsupported state snapshot version %q, expected %q: %w",
			snap.Version, SchemaVersion, ErrUnsupportedVersion)
	}

	if snap.Files == nil {
		snap.Files = map[string]FileState{}
	}

	return &snap, nil
}
