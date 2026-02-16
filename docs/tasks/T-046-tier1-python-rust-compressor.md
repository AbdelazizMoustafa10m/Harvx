# T-046: Tier 1 Compressor -- Python and Rust

**Priority:** Must Have
**Effort:** Large (14-18hrs)
**Dependencies:** T-042, T-043
**Phase:** 3 - Security & Compression

---

## Description

Implement `LanguageCompressor` for Python and Rust -- the remaining two Tier 1 languages. Python extraction focuses on `def`, `class`, `import`, type hints, and docstrings. Rust extraction covers `fn`, `struct`, `enum`, `trait`, `impl`, `type`, `use`, and `const`. Both languages have distinct AST structures that require dedicated extraction logic.

## User Story

As a developer using Harvx with `--compress` on Python or Rust projects, I want the output to include all function signatures, class/struct definitions, trait/protocol declarations, and imports so that an LLM understands the project's architecture.

## Acceptance Criteria

### Python
- [ ] Extracts `function_definition` (name, parameters with type hints, return type hint)
- [ ] Extracts `class_definition` (name, base classes, class body: method signatures + class variables)
- [ ] Extracts `import_statement` and `import_from_statement` (full verbatim line)
- [ ] Extracts docstrings (`"""..."""`) attached to functions, classes, and modules
- [ ] Extracts top-level variable assignments with type annotations (e.g., `MAX_SIZE: int = 100`)
- [ ] Extracts `decorated_definition` (decorator + function/class)
- [ ] Handles `@dataclass`, `@property`, `@staticmethod`, `@classmethod` decorators
- [ ] Handles `__init__` method signatures with parameter types
- [ ] Handles `*args`, `**kwargs`, default parameter values
- [ ] Handles async functions (`async def`)
- [ ] Handles Protocol classes (from typing module)

### Rust
- [ ] Extracts `function_item` (name, parameters with types, return type, visibility modifier)
- [ ] Extracts `struct_item` (name, fields with types, visibility, derive macros)
- [ ] Extracts `enum_item` (name, variants with associated data)
- [ ] Extracts `trait_item` (name, method signatures, associated types)
- [ ] Extracts `impl_item` (target type, trait being implemented, method signatures)
- [ ] Extracts `type_item` (type aliases)
- [ ] Extracts `use_declaration` (full use statement, including `use crate::...`)
- [ ] Extracts `const_item` and `static_item` (name, type, value for simple literals)
- [ ] Extracts `mod_item` declarations (module structure)
- [ ] Handles `pub`, `pub(crate)`, `pub(super)` visibility modifiers
- [ ] Handles `#[derive(...)]` and other attribute macros on declarations
- [ ] Handles generic type parameters and where clauses
- [ ] Handles lifetime annotations

### Both
- [ ] Extraction is verbatim -- exact source text at AST node boundaries
- [ ] Function/method bodies are excluded (only signatures)
- [ ] Achieves 50-70% token reduction on representative files
- [ ] Unit tests with at least 6 fixture files per language

## Technical Notes

### Tree-Sitter Node Types for Python

```
module
  import_statement
    dotted_name
  import_from_statement
    module_name: dotted_name
    name: (dotted_name | import_alias | wildcard_import)
  function_definition
    name: identifier
    parameters: parameters
      (identifier | typed_parameter | default_parameter | typed_default_parameter |
       list_splat_pattern | dictionary_splat_pattern)
    return_type: type
    body: block
  decorated_definition
    decorator
    definition: (function_definition | class_definition)
  class_definition
    name: identifier
    superclasses: argument_list
    body: block
      function_definition  (methods)
      expression_statement
        assignment (class variables)
  expression_statement
    string  (module/class/function docstrings -- first statement in body)
  type_alias_statement (Python 3.12+)
```

### Python Extraction Strategy

1. **Module docstring**: First statement in module if it is a string expression
2. **Imports**: Extract all `import_statement` and `import_from_statement` nodes verbatim
3. **Functions**: Extract decorator chain + `def name(params) -> return_type:` (exclude body)
   - Include docstring (first expression in body if string) as part of signature
4. **Classes**: Extract decorator chain + `class Name(bases):` + method signatures within body
   - Include class docstring
   - Extract class-level type-annotated assignments
5. **Top-level assignments**: Only those with type annotations (e.g., `x: int = 5`)

```python
# Input:
@dataclass
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

# Output:
@dataclass
class Config:
    """Application configuration."""
    host: str = "localhost"
    port: int = 8080
    debug: bool = False

    def validate(self) -> bool:
        """Validate the configuration."""
```

### Tree-Sitter Node Types for Rust

