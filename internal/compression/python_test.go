package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// PythonCompressor metadata tests
// ---------------------------------------------------------------------------

func TestPythonCompressor_Language(t *testing.T) {
	c := NewPythonCompressor()
	assert.Equal(t, "python", c.Language())
}

func TestPythonCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewPythonCompressor()
	types := c.SupportedNodeTypes()
	assert.Contains(t, types, "import_statement")
	assert.Contains(t, types, "import_from_statement")
	assert.Contains(t, types, "function_definition")
	assert.Contains(t, types, "class_definition")
	assert.Contains(t, types, "decorated_definition")
}

func TestPythonCompressor_EmptyInput(t *testing.T) {
	c := NewPythonCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "python", output.Language)
}

func TestPythonCompressor_ContextCancellation(t *testing.T) {
	c := NewPythonCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Generate a large source to ensure cancellation is checked.
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("# line\n")
	}

	_, err := c.Compress(ctx, []byte(b.String()))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Import statements
// ---------------------------------------------------------------------------

func TestPythonCompressor_SimpleImport(t *testing.T) {
	c := NewPythonCompressor()
	source := `import os`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindImport, sig.Kind)
	assert.Equal(t, "import os", sig.Source)
}

func TestPythonCompressor_FromImport(t *testing.T) {
	c := NewPythonCompressor()
	source := `from typing import Optional, Protocol`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindImport, sig.Kind)
	assert.Contains(t, sig.Source, "from typing import Optional, Protocol")
}

func TestPythonCompressor_MultipleImports(t *testing.T) {
	c := NewPythonCompressor()
	source := `import os
import sys
from pathlib import Path
from typing import Optional, List
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	importCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindImport {
			importCount++
		}
	}
	assert.Equal(t, 4, importCount)
}

// ---------------------------------------------------------------------------
// Function definitions
// ---------------------------------------------------------------------------

func TestPythonCompressor_SimpleFunction(t *testing.T) {
	c := NewPythonCompressor()
	source := `def greet(name: str) -> str:
    return f"Hello, {name}"
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
	assert.Equal(t, "greet", funcSig.Name)
	assert.Contains(t, funcSig.Source, "def greet(name: str) -> str:")
	assert.NotContains(t, funcSig.Source, "return")
}

func TestPythonCompressor_AsyncFunction(t *testing.T) {
	c := NewPythonCompressor()
	source := `async def fetch_data(url: str, timeout: float = 30.0) -> dict:
    async with aiohttp.ClientSession() as session:
        response = await session.get(url, timeout=timeout)
        return await response.json()
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
	assert.Equal(t, "fetch_data", funcSig.Name)
	assert.Contains(t, funcSig.Source, "async def fetch_data(url: str, timeout: float = 30.0) -> dict:")
	assert.NotContains(t, funcSig.Source, "aiohttp")
}

func TestPythonCompressor_FunctionWithDocstring(t *testing.T) {
	c := NewPythonCompressor()
	source := `def process(data: list) -> None:
    """Process the given data.

    Args:
        data: The data to process.
    """
    for item in data:
        handle(item)
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
	assert.Equal(t, "process", funcSig.Name)
	assert.Contains(t, funcSig.Source, "def process(data: list) -> None:")
	assert.Contains(t, funcSig.Source, `"""Process the given data.`)
	assert.NotContains(t, funcSig.Source, "for item")
}

func TestPythonCompressor_FunctionWithSingleLineDocstring(t *testing.T) {
	c := NewPythonCompressor()
	source := `def add(a: int, b: int) -> int:
    """Add two numbers."""
    return a + b
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
	assert.Contains(t, funcSig.Source, `"""Add two numbers."""`)
	assert.NotContains(t, funcSig.Source, "return a + b")
}

func TestPythonCompressor_FunctionArgsKwargs(t *testing.T) {
	c := NewPythonCompressor()
	source := `def flexible(*args, **kwargs) -> None:
    pass
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
	assert.Contains(t, funcSig.Source, "*args")
	assert.Contains(t, funcSig.Source, "**kwargs")
}

