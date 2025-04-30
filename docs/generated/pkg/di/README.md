# Package di

## Variables

### ErrInterfaceMustBePointer

DI-specific error variables re-exported from pkg/errors.

## Types

### Container

Container manages dependency injection.

#### Methods

##### Clear

Clear removes a specific service or mock.

##### GetConfig

GetConfig retrieves a configuration value.

##### GetInt

GetInt retrieves the configuration value as an int.

##### GetString

GetString retrieves the configuration value as a string.

##### MustResolve

MustResolve resolves a service instance or returns an error.

##### Register

Register registers a service factory.

##### RegisterConfig

RegisterConfig registers a configuration value.

##### RegisterMock

RegisterMock registers a mock implementation for testing.

##### Reset

Reset clears all registered services and mocks.

##### Resolve

Resolve resolves a service instance.

### Factory

Factory is a function that creates an instance of a service.
