//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"syscall/js"
	"time"
)

// ParticleWorkerPool manages concurrent particle processing
type ParticleWorkerPool struct {
	workers int
	tasks   chan ParticleTask
	results chan ParticleResult
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	active  bool
	mutex   sync.Mutex
}

// ParticleTask represents work for a single chunk of particles
type ParticleTask struct {
	ID            string
	ChunkIndex    int
	Positions     []float32
	StartIndex    int
	EndIndex      int
	DeltaTime     float64
	AnimationMode float64
	Priority      int // 0=high, 1=normal, 2=low
}

// ParticleResult contains processed particle data
type ParticleResult struct {
	ID                 string
	ChunkIndex         int
	ProcessedPositions []float32
	StartIndex         int
	EndIndex           int
	ProcessingTime     float64
	MemoryUsed         int
}

// ComputeTask represents a general computation task
type ComputeTask struct {
	ID        string
	Type      string // "particles", "physics", "ai", "transform"
	Data      []float32
	Params    map[string]float64
	Callback  js.Value
	Priority  int
	Timestamp time.Time
}

// NewParticleWorkerPool creates a new worker pool optimized for particle processing
func NewParticleWorkerPool(workerCount int) *ParticleWorkerPool {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &ParticleWorkerPool{
		workers: workerCount,
		tasks:   make(chan ParticleTask, workerCount*4), // Buffer for burst processing
		results: make(chan ParticleResult, workerCount*4),
		ctx:     ctx,
		cancel:  cancel,
		active:  false,
	}
}

// Start initializes and starts all worker goroutines
func (pool *ParticleWorkerPool) Start() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if pool.active {
		return // Already started
	}

	for i := 0; i < pool.workers; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	pool.active = true
}

// Stop gracefully shuts down the worker pool
func (pool *ParticleWorkerPool) Stop() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if !pool.active {
		return
	}

	pool.cancel()
	pool.wg.Wait()
	pool.active = false
	wasmLog("[WORKER-POOL] Stopped all workers")
}

// ProcessParticlesConcurrently distributes particle processing across workers
func (pool *ParticleWorkerPool) ProcessParticlesConcurrently(
	positions []float32,
	deltaTime float64,
	animationMode float64,
) []float32 {
	if !pool.active {
		pool.Start()
	}

	numParticles := len(positions) / 8 // 8 values per particle
	chunkSize := numParticles / pool.workers
	if chunkSize < 1000 {
		chunkSize = 1000
	}
	numChunks := (numParticles + chunkSize - 1) / chunkSize
	taskID := fmt.Sprintf("concurrent_%d", time.Now().UnixNano())
	tasks := make([]ParticleTask, numChunks)
	for idx := range tasks {
		start := idx * chunkSize
		end := start + chunkSize
		if end > numParticles {
			end = numParticles
		}
		tasks[idx] = ParticleTask{
			ID:            fmt.Sprintf("%s_chunk_%d", taskID, idx),
			ChunkIndex:    idx,
			Positions:     positions,
			StartIndex:    start * 8,
			EndIndex:      end * 8,
			DeltaTime:     deltaTime,
			AnimationMode: animationMode,
			Priority:      1,
		}
	}
	// Distribute work to workers
	for _, task := range tasks {
		select {
		case pool.tasks <- task:
		case <-time.After(time.Millisecond * 100):
			wasmLog("[WORKER-POOL] Task queue full, processing synchronously")
			return pool.processSynchronously(positions, deltaTime, animationMode)
		}
	}
	// Collect results
	results := make([]ParticleResult, numChunks)
	totalProcessingTime := 0.0
	for i := 0; i < len(results); i++ {
		select {
		case result := <-pool.results:
			results[result.ChunkIndex] = result
			totalProcessingTime += result.ProcessingTime
		case <-time.After(time.Second):
			wasmLog("[WORKER-POOL] Result timeout, falling back to synchronous processing")
			return pool.processSynchronously(positions, deltaTime, animationMode)
		}
	}
	// Combine results efficiently
	output := memoryPools.GetFloat32Buffer(len(positions))
	for _, result := range results {
		copy(output[result.StartIndex:result.EndIndex], result.ProcessedPositions)
		memoryPools.PutFloat32Buffer(result.ProcessedPositions)
	}
	return output
}

// processSynchronously is a fallback for when concurrent processing fails
func (pool *ParticleWorkerPool) processSynchronously(positions []float32, deltaTime float64, animationMode float64) []float32 {
	output := memoryPools.GetFloat32Buffer(len(positions))
	copy(output, positions)

	// Simple synchronous processing
	// Data format: position(3) + velocity(3) + time(1) + intensity(1) = 8 values per particle
	for i := 0; i < len(positions); i += 8 {
		particleIndex := i / 8

		switch int(animationMode) {
		case 1: // Simple rotation
			x, z := output[i], output[i+2]
			radius := math.Sqrt(float64(x*x + z*z))
			if radius > 0.001 {
				angle := math.Atan2(float64(z), float64(x)) + deltaTime*0.5
				output[i] = float32(radius * math.Cos(angle))
				output[i+2] = float32(radius * math.Sin(angle))
			}
		default: // Simple wave
			output[i+1] += float32(math.Sin(deltaTime*3.0+float64(particleIndex)*0.01) * 0.1)
		}
	}

	return output
}

// getWorkerPoolStatus returns current worker pool metrics
func getWorkerPoolStatus(this js.Value, args []js.Value) interface{} {
	if particleWorkerPool == nil {
		return js.ValueOf(map[string]interface{}{
			"initialized": false,
			"workers":     0,
			"active":      false,
		})
	}

	particleWorkerPool.mutex.Lock()
	active := particleWorkerPool.active
	workers := particleWorkerPool.workers
	taskQueueLength := len(particleWorkerPool.tasks)
	resultQueueLength := len(particleWorkerPool.results)
	particleWorkerPool.mutex.Unlock()

	return js.ValueOf(map[string]interface{}{
		"initialized":        true,
		"workers":            workers,
		"active":             active,
		"taskQueueLength":    taskQueueLength,
		"resultQueueLength":  resultQueueLength,
		"computeQueueLength": len(computeTaskQueue),
		"memoryPools":        getMemoryPoolStats(),
	})
}