func TestPythonCompressor_MultiLineFunctionSignature(t *testing.T) {
	c := NewPythonCompressor()
	source := `def complex_function(
    name: str,
    age: int,
    email: Optional[str] = None,
    *args,
    **kwargs,
) -> dict:
    result = {"name": name}
    return result
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
	assert.Equal(t, "complex_function", funcSig.Name)
	assert.Contains(t, funcSig.Source, "name: str")
	assert.Contains(t, funcSig.Source, "age: int")
	assert.Contains(t, funcSig.Source, "Optional[str] = None")
	assert.Contains(t, funcSig.Source, "-> dict:")
	assert.NotContains(t, funcSig.Source, "result =")
}

// ---------------------------------------------------------------------------
// Decorated functions
// ---------------------------------------------------------------------------

func TestPythonCompressor_DecoratedFunction(t *testing.T) {
	c := NewPythonCompressor()
	source := `@app.route("/api/data")
def get_data():
    return jsonify(data)
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
	assert.Contains(t, funcSig.Source, `@app.route("/api/data")`)
	assert.Contains(t, funcSig.Source, "def get_data():")
	assert.NotContains(t, funcSig.Source, "jsonify")
}

func TestPythonCompressor_MultipleDecorators(t *testing.T) {
	c := NewPythonCompressor()
	source := `@login_required
@cache_response(timeout=300)
async def protected_view(request: Request) -> Response:
    return Response(data)
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
	assert.Contains(t, funcSig.Source, "@login_required")
	assert.Contains(t, funcSig.Source, "@cache_response(timeout=300)")
	assert.Contains(t, funcSig.Source, "async def protected_view")
}

// ---------------------------------------------------------------------------
// Class definitions
// ---------------------------------------------------------------------------

func TestPythonCompressor_SimpleClass(t *testing.T) {
	c := NewPythonCompressor()
	source := `class MyClass:
    def method(self) -> None:
        pass
`
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
	assert.Equal(t, "MyClass", classSig.Name)
	assert.Contains(t, classSig.Source, "class MyClass:")
	assert.Contains(t, classSig.Source, "def method(self) -> None:")
	assert.NotContains(t, classSig.Source, "pass")
}

func TestPythonCompressor_ClassWithBases(t *testing.T) {
	c := NewPythonCompressor()
	source := `class Child(Parent, Mixin):
    def __init__(self, name: str) -> None:
        super().__init__()
        self.name = name
`
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
	assert.Contains(t, classSig.Source, "class Child(Parent, Mixin):")
	assert.Contains(t, classSig.Source, "def __init__(self, name: str) -> None:")
	assert.NotContains(t, classSig.Source, "super()")
}

func TestPythonCompressor_ClassWithDocstring(t *testing.T) {
	c := NewPythonCompressor()
	source := `class Server:
    """HTTP server implementation."""

    def start(self) -> None:
        self.running = True
`
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
	assert.Contains(t, classSig.Source, `"""HTTP server implementation."""`)
	assert.Contains(t, classSig.Source, "def start(self) -> None:")
}

func TestPythonCompressor_DataclassWithFields(t *testing.T) {
	c := NewPythonCompressor()
	source := `@dataclass
class Config:
    """Application configuration."""
    host: str = "localhost"
    port: int = 8080
    debug: bool = False

    def validate(self) -> bool:
        """Validate the configuration."""
        if self.port < 0:
            return False
        return True
`
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
	assert.Equal(t, "Config", classSig.Name)
	assert.Contains(t, classSig.Source, "@dataclass")
	assert.Contains(t, classSig.Source, `"""Application configuration."""`)
	assert.Contains(t, classSig.Source, `host: str = "localhost"`)
	assert.Contains(t, classSig.Source, "port: int = 8080")
	assert.Contains(t, classSig.Source, "debug: bool = False")
	assert.Contains(t, classSig.Source, "def validate(self) -> bool:")
	assert.Contains(t, classSig.Source, `"""Validate the configuration."""`)
	assert.NotContains(t, classSig.Source, "if self.port < 0")
	assert.NotContains(t, classSig.Source, "return False")
	assert.NotContains(t, classSig.Source, "return True")
}

func TestPythonCompressor_ClassWithMethodDecorators(t *testing.T) {
	c := NewPythonCompressor()
	source := `class MyClass:
    @property
    def name(self) -> str:
        return self._name

    @staticmethod
    def create() -> "MyClass":
        return MyClass()

    @classmethod
    def from_dict(cls, data: dict) -> "MyClass":
        return cls(**data)
`
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
	assert.Contains(t, classSig.Source, "@property")
	assert.Contains(t, classSig.Source, "def name(self) -> str:")
	assert.Contains(t, classSig.Source, "@staticmethod")
	assert.Contains(t, classSig.Source, `def create() -> "MyClass":`)
	assert.Contains(t, classSig.Source, "@classmethod")
	assert.Contains(t, classSig.Source, `def from_dict(cls, data: dict) -> "MyClass":`)
	assert.NotContains(t, classSig.Source, "return self._name")
	assert.NotContains(t, classSig.Source, "return MyClass()")
}

// ---------------------------------------------------------------------------
// Module-level docstring
// ---------------------------------------------------------------------------

func TestPythonCompressor_ModuleDocstring(t *testing.T) {
	c := NewPythonCompressor()
	source := `"""Module docstring."""

import os
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(output.Signatures), 1)

	assert.Equal(t, KindDocComment, output.Signatures[0].Kind)
	assert.Contains(t, output.Signatures[0].Source, `"""Module docstring."""`)
}

