package status

import "fmt"

// Code represents an HTTP-like status code.
type Code int

const (
	// OK indicates success.
	OK Code = iota
	// NotFound indicates the resource was not found.
	NotFound
	// InternalError indicates an internal error.
	InternalError
	// Unauthorized indicates insufficient permissions.
	Unauthorized
)

const MaxRetries = 3

const DefaultTimeout = 30

// Pi is the mathematical constant.
const Pi = 3.14159265358979

// Version information
const (
	Major = 1
	Minor = 2
	Patch = 0
)

var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = fmt.Errorf("not found")
	// ErrTimeout is returned on timeout.
	ErrTimeout = fmt.Errorf("timeout")
)

var globalCount int