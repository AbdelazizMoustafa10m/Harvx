# T-029: Tokenizer Interface and Implementations (cl100k, o200k, none)

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** None (standalone module)
**Phase:** 2 - Intelligence (Relevance & Tokens)

---

## Description

Design and implement the `Tokenizer` interface and three concrete implementations: `cl100k_base` (GPT-4/Claude compatible, default), `o200k_base` (GPT-4o/o1/newer OpenAI models), and a `none` fallback that uses character-count estimation for speed. This is the foundational abstraction for all token counting in Harvx. The interface must be simple, goroutine-safe, and support both single-text counting and batch counting.

## User Story

As a developer, I want Harvx to accurately count tokens using the same tokenizer my target LLM uses so that I can trust the token budget is correct and I stay within my model's context window.

## Acceptance Criteria

- [ ] `Tokenizer` interface defined with methods: `Count(text string) int`, `Name() string`
- [ ] `cl100k_base` implementation using `pkoukk/tiktoken-go` with `tiktoken.GetEncoding("cl100k_base")`
- [ ] `o200k_base` implementation using `pkoukk/tiktoken-go` with `tiktoken.GetEncoding("o200k_base")`
- [ ] `none` (estimator) implementation that returns `len(text) / 4` as a rough token estimate (industry standard ~4 chars per token)
- [ ] Factory function `NewTokenizer(name string) (Tokenizer, error)` that returns the appropriate implementation
- [ ] Supported tokenizer names: `"cl100k_base"`, `"o200k_base"`, `"none"` -- error on unknown name
- [ ] Default tokenizer is `cl100k_base` (returned when empty string passed)
- [ ] tiktoken encoding is initialized once per Tokenizer instance (not per Count call)
- [ ] Tokenizer instances are goroutine-safe (tiktoken-go's Encode is documented as safe for concurrent use)
- [ ] Labels the tokenizer name in output metadata via `Name()` for pipeline debuggability
- [ ] `TIKTOKEN_CACHE_DIR` environment variable is respected for BPE dictionary caching
- [ ] Unit tests achieve 95%+ coverage
- [ ] Benchmark tests comparing cl100k, o200k, and none estimator performance

## Technical Notes

- Create in `internal/tokenizer/tokenizer.go` (interface + factory)
- Create in `internal/tokenizer/tiktoken.go` (cl100k and o200k implementations)
- Create in `internal/tokenizer/estimator.go` (none/fallback implementation)

### tiktoken-go API Usage

```go
import tiktoken "github.com/pkoukk/tiktoken-go"

// Initialize once
enc, err := tiktoken.GetEncoding("cl100k_base")
if err != nil {
    return nil, fmt.Errorf("failed to get encoding: %w", err)
}

// Count tokens (goroutine-safe)
tokens := enc.Encode(text, nil, nil)
count := len(tokens)
```

### BPE Caching

tiktoken-go downloads BPE dictionaries on first use. To avoid network calls at runtime:
- Set `TIKTOKEN_CACHE_DIR` to a writable directory
- Alternatively, use `tiktoken.SetBpeLoader()` with an embedded/offline loader for the single-binary distribution
- Consider embedding the BPE data using `//go:embed` for zero-network operation (investigate feasibility -- the BPE files are ~1-4MB each)

### Interface Design

```go
// Tokenizer counts tokens in text content.
type Tokenizer interface {
    // Count returns the number of tokens in the given text.
    Count(text string) int
    // Name returns the tokenizer encoding name (e.g., "cl100k_base").
    Name() string
}
```

### Dependencies & Versions

| Package/Library | Version | Purpose |
|-----------------|---------|---------|
| github.com/pkoukk/tiktoken-go | v0.1.7+ | BPE tokenization for cl100k_base and o200k_base |

## Files to Create/Modify

- `internal/tokenizer/tokenizer.go` - Tokenizer interface, NewTokenizer() factory
- `internal/tokenizer/tiktoken.go` - tiktokenTokenizer struct implementing Tokenizer for cl100k and o200k
- `internal/tokenizer/estimator.go` - estimatorTokenizer struct implementing Tokenizer for "none" mode
- `internal/tokenizer/tokenizer_test.go` - Interface and factory tests
- `internal/tokenizer/tiktoken_test.go` - tiktoken implementation tests
- `internal/tokenizer/estimator_test.go` - Estimator tests

## Testing Requirements

- Unit test: NewTokenizer("cl100k_base") returns a valid Tokenizer with Name() == "cl100k_base"
- Unit test: NewTokenizer("o200k_base") returns a valid Tokenizer with Name() == "o200k_base"
- Unit test: NewTokenizer("none") returns a valid Tokenizer with Name() == "none"
- Unit test: NewTokenizer("") returns cl100k_base (default)
- Unit test: NewTokenizer("invalid") returns error
- Unit test: cl100k Count("hello world") returns expected token count (should be 2)
- Unit test: o200k Count("hello world") returns expected token count
- Unit test: none estimator Count("hello world") returns len("hello world") / 4 = 2 (integer division)
- Unit test: Count("") returns 0 for all implementations
- Unit test: Count with large text (10KB) returns reasonable values
- Unit test: Count with Unicode/multi-byte characters
- Benchmark: Count on 1KB, 10KB, 100KB texts for each implementation
- Test goroutine safety: run Count concurrently from 10 goroutines