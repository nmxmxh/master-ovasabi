# Package tracing

Package tracing provides OpenTelemetry tracing initialization and configuration.

## Types

### Config

Config holds the configuration for tracing initialization.

## Functions

### Init

Init initializes OpenTelemetry tracing with the provided configuration. It returns a TracerProvider
and a shutdown function that should be called when the application exits.

### Shutdown

Shutdown gracefully shuts down the TracerProvider.
