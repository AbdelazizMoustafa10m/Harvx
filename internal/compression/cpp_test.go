package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CppCompressor metadata tests
// ---------------------------------------------------------------------------

func TestCppCompressor_Language(t *testing.T) {
	c := NewCppCompressor()
	assert.Equal(t, "cpp", c.Language())
}

func TestCppCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewCppCompressor()
	types := c.SupportedNodeTypes()
	assert.Contains(t, types, "preproc_include")
	assert.Contains(t, types, "preproc_def")
	assert.Contains(t, types, "function_definition")
	assert.Contains(t, types, "declaration")
	assert.Contains(t, types, "struct_specifier")
	assert.Contains(t, types, "enum_specifier")
	assert.Contains(t, types, "type_definition")
	assert.Contains(t, types, "class_specifier")
	assert.Contains(t, types, "template_declaration")
	assert.Contains(t, types, "namespace_definition")
	assert.Contains(t, types, "using_declaration")
}

func TestCppCompressor_EmptyInput(t *testing.T) {
	c := NewCppCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "cpp", output.Language)
}

func TestCppCompressor_ContextCancellation(t *testing.T) {
	c := NewCppCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("// line\n")
	}

	_, err := c.Compress(ctx, []byte(b.String()))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// C-compatible features (inherited from C)
// ---------------------------------------------------------------------------

func TestCppCompressor_Includes(t *testing.T) {
	c := NewCppCompressor()
	source := `#include <iostream>
#include <vector>
#include "config.h"`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	importCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindImport {
			importCount++
		}
	}
	assert.Equal(t, 3, importCount)
}

func TestCppCompressor_Defines(t *testing.T) {
	c := NewCppCompressor()
	source := `#define VERSION "1.0.0"
#define MAX(a, b) ((a) > (b) ? (a) : (b))`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	constCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindConstant {
			constCount++
		}
	}
	assert.Equal(t, 2, constCount)
}

func TestCppCompressor_Struct(t *testing.T) {
	c := NewCppCompressor()
	source := `struct Point {
    double x;
    double y;
};`
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
	assert.Equal(t, "Point", structSig.Name)
	assert.Contains(t, structSig.Source, "double x;")
	assert.Contains(t, structSig.Source, "double y;")
}

func TestCppCompressor_FuncDefinition(t *testing.T) {
	c := NewCppCompressor()
	source := `int add(int a, int b) {
    return a + b;
}`
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
	assert.Equal(t, "add", funcSig.Name)
	assert.Contains(t, funcSig.Source, "int add(int a, int b)")
	assert.NotContains(t, funcSig.Source, "return a + b")
}

func TestCppCompressor_FuncPrototype(t *testing.T) {
	c := NewCppCompressor()
	source := `int process(const std::string& data);`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "process", sig.Name)
}

// ---------------------------------------------------------------------------
// Class declarations
// ---------------------------------------------------------------------------