func TestPythonCompressor_MultiLineModuleDocstring(t *testing.T) {
	c := NewPythonCompressor()
	source := `"""
Module docstring that
spans multiple lines.
"""

import os
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(output.Signatures), 1)

	assert.Equal(t, KindDocComment, output.Signatures[0].Kind)
	assert.Contains(t, output.Signatures[0].Source, "Module docstring that")
	assert.Contains(t, output.Signatures[0].Source, "spans multiple lines.")
}

// ---------------------------------------------------------------------------
// Top-level constants
// ---------------------------------------------------------------------------

func TestPythonCompressor_TypeAnnotatedConstant(t *testing.T) {
	c := NewPythonCompressor()
	source := `MAX_RETRIES: int = 3
DEFAULT_TIMEOUT: float = 30.0
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	constCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindConstant {
			constCount++
		}
	}
	assert.Equal(t, 2, constCount)

	assert.Equal(t, "MAX_RETRIES", output.Signatures[0].Name)
	assert.Contains(t, output.Signatures[0].Source, "MAX_RETRIES: int = 3")
}

// ---------------------------------------------------------------------------
// __all__ export list
// ---------------------------------------------------------------------------

func TestPythonCompressor_AllExport(t *testing.T) {
	c := NewPythonCompressor()
	source := `__all__ = ["Config", "Server", "run"]
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	sig := output.Signatures[0]
	assert.Equal(t, KindExport, sig.Kind)
	assert.Contains(t, sig.Source, `__all__ = ["Config", "Server", "run"]`)
}

// ---------------------------------------------------------------------------
// Protocol classes
// ---------------------------------------------------------------------------

func TestPythonCompressor_ProtocolClass(t *testing.T) {
	c := NewPythonCompressor()
	source := `from typing import Protocol

class Renderable(Protocol):
    """Something that can be rendered."""

    def render(self) -> str:
        ...
`
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
	assert.Equal(t, "Renderable", classSig.Name)
	assert.Contains(t, classSig.Source, "class Renderable(Protocol):")
	assert.Contains(t, classSig.Source, `"""Something that can be rendered."""`)
	assert.Contains(t, classSig.Source, "def render(self) -> str:")
}

// ---------------------------------------------------------------------------
// Source order preservation
// ---------------------------------------------------------------------------

func TestPythonCompressor_SourceOrderPreserved(t *testing.T) {
	c := NewPythonCompressor()
	source := `"""Module doc."""

import os
from typing import Optional

MAX_SIZE: int = 100

class Config:
    host: str = "localhost"

def run() -> None:
    pass
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(output.Signatures), 5)

	assert.Equal(t, KindDocComment, output.Signatures[0].Kind) // module doc
	assert.Equal(t, KindImport, output.Signatures[1].Kind)     // import os
	assert.Equal(t, KindImport, output.Signatures[2].Kind)     // from typing
	assert.Equal(t, KindConstant, output.Signatures[3].Kind)   // MAX_SIZE
	assert.Equal(t, KindClass, output.Signatures[4].Kind)      // Config
	assert.Equal(t, KindFunction, output.Signatures[5].Kind)   // run
}

