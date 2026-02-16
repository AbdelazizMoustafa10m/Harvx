# T-047: Tier 2 Compressor -- Java, C, and C++

**Priority:** Should Have
**Effort:** Medium (10-14hrs)
**Dependencies:** T-042, T-043
**Phase:** 3 - Security & Compression

---

## Description

Implement `LanguageCompressor` for the Tier 2 compiled languages: Java, C, and C++. These compressors provide basic structural extraction -- function/method signatures, class/struct declarations, import/include statements, and type definitions. Tier 2 support is intentionally simpler than Tier 1: it targets the most common declaration patterns without exhaustive handling of every edge case. C and C++ share some extraction logic but differ significantly in class/template handling.

## User Story

As a developer using Harvx with `--compress` on Java, C, or C++ projects, I want at least the key structural elements extracted so that the compressed output is more useful than raw file content while keeping within token budgets.

## Acceptance Criteria

### Java
- [ ] Extracts `class_declaration` (modifiers, name, extends, implements, field declarations)
- [ ] Extracts `interface_declaration` (name, extends, method signatures)
- [ ] Extracts `method_declaration` (modifiers, return type, name, parameters, throws)
- [ ] Extracts `constructor_declaration` (modifiers, name, parameters)
- [ ] Extracts `enum_declaration` (name, constants)
- [ ] Extracts `import_declaration` (full verbatim line)
- [ ] Extracts `package_declaration` (full verbatim line)
- [ ] Extracts `annotation_type_declaration` (name, elements)
- [ ] Handles `@Override`, `@Deprecated`, and other annotations on declarations
- [ ] Handles `record` declarations (Java 16+)
- [ ] Handles Javadoc comments (`/** ... */`) attached to declarations

### C
- [ ] Extracts `function_definition` signatures (return type, name, parameters -- exclude body)
- [ ] Extracts `function_declarator` in header files (prototypes)
- [ ] Extracts `struct_specifier` declarations (name, field list)
- [ ] Extracts `enum_specifier` (name, enumerators)
- [ ] Extracts `type_definition` (`typedef` statements)
- [ ] Extracts `preproc_include` (`#include` directives)
- [ ] Extracts `preproc_def` / `preproc_function_def` (`#define` macros -- name and parameters)
- [ ] Extracts top-level `declaration` (global variables with types)
- [ ] Handles forward declarations

### C++
- [ ] All C extractions plus:
- [ ] Extracts `class_specifier` (name, base classes, member declarations)
- [ ] Extracts `template_declaration` (template parameters + underlying declaration)
- [ ] Extracts `namespace_definition` (name, recurse for nested declarations)
- [ ] Extracts `using_declaration` (type aliases, namespace using)
- [ ] Handles `public:`/`private:`/`protected:` access specifiers within classes
- [ ] Handles `virtual`, `override`, `const`, `noexcept` method qualifiers

### All Languages
- [ ] Extraction is verbatim at AST node boundaries
- [ ] Function/method bodies are excluded
- [ ] Achieves 40-60% token reduction (slightly lower target for Tier 2)
- [ ] Unit tests with at least 4 fixture files per language

## Technical Notes

### Tree-Sitter Node Types for Java

```
program
  package_declaration
  import_declaration
  class_declaration
    modifiers
    name: identifier
    superclass: type_identifier
    interfaces: super_interfaces
    body: class_body
      method_declaration
        modifiers
        type: type_identifier
        name: identifier
        parameters: formal_parameters
        throws: throws
        body: block
      constructor_declaration
      field_declaration
      enum_declaration
  interface_declaration
    name: type_identifier
    extends_interfaces
    body: interface_body
      method_declaration (abstract by default)
  enum_declaration
    name: identifier
    body: enum_body
      enum_constant
  annotation_type_declaration
  record_declaration (Java 16+)
```

### Tree-Sitter Node Types for C

```
translation_unit
  preproc_include
  preproc_def / preproc_function_def
  function_definition
    type: type_identifier / primitive_type
    declarator: function_declarator
      declarator: identifier
      parameters: parameter_list
    body: compound_statement
  declaration (global variables, function prototypes)
  struct_specifier
    name: type_identifier
    body: field_declaration_list
  enum_specifier
    name: type_identifier
    body: enumerator_list
  type_definition
```