func TestCppCompressor_SimpleClass(t *testing.T) {
	c := NewCppCompressor()
	source := `class Animal {
public:
    virtual ~Animal() {}
    virtual void speak() const = 0;
    std::string name;
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig, "expected a class signature")
	assert.Equal(t, "Animal", classSig.Name)
	assert.Contains(t, classSig.Source, "public:")
	assert.Contains(t, classSig.Source, "virtual ~Animal()")
	assert.Contains(t, classSig.Source, "virtual void speak() const = 0;")
	assert.Contains(t, classSig.Source, "std::string name;")
}

func TestCppCompressor_ClassWithInheritance(t *testing.T) {
	c := NewCppCompressor()
	source := `class Dog : public Animal {
public:
    Dog(const std::string& name) : Animal(name) {}
    void speak() const override {
        std::cout << "Woof!" << std::endl;
    }
private:
    int tricks_count;
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Equal(t, "Dog", classSig.Name)
	assert.Contains(t, classSig.Source, "class Dog : public Animal {")
	assert.Contains(t, classSig.Source, "public:")
	assert.Contains(t, classSig.Source, "private:")
	assert.Contains(t, classSig.Source, "int tricks_count;")
	// Method bodies should be excluded.
	assert.NotContains(t, classSig.Source, "std::cout")
}

func TestCppCompressor_ClassWithMultipleInheritance(t *testing.T) {
	c := NewCppCompressor()
	source := `class Widget : public Drawable, public Clickable {
public:
    void draw() override {
        render();
    }
    void on_click() override {
        handle();
    }
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Equal(t, "Widget", classSig.Name)
	assert.Contains(t, classSig.Source, "class Widget : public Drawable, public Clickable {")
}

func TestCppCompressor_ClassAccessSpecifiers(t *testing.T) {
	c := NewCppCompressor()
	source := `class Server {
public:
    void start();
    void stop();
protected:
    int port;
private:
    bool running;
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "public:")
	assert.Contains(t, classSig.Source, "protected:")
	assert.Contains(t, classSig.Source, "private:")
	assert.Contains(t, classSig.Source, "void start();")
	assert.Contains(t, classSig.Source, "void stop();")
	assert.Contains(t, classSig.Source, "int port;")
	assert.Contains(t, classSig.Source, "bool running;")
}

func TestCppCompressor_VirtualDestructor(t *testing.T) {
	c := NewCppCompressor()
	source := `class Base {
public:
    virtual ~Base() = default;
    virtual void process() = 0;
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "virtual ~Base() = default;")
	assert.Contains(t, classSig.Source, "virtual void process() = 0;")
}

// ---------------------------------------------------------------------------
// Template declarations
// ---------------------------------------------------------------------------

func TestCppCompressor_TemplateFunction(t *testing.T) {
	c := NewCppCompressor()
	source := `template<typename T>
T max_val(T a, T b) {
    return (a > b) ? a : b;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			funcSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, funcSig, "expected a template function signature")
	assert.Contains(t, funcSig.Source, "template<typename T>")
	assert.Contains(t, funcSig.Source, "T max_val(T a, T b)")
	assert.NotContains(t, funcSig.Source, "return (a > b)")
}

func TestCppCompressor_TemplateClass(t *testing.T) {
	c := NewCppCompressor()
	source := `template<typename T>
class Stack {
public:
    void push(const T& value) {
        data.push_back(value);
    }
    T pop() {
        T val = data.back();
        data.pop_back();
        return val;
    }
private:
    std::vector<T> data;
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig, "expected a template class signature")
	assert.Equal(t, "Stack", classSig.Name)
	assert.Contains(t, classSig.Source, "template<typename T>")
	assert.Contains(t, classSig.Source, "class Stack {")
	assert.Contains(t, classSig.Source, "void push(const T& value)")
	assert.Contains(t, classSig.Source, "std::vector<T> data;")
	assert.NotContains(t, classSig.Source, "data.push_back")
}

func TestCppCompressor_TemplatePrototype(t *testing.T) {
	c := NewCppCompressor()
	source := `template<typename T, typename U>
U convert(const T& input);`
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
	assert.Contains(t, funcSig.Source, "template<typename T, typename U>")
	assert.Contains(t, funcSig.Source, "U convert(const T& input);")
}

// ---------------------------------------------------------------------------
// Namespace definitions
// ---------------------------------------------------------------------------

func TestCppCompressor_SimpleNamespace(t *testing.T) {
	c := NewCppCompressor()
	source := `namespace mylib {
void init();
void shutdown();
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var nsSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType && output.Signatures[i].Name == "mylib" {
			nsSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, nsSig, "expected a namespace signature")
	assert.Equal(t, "mylib", nsSig.Name)
	assert.Contains(t, nsSig.Source, "namespace mylib {")
	assert.Contains(t, nsSig.Source, "void init();")
	assert.Contains(t, nsSig.Source, "void shutdown();")
}

func TestCppCompressor_NestedNamespace(t *testing.T) {
	c := NewCppCompressor()
	source := `namespace a::b::c {
int value();
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var nsSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType && strings.Contains(output.Signatures[i].Name, "a") {
			nsSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, nsSig)
	assert.Contains(t, nsSig.Source, "namespace a::b::c {")
	assert.Contains(t, nsSig.Source, "int value();")
}

func TestCppCompressor_InlineNamespace(t *testing.T) {
	c := NewCppCompressor()
	source := `inline namespace v2 {
void process();
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var nsSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType {
			nsSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, nsSig)
	assert.Contains(t, nsSig.Source, "inline namespace v2 {")
}

func TestCppCompressor_NamespaceWithFuncDef(t *testing.T) {
	c := NewCppCompressor()
	source := `namespace utils {
int helper(int x) {
    return x * 2;
}
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var nsSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType && output.Signatures[i].Name == "utils" {
			nsSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, nsSig)
	assert.Contains(t, nsSig.Source, "namespace utils {")
	// Function signature (without body) should be extracted.
	assert.Contains(t, nsSig.Source, "int helper(int x)")
	assert.NotContains(t, nsSig.Source, "return x * 2")
}

// ---------------------------------------------------------------------------
// Using declarations
// ---------------------------------------------------------------------------

func TestCppCompressor_UsingNamespace(t *testing.T) {
	c := NewCppCompressor()
	source := `using namespace std;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindType, sig.Kind)
	assert.Contains(t, sig.Source, "using namespace std;")
}

func TestCppCompressor_UsingTypeAlias(t *testing.T) {
	c := NewCppCompressor()
	source := `using StringVec = std::vector<std::string>;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindType, sig.Kind)
	assert.Contains(t, sig.Source, "using StringVec = std::vector<std::string>;")
}

// ---------------------------------------------------------------------------
// Enum class
// ---------------------------------------------------------------------------

func TestCppCompressor_EnumClass(t *testing.T) {
	c := NewCppCompressor()
	source := `enum class Color : int {
    Red,
    Green,
    Blue
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var enumSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType {
			enumSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, enumSig, "expected an enum class signature")
	assert.Equal(t, "Color", enumSig.Name)
	assert.Contains(t, enumSig.Source, "enum class Color : int {")
	assert.Contains(t, enumSig.Source, "Red,")
	assert.Contains(t, enumSig.Source, "Green,")
	assert.Contains(t, enumSig.Source, "Blue")
}

func TestCppCompressor_EnumStruct(t *testing.T) {
	c := NewCppCompressor()
	source := `enum struct Status {
    Active,
    Inactive
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var enumSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindType {
			enumSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, enumSig)
	assert.Equal(t, "Status", enumSig.Name)
}

// ---------------------------------------------------------------------------
// Forward declarations
// ---------------------------------------------------------------------------

func TestCppCompressor_ClassForwardDecl(t *testing.T) {
	c := NewCppCompressor()
	source := `class Widget;
struct Point;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 2)

	// class Widget; is caught by isCppClassForwardDecl -> KindType.
	assert.Equal(t, KindType, output.Signatures[0].Kind)
	assert.Contains(t, output.Signatures[0].Source, "class Widget;")

	// struct Point; is caught by isCStructDecl -> handleStructDecl -> KindStruct.
	assert.Equal(t, KindStruct, output.Signatures[1].Kind)
	assert.Contains(t, output.Signatures[1].Source, "struct Point;")
}

// ---------------------------------------------------------------------------
// Out-of-class method definitions
// ---------------------------------------------------------------------------

func TestCppCompressor_OutOfClassMethodDef(t *testing.T) {
	c := NewCppCompressor()
	source := `void Server::start() {
    running = true;
    listen();
}`
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
	assert.Contains(t, funcSig.Source, "void Server::start()")
	assert.NotContains(t, funcSig.Source, "running = true")
}

func TestCppCompressor_OutOfClassMethodPrototype(t *testing.T) {
	c := NewCppCompressor()
	source := `void Server::start();`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Contains(t, sig.Source, "void Server::start();")
}

// ---------------------------------------------------------------------------
// Constexpr functions
// ---------------------------------------------------------------------------

func TestCppCompressor_ConstexprFunc(t *testing.T) {
	c := NewCppCompressor()
	source := `constexpr int factorial(int n) {
    return n <= 1 ? 1 : n * factorial(n - 1);
}`
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
	assert.Contains(t, funcSig.Source, "constexpr int factorial(int n)")
	assert.NotContains(t, funcSig.Source, "return n <= 1")
}

// ---------------------------------------------------------------------------
// Doc comments
// ---------------------------------------------------------------------------

func TestCppCompressor_DocCommentOnClass(t *testing.T) {
	c := NewCppCompressor()
	source := `/**
 * Manages database connections.
 */
class ConnectionPool {
public:
    void connect();
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "/**")
	assert.Contains(t, classSig.Source, "Manages database connections.")
	assert.Contains(t, classSig.Source, "class ConnectionPool {")
}

// ---------------------------------------------------------------------------
// C++-specific detection helper tests
// ---------------------------------------------------------------------------

func TestIsCppClassDecl(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"simple class", "class Foo {", true},
		{"class with base", "class Dog : public Animal {", true},
		{"forward decl", "class Foo;", false},
		{"no brace", "class Foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCppClassDecl(tt.line)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCppTemplateDecl(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"no space", "template<typename T>", true},
		{"with space", "template <typename T>", true},
		{"not template", "void template_func() {", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCppTemplateDecl(tt.line)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCppNamespaceDecl(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"simple", "namespace foo {", true},
		{"nested", "namespace a::b::c {", true},
		{"inline", "inline namespace v2 {", true},
		{"not namespace", "int namespace_var;", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCppNamespaceDecl(tt.line)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCppAccessSpecifier(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"public", "public:", true},
		{"private", "private:", true},
		{"protected", "protected:", true},
		{"not specifier", "public_api:", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCppAccessSpecifier(tt.line)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractCppClassName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "class Foo {", "Foo"},
		{"with base", "class Dog : public Animal {", "Dog"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCppClassName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractCppNamespaceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "namespace foo {", "foo"},
		{"nested", "namespace a::b::c {", "a::b::c"},
		{"inline", "inline namespace v2 {", "v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCppNamespaceName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractCppEnumName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain enum", "enum Color {", "Color"},
		{"enum class", "enum class Status {", "Status"},
		{"enum struct", "enum struct Direction {", "Direction"},
		{"enum class with base type", "enum class Color : int {", "Color"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCppEnumName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Complex/realistic tests
// ---------------------------------------------------------------------------

func TestCppCompressor_CompleteFile(t *testing.T) {
	c := NewCppCompressor()
	source := `#include <iostream>
#include <vector>
#include <string>

#define MAX_SIZE 1024

namespace myapp {

using StringVec = std::vector<std::string>;

enum class LogLevel {
    Debug,
    Info,
    Error
};

class Logger {
public:
    Logger(const std::string& name) : name_(name) {}
    void log(LogLevel level, const std::string& msg) {
        std::cout << name_ << ": " << msg << std::endl;
    }
    const std::string& name() const { return name_; }
private:
    std::string name_;
};

template<typename T>
T clamp(T value, T low, T high) {
    if (value < low) return low;
    if (value > high) return high;
    return value;
}

} // namespace myapp
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()

	// Includes.
	assert.Contains(t, rendered, "#include <iostream>")
	assert.Contains(t, rendered, "#include <vector>")
	assert.Contains(t, rendered, "#include <string>")

	// Define.
	assert.Contains(t, rendered, "#define MAX_SIZE")

	// Namespace.
	assert.Contains(t, rendered, "namespace myapp {")
}

func TestCppCompressor_SourceOrderPreserved(t *testing.T) {
	c := NewCppCompressor()
	source := `#include <iostream>

#define VERSION 1

struct Point {
    int x;
    int y;
};

int add(int a, int b) {
    return a + b;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 4)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)   // #include
	assert.Equal(t, KindConstant, output.Signatures[1].Kind) // #define
	assert.Equal(t, KindStruct, output.Signatures[2].Kind)   // struct
	assert.Equal(t, KindFunction, output.Signatures[3].Kind) // function
}

func TestCppCompressor_CompressionRatio(t *testing.T) {
	c := NewCppCompressor()
	source := `#include <iostream>
#include <vector>

class Server {
public:
    Server(int port) : port_(port), running_(false) {}

    void start() {
        running_ = true;
        while (running_) {
            accept_connections();
            handle_requests();
        }
    }

    void stop() {
        running_ = false;
    }

    int port() const { return port_; }

private:
    int port_;
    bool running_;

    void accept_connections() {
        // complex implementation
        for (int i = 0; i < 100; i++) {
            process(i);
        }
    }

    void handle_requests() {
        // complex implementation
        std::vector<int> requests;
        for (auto& r : requests) {
            handle(r);
        }
    }
};
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	output.Render()
	ratio := output.CompressionRatio()
	assert.Greater(t, ratio, 0.1, "ratio should be > 0.1")
	assert.Less(t, ratio, 0.85, "ratio should be < 0.85 (at least 15%% reduction)")
}

// ---------------------------------------------------------------------------
// Friend declarations and using inside class
// ---------------------------------------------------------------------------

func TestCppCompressor_FriendDeclaration(t *testing.T) {
	c := NewCppCompressor()
	source := `class Foo {
    friend class Bar;
    friend void helper(Foo& f);
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "friend class Bar;")
	assert.Contains(t, classSig.Source, "friend void helper(Foo& f);")
}

func TestCppCompressor_UsingInsideClass(t *testing.T) {
	c := NewCppCompressor()
	source := `class Derived : public Base {
public:
    using Base::method;
    using value_type = int;
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			classSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, classSig)
	assert.Contains(t, classSig.Source, "using Base::method;")
	assert.Contains(t, classSig.Source, "using value_type = int;")
}

// ---------------------------------------------------------------------------
// Golden tests
// ---------------------------------------------------------------------------

func TestCppCompressor_GoldenTests(t *testing.T) {
	compressor := NewCppCompressor()
	ctx := context.Background()

	tests := []struct {
		name     string
		fixture  string
		expected string
	}{
		{
			name:     "class with inheritance and templates",
			fixture:  "cpp/class_templates.cpp",
			expected: "cpp/class_templates.cpp.expected",
		},
		{
			name:     "namespaces and using declarations",
			fixture:  "cpp/namespaces.cpp",
			expected: "cpp/namespaces.cpp.expected",
		},
		{
			name:     "enum class and structs",
			fixture:  "cpp/enums_structs.cpp",
			expected: "cpp/enums_structs.cpp.expected",
		},
		{
			name:     "complete realistic file",
			fixture:  "cpp/complete_file.cpp",
			expected: "cpp/complete_file.cpp.expected",
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

func BenchmarkCppCompressor(b *testing.B) {
	c := NewCppCompressor()
	source := []byte(`#include <iostream>
#include <vector>
#include <string>

#define MAX_SIZE 1024

namespace myapp {

using StringVec = std::vector<std::string>;

enum class LogLevel {
    Debug,
    Info,
    Error
};

class Logger {
public:
    Logger(const std::string& name) : name_(name) {}
    void log(LogLevel level, const std::string& msg) {
        std::cout << name_ << ": " << msg << std::endl;
    }
private:
    std::string name_;
};

template<typename T>
T clamp(T value, T low, T high) {
    if (value < low) return low;
    if (value > high) return high;
    return value;
}

} // namespace myapp
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
