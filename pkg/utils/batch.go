package utils

import "sync"

// BatchProcess runs a function on each item in batches with parallelism.
func BatchProcess[T any](items []T, batchSize int, fn func(batch []T)) {
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]
		var wg sync.WaitGroup
		for _, item := range batch {
			wg.Add(1)
			go func(it T) {
				defer wg.Done()
				fn([]T{it})
			}(item)
		}
		wg.Wait()
	}
}
