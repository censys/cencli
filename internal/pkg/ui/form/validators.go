package form

import (
	"errors"

	"github.com/google/uuid"
)

// NonEmpty returns a validator that errors with msg if the input is empty.
func NonEmpty(msg string) func(string) error {
	return func(s string) error {
		if s == "" {
			return errors.New(msg)
		}
		return nil
	}
}

func OptionalUUID(msg string) func(string) error {
	return func(s string) error {
		if s == "" {
			return nil
		}
		_, err := uuid.Parse(s)
		if err != nil {
			return errors.New(msg)
		}
		return nil
	}
}

func NonEmptyUUID(msg string) func(string) error {
	return func(s string) error {
		_, err := uuid.Parse(s)
		if err != nil {
			return errors.New(msg)
		}
		return nil
	}
}
