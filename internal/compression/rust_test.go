package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// RustCompressor metadata tests
// ---------------------------------------------------------------------------

func TestRustCompressor_Language(t *testing.T) {
	c := NewRustCompressor()
	assert.Equal(t, "rust", c.Language())
}

func TestRustCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewRustCompressor()
	types := c.SupportedNodeTypes()
	assert.Contains(t, types, "use_declaration")
	assert.Contains(t, types, "function_item")
	assert.Contains(t, types, "struct_item")
	assert.Contains(t, types, "enum_item")
	assert.Contains(t, types, "trait_item")
	assert.Contains(t, types, "impl_item")
	assert.Contains(t, types, "type_alias")
	assert.Contains(t, types, "const_item")
	assert.Contains(t, types, "static_item")
	assert.Contains(t, types, "mod_item")
	assert.Contains(t, types, "macro_definition")
	assert.Contains(t, types, "extern_block")
}

func TestRustCompressor_EmptyInput(t *testing.T) {
	c := NewRustCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "rust", output.Language)
}

func TestRustCompressor_ContextCancellation(t *testing.T) {
	c := NewRustCompressor()
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
// Use declarations
// ---------------------------------------------------------------------------

func TestRustCompressor_UseDeclaration(t *testing.T) {
	c := NewRustCompressor()
	source := `use std::collections::HashMap;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindImport, sig.Kind)
	assert.Contains(t, sig.Source, "use std::collections::HashMap;")
}

func TestRustCompressor_PubUseDeclaration(t *testing.T) {
	c := NewRustCompressor()
	source := `pub use crate::config::Config;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)
}

