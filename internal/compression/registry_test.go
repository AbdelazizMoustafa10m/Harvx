package compression

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock compressor
// ---------------------------------------------------------------------------

type mockCompressor struct {
	lang      string
	nodeTypes []string
}

func (m *mockCompressor) Compress(_ context.Context, _ []byte) (*CompressedOutput, error) {
	return &CompressedOutput{Language: m.lang}, nil
}

func (m *mockCompressor) Language() string {
	return m.lang
}

func (m *mockCompressor) SupportedNodeTypes() []string {
	return m.nodeTypes
}

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestNewCompressorRegistry(t *testing.T) {
	detector := NewLanguageDetector()
	reg := NewCompressorRegistry(detector)

	require.NotNil(t, reg, "registry should not be nil")
	assert.Empty(t, reg.Languages(), "new registry should have no registered languages")
}

func TestRegister_And_Get(t *testing.T) {
	detector := NewLanguageDetector()
	reg := NewCompressorRegistry(detector)

	goCompressor := &mockCompressor{lang: "go", nodeTypes: []string{"function_declaration"}}
	reg.Register(goCompressor)

	t.Run("registered extension returns compressor", func(t *testing.T) {
		got := reg.Get("main.go")
		require.NotNil(t, got, "Get should return compressor for .go files")
		assert.Equal(t, "go", got.Language())
	})

	t.Run("unknown extension returns nil", func(t *testing.T) {
		got := reg.Get("unknown.xyz")
		assert.Nil(t, got, "Get should return nil for unrecognized extension")
	})
}

func TestRegister_Replaces(t *testing.T) {
	detector := NewLanguageDetector()
	reg := NewCompressorRegistry(detector)

	first := &mockCompressor{lang: "go", nodeTypes: []string{"function_declaration"}}
	second := &mockCompressor{lang: "go", nodeTypes: []string{"function_declaration", "type_declaration"}}

	reg.Register(first)
	reg.Register(second)

	got := reg.GetByLanguage("go")
	require.NotNil(t, got)
	// The second compressor has two node types; the first has one.
	assert.Equal(t, second.SupportedNodeTypes(), got.SupportedNodeTypes(),
		"second registration should replace the first")
}

func TestGet_UnsupportedLanguage(t *testing.T) {
	detector := NewLanguageDetector()
	reg := NewCompressorRegistry(detector)

	// Register only "go" -- detector recognises .py as "python" but no compressor is registered.
	reg.Register(&mockCompressor{lang: "go"})

	got := reg.Get("script.py")
	assert.Nil(t, got, "Get should return nil when language is detected but no compressor is registered")
}

