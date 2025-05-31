package errors

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

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

// New creates a new error with the given message.
func New(msg string) error {
	return errors.New(msg)
}

// Wrap wraps an error with additional context.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return errors.New(msg + ": " + err.Error())
}

// LogWithError logs the error with context and returns a wrapped error. Use this for standardized error logging across services.
func LogWithError(ctx context.Context, log *zap.Logger, msg string, err error, fields ...zap.Field) error {
	if log != nil {
		if ctx != nil {
			if reqID, ok := ctx.Value("request_id").(string); ok && reqID != "" {
				fields = append(fields, zap.String("request_id", reqID))
			}
		}
		log.Error(msg, append(fields, zap.Error(err))...)
	}
	return Wrap(err, msg)
}
