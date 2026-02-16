package pipeline

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestExitCodeConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code ExitCode
		want int
	}{
		{name: "ExitSuccess is 0", code: ExitSuccess, want: 0},
		{name: "ExitError is 1", code: ExitError, want: 1},
		{name: "ExitPartial is 2", code: ExitPartial, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if int(tt.code) != tt.want {
				t.Errorf("got %d, want %d", int(tt.code), tt.want)
			}
		})
	}
}

func TestOutputFormatConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format OutputFormat
		want   string
	}{
		{name: "FormatMarkdown", format: FormatMarkdown, want: "markdown"},
		{name: "FormatXML", format: FormatXML, want: "xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.format) != tt.want {
				t.Errorf("got %q, want %q", string(tt.format), tt.want)
			}
		})
	}
}

func TestLLMTargetConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target LLMTarget
		want   string
	}{
		{name: "TargetClaude", target: TargetClaude, want: "claude"},
		{name: "TargetChatGPT", target: TargetChatGPT, want: "chatgpt"},
		{name: "TargetGeneric", target: TargetGeneric, want: "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.target) != tt.want {
				t.Errorf("got %q, want %q", string(tt.target), tt.want)
			}
		})
	}
}

func TestDefaultTier(t *testing.T) {
	t.Parallel()

	if DefaultTier != 2 {
		t.Errorf("DefaultTier = %d, want 2", DefaultTier)
	}
}

func TestFileDescriptor_ZeroValue(t *testing.T) {
	t.Parallel()

	var fd FileDescriptor

	if fd.Path != "" {
		t.Errorf("zero-value Path = %q, want empty", fd.Path)
	}
	if fd.AbsPath != "" {
		t.Errorf("zero-value AbsPath = %q, want empty", fd.AbsPath)
	}
	if fd.Size != 0 {
		t.Errorf("zero-value Size = %d, want 0", fd.Size)
	}
	if fd.Tier != 0 {
		t.Errorf("zero-value Tier = %d, want 0", fd.Tier)
	}
	if fd.TokenCount != 0 {
		t.Errorf("zero-value TokenCount = %d, want 0", fd.TokenCount)
	}
	if fd.ContentHash != 0 {
		t.Errorf("zero-value ContentHash = %d, want 0", fd.ContentHash)
	}
	if fd.Content != "" {
		t.Errorf("zero-value Content = %q, want empty", fd.Content)
	}
	if fd.IsCompressed {
		t.Error("zero-value IsCompressed = true, want false")
	}
	if fd.Redactions != 0 {
		t.Errorf("zero-value Redactions = %d, want 0", fd.Redactions)
	}
	if fd.Language != "" {
		t.Errorf("zero-value Language = %q, want empty", fd.Language)
	}
	if fd.IsSymlink {
		t.Error("zero-value IsSymlink = true, want false")
	}
	if fd.IsBinary {
		t.Error("zero-value IsBinary = true, want false")
	}
	if fd.Error != nil {
		t.Errorf("zero-value Error = %v, want nil", fd.Error)
	}
}

