package tokenizer

import (
	"fmt"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// tiktokenTokenizer is a Tokenizer backed by pkoukk/tiktoken-go.
// It uses the BPE encoding specified by encodingName to count tokens.
// The encoding is initialised once on construction; Count is goroutine-safe
// because tiktoken-go's Encode method does not mutate shared state.
type tiktokenTokenizer struct {
	name string
	enc  *tiktoken.Tiktoken
}

// newTiktokenTokenizer constructs a tiktokenTokenizer for the given BPE
// encoding name (e.g. "cl100k_base" or "o200k_base").
//
// The encoding is loaded from disk (or the TIKTOKEN_CACHE_DIR cache) exactly
// once. An error is returned if the encoding cannot be loaded.
func newTiktokenTokenizer(encodingName string) (*tiktokenTokenizer, error) {
	enc, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, fmt.Errorf("initialising tiktoken encoding %q: %w", encodingName, err)
	}

	return &tiktokenTokenizer{
		name: encodingName,
		enc:  enc,
	}, nil
}

// Count returns the exact number of BPE tokens in text.
// Returns 0 for empty text. Safe for concurrent use.
func (t *tiktokenTokenizer) Count(text string) int {
	if text == "" {
		return 0
	}
	tokens := t.enc.Encode(text, nil, nil)
	return len(tokens)
}

// Name returns the BPE encoding name (e.g. "cl100k_base").
func (t *tiktokenTokenizer) Name() string {
	return t.name
}
