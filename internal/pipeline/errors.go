// Package pipeline defines the central data types shared across all pipeline
// stages in Harvx. This file defines the HarvxError type for structured error
// handling with exit codes, enabling commands to communicate specific exit
// codes back to main.go.
package pipeline

import "fmt"

// HarvxError is a custom error type that carries an exit code for structured
// error handling. Commands in the CLI use this to communicate specific exit
// codes back to main.go. It implements the error interface and supports
// unwrapping via errors.Is and errors.As.
type HarvxError struct {
	// Code is the process exit code associated with this error.
	Code int

	// Message is a human-readable description of what went wrong.
	Message string

	// Err is the underlying error that caused this HarvxError, if any.
	Err error
}

// Error returns the formatted error message. If an underlying error is present,
// it is included in the output separated by a colon.
func (e *HarvxError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error, enabling errors.Is and errors.As to
// traverse the error chain.
func (e *HarvxError) Unwrap() error {
	return e.Err
}

// NewError creates a HarvxError with ExitError (1) code for fatal errors.
func NewError(msg string, err error) *HarvxError {
	return &HarvxError{Code: int(ExitError), Message: msg, Err: err}
}

// NewPartialError creates a HarvxError with ExitPartial (2) code for scenarios
// where some files failed processing but output was still generated.
func NewPartialError(msg string, err error) *HarvxError {
	return &HarvxError{Code: int(ExitPartial), Message: msg, Err: err}
}

// NewRedactionError creates a HarvxError with ExitError (1) code for redaction
// failures, such as when --fail-on-redaction is triggered by detected secrets.
func NewRedactionError(msg string) *HarvxError {
	return &HarvxError{Code: int(ExitError), Message: msg}
}
