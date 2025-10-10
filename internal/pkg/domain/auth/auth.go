package auth

import (
	"errors"
)

// ErrAuthNotFound is returned when no auth value is found.
var ErrAuthNotFound = errors.New("auth not found")