func TestFileDescriptor_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fd   FileDescriptor
		want bool
	}{
		{
			name: "valid with path",
			fd:   FileDescriptor{Path: "src/main.go"},
			want: true,
		},
		{
			name: "valid with all fields",
			fd: FileDescriptor{
				Path:         "internal/config/config.go",
				AbsPath:      "/home/user/project/internal/config/config.go",
				Size:         4096,
				Tier:         1,
				TokenCount:   500,
				ContentHash:  12345678,
				Content:      "package config",
				IsCompressed: false,
				Redactions:   0,
				Language:     "go",
			},
			want: true,
		},
		{
			name: "invalid with empty path",
			fd:   FileDescriptor{},
			want: false,
		},
		{
			name: "invalid with only abs path",
			fd:   FileDescriptor{AbsPath: "/home/user/project/main.go"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.fd.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileDescriptor_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	fd := FileDescriptor{
		Path:         "internal/pipeline/types.go",
		AbsPath:      "/home/user/harvx/internal/pipeline/types.go",
		Size:         2048,
		Tier:         1,
		TokenCount:   350,
		ContentHash:  9876543210,
		Content:      "package pipeline\n\ntype FileDescriptor struct {}",
		IsCompressed: true,
		Redactions:   3,
		Language:     "go",
		IsSymlink:    false,
		IsBinary:     false,
		Error:        errors.New("test error"),
	}

	data, err := json.Marshal(fd)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got FileDescriptor
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Verify all JSON-serializable fields round-trip correctly.
	if got.Path != fd.Path {
		t.Errorf("Path = %q, want %q", got.Path, fd.Path)
	}
	if got.AbsPath != fd.AbsPath {
		t.Errorf("AbsPath = %q, want %q", got.AbsPath, fd.AbsPath)
	}
	if got.Size != fd.Size {
		t.Errorf("Size = %d, want %d", got.Size, fd.Size)
	}
	if got.Tier != fd.Tier {
		t.Errorf("Tier = %d, want %d", got.Tier, fd.Tier)
	}
	if got.TokenCount != fd.TokenCount {
		t.Errorf("TokenCount = %d, want %d", got.TokenCount, fd.TokenCount)
	}
	if got.ContentHash != fd.ContentHash {
		t.Errorf("ContentHash = %d, want %d", got.ContentHash, fd.ContentHash)
	}
	if got.Content != fd.Content {
		t.Errorf("Content = %q, want %q", got.Content, fd.Content)
	}
	if got.IsCompressed != fd.IsCompressed {
		t.Errorf("IsCompressed = %v, want %v", got.IsCompressed, fd.IsCompressed)
	}
	if got.Redactions != fd.Redactions {
		t.Errorf("Redactions = %d, want %d", got.Redactions, fd.Redactions)
	}
	if got.Language != fd.Language {
		t.Errorf("Language = %q, want %q", got.Language, fd.Language)
	}
	if got.IsSymlink != fd.IsSymlink {
		t.Errorf("IsSymlink = %v, want %v", got.IsSymlink, fd.IsSymlink)
	}
	if got.IsBinary != fd.IsBinary {
		t.Errorf("IsBinary = %v, want %v", got.IsBinary, fd.IsBinary)
	}

	// Error field must NOT be serialized (json:"-" tag).
	if got.Error != nil {
		t.Errorf("Error should be nil after JSON round-trip, got %v", got.Error)
	}
}

func TestFileDescriptor_ErrorFieldOmittedFromJSON(t *testing.T) {
	t.Parallel()

	fd := FileDescriptor{
		Path:  "broken.go",
		Error: errors.New("permission denied"),
	}

	data, err := json.Marshal(fd)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// The JSON output must not contain an "error" key.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal raw: %v", err)
	}

	if _, found := raw["error"]; found {
		t.Error("Error field should be omitted from JSON (json:\"-\" tag), but was present")
	}
}

func TestDiscoveryResult_ZeroValue(t *testing.T) {
	t.Parallel()

	var dr DiscoveryResult

	if dr.Files != nil {
		t.Errorf("zero-value Files = %v, want nil", dr.Files)
	}
	if dr.TotalFound != 0 {
		t.Errorf("zero-value TotalFound = %d, want 0", dr.TotalFound)
	}
	if dr.TotalSkipped != 0 {
		t.Errorf("zero-value TotalSkipped = %d, want 0", dr.TotalSkipped)
	}
	if dr.SkipReasons != nil {
		t.Errorf("zero-value SkipReasons = %v, want nil", dr.SkipReasons)
	}
}

func TestDiscoveryResult_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	dr := DiscoveryResult{
		Files: []FileDescriptor{
			{
				Path:    "main.go",
				AbsPath: "/project/main.go",
				Size:    512,
				Tier:    DefaultTier,
			},
			{
				Path:     "README.md",
				AbsPath:  "/project/README.md",
				Size:     1024,
				Tier:     4,
				Language: "markdown",
			},
		},
		TotalFound:   100,
		TotalSkipped: 98,
		SkipReasons: map[string]int{
			"gitignore":  50,
			"binary":     30,
			"size_limit": 18,
		},
	}

	data, err := json.Marshal(dr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got DiscoveryResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(got.Files) != len(dr.Files) {
		t.Fatalf("Files length = %d, want %d", len(got.Files), len(dr.Files))
	}
	if got.Files[0].Path != "main.go" {
		t.Errorf("Files[0].Path = %q, want %q", got.Files[0].Path, "main.go")
	}
	if got.Files[1].Path != "README.md" {
		t.Errorf("Files[1].Path = %q, want %q", got.Files[1].Path, "README.md")
	}
	if got.TotalFound != dr.TotalFound {
		t.Errorf("TotalFound = %d, want %d", got.TotalFound, dr.TotalFound)
	}
	if got.TotalSkipped != dr.TotalSkipped {
		t.Errorf("TotalSkipped = %d, want %d", got.TotalSkipped, dr.TotalSkipped)
	}
	if len(got.SkipReasons) != len(dr.SkipReasons) {
		t.Fatalf("SkipReasons length = %d, want %d", len(got.SkipReasons), len(dr.SkipReasons))
	}
	for reason, count := range dr.SkipReasons {
		if got.SkipReasons[reason] != count {
			t.Errorf("SkipReasons[%q] = %d, want %d", reason, got.SkipReasons[reason], count)
		}
	}
}

func TestFileDescriptor_TierDefaultIsZeroValue(t *testing.T) {
	t.Parallel()

	// The Go zero value for int is 0, which means newly created FileDescriptors
	// have Tier=0 (unset). Pipeline stages should explicitly set Tier to
	// DefaultTier (2) for unmatched files. This test documents that the zero
	// value is 0, not DefaultTier, to prevent confusion.
	var fd FileDescriptor
	if fd.Tier != 0 {
		t.Errorf("zero-value Tier = %d, want 0 (unset); DefaultTier (%d) must be set explicitly", fd.Tier, DefaultTier)
	}
}

func TestOutputFormat_StringType(t *testing.T) {
	t.Parallel()

	// Verify OutputFormat is usable as a string in switch statements and maps.
	formats := map[OutputFormat]bool{
		FormatMarkdown: true,
		FormatXML:      true,
	}

	if !formats[FormatMarkdown] {
		t.Error("FormatMarkdown not found in format map")
	}
	if !formats[FormatXML] {
		t.Error("FormatXML not found in format map")
	}
	if formats[OutputFormat("json")] {
		t.Error("unexpected format 'json' found in format map")
	}
}

func TestLLMTarget_StringType(t *testing.T) {
	t.Parallel()

	// Verify LLMTarget is usable as a string in switch statements.
	targets := map[LLMTarget]bool{
		TargetClaude:  true,
		TargetChatGPT: true,
		TargetGeneric: true,
	}

	if !targets[TargetClaude] {
		t.Error("TargetClaude not found in target map")
	}
	if !targets[TargetChatGPT] {
		t.Error("TargetChatGPT not found in target map")
	}
	if !targets[TargetGeneric] {
		t.Error("TargetGeneric not found in target map")
	}
	if targets[LLMTarget("gemini")] {
		t.Error("unexpected target 'gemini' found in target map")
	}
}