// ---------------------------------------------------------------------------
// Compression ratio
// ---------------------------------------------------------------------------

func TestPythonCompressor_CompressionRatio(t *testing.T) {
	c := NewPythonCompressor()
	source := `"""Example module."""

import os
from typing import Optional, List

MAX_RETRIES: int = 3

class Config:
    """Application configuration."""
    host: str = "localhost"
    port: int = 8080

    def validate(self) -> bool:
        """Validate the configuration."""
        if self.port < 0:
            return False
        if not self.host:
            return False
        return True

    def to_dict(self) -> dict:
        return {"host": self.host, "port": self.port}

async def fetch_data(url: str, timeout: float = 30.0) -> dict:
    """Fetch data from URL."""
    async with aiohttp.ClientSession() as session:
        response = await session.get(url, timeout=timeout)
        data = await response.json()
        if response.status != 200:
            raise ValueError(f"Bad status: {response.status}")
        return data

def process_results(results: List[dict]) -> None:
    """Process a list of results."""
    for result in results:
        if result.get("status") == "error":
            log_error(result)
            continue
        handle_result(result)
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	output.Render()
	ratio := output.CompressionRatio()
	// Expect meaningful compression -- function/method bodies stripped.
	assert.Greater(t, ratio, 0.1, "ratio should be > 0.1")
	assert.Less(t, ratio, 0.85, "ratio should be < 0.85 (at least 15%% reduction)")
}

// ---------------------------------------------------------------------------
// Complex real-world patterns
// ---------------------------------------------------------------------------

func TestPythonCompressor_CompleteFile(t *testing.T) {
	c := NewPythonCompressor()
	source := `"""Module docstring."""

import os
from typing import Optional, Protocol

MAX_RETRIES: int = 3

@dataclass
class Config:
    """Application configuration."""
    host: str = "localhost"
    port: int = 8080

    def validate(self) -> bool:
        """Validate the configuration."""
        if self.port < 0:
            return False
        return True

async def fetch_data(url: str, timeout: float = 30.0) -> dict:
    """Fetch data from URL."""
    async with aiohttp.ClientSession() as session:
        response = await session.get(url, timeout=timeout)
        return await response.json()
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()

	// Should contain module docstring.
	assert.Contains(t, rendered, `"""Module docstring."""`)

	// Should contain imports.
	assert.Contains(t, rendered, "import os")
	assert.Contains(t, rendered, "from typing import Optional, Protocol")

	// Should contain constant.
	assert.Contains(t, rendered, "MAX_RETRIES: int = 3")

	// Should contain class with decorator and docstring.
	assert.Contains(t, rendered, "@dataclass")
	assert.Contains(t, rendered, "class Config:")
	assert.Contains(t, rendered, `"""Application configuration."""`)
	assert.Contains(t, rendered, `host: str = "localhost"`)
	assert.Contains(t, rendered, "port: int = 8080")

	// Should contain method signature with docstring.
	assert.Contains(t, rendered, "def validate(self) -> bool:")
	assert.Contains(t, rendered, `"""Validate the configuration."""`)

	// Should NOT contain method body.
	assert.NotContains(t, rendered, "if self.port < 0")
	assert.NotContains(t, rendered, "return False")
	assert.NotContains(t, rendered, "return True")

	// Should contain async function with docstring.
	assert.Contains(t, rendered, "async def fetch_data(url: str, timeout: float = 30.0) -> dict:")
	assert.Contains(t, rendered, `"""Fetch data from URL."""`)

	// Should NOT contain function body.
	assert.NotContains(t, rendered, "aiohttp.ClientSession")
	assert.NotContains(t, rendered, "await session.get")
}

