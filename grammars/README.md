# Tree-sitter Grammar WASM Files

This directory contains prebuilt tree-sitter grammar files compiled to WebAssembly.
These files are embedded into the `harvx` binary at compile time via `//go:embed` and
executed at runtime using the [wazero](https://github.com/tetratelabs/wazero) WASM runtime.

## Source

All grammar WASM files originate from the
[tree-sitter-wasms](https://www.npmjs.com/package/tree-sitter-wasms) npm package
published by Sourcegraph. This package provides tree-sitter parsers compiled to
WASM using Emscripten.

## Included Grammars

| File | Language | Size |
|------|----------|------|
| `tree-sitter-typescript.wasm` | TypeScript | 2.3 MB |
| `tree-sitter-javascript.wasm` | JavaScript | 647 KB |
| `tree-sitter-go.wasm` | Go | 236 KB |
| `tree-sitter-python.wasm` | Python | 476 KB |
| `tree-sitter-rust.wasm` | Rust | 819 KB |
| `tree-sitter-java.wasm` | Java | 430 KB |
| `tree-sitter-c.wasm` | C | 793 KB |
| `tree-sitter-cpp.wasm` | C++ | 4.7 MB |

**Total embedded size:** ~10.4 MB (added to binary size). Sizes from version 0.1.13.

## How to Download / Update

Run the fetch script from the project root:

```bash
# Download missing grammars
./scripts/fetch-grammars.sh

# Force re-download all grammars (e.g., to update to a newer version)
./scripts/fetch-grammars.sh --force
```

The script downloads from the unpkg.com CDN with a jsDelivr fallback.

## Build Requirement

The `.wasm` files **must** be present in this directory before compiling `harvx`.
The `grammars/embed.go` file uses `//go:embed *.wasm` which requires at least one
matching file at compile time. If the files are missing, `go build` will fail with:

```
pattern *.wasm: no matching files found
```

## License

Tree-sitter itself is licensed under the [MIT License](https://github.com/tree-sitter/tree-sitter/blob/master/LICENSE).

Individual grammar repositories maintain their own licenses:

- [tree-sitter-typescript](https://github.com/tree-sitter/tree-sitter-typescript) -- MIT
- [tree-sitter-javascript](https://github.com/tree-sitter/tree-sitter-javascript) -- MIT
- [tree-sitter-go](https://github.com/tree-sitter/tree-sitter-go) -- MIT
- [tree-sitter-python](https://github.com/tree-sitter/tree-sitter-python) -- MIT
- [tree-sitter-rust](https://github.com/tree-sitter/tree-sitter-rust) -- MIT
- [tree-sitter-java](https://github.com/tree-sitter/tree-sitter-java) -- MIT
- [tree-sitter-c](https://github.com/tree-sitter/tree-sitter-c) -- MIT
- [tree-sitter-cpp](https://github.com/tree-sitter/tree-sitter-cpp) -- MIT

The `tree-sitter-wasms` npm package by Sourcegraph is also MIT-licensed.

## Git Tracking

These `.wasm` files are tracked in git (excluded from the global `*.wasm` gitignore
rule via `!grammars/*.wasm`) to ensure reproducible builds without requiring
developers to run the fetch script before their first build.