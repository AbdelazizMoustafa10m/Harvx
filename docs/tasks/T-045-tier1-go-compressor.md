# T-045: Tier 1 Compressor -- Go

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-042, T-043
**Phase:** 3 - Security & Compression

---

## Description

Implement `LanguageCompressor` for Go -- a Tier 1 language. Go has a notably clean and regular syntax, making its AST extraction straightforward compared to TypeScript. This compressor extracts function/method signatures (including receiver types), struct declarations with field types, interface declarations, type aliases, const/var blocks, import statements, and doc comments. Since Harvx itself is written in Go, this compressor is also critical for dogfooding.

## User Story

As a developer using Harvx with `--compress` on a Go project, I want the output to capture all exported and unexported function signatures, struct definitions, interfaces, and imports so that an LLM can reason about my Go codebase architecture.

## Acceptance Criteria

- [ ] Go compressor extracts all of the following node types:
  - `function_declaration` (name, parameters, return types)
  - `method_declaration` (receiver type, name, parameters, return types)
  - `type_declaration` containing `struct_type` (name, field names and types, struct tags)
  - `type_declaration` containing `interface_type` (name, method signatures)
  - `type_declaration` containing type aliases (e.g., `type ID = string`)
  - `type_declaration` containing type definitions (e.g., `type ID string`)
  - `const_declaration` / `const` blocks (iota enums, typed constants)
  - `var_declaration` at package level (name, type)
  - `import_declaration` (full import block)
  - Doc comments (`//` or `/* */` blocks immediately preceding declarations)
- [ ] Extraction is verbatim -- exact source text at AST node boundaries
- [ ] Function/method bodies are excluded (only signature portion up to opening `{`)
- [ ] Struct fields include tags (e.g., `` `json:"name"` ``)
- [ ] Interface method sets are fully preserved
- [ ] Grouped declarations (`type ( ... )`, `const ( ... )`, `var ( ... )`) are handled correctly
- [ ] Achieves 50-70% token reduction on representative Go files
- [ ] Unit tests with at least 8 Go fixture files covering all node types
- [ ] Golden tests compare extracted output against expected signatures

## Technical Notes

### Tree-Sitter Node Types for Go

Key AST node types in `tree-sitter-go`:

```
source_file
  package_clause
  import_declaration
    import_spec_list
      import_spec (path, name)
  function_declaration
    name: identifier
    parameters: parameter_list
    result: (type_identifier | parameter_list)
    body: block
  method_declaration
    receiver: parameter_list  // (r *Router)
    name: field_identifier
    parameters: parameter_list
    result: ...
    body: block
  type_declaration
    type_spec
      name: type_identifier
      type: struct_type
        field_declaration_list
          field_declaration
            name: field_identifier
            type: ...
            tag: raw_string_literal
      type: interface_type
        method_spec  // method signatures
        type_elem    // embedded types
      type: type_identifier  // type alias or definition
  const_declaration
    const_spec
      name: identifier
      type: type_identifier (optional)
      value: expression
  var_declaration
    var_spec
      name: identifier
      type: type_identifier
```

### Extraction Strategy

1. **Package clause**: Always include
2. **Imports**: Extract entire `import_declaration` block verbatim
3. **Functions**: Extract from `func` keyword through end of parameter list and return type, excluding `body` (block)
4. **Methods**: Same as functions but including the receiver parameter
5. **Structs**: Extract the full `type_declaration` including all field declarations and tags
6. **Interfaces**: Extract full `type_declaration` (interfaces are already signature-only)
7. **Type aliases/definitions**: Extract full `type_declaration`
8. **Constants**: Extract full `const_declaration` or `const` block
9. **Package-level vars**: Extract full `var_declaration` (name + type, exclude complex initializers)
10. **Doc comments**: Check previous sibling for comment nodes; include if immediately preceding

### Go-Specific Patterns

```go
// Function signature extraction:
// Input:
//   // Add inserts a value into the map.
//   func (m *Map[K, V]) Add(key K, value V) error {
//       // implementation...
//   }
// Output:
//   // Add inserts a value into the map.
//   func (m *Map[K, V]) Add(key K, value V) error

// Struct extraction:
// Input:
//   // FileDescriptor holds metadata about a processed file.
//   type FileDescriptor struct {
//       Path         string   `json:"path"`
//       Size         int64    `json:"size"`
//       Tier         int      `json:"tier"`
//       TokenCount   int      `json:"token_count"`
//   }
// Output: (full struct, unchanged -- structs are inherently signatures)

// Interface extraction:
// Input:
//   type LanguageCompressor interface {
//       Compress(ctx context.Context, source []byte) (*CompressedOutput, error)
//       Language() string
//   }
// Output: (full interface, unchanged)

// Const block with iota:
// Input:
//   type Tier int
//   const (
//       Tier0 Tier = iota
//       Tier1
//       Tier2
//   )
// Output: (full const block, unchanged)
```

### Edge Cases

- **Multi-return functions**: `func Foo() (int, error)` -- both return values captured
- **Named returns**: `func Foo() (result int, err error)` -- names captured
- **Generic functions/types**: `func Map[T any](s []T, f func(T) T) []T` -- type params included
- **Embedded structs**: `type Foo struct { Bar; baz string }` -- embedded types captured
- **Blank imports**: `import _ "net/http/pprof"` -- include in import block
- **Dot imports**: `import . "math"` -- include in import block
- **Build constraints**: `//go:build` lines preceding package clause
- **CGO blocks**: `import "C"` with preceding comment block -- include verbatim

## Files to Create/Modify

- `internal/compression/golang.go` -- Go compressor implementation
- `internal/compression/golang_test.go` -- Unit tests
- `testdata/compression/go/` -- Go fixture files (input + expected output)
- `testdata/compression/go/simple_func.go` -- Basic function declarations
- `testdata/compression/go/methods.go` -- Methods with receivers
- `testdata/compression/go/structs.go` -- Struct declarations with tags
- `testdata/compression/go/interfaces.go` -- Interface declarations
- `testdata/compression/go/generics.go` -- Generic types and functions
- `testdata/compression/go/const_iota.go` -- Const blocks with iota
- `testdata/compression/go/imports.go` -- Various import patterns
- `testdata/compression/go/full_file.go` -- Realistic complete file

## Testing Requirements

- Golden test: Parse Harvx's own `internal/compression/types.go` and verify signatures match
- Golden test: Parse a Go HTTP handler file (function + struct + interface)
- Golden test: Parse a file with generics (Go 1.18+)
- Golden test: Parse a file with embedded structs and interface composition
- Unit test: Package clause extraction
- Unit test: Import block extraction (single and grouped)
- Unit test: Method receiver extraction
- Unit test: Const/iota block extraction
- Unit test: Doc comment attachment
- Unit test: Named return values
- Benchmark: Compression ratio on 10 representative Go files (target: 50-70% reduction)

## References

- [tree-sitter-go Grammar](https://github.com/tree-sitter/tree-sitter-go)
- [tree-sitter-go Node Types](https://github.com/tree-sitter/tree-sitter-go/blob/master/src/node-types.json)
- [Go Specification - Declarations](https://go.dev/ref/spec#Declarations_and_scope)
- PRD Section 5.6: Extraction rules per language