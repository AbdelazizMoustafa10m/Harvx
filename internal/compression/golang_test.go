package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GoCompressor metadata tests
// ---------------------------------------------------------------------------

func TestGoCompressor_Language(t *testing.T) {
	c := NewGoCompressor()
	assert.Equal(t, "go", c.Language())
}

func TestGoCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewGoCompressor()
	types := c.SupportedNodeTypes()
	assert.Contains(t, types, "package_clause")
	assert.Contains(t, types, "import_declaration")
	assert.Contains(t, types, "function_declaration")
	assert.Contains(t, types, "method_declaration")
	assert.Contains(t, types, "type_declaration")
	assert.Contains(t, types, "const_declaration")
	assert.Contains(t, types, "var_declaration")
}

func TestGoCompressor_EmptyInput(t *testing.T) {
	c := NewGoCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "go", output.Language)
}

func TestGoCompressor_ContextCancellation(t *testing.T) {
	c := NewGoCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Generate a large source to ensure cancellation is checked.
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("// line\n")
	}

	_, err := c.Compress(ctx, []byte(b.String()))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Package clause
// ---------------------------------------------------------------------------

func TestGoCompressor_PackageClause(t *testing.T) {
	c := NewGoCompressor()
	source := `package main`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindImport, sig.Kind)
	assert.Equal(t, "main", sig.Name)
	assert.Equal(t, "package main", sig.Source)
}

// ---------------------------------------------------------------------------
// Import declarations
// ---------------------------------------------------------------------------

func TestGoCompressor_SingleImport(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

import "fmt"
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 2) // package + import
	sig := output.Signatures[1]
	assert.Equal(t, KindImport, sig.Kind)
	assert.Contains(t, sig.Source, `import "fmt"`)
}

func TestGoCompressor_GroupedImport(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

import (
	"fmt"
	"strings"
)
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 2) // package + import
	sig := output.Signatures[1]
	assert.Equal(t, KindImport, sig.Kind)
	assert.Contains(t, sig.Source, `"fmt"`)
	assert.Contains(t, sig.Source, `"strings"`)
	assert.Contains(t, sig.Source, "import (")
}

// ---------------------------------------------------------------------------
// Function declarations
// ---------------------------------------------------------------------------

func TestGoCompressor_FunctionDeclaration(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

func Foo(x int, y string) (int, error) {
	return 0, nil
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig, "expected a function signature")
	assert.Equal(t, "Foo", funcSig.Name)
	assert.Contains(t, funcSig.Source, "func Foo(x int, y string) (int, error)")
	assert.NotContains(t, funcSig.Source, "return 0, nil")
}