```
source_file
  use_declaration
    argument: (scoped_identifier | use_wildcard | use_list | use_as_clause)
  function_item
    visibility_modifier
    name: identifier
    type_parameters: type_parameters
    parameters: parameters
    return_type: type
    where_clause
    body: block
  struct_item
    visibility_modifier
    name: type_identifier
    type_parameters
    body: field_declaration_list
      field_declaration (visibility, name, type)
  enum_item
    name: type_identifier
    body: enum_variant_list
      enum_variant (name, optional body/tuple)
  trait_item
    visibility_modifier
    name: type_identifier
    type_parameters
    body: declaration_list
      function_item (method signatures)
      associated_type
  impl_item
    type: type_identifier
    trait: type_identifier (optional, for trait impls)
    body: declaration_list
      function_item
  type_item
    name: type_identifier
    type: type
  const_item
    name: identifier
    type: type
    value: expression
  static_item
    name: identifier
    type: type
  attribute_item (#[...])
    attribute (derive, cfg, etc.)
```

### Rust Extraction Strategy

1. **Use declarations**: Extract all `use_declaration` nodes verbatim
2. **Functions**: Extract attributes + visibility + `fn name<T>(params) -> ReturnType where ...` (exclude body block)
3. **Structs**: Extract attributes + full struct declaration including field list
4. **Enums**: Extract attributes + full enum declaration including variants
5. **Traits**: Extract attributes + full trait declaration including method signatures
6. **Impl blocks**: Extract `impl Type` or `impl Trait for Type` + method signatures (exclude bodies)
7. **Type aliases**: Extract full `type_item`
8. **Constants/Statics**: Extract full declaration

```rust
// Input:
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

// Output:
/// A thread-safe connection pool.
#[derive(Debug, Clone)]
pub struct Pool<T: Connection> {
    connections: Vec<T>,
    max_size: usize,
}

impl<T: Connection> Pool<T> {
    /// Create a new pool with the given capacity.
    pub fn new(max_size: usize) -> Self

    pub fn acquire(&self) -> Option<&T>
}
```

### Edge Cases

**Python:**
- `__all__` list at module level (include verbatim -- defines public API)
- Conditional imports (`if TYPE_CHECKING: import ...`) -- include both branches
- Walrus operator in assignments (skip -- not structural)
- Multi-line function signatures with many parameters
- Type aliases: `Vector = list[float]`

**Rust:**
- `#[cfg(...)]` conditional compilation attributes on declarations
- `macro_rules!` definitions (include the signature/name, skip body)
- `extern "C"` blocks
- Lifetime annotations in function signatures and struct fields
- Complex where clauses spanning multiple lines
- Tuple structs: `pub struct Wrapper(pub T);`
- Unit structs: `pub struct Marker;`

## Files to Create/Modify

- `internal/compression/python.go` -- Python compressor implementation
- `internal/compression/rust.go` -- Rust compressor implementation
- `internal/compression/python_test.go` -- Python unit tests
- `internal/compression/rust_test.go` -- Rust unit tests
- `testdata/compression/python/` -- Python fixture files
- `testdata/compression/rust/` -- Rust fixture files

## Testing Requirements

### Python
- Golden test: Parse a Django model file (class with fields, Meta, methods)
- Golden test: Parse a FastAPI router file (decorated async functions)
- Golden test: Parse a dataclass file with type hints
- Golden test: Parse a file with Protocol classes and type aliases
- Unit test: Decorator preservation
- Unit test: Docstring extraction for module, class, and function levels
- Unit test: `*args` / `**kwargs` handling
- Benchmark: Compression ratio (target: 50-70%)

### Rust
- Golden test: Parse a Rust struct + impl block file
- Golden test: Parse a trait definition with associated types
- Golden test: Parse a file with enum variants (unit, tuple, struct)
- Golden test: Parse a file with lifetime annotations and generics
- Unit test: `#[derive]` attribute preservation
- Unit test: Visibility modifier extraction
- Unit test: `use` statement grouping
- Benchmark: Compression ratio (target: 50-70%)

## References

- [tree-sitter-python Grammar](https://github.com/tree-sitter/tree-sitter-python)
- [tree-sitter-python Node Types](https://github.com/tree-sitter/tree-sitter-python/blob/master/src/node-types.json)
- [tree-sitter-rust Grammar](https://github.com/tree-sitter/tree-sitter-rust)
- [tree-sitter-rust Node Types](https://github.com/tree-sitter/tree-sitter-rust/blob/master/src/node-types.json)
- [Rust Reference - Items](https://doc.rust-lang.org/reference/items.html)
- [Python AST Specification](https://docs.python.org/3/library/ast.html)
- PRD Section 5.6: Extraction rules per language