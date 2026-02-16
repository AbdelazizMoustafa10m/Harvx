# T-044: Tier 1 Compressor -- TypeScript and JavaScript

**Priority:** Must Have
**Effort:** Large (16-20hrs)
**Dependencies:** T-042, T-043
**Phase:** 3 - Security & Compression

---

## Description

Implement `LanguageCompressor` for TypeScript and JavaScript -- the two most critical Tier 1 languages. This involves writing tree-sitter queries that walk the AST to extract function/method signatures, class declarations, interface declarations, type aliases, enums, import/export statements, top-level constants, and doc comments. TypeScript and JavaScript share a grammar lineage, so much of the extraction logic can be shared with TypeScript-specific extensions for types, interfaces, enums, and type aliases.

This is the most complex compressor because TypeScript/JavaScript have the richest set of declaration forms (arrow functions, class methods, default exports, re-exports, namespace imports, destructured imports, etc.).

## User Story

As a developer using Harvx with `--compress` on a TypeScript/JavaScript project, I want the output to include all function signatures, class structures, types, and imports so that an LLM understands my code architecture without reading every function body.

## Acceptance Criteria

- [ ] TypeScript compressor extracts all of the following node types:
  - `function_declaration` (name, parameters, return type annotation)
  - `method_definition` within classes (name, parameters, return type, visibility modifier)
  - `arrow_function` assigned to `const`/`let`/`var` at top level or exported
  - `class_declaration` (name, extends, implements, field declarations with types)
  - `interface_declaration` (name, extends, property signatures with types)
  - `type_alias_declaration` (name, type expression)
  - `enum_declaration` (name, members)
  - `import_statement` (full verbatim import line)
  - `export_statement` (full verbatim export line, including re-exports)
  - Top-level `const`/`let` declarations (identifier and type annotation, not value)
  - JSDoc/TSDoc comments (`/** ... */`) attached to declarations
- [ ] JavaScript compressor extracts the same set minus TypeScript-specific items (interfaces, type aliases, enums, type annotations)
- [ ] Extraction is verbatim -- exact source text at AST node boundaries, never summarized
- [ ] Compressed output preserves source order (signatures appear in file order)
- [ ] Handles common patterns: default exports, named exports, re-exports, barrel files
- [ ] Handles decorators (e.g., `@Component`) attached to class declarations
- [ ] Doc comments are attached to the following declaration, not standalone
- [ ] Achieves 50-70% token reduction on representative TypeScript files
- [ ] Unit tests with at least 10 TypeScript and 5 JavaScript fixture files
- [ ] Golden tests compare extracted output against expected signatures

## Technical Notes

### Tree-Sitter Node Types for TypeScript

Key AST node types in `tree-sitter-typescript`:

```
program
  import_statement
  export_statement
  function_declaration
    name: identifier
    parameters: formal_parameters
    return_type: type_annotation
    body: statement_block
  class_declaration
    name: type_identifier
    type_parameters: type_parameters
    class_heritage (extends_clause, implements_clause)
    body: class_body
      method_definition
      public_field_definition
  interface_declaration
    name: type_identifier
    extends_type_clause
    body: object_type (property_signature, method_signature)
  type_alias_declaration
    name: type_identifier
    value: ... (any type expression)
  enum_declaration
    name: identifier
    body: enum_body (enum_member)
  lexical_declaration (const/let)
    variable_declarator
      name: identifier
      type: type_annotation
      value: ...
```

### Tree-Sitter Node Types for JavaScript

JavaScript uses `tree-sitter-javascript` which is a subset of TypeScript's grammar:

```
program
  import_statement
  export_statement
  function_declaration
  class_declaration
    class_body
      method_definition
      field_definition
  variable_declaration / lexical_declaration
  arrow_function (when assigned to variable)
```

### Extraction Strategy

