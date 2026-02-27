package io

// Reader reads bytes from a source.
type Reader interface {
	Read(p []byte) (n int, err error)
}

// Writer writes bytes to a destination.
type Writer interface {
	Write(p []byte) (n int, err error)
}

// ReadWriter combines Reader and Writer.
type ReadWriter interface {
	Reader
	Writer
}

// Closer closes a resource.
type Closer interface {
	Close() error
}

// ReadWriteCloser combines all three.
type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

// Stringer converts to string.
type Stringer interface {
	String() string
}