func TestRustCompressor_MultipleUseDeclarations(t *testing.T) {
	c := NewRustCompressor()
	source := `use std::collections::HashMap;
use crate::config::Config;
use std::io::{self, Read, Write};`
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
// Function declarations
// ---------------------------------------------------------------------------

func TestRustCompressor_SimpleFn(t *testing.T) {
	c := NewRustCompressor()
	source := `fn main() {
    println!("hello");
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
	assert.Contains(t, funcSig.Source, "fn main()")
	assert.NotContains(t, funcSig.Source, "println!")
}

func TestRustCompressor_PubFn(t *testing.T) {
	c := NewRustCompressor()
	source := `pub fn process(data: &[u8]) -> Result<(), Error> {
    validate(data)?;
    Ok(())
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
	assert.Contains(t, funcSig.Source, "pub fn process(data: &[u8]) -> Result<(), Error>")
	assert.NotContains(t, funcSig.Source, "validate")
}

func TestRustCompressor_AsyncFn(t *testing.T) {
	c := NewRustCompressor()
	source := `pub async fn fetch(url: &str) -> Result<Response, Error> {
    let resp = client.get(url).await?;
    Ok(resp)
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
	assert.Equal(t, "fetch", funcSig.Name)
	assert.Contains(t, funcSig.Source, "pub async fn fetch")
	assert.NotContains(t, funcSig.Source, "client.get")
}

func TestRustCompressor_UnsafeFn(t *testing.T) {
	c := NewRustCompressor()
	source := `pub unsafe fn deref_raw(ptr: *const u8) -> u8 {
    *ptr
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
	assert.Contains(t, funcSig.Source, "pub unsafe fn deref_raw")
}

func TestRustCompressor_GenericFn(t *testing.T) {
	c := NewRustCompressor()
	source := `pub fn map<T, U>(items: Vec<T>, f: fn(T) -> U) -> Vec<U> {
    items.into_iter().map(f).collect()
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
	assert.Equal(t, "map", funcSig.Name)
	assert.Contains(t, funcSig.Source, "pub fn map<T, U>(items: Vec<T>, f: fn(T) -> U) -> Vec<U>")
}

func TestRustCompressor_FnWithWhereClause(t *testing.T) {
	c := NewRustCompressor()
	source := `pub fn print_all<T>(items: &[T]) where T: Display + Debug {
    for item in items {
        println!("{:?}", item);
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
	assert.Contains(t, funcSig.Source, "where T: Display + Debug")
}

func TestRustCompressor_FnWithLifetimes(t *testing.T) {
	c := NewRustCompressor()
	source := `pub fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() { x } else { y }
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
	assert.Contains(t, funcSig.Source, "fn longest<'a>(x: &'a str, y: &'a str) -> &'a str")
}

func TestRustCompressor_PubCrateFn(t *testing.T) {
	c := NewRustCompressor()
	source := `pub(crate) fn internal_helper() -> u32 {
    42
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
	assert.Equal(t, "internal_helper", funcSig.Name)
	assert.Contains(t, funcSig.Source, "pub(crate) fn internal_helper()")
}

func TestRustCompressor_ConstFn(t *testing.T) {
	c := NewRustCompressor()
	source := `pub const fn max_value() -> u32 {
    u32::MAX
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
	assert.Equal(t, "max_value", funcSig.Name)
	assert.Contains(t, funcSig.Source, "pub const fn max_value()")
}

// ---------------------------------------------------------------------------
// Struct declarations
// ---------------------------------------------------------------------------

func TestRustCompressor_StructWithFields(t *testing.T) {
	c := NewRustCompressor()
	source := `pub struct Config {
    host: String,
    port: u16,
}`
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
	assert.Contains(t, structSig.Source, "host: String")
	assert.Contains(t, structSig.Source, "port: u16")
}

func TestRustCompressor_StructWithDeriveAndDoc(t *testing.T) {
	c := NewRustCompressor()
	source := `/// A thread-safe connection pool.
#[derive(Debug, Clone)]
pub struct Pool<T: Connection> {
    connections: Vec<T>,
    max_size: usize,
}`
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
	assert.Equal(t, "Pool", structSig.Name)
	assert.Contains(t, structSig.Source, "/// A thread-safe connection pool.")
	assert.Contains(t, structSig.Source, "#[derive(Debug, Clone)]")
	assert.Contains(t, structSig.Source, "connections: Vec<T>")
}

func TestRustCompressor_TupleStruct(t *testing.T) {
	c := NewRustCompressor()
	source := `pub struct Wrapper(pub String);`
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
	assert.Equal(t, "Wrapper", structSig.Name)
	assert.Contains(t, structSig.Source, "pub struct Wrapper(pub String);")
}

func TestRustCompressor_UnitStruct(t *testing.T) {
	c := NewRustCompressor()
	source := `pub struct Marker;`
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
	assert.Equal(t, "Marker", structSig.Name)
	assert.Contains(t, structSig.Source, "pub struct Marker;")
}

func TestRustCompressor_GenericStruct(t *testing.T) {
	c := NewRustCompressor()
	source := `pub struct Cache<K: Hash + Eq, V> {
    data: HashMap<K, V>,
    capacity: usize,
}`
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
	assert.Equal(t, "Cache", structSig.Name)
	assert.Contains(t, structSig.Source, "Cache<K: Hash + Eq, V>")
}

// ---------------------------------------------------------------------------
// Enum declarations
// ---------------------------------------------------------------------------

func TestRustCompressor_SimpleEnum(t *testing.T) {
	c := NewRustCompressor()
	source := `pub enum Status {
    Active,
    Idle(Duration),
    Error { code: u32, message: String },
}`
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
	assert.Equal(t, "Status", enumSig.Name)
	assert.Contains(t, enumSig.Source, "Active,")
	assert.Contains(t, enumSig.Source, "Idle(Duration),")
	assert.Contains(t, enumSig.Source, "Error { code: u32, message: String },")
}

func TestRustCompressor_EnumWithDerive(t *testing.T) {
	c := NewRustCompressor()
	source := `/// Represents possible errors.
#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    #[error("Parse error: {msg}")]
    Parse { msg: String },
}`
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
	assert.Contains(t, enumSig.Source, "/// Represents possible errors.")
	assert.Contains(t, enumSig.Source, "#[derive(Debug, thiserror::Error)]")
}

// ---------------------------------------------------------------------------
// Trait declarations
// ---------------------------------------------------------------------------

func TestRustCompressor_TraitDeclaration(t *testing.T) {
	c := NewRustCompressor()
	source := `pub trait Connection: Send + Sync {
    type Error;
    fn connect(addr: &str) -> Result<Self, Self::Error> where Self: Sized;
    fn is_alive(&self) -> bool;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var traitSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			traitSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, traitSig, "expected a trait signature")
	assert.Equal(t, "Connection", traitSig.Name)
	assert.Contains(t, traitSig.Source, "type Error;")
	assert.Contains(t, traitSig.Source, "fn connect(addr: &str)")
	assert.Contains(t, traitSig.Source, "fn is_alive(&self) -> bool;")
}

func TestRustCompressor_TraitWithDefaultMethod(t *testing.T) {
	c := NewRustCompressor()
	source := `pub trait Logger {
    fn log(&self, msg: &str);
    fn debug(&self, msg: &str) {
        self.log(&format!("DEBUG: {}", msg));
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var traitSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			traitSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, traitSig)
	assert.Equal(t, "Logger", traitSig.Name)
	assert.Contains(t, traitSig.Source, "fn log(&self, msg: &str);")
	assert.Contains(t, traitSig.Source, "fn debug(&self, msg: &str)")
	// Should NOT contain the default implementation body content.
	assert.NotContains(t, traitSig.Source, "format!")
}

func TestRustCompressor_TraitWithSupertraits(t *testing.T) {
	c := NewRustCompressor()
	source := `pub trait Handler: Send + Sync + 'static {
    fn handle(&self, req: Request) -> Response;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var traitSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindInterface {
			traitSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, traitSig)
	assert.Contains(t, traitSig.Source, "Handler: Send + Sync + 'static")
}

// ---------------------------------------------------------------------------
// Impl blocks
// ---------------------------------------------------------------------------

func TestRustCompressor_ImplBlock(t *testing.T) {
	c := NewRustCompressor()
	source := `impl Config {
    pub fn new() -> Self {
        Config { host: String::new(), port: 8080 }
    }

    pub fn with_host(mut self, host: &str) -> Self {
        self.host = host.to_string();
        self
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var implSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			implSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, implSig, "expected an impl signature")
	assert.Equal(t, "Config", implSig.Name)
	assert.Contains(t, implSig.Source, "pub fn new() -> Self")
	assert.Contains(t, implSig.Source, "pub fn with_host(mut self, host: &str) -> Self")
	// Bodies should be excluded.
	assert.NotContains(t, implSig.Source, "Config { host:")
	assert.NotContains(t, implSig.Source, "host.to_string()")
}

func TestRustCompressor_ImplTraitForType(t *testing.T) {
	c := NewRustCompressor()
	source := `impl Display for Config {
    fn fmt(&self, f: &mut Formatter) -> fmt::Result {
        write!(f, "{}:{}", self.host, self.port)
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var implSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			implSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, implSig)
	assert.Equal(t, "Display for Config", implSig.Name)
	assert.Contains(t, implSig.Source, "fn fmt(&self, f: &mut Formatter) -> fmt::Result")
	assert.NotContains(t, implSig.Source, "write!")
}

func TestRustCompressor_GenericImpl(t *testing.T) {
	c := NewRustCompressor()
	source := `impl<T: Connection> Pool<T> {
    /// Create a new pool with the given capacity.
    pub fn new(max_size: usize) -> Self {
        Pool {
            connections: Vec::with_capacity(max_size),
            max_size,
        }
    }

    pub fn acquire(&self) -> Option<&T> {
        self.connections.first()
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var implSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			implSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, implSig)
	assert.Contains(t, implSig.Source, "impl<T: Connection> Pool<T>")
	assert.Contains(t, implSig.Source, "/// Create a new pool with the given capacity.")
	assert.Contains(t, implSig.Source, "pub fn new(max_size: usize) -> Self")
	assert.Contains(t, implSig.Source, "pub fn acquire(&self) -> Option<&T>")
}

func TestRustCompressor_ImplDocCommentOnMethods(t *testing.T) {
	c := NewRustCompressor()
	source := `impl Server {
    /// Start the server.
    pub fn start(&self) {
        self.listen();
    }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var implSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindClass {
			implSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, implSig)
	assert.Contains(t, implSig.Source, "/// Start the server.")
	assert.Contains(t, implSig.Source, "pub fn start(&self)")
}

// ---------------------------------------------------------------------------
// Type aliases
// ---------------------------------------------------------------------------

func TestRustCompressor_TypeAlias(t *testing.T) {
	c := NewRustCompressor()
	source := `pub type Result<T> = std::result::Result<T, Error>;`
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
	assert.Equal(t, "Result", typeSig.Name)
	assert.Contains(t, typeSig.Source, "pub type Result<T> = std::result::Result<T, Error>;")
}

// ---------------------------------------------------------------------------
// Const and static declarations
// ---------------------------------------------------------------------------

func TestRustCompressor_ConstItem(t *testing.T) {
	c := NewRustCompressor()
	source := `const MAX_CONNECTIONS: usize = 100;`
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
	assert.Equal(t, "MAX_CONNECTIONS", constSig.Name)
	assert.Contains(t, constSig.Source, "const MAX_CONNECTIONS: usize = 100;")
}

func TestRustCompressor_StaticItem(t *testing.T) {
	c := NewRustCompressor()
	source := `static GLOBAL_COUNT: AtomicUsize = AtomicUsize::new(0);`
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
	assert.Equal(t, "GLOBAL_COUNT", constSig.Name)
}

func TestRustCompressor_StaticMut(t *testing.T) {
	c := NewRustCompressor()
	source := `static mut BUFFER: [u8; 1024] = [0; 1024];`
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
	assert.Equal(t, "BUFFER", constSig.Name)
}

// ---------------------------------------------------------------------------
// Mod declarations
// ---------------------------------------------------------------------------

func TestRustCompressor_ModDeclaration(t *testing.T) {
	c := NewRustCompressor()
	source := `pub mod config;
mod internal;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	importCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindImport {
			importCount++
		}
	}
	assert.Equal(t, 2, importCount)
}

// ---------------------------------------------------------------------------
// Macro rules
// ---------------------------------------------------------------------------

func TestRustCompressor_MacroRules(t *testing.T) {
	c := NewRustCompressor()
	source := `macro_rules! vec_of_strings {
    ($($s:expr),*) => {
        vec![$($s.to_string()),*]
    };
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	var macroSig *Signature
	for i := range output.Signatures {
		if output.Signatures[i].Kind == KindFunction {
			macroSig = &output.Signatures[i]
			break
		}
	}
	require.NotNil(t, macroSig, "expected a macro signature")
	assert.Equal(t, "vec_of_strings", macroSig.Name)
	assert.Contains(t, macroSig.Source, "macro_rules! vec_of_strings")
	// Body should be excluded.
	assert.NotContains(t, macroSig.Source, "to_string()")
}

// ---------------------------------------------------------------------------
// Doc comments
// ---------------------------------------------------------------------------

func TestRustCompressor_DocCommentOnFn(t *testing.T) {
	c := NewRustCompressor()
	source := `/// Process the input data.
/// Returns the result or an error.
pub fn process(data: &[u8]) -> Result<Vec<u8>> {
    Ok(data.to_vec())
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
	assert.Contains(t, funcSig.Source, "/// Process the input data.")
	assert.Contains(t, funcSig.Source, "/// Returns the result or an error.")
	assert.Contains(t, funcSig.Source, "pub fn process")
}

func TestRustCompressor_InnerDocComment(t *testing.T) {
	c := NewRustCompressor()
	source := `//! This is a crate-level doc comment.
//! It describes the crate.

use std::io;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	docCount := 0
	for _, sig := range output.Signatures {
		if sig.Kind == KindDocComment {
			docCount++
		}
	}
	assert.Equal(t, 2, docCount)
}

func TestRustCompressor_EmptyLineClearsDoc(t *testing.T) {
	c := NewRustCompressor()
	source := `/// This comment is separated.

pub fn orphan() {
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
	assert.NotContains(t, funcSig.Source, "This comment is separated")
	assert.Contains(t, funcSig.Source, "pub fn orphan()")
}

// ---------------------------------------------------------------------------
// Attributes
// ---------------------------------------------------------------------------

func TestRustCompressor_AttributeOnStruct(t *testing.T) {
	c := NewRustCompressor()
	source := `#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Config {
    name: String,
}`
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
	assert.Contains(t, structSig.Source, "#[derive(Debug, Clone, Serialize)]")
	assert.Contains(t, structSig.Source, `#[serde(rename_all = "camelCase")]`)
}

// ---------------------------------------------------------------------------
// Brace counting
// ---------------------------------------------------------------------------

func TestRustCountBraces(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		depth int
	}{
		{"empty", "", 0},
		{"open brace", "fn main() {", 1},
		{"close brace", "}", -1},
		{"balanced", "if x { y() }", 0},
		{"brace in string", `let s = "{}";`, 0},
		{"brace in raw string", `let s = r#"{ }"#;`, 0},
		{"brace in char", "let c = '{';", 0},
		{"brace after line comment", "x = 1; // {", 0},
		{"nested braces", "HashMap::new() { }", 0},
		{"multiple open", "struct Foo { bar: HashMap<K, V> {", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rustCountBraces(tt.line)
			assert.Equal(t, tt.depth, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Complex/realistic tests
// ---------------------------------------------------------------------------

func TestRustCompressor_CompleteFile(t *testing.T) {
	c := NewRustCompressor()
	source := `use std::collections::HashMap;
use crate::config::Config;

/// A thread-safe connection pool.
#[derive(Debug, Clone)]
pub struct Pool<T: Connection> {
    connections: Vec<T>,
    max_size: usize,
}

impl<T: Connection> Pool<T> {
    /// Create a new pool with the given capacity.
    pub fn new(max_size: usize) -> Self {
        Pool {
            connections: Vec::with_capacity(max_size),
            max_size,
        }
    }

    pub fn acquire(&self) -> Option<&T> {
        self.connections.first()
    }
}

pub trait Connection: Send + Sync {
    type Error;
    fn connect(addr: &str) -> Result<Self, Self::Error> where Self: Sized;
    fn is_alive(&self) -> bool;
}

pub enum Status {
    Active,
    Idle(Duration),
    Error { code: u32, message: String },
}

pub type Result<T> = std::result::Result<T, Error>;

const MAX_CONNECTIONS: usize = 100;
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()

	// Should contain use declarations.
	assert.Contains(t, rendered, "use std::collections::HashMap;")
	assert.Contains(t, rendered, "use crate::config::Config;")

	// Should contain struct with fields, doc, and derive.
	assert.Contains(t, rendered, "/// A thread-safe connection pool.")
	assert.Contains(t, rendered, "#[derive(Debug, Clone)]")
	assert.Contains(t, rendered, "pub struct Pool<T: Connection>")
	assert.Contains(t, rendered, "connections: Vec<T>")

	// Should contain impl block with method signatures.
	assert.Contains(t, rendered, "impl<T: Connection> Pool<T>")
	assert.Contains(t, rendered, "pub fn new(max_size: usize) -> Self")
	assert.Contains(t, rendered, "pub fn acquire(&self) -> Option<&T>")

	// Should NOT contain function bodies.
	assert.NotContains(t, rendered, "Vec::with_capacity")
	assert.NotContains(t, rendered, "self.connections.first()")

	// Should contain trait.
	assert.Contains(t, rendered, "pub trait Connection: Send + Sync")
	assert.Contains(t, rendered, "type Error;")
	assert.Contains(t, rendered, "fn connect(addr: &str)")
	assert.Contains(t, rendered, "fn is_alive(&self) -> bool;")

	// Should contain enum.
	assert.Contains(t, rendered, "pub enum Status")
	assert.Contains(t, rendered, "Active,")

	// Should contain type alias.
	assert.Contains(t, rendered, "pub type Result<T> = std::result::Result<T, Error>;")

	// Should contain const.
	assert.Contains(t, rendered, "const MAX_CONNECTIONS: usize = 100;")
}

func TestRustCompressor_SourceOrderPreserved(t *testing.T) {
	c := NewRustCompressor()
	source := `use std::io;

const VERSION: &str = "1.0";

pub struct Config {
    name: String,
}

pub fn run() {
    println!("run");
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 4)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)   // use
	assert.Equal(t, KindConstant, output.Signatures[1].Kind) // const
	assert.Equal(t, KindStruct, output.Signatures[2].Kind)   // struct
	assert.Equal(t, KindFunction, output.Signatures[3].Kind) // fn
}

func TestRustCompressor_CompressionRatio(t *testing.T) {
	c := NewRustCompressor()
	source := `use std::collections::HashMap;

pub struct Config {
    host: String,
    port: u16,
}

impl Config {
    pub fn new() -> Self {
        Config {
            host: "localhost".to_string(),
            port: 8080,
        }
    }

    pub fn addr(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }

    pub fn validate(&self) -> Result<(), String> {
        if self.port == 0 {
            return Err("port cannot be 0".to_string());
        }
        if self.host.is_empty() {
            return Err("host cannot be empty".to_string());
        }
        Ok(())
    }
}
`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	output.Render()
	ratio := output.CompressionRatio()
	assert.Greater(t, ratio, 0.1, "ratio should be > 0.1")
	assert.Less(t, ratio, 0.85, "ratio should be < 0.85 (at least 15%% reduction)")
}

func TestRustCompressor_NestedBraces(t *testing.T) {
	c := NewRustCompressor()
	source := `pub fn complex() {
    let m: HashMap<String, Vec<u8>> = HashMap::new();
    if let Some(v) = m.get("key") {
        for b in v {
            println!("{}", b);
        }
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
	assert.Equal(t, "complex", funcSig.Name)
	assert.NotContains(t, funcSig.Source, "HashMap::new()")
}

func TestRustCompressor_ExternCBlock(t *testing.T) {
	c := NewRustCompressor()
	source := `extern "C" {
    fn abs(input: i32) -> i32;
    fn strlen(s: *const c_char) -> usize;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	// Should have an extern block signature.
	found := false
	for _, sig := range output.Signatures {
		if strings.Contains(sig.Source, "extern") {
			found = true
			assert.Contains(t, sig.Source, "fn abs(input: i32) -> i32;")
			assert.Contains(t, sig.Source, "fn strlen(s: *const c_char) -> usize;")
			break
		}
	}
	assert.True(t, found, "expected an extern block signature")
}

func TestRustCompressor_PubSuperFn(t *testing.T) {
	c := NewRustCompressor()
	source := `pub(super) fn parent_visible() -> bool {
    true
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
	assert.Equal(t, "parent_visible", funcSig.Name)
	assert.Contains(t, funcSig.Source, "pub(super) fn parent_visible()")
}

// ---------------------------------------------------------------------------
// Golden tests using fixture files
// ---------------------------------------------------------------------------

func TestRustCompressor_GoldenTests(t *testing.T) {
	compressor := NewRustCompressor()
	ctx := context.Background()

	tests := []struct {
		name     string
		fixture  string
		expected string
	}{
		{
			name:     "struct and impl blocks",
			fixture:  "rust/struct_impl.rs",
			expected: "rust/struct_impl.rs.expected",
		},
		{
			name:     "trait definitions",
			fixture:  "rust/trait_definition.rs",
			expected: "rust/trait_definition.rs.expected",
		},
		{
			name:     "enum variants",
			fixture:  "rust/enum_variants.rs",
			expected: "rust/enum_variants.rs.expected",
		},
		{
			name:     "lifetimes and generics",
			fixture:  "rust/lifetimes_generics.rs",
			expected: "rust/lifetimes_generics.rs.expected",
		},
		{
			name:     "visibility modifiers",
			fixture:  "rust/visibility.rs",
			expected: "rust/visibility.rs.expected",
		},
		{
			name:     "use statements",
			fixture:  "rust/use_statements.rs",
			expected: "rust/use_statements.rs.expected",
		},
		{
			name:     "complete realistic file",
			fixture:  "rust/complete_file.rs",
			expected: "rust/complete_file.rs.expected",
		},
		{
			name:     "macros and extern blocks",
			fixture:  "rust/macros_extern.rs",
			expected: "rust/macros_extern.rs.expected",
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

func BenchmarkRustCompressor(b *testing.B) {
	c := NewRustCompressor()
	source := []byte(`use std::collections::HashMap;
use std::io::{self, Read, Write};

/// Server configuration.
#[derive(Debug, Clone)]
pub struct Config {
    host: String,
    port: u16,
    max_connections: usize,
}

pub trait Handler: Send + Sync {
    fn handle(&self, req: Request) -> Response;
}

impl Config {
    pub fn new(host: &str, port: u16) -> Self {
        Config { host: host.to_string(), port, max_connections: 100 }
    }

    pub fn addr(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }
}

pub enum Status {
    Active,
    Idle(u64),
    Error { code: u32, message: String },
}

pub type Result<T> = std::result::Result<T, Error>;

const MAX_SIZE: usize = 1024;

pub fn process(data: &[u8]) -> Result<()> {
    validate(data)?;
    Ok(())
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