func TestGoCompressor_FunctionNoBody(t *testing.T) {
	c := NewGoCompressor()
	// External function declaration (no body).
	source := `package main

func ExternalFunc(x int) int
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.Equal(t, "ExternalFunc", funcSig.Name)
	assert.Contains(t, funcSig.Source, "func ExternalFunc(x int) int")
}

// ---------------------------------------------------------------------------
// Method declarations
// ---------------------------------------------------------------------------

func TestGoCompressor_MethodDeclaration(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig, "expected a method signature")
	assert.Equal(t, "Handle", funcSig.Name)
	assert.Contains(t, funcSig.Source, "func (s *Server) Handle(w http.ResponseWriter, r *http.Request)")
	assert.NotContains(t, funcSig.Source, "w.Write")
}

func TestGoCompressor_MethodValueReceiver(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

func (p Point) String() string {
	return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.Equal(t, "String", funcSig.Name)
	assert.Contains(t, funcSig.Source, "func (p Point) String() string")
}

// ---------------------------------------------------------------------------
// Struct declarations
// ---------------------------------------------------------------------------

func TestGoCompressor_StructDeclaration(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type Config struct {
	Host string ` + "`json:\"host\" yaml:\"host\"`" + `
	Port int    ` + "`json:\"port\"`" + `
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var structSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindStruct {
			structSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, structSig, "expected a struct signature")
	assert.Equal(t, "Config", structSig.Name)
	assert.Contains(t, structSig.Source, "Host string")
	assert.Contains(t, structSig.Source, "Port int")
}

func TestGoCompressor_EmptyStruct(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type Empty struct{}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var structSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindStruct {
			structSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, structSig)
	assert.Equal(t, "Empty", structSig.Name)
}

// ---------------------------------------------------------------------------
// Interface declarations
// ---------------------------------------------------------------------------

func TestGoCompressor_InterfaceDeclaration(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type Reader interface {
	Read(p []byte) (n int, err error)
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var ifaceSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			ifaceSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, ifaceSig, "expected an interface signature")
	assert.Equal(t, "Reader", ifaceSig.Name)
	assert.Contains(t, ifaceSig.Source, "Read(p []byte) (n int, err error)")
}

func TestGoCompressor_EmptyInterface(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type Any interface{}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var ifaceSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			ifaceSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, ifaceSig)
	assert.Equal(t, "Any", ifaceSig.Name)
}

// ---------------------------------------------------------------------------
// Type aliases and definitions
// ---------------------------------------------------------------------------

func TestGoCompressor_TypeAlias(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type ID = string
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var typeSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType {
			typeSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, typeSig, "expected a type signature")
	assert.Equal(t, "ID", typeSig.Name)
	assert.Contains(t, typeSig.Source, "type ID = string")
}

func TestGoCompressor_TypeDefinition(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type Duration int64
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var typeSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType {
			typeSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, typeSig)
	assert.Equal(t, "Duration", typeSig.Name)
}

// ---------------------------------------------------------------------------
// Const declarations
// ---------------------------------------------------------------------------

func TestGoCompressor_SingleConst(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

const MaxSize = 1024
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var constSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindConstant {
			constSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, constSig, "expected a constant signature")
	assert.Equal(t, "MaxSize", constSig.Name)
	assert.Contains(t, constSig.Source, "const MaxSize = 1024")
}

func TestGoCompressor_GroupedConst(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

const (
	Tier0 Tier = iota
	Tier1
	Tier2
)
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var constSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindConstant {
			constSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, constSig, "expected a const group signature")
	assert.Equal(t, "Tier0", constSig.Name)
	assert.Contains(t, constSig.Source, "Tier0 Tier = iota")
	assert.Contains(t, constSig.Source, "Tier1")
	assert.Contains(t, constSig.Source, "Tier2")
}

// ---------------------------------------------------------------------------
// Var declarations
// ---------------------------------------------------------------------------

func TestGoCompressor_SingleVar(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

var DefaultTimeout = 30
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var varSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindConstant && output.Signatures[i].Name == "DefaultTimeout" {
			varSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, varSig, "expected a var signature")
	assert.Contains(t, varSig.Source, "var DefaultTimeout = 30")
}

func TestGoCompressor_GroupedVar(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

var (
	ErrNotFound = errors.New("not found")
	ErrTimeout  = errors.New("timeout")
)
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var varSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindConstant && output.Signatures[i].Name == "ErrNotFound" {
			varSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, varSig, "expected a var group signature")
	assert.Contains(t, varSig.Source, "ErrNotFound")
	assert.Contains(t, varSig.Source, "ErrTimeout")
}

// ---------------------------------------------------------------------------
// Doc comment attachment
// ---------------------------------------------------------------------------

func TestGoCompressor_DocCommentOnFunc(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

// Greet returns a greeting message.
// It is used for testing.
func Greet(name string) string {
	return "Hello, " + name
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.Contains(t, funcSig.Source, "// Greet returns a greeting message.")
	assert.Contains(t, funcSig.Source, "// It is used for testing.")
	assert.Contains(t, funcSig.Source, "func Greet(name string) string")
}

func TestGoCompressor_BlockDocComment(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

/*
Package main provides the entry point.
It does important things.
*/
func Main() {
	run()
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.Contains(t, funcSig.Source, "Package main provides the entry point.")
	assert.Contains(t, funcSig.Source, "func Main()")
}

func TestGoCompressor_EmptyLineClearsDocComment(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

// This comment is separated by an empty line.

func Orphan() {
	return
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.NotContains(t, funcSig.Source, "This comment is separated")
	assert.Contains(t, funcSig.Source, "func Orphan()")
}

// ---------------------------------------------------------------------------
// Generic types and functions
// ---------------------------------------------------------------------------

func TestGoCompressor_GenericFunction(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

func Map[T any](s []T, f func(T) T) []T {
	result := make([]T, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.Equal(t, "Map", funcSig.Name)
	assert.Contains(t, funcSig.Source, "func Map[T any](s []T, f func(T) T) []T")
	assert.NotContains(t, funcSig.Source, "result :=")
}

func TestGoCompressor_GenericStruct(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type Set[T comparable] struct {
	items map[T]struct{}
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var structSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindStruct {
			structSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, structSig)
	assert.Equal(t, "Set", structSig.Name)
	assert.Contains(t, structSig.Source, "Set[T comparable]")
	assert.Contains(t, structSig.Source, "items map[T]struct{}")
}

// ---------------------------------------------------------------------------
// Build constraints and go directives
// ---------------------------------------------------------------------------

func TestGoCompressor_BuildConstraint(t *testing.T) {
	c := NewGoCompressor()
	// Build constraint directly preceding package clause (no blank line).
	source := `//go:build linux
package main
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	// The build constraint should be attached as a doc comment to the package clause.
	var pkgSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindImport && output.Signatures[i].Name == "main" {
			pkgSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, pkgSig)
	assert.Contains(t, pkgSig.Source, "//go:build linux")
	assert.Contains(t, pkgSig.Source, "package main")
}

