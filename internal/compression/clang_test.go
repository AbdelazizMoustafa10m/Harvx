package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CCompressor metadata tests
// ---------------------------------------------------------------------------

func TestCCompressor_Language(t *testing.T) {
	c := NewCCompressor()
	assert.Equal(t, "c", c.Language())
}

func TestCCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewCCompressor()
	types := c.SupportedNodeTypes()
	assert.Contains(t, types, "preproc_include")
	assert.Contains(t, types, "preproc_def")
	assert.Contains(t, types, "function_definition")
	assert.Contains(t, types, "declaration")
	assert.Contains(t, types, "struct_specifier")
	assert.Contains(t, types, "enum_specifier")
	assert.Contains(t, types, "type_definition")
}

func TestCCompressor_EmptyInput(t *testing.T) {
	c := NewCCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "c", output.Language)
}

func TestCCompressor_ContextCancellation(t *testing.T) {
	c := NewCCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Generate large source to ensure cancellation is checked.
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("// line\n")
	}

	_, err := c.Compress(ctx, []byte(b.String()))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// #include directives
// ---------------------------------------------------------------------------

func TestCCompressor_IncludeAngleBrackets(t *testing.T) {
	c := NewCCompressor()
	source := `#include <stdio.h>`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindImport, sig.Kind)
	assert.Contains(t, sig.Source, "#include <stdio.h>")
}

func TestCCompressor_IncludeQuotes(t *testing.T) {
	c := NewCCompressor()
	source := `#include "myheader.h"`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)
	assert.Contains(t, output.Signatures[0].Source, `#include "myheader.h"`)
}

func TestCCompressor_MultipleIncludes(t *testing.T) {
	c := NewCCompressor()
	source := `#include <stdio.h>
#include <stdlib.h>
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

// ---------------------------------------------------------------------------
// #define macros
// ---------------------------------------------------------------------------

func TestCCompressor_SimpleDefine(t *testing.T) {
	c := NewCCompressor()
	source := `#define MAX_SIZE 1024`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindConstant, sig.Kind)
	assert.Equal(t, "MAX_SIZE", sig.Name)
	assert.Equal(t, "#define MAX_SIZE", sig.Source)
}

func TestCCompressor_FunctionLikeDefine(t *testing.T) {
	c := NewCCompressor()
	source := `#define MIN(a, b) ((a) < (b) ? (a) : (b))`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindConstant, sig.Kind)
	assert.Equal(t, "MIN", sig.Name)
	assert.Equal(t, "#define MIN(a, b)", sig.Source)
}

func TestCCompressor_MultiLineDefine(t *testing.T) {
	c := NewCCompressor()
	source := `#define MULTI_LINE_MACRO(x) \
    do { \
        printf("%d\n", x); \
    } while(0)`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindConstant, sig.Kind)
	assert.Equal(t, "MULTI_LINE_MACRO", sig.Name)
	assert.Equal(t, "#define MULTI_LINE_MACRO(x)", sig.Source)
}

func TestCCompressor_DefineWithoutValue(t *testing.T) {
	c := NewCCompressor()
	source := `#define FEATURE_ENABLED`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindConstant, sig.Kind)
	assert.Equal(t, "FEATURE_ENABLED", sig.Name)
	assert.Equal(t, "#define FEATURE_ENABLED", sig.Source)
}

// ---------------------------------------------------------------------------
// Function definitions
// ---------------------------------------------------------------------------

