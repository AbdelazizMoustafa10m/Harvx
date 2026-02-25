package diff

import "errors"

// ErrBranchMismatch is returned when loading a cached state whose GitBranch
// differs from the current branch.
var ErrBranchMismatch = errors.New("cached state branch mismatch")

// ErrNoState is returned when attempting to load a state that does not exist.
var ErrNoState = errors.New("no cached state found")

// ErrInvalidVersion is returned when the state file has an invalid or empty
// version field.
var ErrInvalidVersion = errors.New("invalid state version")
