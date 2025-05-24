package user

import "errors"

var (
	ErrUserExists            = errors.New("user already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidUsername       = errors.New("invalid username")
	ErrUsernameReserved      = errors.New("username is reserved")
	ErrUsernameTaken         = errors.New("username is already taken")
	ErrUsernameBadWord       = errors.New("username contains inappropriate content")
	ErrUsernameInvalidFormat = errors.New("username contains invalid characters or format")
)