For each top-level declaration:
1. **Check for preceding doc comment** -- look at the previous sibling node; if it is a `comment` node starting with `/**`, include it
2. **Extract the signature** -- capture the source text from the node start to:
   - For functions: end of `formal_parameters` and `return_type` (exclude `statement_block`)
   - For classes: start of `class_body` (include the `{` but exclude method bodies)
   - For interfaces: full node (interfaces are already signatures)
   - For imports/exports: full node text
   - For type aliases: full node text
   - For enums: full node text
   - For constants: identifier + type annotation (exclude initializer if complex)
3. **For class bodies** -- recurse into `class_body` and extract method signatures (exclude bodies)

### Source Text Extraction

```go
// extractNodeText returns the verbatim source text for a node.
// For declarations with bodies, it extracts only the signature portion.
func extractNodeText(source []byte, node *Node) string {
    return string(source[node.StartByte():node.EndByte()])
}

// extractSignatureOnly returns just the signature, excluding the body.
// For a function: everything before the opening { of the body.
func extractSignatureOnly(source []byte, node *Node) string {
    bodyNode := node.ChildByFieldName("body")
    if bodyNode != nil {
        return strings.TrimRight(string(source[node.StartByte():bodyNode.StartByte()]), " \t\n")
    }
    return extractNodeText(source, node)
}
```

### Shared Base Compressor

Since TypeScript is a superset of JavaScript, create a shared base:

```go
type jsBaseCompressor struct {
    grammarRegistry *GrammarRegistry
}

type TypeScriptCompressor struct {
    jsBaseCompressor
}

type JavaScriptCompressor struct {
    jsBaseCompressor
}
```

### Edge Cases to Handle

- **Arrow functions**: `const handler = async (req: Request): Promise<Response> => { ... }` -- extract entire left-hand side + arrow + return type
- **Exported declarations**: `export default function` -- include the `export default` prefix
- **Re-exports**: `export { foo } from './bar'` -- include full statement
- **Barrel files**: Files that are only exports (e.g., `index.ts` with `export * from './module'`)
- **Nested functions**: Only extract top-level and class-level functions, not nested closures
- **Overloaded functions** (TypeScript): Multiple signature declarations before the implementation
- **Abstract classes and methods**: Include `abstract` modifier
- **Getter/setter methods**: Include `get`/`set` keyword

## Files to Create/Modify

- `internal/compression/typescript.go` -- TypeScript compressor implementation
- `internal/compression/javascript.go` -- JavaScript compressor implementation
- `internal/compression/js_base.go` -- Shared extraction logic for JS/TS
- `internal/compression/typescript_test.go` -- TypeScript unit tests
- `internal/compression/javascript_test.go` -- JavaScript unit tests
- `testdata/compression/typescript/` -- TypeScript fixture files (input + expected output)
- `testdata/compression/javascript/` -- JavaScript fixture files (input + expected output)

## Testing Requirements

- Golden test: Parse a representative Next.js API route file, compare extracted signatures
- Golden test: Parse a React component file with hooks, props interface, and JSX
- Golden test: Parse a TypeScript service class with methods, constructor, and decorators
- Golden test: Parse a barrel file (`index.ts` with only re-exports)
- Golden test: Parse a file with enums, type aliases, and interfaces
- Unit test: Arrow function extraction (assigned to const, exported)
- Unit test: Class with inheritance and implements
- Unit test: Default export handling
- Unit test: Doc comment attachment to declarations
- Unit test: File with no extractable signatures returns empty output
- Benchmark: Compression ratio on 10 representative TS files (target: 50-70% reduction)
- Benchmark: Parsing time per file (target: < 50ms for typical files)

## References

- [tree-sitter-typescript Grammar](https://github.com/tree-sitter/tree-sitter-typescript)
- [tree-sitter-javascript Grammar](https://github.com/tree-sitter/tree-sitter-javascript)
- [TypeScript Node Types JSON](https://github.com/tree-sitter/tree-sitter-typescript/blob/master/tsx/src/node-types.json)
- [Tree-sitter Query Syntax](https://tree-sitter.github.io/tree-sitter/using-parsers/queries/1-syntax.html)
- PRD Section 5.6: Extraction rules per language