func TestPythonCompressor_MultipleFunctions(t *testing.T) {
	c := NewPythonCompressor()
	source := `def first() -> None:
    pass

def second() -> None:
    pass

def third() -> None:
    pass
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	funcCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindFunction {
			funcCount++
		}
	}
	assert.Equal(t, 3, funcCount, "expected 3 function signatures")
}

func TestPythonCompressor_ClassWithMultipleMethods(t *testing.T) {
	c := NewPythonCompressor()
	source := `class Calculator:
    def add(self, a: int, b: int) -> int:
        return a + b

    def subtract(self, a: int, b: int) -> int:
        return a - b

    def multiply(self, a: int, b: int) -> int:
        return a * b
`
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
	assert.Contains(t, classSig.Source, "def add(self, a: int, b: int) -> int:")
	assert.Contains(t, classSig.Source, "def subtract(self, a: int, b: int) -> int:")
	assert.Contains(t, classSig.Source, "def multiply(self, a: int, b: int) -> int:")
	assert.NotContains(t, classSig.Source, "return a + b")
	assert.NotContains(t, classSig.Source, "return a - b")
	assert.NotContains(t, classSig.Source, "return a * b")
}

func TestPythonCompressor_MethodWithDocstring(t *testing.T) {
	c := NewPythonCompressor()
	source := `class MyClass:
    def method(self) -> str:
        """Return a string."""
        return "hello"
`
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
	assert.Contains(t, classSig.Source, "def method(self) -> str:")
	assert.Contains(t, classSig.Source, `"""Return a string."""`)
	assert.NotContains(t, classSig.Source, `return "hello"`)
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestPythonCompressor_FunctionDefaultValues(t *testing.T) {
	c := NewPythonCompressor()
	source := `def connect(host: str = "localhost", port: int = 5432, ssl: bool = True) -> None:
    do_connect()
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
	assert.Contains(t, funcSig.Source, `host: str = "localhost"`)
	assert.Contains(t, funcSig.Source, "port: int = 5432")
	assert.Contains(t, funcSig.Source, "ssl: bool = True")
}

func TestPythonCompressor_ClassFollowedByFunction(t *testing.T) {
	c := NewPythonCompressor()
	source := `class Foo:
    x: int = 1

def bar() -> None:
    pass
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var classSig *Signature
	var funcSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass && classSig == nil {
			classSig = &output.Signatures[i]
		}
		if output.Signatures[i].Kind == KindFunction && funcSig == nil {
			funcSig = &output.Signatures[i]
		}
	}
	require.NotNil(t, classSig, "expected class signature")
	require.NotNil(t, funcSig, "expected function signature")
	assert.Equal(t, "Foo", classSig.Name)
	assert.Equal(t, "bar", funcSig.Name)
}

func TestPythonCompressor_FunctionNoTypeHints(t *testing.T) {
	c := NewPythonCompressor()
	source := `def legacy_function(x, y, z=None):
    return x + y
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
	assert.Contains(t, funcSig.Source, "def legacy_function(x, y, z=None):")
}