func TestGetByLanguage(t *testing.T) {
	detector := NewLanguageDetector()
	reg := NewCompressorRegistry(detector)

	reg.Register(&mockCompressor{lang: "typescript", nodeTypes: []string{"function_declaration"}})

	tests := []struct {
		name  string
		lang  string
		found bool
	}{
		{name: "registered language", lang: "typescript", found: true},
		{name: "unregistered language", lang: "ruby", found: false},
		{name: "empty string", lang: "", found: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reg.GetByLanguage(tt.lang)
			if tt.found {
				require.NotNil(t, got)
				assert.Equal(t, tt.lang, got.Language())
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	detector := NewLanguageDetector()
	reg := NewCompressorRegistry(detector)

	reg.Register(&mockCompressor{lang: "go"})
	reg.Register(&mockCompressor{lang: "python"})

	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{name: "go file", filePath: "main.go", want: true},
		{name: "python file", filePath: "app.py", want: true},
		{name: "typescript not registered", filePath: "index.ts", want: false},
		{name: "unknown extension", filePath: "data.bin", want: false},
		{name: "no extension", filePath: "Makefile", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reg.IsSupported(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLanguages(t *testing.T) {
	detector := NewLanguageDetector()
	reg := NewCompressorRegistry(detector)

	reg.Register(&mockCompressor{lang: "go"})
	reg.Register(&mockCompressor{lang: "python"})
	reg.Register(&mockCompressor{lang: "typescript"})

	langs := reg.Languages()
	sort.Strings(langs) // Languages() order is non-deterministic (map iteration).

	assert.Equal(t, []string{"go", "python", "typescript"}, langs)
}

func TestRegistry_MultipleCompressors(t *testing.T) {
	detector := NewLanguageDetector()
	reg := NewCompressorRegistry(detector)

	goComp := &mockCompressor{lang: "go", nodeTypes: []string{"function_declaration"}}
	tsComp := &mockCompressor{lang: "typescript", nodeTypes: []string{"function_declaration", "class_declaration"}}
	pyComp := &mockCompressor{lang: "python", nodeTypes: []string{"function_definition", "class_definition"}}

	reg.Register(goComp)
	reg.Register(tsComp)
	reg.Register(pyComp)

	tests := []struct {
		name     string
		filePath string
		wantLang string
	}{
		{name: "go file", filePath: "cmd/main.go", wantLang: "go"},
		{name: "ts file", filePath: "src/index.ts", wantLang: "typescript"},
		{name: "tsx file", filePath: "src/App.tsx", wantLang: "typescript"},
		{name: "py file", filePath: "scripts/build.py", wantLang: "python"},
		{name: "pyi stub", filePath: "types/stub.pyi", wantLang: "python"},
		{name: "js file (not registered)", filePath: "lib/util.js", wantLang: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reg.Get(tt.filePath)
			if tt.wantLang == "" {
				assert.Nil(t, got, "expected nil compressor for %s", tt.filePath)
			} else {
				require.NotNil(t, got, "expected compressor for %s", tt.filePath)
				assert.Equal(t, tt.wantLang, got.Language())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Types tests -- SignatureKind
// ---------------------------------------------------------------------------

func TestSignatureKind_String(t *testing.T) {
	tests := []struct {
		kind SignatureKind
		want string
	}{
		{KindFunction, "function"},
		{KindClass, "class"},
		{KindStruct, "struct"},
		{KindInterface, "interface"},
		{KindType, "type"},
		{KindImport, "import"},
		{KindExport, "export"},
		{KindConstant, "constant"},
		{KindDocComment, "doc_comment"},
		{SignatureKind(99), "unknown"},
		{SignatureKind(-1), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.kind.String())
		})
	}
}

// ---------------------------------------------------------------------------
// Types tests -- CompressedOutput.Render
// ---------------------------------------------------------------------------

func TestCompressedOutput_Render_Empty(t *testing.T) {
	co := &CompressedOutput{
		Language:     "go",
		OriginalSize: 100,
	}

	result := co.Render()
	assert.Equal(t, "", result, "Render with no signatures should return empty string")
}

func TestCompressedOutput_Render(t *testing.T) {
	co := &CompressedOutput{
		Signatures: []Signature{
			{Kind: KindFunction, Name: "Foo", Source: "func Foo() {}", StartLine: 1, EndLine: 1},
			{Kind: KindFunction, Name: "Bar", Source: "func Bar() {}", StartLine: 3, EndLine: 3},
			{Kind: KindStruct, Name: "Baz", Source: "type Baz struct{}", StartLine: 5, EndLine: 5},
		},
		Language:     "go",
		OriginalSize: 200,
	}

	result := co.Render()
	expected := "func Foo() {}\n\nfunc Bar() {}\n\ntype Baz struct{}"

	assert.Equal(t, expected, result)
	assert.Equal(t, len(expected), co.OutputSize,
		"Render should update OutputSize to the length of the rendered string")
}

func TestCompressedOutput_Render_Single(t *testing.T) {
	co := &CompressedOutput{
		Signatures: []Signature{
			{Kind: KindFunction, Name: "Main", Source: "func main() {}", StartLine: 1, EndLine: 1},
		},
		Language:     "go",
		OriginalSize: 50,
	}

	result := co.Render()
	assert.Equal(t, "func main() {}", result,
		"single signature should render without separators")
	assert.Equal(t, len("func main() {}"), co.OutputSize)
}

// ---------------------------------------------------------------------------
// Types tests -- CompressedOutput.CompressionRatio
// ---------------------------------------------------------------------------

func TestCompressedOutput_CompressionRatio(t *testing.T) {
	tests := []struct {
		name         string
		originalSize int
		outputSize   int
		wantRatio    float64
	}{
		{
			name:         "50 percent compression",
			originalSize: 100,
			outputSize:   50,
			wantRatio:    0.5,
		},
		{
			name:         "no compression",
			originalSize: 100,
			outputSize:   100,
			wantRatio:    1.0,
		},
		{
			name:         "full compression",
			originalSize: 100,
			outputSize:   0,
			wantRatio:    0.0,
		},
		{
			name:         "zero original size",
			originalSize: 0,
			outputSize:   0,
			wantRatio:    0.0,
		},
		{
			name:         "zero original nonzero output",
			originalSize: 0,
			outputSize:   50,
			wantRatio:    0.0,
		},
		{
			name:         "expansion ratio above 1",
			originalSize: 100,
			outputSize:   150,
			wantRatio:    1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			co := &CompressedOutput{
				OriginalSize: tt.originalSize,
				OutputSize:   tt.outputSize,
			}
			assert.InDelta(t, tt.wantRatio, co.CompressionRatio(), 1e-9)
		})
	}
}

// ---------------------------------------------------------------------------
// Types tests -- Signature source order in Render
// ---------------------------------------------------------------------------

func TestSignature_SourceOrder(t *testing.T) {
	co := &CompressedOutput{
		Signatures: []Signature{
			{Kind: KindImport, Source: "import \"fmt\"", StartLine: 1, EndLine: 1},
			{Kind: KindConstant, Source: "const Version = \"1.0\"", StartLine: 3, EndLine: 3},
			{Kind: KindStruct, Source: "type Config struct{}", StartLine: 5, EndLine: 5},
			{Kind: KindFunction, Source: "func New() *Config { return nil }", StartLine: 7, EndLine: 7},
		},
		Language:     "go",
		OriginalSize: 300,
	}

	result := co.Render()
	expected := "import \"fmt\"\n\nconst Version = \"1.0\"\n\ntype Config struct{}\n\nfunc New() *Config { return nil }"

	assert.Equal(t, expected, result,
		"Render must preserve the slice order of Signatures")
}