### Tree-Sitter Node Types for C++

Extends C grammar with:
```
class_specifier
  name: type_identifier
  base_class_clause
  body: field_declaration_list
    access_specifier
    function_definition (methods)
    field_declaration
template_declaration
  parameters: template_parameter_list
  (class_specifier | function_definition | ...)
namespace_definition
  name: identifier (or namespace_identifier)
  body: declaration_list
using_declaration
```

### Extraction Strategy

**Java:**
- Package and imports: full verbatim
- Classes: modifiers + name + extends/implements + `{`, then recurse for field declarations and method signatures
- Methods: modifiers + return type + name + parameters + throws (exclude body)
- Enums: full declaration (constants are structural)
- Interfaces: full declaration (all members are signatures)
- Javadoc: attach to following declaration

**C:**
- Includes: all `#include` lines verbatim
- Defines: `#define` name + parameters (exclude multi-line macro bodies)
- Functions: return type + name + parameters (exclude compound_statement body)
- Structs/Enums: full declaration
- Typedefs: full line
- Function prototypes in headers: full declaration

**C++:**
- Everything from C, plus:
- Classes: similar to Java class extraction with access specifiers
- Templates: `template<...>` prefix + underlying declaration
- Namespaces: `namespace Name {` + recurse for nested declarations at top level

### Shared C/C++ Base

```go
type cBaseCompressor struct {
    grammarRegistry *GrammarRegistry
}

type CCompressor struct {
    cBaseCompressor
}

type CppCompressor struct {
    cBaseCompressor
}
```

### Edge Cases

**Java:**
- Inner/nested classes -- extract signatures for nested classes too
- Anonymous classes -- skip (no meaningful signature)
- Lambda expressions -- skip (not structural declarations)
- Sealed classes/interfaces (Java 17+)
- Annotation elements (`@interface` members)

**C:**
- Function pointer typedefs: `typedef void (*handler_t)(int, const char*);`
- Bit fields in structs: `unsigned int flag : 1;`
- Variadic functions: `int printf(const char *fmt, ...);`
- `extern "C"` blocks in headers (technically C++ but common in C headers)

**C++:**
- Multiple inheritance
- Virtual destructors
- Operator overloading declarations
- `constexpr` functions (include as signatures)
- Nested namespaces: `namespace a::b::c { ... }`

## Files to Create/Modify

- `internal/compression/java.go` -- Java compressor
- `internal/compression/clang.go` -- C compressor
- `internal/compression/cpp.go` -- C++ compressor
- `internal/compression/c_base.go` -- Shared C/C++ extraction logic
- `internal/compression/java_test.go` -- Java unit tests
- `internal/compression/clang_test.go` -- C unit tests
- `internal/compression/cpp_test.go` -- C++ unit tests
- `testdata/compression/java/` -- Java fixtures
- `testdata/compression/c/` -- C fixtures
- `testdata/compression/cpp/` -- C++ fixtures

## Testing Requirements

### Java
- Golden test: Parse a Spring Boot controller class
- Golden test: Parse an interface with generic type parameters
- Golden test: Parse an enum with methods
- Unit test: Annotation handling on classes and methods
- Unit test: Javadoc attachment

### C
- Golden test: Parse a C header file with prototypes, structs, and typedefs
- Golden test: Parse a C source file with function definitions
- Unit test: #include and #define extraction
- Unit test: Function pointer typedef handling

### C++
- Golden test: Parse a class with inheritance, templates, and access specifiers
- Golden test: Parse a file with namespaces and nested classes
- Unit test: Template declaration extraction
- Unit test: `using` declaration handling

### All
- Benchmark: Compression ratio per language (target: 40-60%)

## References

- [tree-sitter-java Grammar](https://github.com/tree-sitter/tree-sitter-java)
- [tree-sitter-c Grammar](https://github.com/tree-sitter/tree-sitter-c)
- [tree-sitter-cpp Grammar](https://github.com/tree-sitter/tree-sitter-cpp)
- [Tree-sitter Java Node Types (DeepWiki)](https://deepwiki.com/tree-sitter/tree-sitter-java/4.2-query-system)
- PRD Section 5.6: Tier 2 basic support languages