package tokenizer

// estimatorTokenizer is the "none" Tokenizer implementation.
// It estimates token count as len(text) / 4, which is the widely accepted
// industry heuristic of approximately 4 characters per token for English text.
//
// This implementation involves no network I/O, no disk access, and no
// initialisation overhead. It is ideal when speed is preferred over accuracy
// or when tiktoken BPE data is not available.
//
// estimatorTokenizer is goroutine-safe: it holds no mutable state.
type estimatorTokenizer struct{}

// newEstimatorTokenizer constructs an estimatorTokenizer.
func newEstimatorTokenizer() *estimatorTokenizer {
	return &estimatorTokenizer{}
}

// Count returns an estimate of the number of tokens in text using the
// heuristic: len(text) / 4 (integer division). Returns 0 for empty text.
// Safe for concurrent use.
func (e *estimatorTokenizer) Count(text string) int {
	return len(text) / 4
}

// Name returns "none", indicating this is the character-count estimator.
func (e *estimatorTokenizer) Name() string {
	return NameNone
}