func TestCCompressor_SimpleFuncDef(t *testing.T) {
	c := NewCCompressor()
	source := `int main(int argc, char **argv) {
    printf("hello\n");
    return 0;
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
	require.NotNil(t, funcSig, "expected a function signature")
	assert.Equal(t, "main", funcSig.Name)
	assert.Contains(t, funcSig.Source, "int main(int argc, char **argv)")
	assert.NotContains(t, funcSig.Source, "printf")
	assert.NotContains(t, funcSig.Source, "return 0")
}

func TestCCompressor_StaticFunc(t *testing.T) {
	c := NewCCompressor()
	source := `static int helper(int x) {
    return x * 2;
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
	assert.Equal(t, "helper", funcSig.Name)
	assert.Contains(t, funcSig.Source, "static int helper(int x)")
	assert.NotContains(t, funcSig.Source, "return x * 2")
}

func TestCCompressor_VoidFunc(t *testing.T) {
	c := NewCCompressor()
	source := `void process(const char *data, size_t len) {
    for (size_t i = 0; i < len; i++) {
        handle(data[i]);
    }
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
	assert.Equal(t, "process", funcSig.Name)
	assert.Contains(t, funcSig.Source, "void process(const char *data, size_t len)")
	assert.NotContains(t, funcSig.Source, "handle(data[i])")
}

func TestCCompressor_StructReturnFunc(t *testing.T) {
	c := NewCCompressor()
	source := `struct Config *config_new(const char *host, int port) {
    struct Config *cfg = malloc(sizeof(struct Config));
    cfg->host = strdup(host);
    cfg->port = port;
    return cfg;
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
	require.NotNil(t, funcSig, "expected a function signature for struct-return function")
	assert.Equal(t, "config_new", funcSig.Name)
	assert.Contains(t, funcSig.Source, "struct Config *config_new(const char *host, int port)")
	assert.NotContains(t, funcSig.Source, "malloc")

	// Verify it was NOT detected as a struct.
	for _, sig := range output.Signatures {
		assert.NotEqual(t, KindStruct, sig.Kind, "struct-return function should not be detected as struct")
	}
}

func TestCCompressor_PointerReturnFunc(t *testing.T) {
	c := NewCCompressor()
	source := `char *strdup(const char *s) {
    size_t len = strlen(s);
    char *copy = malloc(len + 1);
    memcpy(copy, s, len + 1);
    return copy;
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
	assert.Equal(t, "strdup", funcSig.Name)
	assert.NotContains(t, funcSig.Source, "malloc")
}

// ---------------------------------------------------------------------------
// Function prototypes
// ---------------------------------------------------------------------------

func TestCCompressor_FuncPrototype(t *testing.T) {
	c := NewCCompressor()
	source := `int process(const char *data, size_t len);`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "process", sig.Name)
	assert.Contains(t, sig.Source, "int process(const char *data, size_t len);")
}

func TestCCompressor_VariadicPrototype(t *testing.T) {
	c := NewCCompressor()
	source := `int printf(const char *fmt, ...);`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "printf", sig.Name)
}

// ---------------------------------------------------------------------------
// Struct declarations
// ---------------------------------------------------------------------------

func TestCCompressor_SimpleStruct(t *testing.T) {
	c := NewCCompressor()
	source := `struct Point {
    int x;
    int y;
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
	require.NotNil(t, structSig, "expected a struct signature")
	assert.Equal(t, "Point", structSig.Name)
	assert.Contains(t, structSig.Source, "int x;")
	assert.Contains(t, structSig.Source, "int y;")
}

func TestCCompressor_TypedefStruct(t *testing.T) {
	c := NewCCompressor()
	source := `typedef struct {
    char name[64];
    int age;
} Person;`
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
	assert.Equal(t, "Person", structSig.Name)
	assert.Contains(t, structSig.Source, "char name[64];")
	assert.Contains(t, structSig.Source, "int age;")
}

func TestCCompressor_TypedefNamedStruct(t *testing.T) {
	c := NewCCompressor()
	source := `typedef struct node {
    int value;
    struct node *next;
} Node;`
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
	assert.Equal(t, "Node", structSig.Name)
	assert.Contains(t, structSig.Source, "int value;")
	assert.Contains(t, structSig.Source, "struct node *next;")
}

// ---------------------------------------------------------------------------
// Enum declarations
// ---------------------------------------------------------------------------

func TestCCompressor_SimpleEnum(t *testing.T) {
	c := NewCCompressor()
	source := `enum Color {
    RED,
    GREEN,
    BLUE
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
	require.NotNil(t, enumSig, "expected an enum signature")
	assert.Equal(t, "Color", enumSig.Name)
	assert.Contains(t, enumSig.Source, "RED,")
	assert.Contains(t, enumSig.Source, "GREEN,")
	assert.Contains(t, enumSig.Source, "BLUE")
}

func TestCCompressor_TypedefEnum(t *testing.T) {
	c := NewCCompressor()
	source := `typedef enum {
    STATUS_OK,
    STATUS_ERROR,
    STATUS_PENDING
} Status;`
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
	assert.Contains(t, enumSig.Source, "STATUS_OK,")
}

// ---------------------------------------------------------------------------
// Typedef statements
// ---------------------------------------------------------------------------

func TestCCompressor_SimpleTypedef(t *testing.T) {
	c := NewCCompressor()
	source := `typedef unsigned long size_t;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindType, sig.Kind)
	assert.Contains(t, sig.Source, "typedef unsigned long size_t;")
}

func TestCCompressor_FunctionPointerTypedef(t *testing.T) {
	c := NewCCompressor()
	source := `typedef void (*handler_t)(int, const char*);`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindType, sig.Kind)
	assert.Contains(t, sig.Source, "typedef void (*handler_t)(int, const char*);")
}

// ---------------------------------------------------------------------------
// Forward declarations
// ---------------------------------------------------------------------------

func TestCCompressor_ForwardDecl(t *testing.T) {
	c := NewCCompressor()
	source := `struct Node;
enum Status;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 2)

	// struct Node; is caught by isCStructDecl -> handleStructDecl -> KindStruct.
	assert.Equal(t, KindStruct, output.Signatures[0].Kind)
	assert.Contains(t, output.Signatures[0].Source, "struct Node;")

	// enum Status; is caught by isCEnumDecl -> handleEnumDecl -> KindType.
	assert.Equal(t, KindType, output.Signatures[1].Kind)
	assert.Contains(t, output.Signatures[1].Source, "enum Status;")
}

// ---------------------------------------------------------------------------
// Global variable declarations
// ---------------------------------------------------------------------------

func TestCCompressor_GlobalVar(t *testing.T) {
	c := NewCCompressor()
	source := `int global_counter;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindConstant, sig.Kind)
	assert.Equal(t, "global_counter", sig.Name)
}

