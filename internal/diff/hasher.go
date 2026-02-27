package diff

// Hasher defines the content hashing operations used for change detection
// between Harvx runs. Implementations must be deterministic: identical input
// always produces identical output.
type Hasher interface {
	// HashBytes computes the 64-bit hash of a byte slice.
	HashBytes(data []byte) uint64

	// HashString computes the 64-bit hash of a string without allocating
	// a copy of the string data.
	HashString(s string) uint64
}
