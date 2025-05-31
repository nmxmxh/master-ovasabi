package utils

import "context"

// StreamItems streams items from a channel to a callback, with cancellation support.
func StreamItems[T any](ctx context.Context, ch <-chan T, fn func(T) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-ch:
			if !ok {
				return nil
			}
			if err := fn(item); err != nil {
				return err
			}
		}
	}
}
