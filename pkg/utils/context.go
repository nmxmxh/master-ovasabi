package utils

import (
	"context"
	"time"
)

// DefaultTimeout is the default timeout for operations
const DefaultTimeout = 30 * time.Second

// ContextWithTimeout creates a context with the default timeout
func ContextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultTimeout)
}

// ContextWithCustomTimeout creates a context with a custom timeout
func ContextWithCustomTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// ContextWithDeadline creates a context with a deadline
func ContextWithDeadline(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, deadline)
}

// MergeContexts creates a new context that is canceled when any of the input contexts are canceled
func MergeContexts(contexts ...context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()

		done := make(chan struct{})
		for _, c := range contexts {
			go func(c context.Context) {
				select {
				case <-c.Done():
					close(done)
				case <-ctx.Done():
				}
			}(c)
		}

		<-done
	}()

	return ctx, cancel
}

// WithValue adds a value to the context with type safety
func WithValue[T any](ctx context.Context, key interface{}, value T) context.Context {
	return context.WithValue(ctx, key, value)
}

// GetValue retrieves a value from the context with type safety
func GetValue[T any](ctx context.Context, key interface{}) (T, bool) {
	value := ctx.Value(key)
	if value == nil {
		var zero T
		return zero, false
	}

	typed, ok := value.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return typed, true
}
