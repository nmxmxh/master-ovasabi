package errors

import "errors"

var (
	// ErrUserNotFound is returned when a user cannot be found.
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")
	// ErrInvalidCredentials is returned when authentication fails.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserExists is returned when trying to create a duplicate user.
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidToken is returned when a token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned when a token has expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrInvalidEmail is returned when an email format is invalid.
	ErrInvalidEmail = errors.New("invalid email format")
	// ErrWeakPassword is returned when a password is too weak.
	ErrWeakPassword = errors.New("password too weak")
)

// DI container errors.
var (
	// ErrInterfaceMustBePointer is returned when a non-pointer interface is registered.
	ErrInterfaceMustBePointer = errors.New("interface must be a pointer type")
	// ErrMockDoesNotImplement is returned when a mock does not implement the interface.
	ErrMockDoesNotImplement = errors.New("mock does not implement interface")
	// ErrTargetMustBePointer is returned when a non-pointer target is passed to Resolve.
	ErrTargetMustBePointer = errors.New("target must be a pointer")
	// ErrNoFactoryRegistered is returned when no factory is registered for a type.
	ErrNoFactoryRegistered = errors.New("no factory registered")
	// ErrFactoryFailed is returned when the factory fails to create an instance.
	ErrFactoryFailed = errors.New("factory failed to create instance")
)
