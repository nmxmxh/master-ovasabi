package di

import (
	errs "github.com/nmxmxh/master-ovasabi/pkg/errors"
)

// DI-specific error variables re-exported from pkg/errors.
var (
	// ErrInterfaceMustBePointer is returned when a non-pointer interface is registered.
	ErrInterfaceMustBePointer = errs.ErrInterfaceMustBePointer
	// ErrMockDoesNotImplement is returned when a mock does not implement the interface.
	ErrMockDoesNotImplement = errs.ErrMockDoesNotImplement
	// ErrTargetMustBePointer is returned when a non-pointer target is passed to Resolve.
	ErrTargetMustBePointer = errs.ErrTargetMustBePointer
	// ErrNoFactoryRegistered is returned when no factory is registered for a type.
	ErrNoFactoryRegistered = errs.ErrNoFactoryRegistered
	// ErrFactoryFailed is returned when the factory fails to create an instance.
	ErrFactoryFailed = errs.ErrFactoryFailed
)