func TestPythonCompressor_SingleQuoteDocstring(t *testing.T) {
	c := NewPythonCompressor()
	source := `def foo():
    '''Single-quote docstring.'''
    pass
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
	assert.Contains(t, funcSig.Source, "'''Single-quote docstring.'''")
}

// ---------------------------------------------------------------------------
// Helper function unit tests
// ---------------------------------------------------------------------------

func TestPyExtractFuncName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple def", input: "def foo():", expected: "foo"},
		{name: "async def", input: "async def bar():", expected: "bar"},
		{name: "with params", input: "def baz(x: int, y: str) -> bool:", expected: "baz"},
		{name: "dunder", input: "def __init__(self):", expected: "__init__"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pyExtractFuncName(tt.input))
		})
	}
}

func TestPyExtractClassName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple", input: "class Foo:", expected: "Foo"},
		{name: "with bases", input: "class Bar(Base, Mixin):", expected: "Bar"},
		{name: "protocol", input: "class Renderable(Protocol):", expected: "Renderable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pyExtractClassName(tt.input))
		})
	}
}

func TestPyIsTypeAnnotatedAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "with value", input: "MAX_SIZE: int = 100", expected: true},
		{name: "annotation only", input: "name: str", expected: true},
		{name: "no annotation", input: "x = 5", expected: false},
		{name: "function call", input: "x = foo()", expected: false},
		{name: "empty", input: "", expected: false},
		{name: "complex type", input: "items: List[str] = []", expected: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pyIsTypeAnnotatedAssignment(tt.input))
		})
	}
}

func TestPyCountParens(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		depth int
	}{
		{name: "empty", line: "", depth: 0},
		{name: "open paren", line: "def foo(", depth: 1},
		{name: "balanced", line: "def foo():", depth: 0},
		{name: "in string", line: `x = "("`, depth: 0},
		{name: "after comment", line: "x = 1 # (", depth: 0},
		{name: "nested", line: "def foo(bar(baz)):", depth: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.depth, pyCountParens(tt.line))
		})
	}
}

func TestPyLineIndent(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{name: "no indent", line: "foo", expected: 0},
		{name: "4 spaces", line: "    foo", expected: 4},
		{name: "8 spaces", line: "        foo", expected: 8},
		{name: "tab", line: "\tfoo", expected: 4},
		{name: "empty", line: "", expected: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pyLineIndent(tt.line))
		})
	}
}

// ---------------------------------------------------------------------------
// Golden tests using fixture files
// ---------------------------------------------------------------------------

func TestPythonCompressor_GoldenTests(t *testing.T) {
	compressor := NewPythonCompressor()
	ctx := context.Background()

	tests := []struct {
		name     string
		fixture  string
		expected string
	}{
		{
			name:     "django model",
			fixture:  "python/django_model.py",
			expected: "python/django_model.py.expected",
		},
		{
			name:     "fastapi router",
			fixture:  "python/fastapi_router.py",
			expected: "python/fastapi_router.py.expected",
		},
		{
			name:     "dataclass types",
			fixture:  "python/dataclass_types.py",
			expected: "python/dataclass_types.py.expected",
		},
		{
			name:     "protocol types",
			fixture:  "python/protocol_types.py",
			expected: "python/protocol_types.py.expected",
		},
		{
			name:     "decorators",
			fixture:  "python/decorators.py",
			expected: "python/decorators.py.expected",
		},
		{
			name:     "docstrings",
			fixture:  "python/docstrings.py",
			expected: "python/docstrings.py.expected",
		},
		{
			name:     "complete file",
			fixture:  "python/complete_file.py",
			expected: "python/complete_file.py.expected",
		},
		{
			name:     "async functions",
			fixture:  "python/async_functions.py",
			expected: "python/async_functions.py.expected",
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

func BenchmarkPythonCompressor(b *testing.B) {
	c := NewPythonCompressor()
	source := []byte(`"""Example module."""

import os
from typing import Optional, List

MAX_RETRIES: int = 3

@dataclass
class Config:
    """Application configuration."""
    host: str = "localhost"
    port: int = 8080

    def validate(self) -> bool:
        """Validate the configuration."""
        if self.port < 0:
            return False
        return True

    def to_dict(self) -> dict:
        return {"host": self.host, "port": self.port}

async def fetch_data(url: str, timeout: float = 30.0) -> dict:
    """Fetch data from URL."""
    async with aiohttp.ClientSession() as session:
        response = await session.get(url, timeout=timeout)
        return await response.json()

def process(data: List[dict]) -> None:
    for item in data:
        handle(item)
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
