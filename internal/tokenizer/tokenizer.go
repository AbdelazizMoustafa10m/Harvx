// Package tokenizer provides token counting implementations for LLM context
// documents. It defines the Tokenizer interface and a factory function for
// creating the appropriate implementation based on an encoding name.
//
// Three implementations are provided:
//   - cl100k_base: OpenAI/Anthropic compatible BPE tokenizer (default)
//   - o200k_base:  GPT-4o/o1 BPE tokenizer
//   - none:        Fast character-count estimator (~4 chars per token)
//
// All implementations are goroutine-safe.
package tokenizer

import (
	"fmt"
)

// Tokenizer counts tokens in text content.
// All implementations must be safe for concurrent use from multiple goroutines.
type Tokenizer interface {
	// Count returns the number of tokens in the given text.
	// Returns 0 for empty text. Never returns a negative value.
	Count(text string) int

	// Name returns the tokenizer encoding name (e.g., "cl100k_base").
	// The name is used in pipeline metadata to identify which tokenizer
	// was used when generating a context document.
	Name() string
}

// Supported tokenizer encoding names.
const (
	// NameCL100K is the cl100k_base BPE encoding used by GPT-4 and Claude models.
	// This is the default encoding when an empty string is passed to NewTokenizer.
	NameCL100K = "cl100k_base"

	// NameO200K is the o200k_base BPE encoding used by GPT-4o and OpenAI o1/o3 models.
	NameO200K = "o200k_base"

	// NameNone selects the character-count estimator: len(text) / 4.
	// Use this for maximum speed when exact token counts are not required.
	NameNone = "none"
)

// ErrUnknownTokenizer is returned by NewTokenizer when an unrecognised
// encoding name is provided. Callers can check for this with errors.Is.
var ErrUnknownTokenizer = fmt.Errorf("unknown tokenizer")

// NewTokenizer returns a Tokenizer for the given encoding name.
//
// Supported names are "cl100k_base", "o200k_base", and "none".
// Passing an empty string returns the default cl100k_base tokenizer.
//
// The tiktoken BPE encodings (cl100k_base, o200k_base) are initialised once
// on construction. Subsequent Count calls are cheap and goroutine-safe.
//
// tiktoken-go respects the TIKTOKEN_CACHE_DIR environment variable for
// caching downloaded BPE dictionaries.
func NewTokenizer(name string) (Tokenizer, error) {
	if name == "" {
		name = NameCL100K
	}

	switch name {
	case NameCL100K, NameO200K:
		return newTiktokenTokenizer(name)
	case NameNone:
		return newEstimatorTokenizer(), nil
	default:
		return nil, fmt.Errorf("%w: %q (supported: cl100k_base, o200k_base, none)", ErrUnknownTokenizer, name)
	}
}
