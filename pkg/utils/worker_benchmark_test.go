package utils

import (
	"context"
	"testing"
)

// noopTask is a Task implementation that does nothing
type noopTask struct{}

func (noopTask) Process(ctx context.Context) error { return nil }

// BenchmarkWorkerPool_HeavyLoad measures throughput of sequential submissions
func BenchmarkWorkerPool_HeavyLoad(b *testing.B) {
	pool := NewWorkerPool(100)
	pool.Start()
	defer pool.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pool.Submit(noopTask{}); err != nil {
			b.Fatalf("Submit failed: %v", err)
		}
	}
}

// BenchmarkWorkerPool_ParallelHeavyLoad measures throughput under parallel submissions
func BenchmarkWorkerPool_ParallelHeavyLoad(b *testing.B) {
	pool := NewWorkerPool(100)
	pool.Start()
	defer pool.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := pool.Submit(noopTask{}); err != nil {
				b.Fatalf("Submit failed in parallel: %v", err)
			}
		}
	})
}
