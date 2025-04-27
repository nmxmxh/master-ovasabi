package user

import "errors"

var (
	// ErrUserNotFound is returned when a user cannot be found.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists is returned when attempting to create a user that already exists.
	ErrUserExists = errors.New("user already exists")
)