func TestGoCompressor_BuildConstraintWithBlankLine(t *testing.T) {
	c := NewGoCompressor()
	// Standard Go layout: build constraint, blank line, package clause.
	// The blank line separates the constraint from being a doc comment.
	source := `//go:build linux

package main
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var pkgSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindImport && output.Signatures[i].Name == "main" {
			pkgSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, pkgSig)
	// With a blank line separator, the build constraint is not attached
	// as a doc comment per Go convention.
	assert.NotContains(t, pkgSig.Source, "//go:build linux")
	assert.Contains(t, pkgSig.Source, "package main")
}

// ---------------------------------------------------------------------------
// Source order preservation
// ---------------------------------------------------------------------------

func TestGoCompressor_SourceOrderPreserved(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

import "fmt"

const Version = "1.0"

type Config struct {
	Name string
}

func Run() {
	fmt.Println("run")
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 5)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)   // package
	assert.Equal(t, KindImport, output.Signatures[1].Kind)   // import
	assert.Equal(t, KindConstant, output.Signatures[2].Kind) // const
	assert.Equal(t, KindStruct, output.Signatures[3].Kind)   // struct
	assert.Equal(t, KindFunction, output.Signatures[4].Kind) // func
}

// ---------------------------------------------------------------------------
// Compression ratio
// ---------------------------------------------------------------------------

func TestGoCompressor_CompressionRatio(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

import (
	"fmt"
	"strings"
)

// Config holds application configuration.
type Config struct {
	Host string
	Port int
}

// NewConfig creates a Config with defaults.
func NewConfig() *Config {
	return &Config{
		Host: "localhost",
		Port: 8080,
	}
}

// Run starts the server.
func Run(cfg *Config) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid address: %s", addr)
	}
	fmt.Printf("Listening on %s\n", addr)
	return nil
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	output.Render()
	ratio := output.CompressionRatio()
	// Expect meaningful compression -- function bodies should be stripped.
	assert.Greater(t, ratio, 0.1, "ratio should be > 0.1")
	assert.Less(t, ratio, 0.85, "ratio should be < 0.85 (at least 15%% reduction)")
}

// ---------------------------------------------------------------------------
// Complex real-world patterns
// ---------------------------------------------------------------------------

func TestGoCompressor_CompleteFile(t *testing.T) {
	c := NewGoCompressor()
	source := `package server

import (
	"context"
	"fmt"
	"net/http"
)

// ErrShutdown is returned when the server is shutting down.
var ErrShutdown = fmt.Errorf("server shutting down")

// Server handles HTTP requests.
type Server struct {
	addr   string
	mux    *http.ServeMux
	logger Logger
}

// Logger defines the logging interface.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Option configures a Server.
type Option func(*Server)

// WithAddr sets the listen address.
func WithAddr(addr string) Option {
	return func(s *Server) {
		s.addr = addr
	}
}

// New creates a new Server with the given options.
func New(opts ...Option) *Server {
	s := &Server{
		addr: ":8080",
		mux:  http.NewServeMux(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start begins serving HTTP requests.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
	}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	s.logger.Info("starting server", "addr", s.addr)
	return srv.ListenAndServe()
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()

	// Should contain package and import.
	assert.Contains(t, rendered, "package server")
	assert.Contains(t, rendered, `"net/http"`)

	// Should contain struct with fields.
	assert.Contains(t, rendered, "type Server struct")
	assert.Contains(t, rendered, "addr   string")

	// Should contain interface.
	assert.Contains(t, rendered, "type Logger interface")

	// Should contain function signatures.
	assert.Contains(t, rendered, "func WithAddr(addr string) Option")
	assert.Contains(t, rendered, "func New(opts ...Option) *Server")
	assert.Contains(t, rendered, "func (s *Server) Start(ctx context.Context) error")

	// Should NOT contain function bodies.
	assert.NotContains(t, rendered, "s.addr = addr")
	assert.NotContains(t, rendered, "srv.ListenAndServe()")
	assert.NotContains(t, rendered, "srv.Shutdown")

	// Should contain var declaration.
	assert.Contains(t, rendered, "var ErrShutdown")

	// Should contain doc comments.
	assert.Contains(t, rendered, "// Server handles HTTP requests.")
	assert.Contains(t, rendered, "// Logger defines the logging interface.")
}

func TestGoCompressor_InitFunc(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

func init() {
	setupLogging()
}

func main() {
	run()
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	funcNames := make([]string, 0)
	for _, sig := range output.Signatures {
		if sig.Kind == KindFunction {
			funcNames = append(funcNames, sig.Name)
		}
	}
	assert.Contains(t, funcNames, "init")
	assert.Contains(t, funcNames, "main")
}

func TestGoCompressor_FuncWithEmptyBody(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

func noop() {}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.Equal(t, "noop", funcSig.Name)
	assert.Contains(t, funcSig.Source, "func noop()")
}

func TestGoCompressor_MultipleMethodsOnSameType(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type Point struct {
	X, Y int
}

func (p Point) Add(other Point) Point {
	return Point{X: p.X + other.X, Y: p.Y + other.Y}
}

func (p *Point) Scale(factor int) {
	p.X *= factor
	p.Y *= factor
}

func (p Point) String() string {
	return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	funcCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindFunction {
			funcCount++
		}
	}
	assert.Equal(t, 3, funcCount, "expected 3 method signatures")
}

func TestGoCompressor_GroupedTypeDeclaration(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type (
	ID     string
	Name   string
	Age    int
)
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var typeSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType {
			typeSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, typeSig)
	assert.Equal(t, "ID", typeSig.Name) // first name in group
	assert.Contains(t, typeSig.Source, "ID     string")
	assert.Contains(t, typeSig.Source, "Name   string")
	assert.Contains(t, typeSig.Source, "Age    int")
}

// ---------------------------------------------------------------------------
// Brace counting with Go-specific strings
// ---------------------------------------------------------------------------

func TestGoCountBraces(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		depth int
	}{
		{"empty", "", 0},
		{"open brace", "func main() {", 1},
		{"close brace", "}", -1},
		{"balanced", "if x { y() }", 0},
		{"brace in double-quoted string", `fmt.Println("{")`, 0},
		{"brace in raw string", "s := `{}`", 0},
		{"brace in rune", "ch := '{'", 0},
		{"brace after line comment", "x := 1 // {", 0},
		{"struct tag with backticks", "Host string `json:\"host\"`", 0},
		{"nested braces", "map[string]struct{}{}", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := goCountBraces(tt.line)
			assert.Equal(t, tt.depth, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestGoCompressor_DocCommentOnStruct(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

// Point represents a 2D point.
type Point struct {
	X int
	Y int
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var structSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindStruct {
			structSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, structSig)
	assert.Contains(t, structSig.Source, "// Point represents a 2D point.")
	assert.Contains(t, structSig.Source, "type Point struct")
}

func TestGoCompressor_DocCommentOnConst(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

// MaxRetries is the maximum number of retries.
const MaxRetries = 3
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var constSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindConstant {
			constSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, constSig)
	assert.Contains(t, constSig.Source, "// MaxRetries is the maximum number of retries.")
}

func TestGoCompressor_InterfaceWithEmbedding(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

type ReadWriter interface {
	Reader
	Writer
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var ifaceSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			ifaceSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, ifaceSig)
	assert.Equal(t, "ReadWriter", ifaceSig.Name)
	assert.Contains(t, ifaceSig.Source, "Reader")
	assert.Contains(t, ifaceSig.Source, "Writer")
}

func TestGoCompressor_ConstTyped(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

const Pi float64 = 3.14159
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var constSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindConstant {
			constSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, constSig)
	assert.Equal(t, "Pi", constSig.Name)
	assert.Contains(t, constSig.Source, "const Pi float64 = 3.14159")
}

func TestGoCompressor_VarWithType(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

var mu sync.Mutex
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var varSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindConstant && output.Signatures[i].Name == "mu" {
			varSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, varSig)
	assert.Contains(t, varSig.Source, "var mu sync.Mutex")
}

func TestGoCompressor_CompileTimeInterfaceCheck(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

var _ LanguageCompressor = (*GoCompressor)(nil)
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	// The var _ line should be captured.
	var varSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindConstant {
			varSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, varSig)
	assert.Contains(t, varSig.Source, "var _ LanguageCompressor")
}

func TestGoCompressor_FuncMultipleReturnValues(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.Equal(t, "Divide", funcSig.Name)
	assert.Contains(t, funcSig.Source, "func Divide(a, b float64) (float64, error)")
	assert.NotContains(t, funcSig.Source, "division by zero")
}

func TestGoCompressor_NestedBraces(t *testing.T) {
	c := NewGoCompressor()
	source := `package main

func Complex() {
	m := map[string]struct{}{
		"a": {},
		"b": {},
	}
	for k := range m {
		if k == "a" {
			fmt.Println(k)
		}
	}
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig)
	assert.Equal(t, "Complex", funcSig.Name)
	assert.NotContains(t, funcSig.Source, "fmt.Println")
}

// ---------------------------------------------------------------------------
// Golden tests using fixture files
// ---------------------------------------------------------------------------

func TestGoCompressor_GoldenTests(t *testing.T) {
	compressor := NewGoCompressor()
	ctx := context.Background()

	tests := []struct {
		name     string
		fixture  string
		expected string
	}{
		{
			name:     "simple functions",
			fixture:  "go/simple_func.go",
			expected: "go/simple_func.go.expected",
		},
		{
			name:     "methods with receivers",
			fixture:  "go/methods.go",
			expected: "go/methods.go.expected",
		},
		{
			name:     "struct declarations",
			fixture:  "go/structs.go",
			expected: "go/structs.go.expected",
		},
		{
			name:     "interface declarations",
			fixture:  "go/interfaces.go",
			expected: "go/interfaces.go.expected",
		},
		{
			name:     "generics",
			fixture:  "go/generics.go",
			expected: "go/generics.go.expected",
		},
		{
			name:     "const and iota",
			fixture:  "go/const_iota.go",
			expected: "go/const_iota.go.expected",
		},
		{
			name:     "import patterns",
			fixture:  "go/imports.go",
			expected: "go/imports.go.expected",
		},
		{
			name:     "full realistic file",
			fixture:  "go/full_file.go",
			expected: "go/full_file.go.expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := readFixture(t, tt.fixture)
			expected := readExpected(t, tt.expected)

			output, err := compressor.Compress(ctx, source)
			require.NoError(t, err)

			rendered := strings.TrimSpace(output.Render())
			assert.Equal(t, expected, rendered,
				"golden test mismatch for %s", tt.fixture)
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkGoCompressor(b *testing.B) {
	c := NewGoCompressor()
	source := []byte(`package server

import (
	"context"
	"fmt"
	"net/http"
)

var ErrShutdown = fmt.Errorf("server shutting down")

type Server struct {
	addr   string
	mux    *http.ServeMux
}

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

func New(addr string) *Server {
	return &Server{addr: addr, mux: http.NewServeMux()}
}

func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{Addr: s.addr, Handler: s.mux}
	go func() { <-ctx.Done(); srv.Shutdown(context.Background()) }()
	return srv.ListenAndServe()
}

func (s *Server) Stop() error {
	return nil
}
`)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}