func TestCCompressor_StaticGlobalVar(t *testing.T) {
	c := NewCCompressor()
	source := `static const int max_size = 100;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindConstant, sig.Kind)
	assert.Equal(t, "max_size", sig.Name)
}

// ---------------------------------------------------------------------------
// Doc comments
// ---------------------------------------------------------------------------

func TestCCompressor_DocCommentOnFunc(t *testing.T) {
	c := NewCCompressor()
	source := `/**
 * Process the input data.
 * @param data The input buffer
 * @param len Length of the buffer
 * @return 0 on success
 */
int process(const char *data, size_t len) {
    return 0;
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
	assert.Contains(t, funcSig.Source, "/**")
	assert.Contains(t, funcSig.Source, "* Process the input data.")
	assert.Contains(t, funcSig.Source, "int process(const char *data, size_t len)")
}

func TestCCompressor_LineCommentOnFunc(t *testing.T) {
	c := NewCCompressor()
	source := `// Initialize the system.
// Must be called before any other function.
void init(void) {
    setup();
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
	assert.Contains(t, funcSig.Source, "// Initialize the system.")
	assert.Contains(t, funcSig.Source, "// Must be called before any other function.")
	assert.Contains(t, funcSig.Source, "void init(void)")
}

func TestCCompressor_EmptyLineClearsDoc(t *testing.T) {
	c := NewCCompressor()
	source := `// This comment is separated.

void orphan(void) {
    return;
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
	assert.NotContains(t, funcSig.Source, "This comment is separated.")
	assert.Contains(t, funcSig.Source, "void orphan(void)")
}

// ---------------------------------------------------------------------------
// Brace counting
// ---------------------------------------------------------------------------

func TestCCountBraces(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		depth int
	}{
		{"empty", "", 0},
		{"open brace", "int main() {", 1},
		{"close brace", "}", -1},
		{"balanced", "if (x) { y(); }", 0},
		{"brace in string", `char *s = "{}";`, 0},
		{"brace in char", "char c = '{';", 0},
		{"brace after line comment", "x = 1; // {", 0},
		{"brace in block comment", "x = 1; /* { */ y = 2;", 0},
		{"nested braces", "struct { int x; } s = { 0 };", 0},
		{"multiple open", "struct Foo { struct Bar {", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cCountBraces(tt.line)
			assert.Equal(t, tt.depth, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Complex/realistic tests
// ---------------------------------------------------------------------------

func TestCCompressor_CompleteHeaderFile(t *testing.T) {
	c := NewCCompressor()
	source := `#include <stdio.h>
#include <stdlib.h>
#include "config.h"

#define MAX_BUFFER 4096
#define MIN(a, b) ((a) < (b) ? (a) : (b))

struct Config {
    char *host;
    int port;
    int max_connections;
};

typedef struct Config Config;

enum LogLevel {
    LOG_DEBUG,
    LOG_INFO,
    LOG_WARN,
    LOG_ERROR
};

typedef void (*log_handler_t)(enum LogLevel, const char *);

struct Config *config_new(const char *host, int port);
void config_free(struct Config *cfg);
int config_validate(const struct Config *cfg);
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()

	// Includes.
	assert.Contains(t, rendered, "#include <stdio.h>")
	assert.Contains(t, rendered, "#include <stdlib.h>")
	assert.Contains(t, rendered, `#include "config.h"`)

	// Defines.
	assert.Contains(t, rendered, "#define MAX_BUFFER")
	assert.Contains(t, rendered, "#define MIN(a, b)")

	// Struct.
	assert.Contains(t, rendered, "struct Config")
	assert.Contains(t, rendered, "char *host;")

	// Enum.
	assert.Contains(t, rendered, "enum LogLevel")
	assert.Contains(t, rendered, "LOG_DEBUG,")

	// Function pointer typedef.
	assert.Contains(t, rendered, "typedef void (*log_handler_t)")

	// Function prototypes.
	assert.Contains(t, rendered, "config_new")
	assert.Contains(t, rendered, "config_free")
	assert.Contains(t, rendered, "config_validate")
}

func TestCCompressor_CompleteSourceFile(t *testing.T) {
	c := NewCCompressor()
	source := `#include "config.h"
#include <string.h>

static int internal_counter = 0;

/**
 * Create a new config.
 */
struct Config *config_new(const char *host, int port) {
    struct Config *cfg = malloc(sizeof(struct Config));
    if (!cfg) return NULL;
    cfg->host = strdup(host);
    cfg->port = port;
    cfg->max_connections = 100;
    internal_counter++;
    return cfg;
}

void config_free(struct Config *cfg) {
    if (cfg) {
        free(cfg->host);
        free(cfg);
    }
}

int config_validate(const struct Config *cfg) {
    if (!cfg) return -1;
    if (cfg->port <= 0 || cfg->port > 65535) return -1;
    if (!cfg->host) return -1;
    return 0;
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()

	// Includes.
	assert.Contains(t, rendered, `#include "config.h"`)
	assert.Contains(t, rendered, "#include <string.h>")

	// Global variable.
	assert.Contains(t, rendered, "internal_counter")

	// Function signatures (without bodies).
	assert.Contains(t, rendered, "config_new")
	assert.Contains(t, rendered, "config_free")
	assert.Contains(t, rendered, "config_validate")

	// Bodies should be excluded.
	assert.NotContains(t, rendered, "malloc(sizeof")
	assert.NotContains(t, rendered, "free(cfg->host)")
	assert.NotContains(t, rendered, "internal_counter++")

	// Doc comment should be attached.
	assert.Contains(t, rendered, "/**")
	assert.Contains(t, rendered, "Create a new config.")
}

func TestCCompressor_SourceOrderPreserved(t *testing.T) {
	c := NewCCompressor()
	source := `#include <stdio.h>

#define VERSION 1

struct Foo {
    int x;
};

int bar(void) {
    return 0;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 4)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)   // #include
	assert.Equal(t, KindConstant, output.Signatures[1].Kind) // #define
	assert.Equal(t, KindStruct, output.Signatures[2].Kind)   // struct
	assert.Equal(t, KindFunction, output.Signatures[3].Kind) // function
}

func TestCCompressor_CompressionRatio(t *testing.T) {
	c := NewCCompressor()
	source := `#include <stdio.h>
#include <stdlib.h>

struct Config {
    char *host;
    int port;
};

struct Config *config_new(const char *host, int port) {
    struct Config *cfg = malloc(sizeof(struct Config));
    if (!cfg) return NULL;
    cfg->host = strdup(host);
    cfg->port = port;
    return cfg;
}

void config_free(struct Config *cfg) {
    if (cfg) {
        free(cfg->host);
        free(cfg);
    }
}

int config_validate(const struct Config *cfg) {
    if (!cfg) return -1;
    if (cfg->port <= 0 || cfg->port > 65535) return -1;
    if (!cfg->host) return -1;
    return 0;
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	output.Render()
	ratio := output.CompressionRatio()
	assert.Greater(t, ratio, 0.1, "ratio should be > 0.1")
	assert.Less(t, ratio, 0.85, "ratio should be < 0.85 (at least 15%% reduction)")
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestExtractCFuncName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "int main(int argc, char **argv) {", "main"},
		{"pointer return", "char *strdup(const char *s) {", "strdup"},
		{"void func", "void process(void) {", "process"},
		{"static", "static int helper(int x) {", "helper"},
		{"const return", "const char *get_name(void) {", "get_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCFuncName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractCDefineNameLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple constant", "#define MAX 100", "#define MAX"},
		{"function-like", "#define MIN(a, b) ((a) < (b) ? (a) : (b))", "#define MIN(a, b)"},
		{"no value", "#define FEATURE_ENABLED", "#define FEATURE_ENABLED"},
		{"single param", "#define SQUARE(x) ((x) * (x))", "#define SQUARE(x)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCDefineNameLine(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestIsCStructDecl(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"simple struct", "struct Foo {", true},
		{"typedef struct", "typedef struct {", true},
		{"typedef struct named", "typedef struct Foo {", true},
		{"struct without body", "struct Foo", true},
		{"static struct", "static struct Config {", true},
		{"func with struct return", "struct Config *config_new(const char *host, int port) {", false},
		{"prototype with struct return", "struct Config *config_new(const char *host, int port);", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCStructDecl(tt.line)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractCStructName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "struct Point {", "Point"},
		{"typedef", "typedef struct {", ""},
		{"typedef named", "typedef struct node {", "node"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCStructName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractCGlobalVarName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "int counter;", "counter"},
		{"with init", "static const int max_size = 100;", "max_size"},
		{"pointer", "char *buffer;", "buffer"},
		{"array", "int data[10];", "data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCGlobalVarName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestIsCFuncDefinition(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"simple func", "int main(int argc, char **argv) {", true},
		{"void func", "void process(void) {", true},
		{"control keyword if", "if (x) {", false},
		{"control keyword while", "while (1) {", false},
		{"control keyword for", "for (int i = 0; i < n; i++) {", false},
		{"struct decl", "struct Foo {", false},
		{"enum decl", "enum Bar {", false},
		{"typedef", "typedef struct {", false},
		{"prototype no brace", "int foo(void);", false},
		{"struct return type func", "struct Config *config_new(const char *host, int port) {", true},
		{"enum return type func", "enum Color get_color(int idx) {", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCFuncDefinition(tt.line)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCFuncPrototype(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"simple prototype", "int process(const char *data);", true},
		{"variadic", "int printf(const char *fmt, ...);", true},
		{"void return", "void init(void);", true},
		{"not prototype - no semi", "int foo(void) {", false},
		{"typedef", "typedef void (*handler)(int);", false},
		{"include", "#include <stdio.h>", false},
		{"control kw", "return(0);", false},
		{"struct return type prototype", "struct Config *config_new(const char *host, int port);", true},
		{"enum return type prototype", "enum Color get_default_color(void);", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCFuncPrototype(tt.line)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Golden tests
// ---------------------------------------------------------------------------

func TestCCompressor_GoldenTests(t *testing.T) {
	compressor := NewCCompressor()
	ctx := context.Background()

	tests := []struct {
		name     string
		fixture  string
		expected string
	}{
		{
			name:     "header file with prototypes and structs",
			fixture:  "c/header_file.h",
			expected: "c/header_file.h.expected",
		},
		{
			name:     "source file with function definitions",
			fixture:  "c/source_file.c",
			expected: "c/source_file.c.expected",
		},
		{
			name:     "typedefs and enums",
			fixture:  "c/typedefs_enums.c",
			expected: "c/typedefs_enums.c.expected",
		},
		{
			name:     "complete realistic file",
			fixture:  "c/complete_file.c",
			expected: "c/complete_file.c.expected",
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

func BenchmarkCCompressor(b *testing.B) {
	c := NewCCompressor()
	source := []byte(`#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define MAX_BUFFER 4096
#define MIN(a, b) ((a) < (b) ? (a) : (b))

struct Config {
    char *host;
    int port;
    int max_connections;
};

typedef void (*log_handler_t)(enum LogLevel, const char *);

struct Config *config_new(const char *host, int port) {
    struct Config *cfg = malloc(sizeof(struct Config));
    cfg->host = strdup(host);
    cfg->port = port;
    return cfg;
}

void config_free(struct Config *cfg) {
    if (cfg) {
        free(cfg->host);
        free(cfg);
    }
}

int config_validate(const struct Config *cfg) {
    if (!cfg) return -1;
    if (cfg->port <= 0) return -1;
    return 0;
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
