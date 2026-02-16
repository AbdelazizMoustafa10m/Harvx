package pipeline

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewError_Code(t *testing.T) {
	t.Parallel()

	err := NewError("something failed", errors.New("underlying"))
	assert.Equal(t, int(ExitError), err.Code)
	assert.Equal(t, 1, err.Code)
}

func TestNewPartialError_Code(t *testing.T) {
	t.Parallel()

	err := NewPartialError("partial failure", errors.New("some files failed"))
	assert.Equal(t, int(ExitPartial), err.Code)
	assert.Equal(t, 2, err.Code)
}

func TestNewRedactionError_Code(t *testing.T) {
	t.Parallel()

	err := NewRedactionError("secrets detected")
	assert.Equal(t, int(ExitError), err.Code)
	assert.Equal(t, 1, err.Code)
}

func TestNewRedactionError_NilUnderlying(t *testing.T) {
	t.Parallel()

	err := NewRedactionError("secrets detected")
	assert.Nil(t, err.Err)
}

func TestHarvxError_ErrorWithUnderlying(t *testing.T) {
	t.Parallel()

	underlying := errors.New("disk full")
	err := NewError("write failed", underlying)
	assert.Equal(t, "write failed: disk full", err.Error())
}

func TestHarvxError_ErrorWithoutUnderlying(t *testing.T) {
	t.Parallel()

	err := NewRedactionError("secrets detected in output")
	assert.Equal(t, "secrets detected in output", err.Error())
}

func TestHarvxError_ErrorMessageFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     *HarvxError
		wantMsg string
	}{
		{
			name:    "error with underlying",
			err:     NewError("processing failed", errors.New("permission denied")),
			wantMsg: "processing failed: permission denied",
		},
		{
			name:    "error without underlying",
			err:     NewRedactionError("redaction triggered"),
			wantMsg: "redaction triggered",
		},
		{
			name:    "partial error with underlying",
			err:     NewPartialError("5 files failed", errors.New("timeout")),
			wantMsg: "5 files failed: timeout",
		},
		{
			name:    "error with nil underlying",
			err:     NewError("generic failure", nil),
			wantMsg: "generic failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestHarvxError_Unwrap(t *testing.T) {
	t.Parallel()

	underlying := errors.New("root cause")
	err := NewError("wrapper", underlying)

	unwrapped := err.Unwrap()
	assert.Equal(t, underlying, unwrapped)
}

func TestHarvxError_UnwrapNil(t *testing.T) {
	t.Parallel()

	err := NewRedactionError("no underlying")
	assert.Nil(t, err.Unwrap())
}

func TestHarvxError_ErrorsIs(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel error")
	harvxErr := NewError("wrapped sentinel", sentinel)

	assert.True(t, errors.Is(harvxErr, sentinel),
		"errors.Is should find the sentinel through HarvxError.Unwrap")
}

func TestHarvxError_ErrorsIsChained(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("deep sentinel")
	wrapped := fmt.Errorf("mid-level: %w", sentinel)
	harvxErr := NewError("top-level", wrapped)

	assert.True(t, errors.Is(harvxErr, sentinel),
		"errors.Is should traverse the full chain")
}

func TestHarvxError_ErrorsAs(t *testing.T) {
	t.Parallel()

	harvxErr := NewPartialError("partial", errors.New("some failed"))

	// Wrap the HarvxError in a standard error chain.
	wrappedErr := fmt.Errorf("command failed: %w", harvxErr)

	var target *HarvxError
	require.True(t, errors.As(wrappedErr, &target),
		"errors.As should extract HarvxError from wrapped chain")
	assert.Equal(t, int(ExitPartial), target.Code)
	assert.Equal(t, "partial", target.Message)
}

func TestHarvxError_ErrorsAsDirectly(t *testing.T) {
	t.Parallel()

	harvxErr := NewError("direct", errors.New("cause"))

	var target *HarvxError
	require.True(t, errors.As(harvxErr, &target))
	assert.Equal(t, int(ExitError), target.Code)
}

func TestHarvxError_ImplementsErrorInterface(t *testing.T) {
	t.Parallel()

	// Compile-time check that *HarvxError implements error.
	var _ error = (*HarvxError)(nil)

	// Runtime check.
	var err error = NewError("test", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "test", err.Error())
}

func TestHarvxError_ErrorsIsWithStdlibErrors(t *testing.T) {
	t.Parallel()

	// Wrap a standard library error type (fs.ErrNotExist) in HarvxError.
	harvxErr := NewError("file not found", fs.ErrNotExist)

	assert.True(t, errors.Is(harvxErr, fs.ErrNotExist),
		"errors.Is should find fs.ErrNotExist through HarvxError")
}

func TestNewError_PreservesMessage(t *testing.T) {
	t.Parallel()

	err := NewError("custom message", errors.New("cause"))
	assert.Equal(t, "custom message", err.Message)
}

func TestNewPartialError_PreservesMessage(t *testing.T) {
	t.Parallel()

	err := NewPartialError("partial message", errors.New("cause"))
	assert.Equal(t, "partial message", err.Message)
}

func TestNewRedactionError_PreservesMessage(t *testing.T) {
	t.Parallel()

	err := NewRedactionError("redaction message")
	assert.Equal(t, "redaction message", err.Message)
}

func TestHarvxError_ErrorsIsNonMatching(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("expected sentinel")
	other := errors.New("different sentinel")
	harvxErr := NewError("wrapped", sentinel)

	assert.False(t, errors.Is(harvxErr, other),
		"errors.Is should return false when sentinel does not match")
}

func TestHarvxError_ErrorsAsNonMatching(t *testing.T) {
	t.Parallel()

	// A plain error that is NOT a *HarvxError should not match errors.As.
	plainErr := fmt.Errorf("plain: %w", errors.New("cause"))

	var target *HarvxError
	assert.False(t, errors.As(plainErr, &target),
		"errors.As should return false when chain contains no HarvxError")
}

func TestNewError_UnwrapNilUnderlying(t *testing.T) {
	t.Parallel()

	// NewError with nil underlying should also return nil from Unwrap,
	// distinct from the NewRedactionError case tested in TestHarvxError_UnwrapNil.
	err := NewError("no cause", nil)
	assert.Nil(t, err.Unwrap())
}

func TestNewPartialError_UnwrapNilUnderlying(t *testing.T) {
	t.Parallel()

	err := NewPartialError("partial no cause", nil)
	assert.Nil(t, err.Unwrap())
}

func TestHarvxError_EmptyMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     *HarvxError
		wantMsg string
	}{
		{
			name:    "NewError empty message no underlying",
			err:     NewError("", nil),
			wantMsg: "",
		},
		{
			name:    "NewError empty message with underlying",
			err:     NewError("", errors.New("cause")),
			wantMsg: ": cause",
		},
		{
			name:    "NewPartialError empty message",
			err:     NewPartialError("", nil),
			wantMsg: "",
		},
		{
			name:    "NewRedactionError empty message",
			err:     NewRedactionError(""),
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestHarvxError_ErrorsIsNilTarget(t *testing.T) {
	t.Parallel()

	// HarvxError with nil underlying should NOT match nil sentinel via errors.Is.
	// errors.Is(err, nil) returns true only when err is nil.
	harvxErr := NewError("msg", nil)
	assert.False(t, errors.Is(harvxErr, nil),
		"errors.Is(nonNilErr, nil) should return false")
}
