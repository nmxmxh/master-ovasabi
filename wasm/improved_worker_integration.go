//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"math"
	"sync"
	"syscall/js"
	"time"
)

// Enhanced worker pool with better task distribution
type EnhancedParticleWorkerPool struct {
	workers    int
	tasks      chan ParticleTask
	results    chan ParticleResult
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mutex      sync.RWMutex
	activeJobs int
}

// Enhanced particle task with better metadata
type EnhancedParticleTask struct {
	ID            string
	ChunkIndex    int
	StartIndex    int
	EndIndex      int
	Positions     []float32
	DeltaTime     float64
	AnimationMode float64
	Priority      int
	Timestamp     time.Time
	Metadata      map[string]interface{}
}

// Enhanced particle result with performance metrics
type EnhancedParticleResult struct {
	ID                 string
	ChunkIndex         int
	ProcessedPositions []float32
	StartIndex         int
	EndIndex           int
	MemoryUsed         int
	ProcessingTime     float64
	WorkerID           int
	Metadata           map[string]interface{}
}

// Global enhanced worker pool
var enhancedParticleWorkerPool *EnhancedParticleWorkerPool

// Create enhanced particle worker pool
func NewEnhancedParticleWorkerPool(workerCount int) *EnhancedParticleWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &EnhancedParticleWorkerPool{
		workers: workerCount,
		tasks:   make(chan ParticleTask, workerCount*2),
		results: make(chan ParticleResult, workerCount*2),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start workers
	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.enhancedWorker(i)
	}

	wasmLog("[ENHANCED-WORKER-POOL] Created with", workerCount, "workers")
	return pool
}

// Enhanced worker with better error handling and performance monitoring
func (pool *EnhancedParticleWorkerPool) enhancedWorker(id int) {
	defer pool.wg.Done()

	wasmLog("[ENHANCED-WORKER", id, "] Started")

	for {
		select {
		case task := <-pool.tasks:
			pool.mutex.Lock()
			pool.activeJobs++
			pool.mutex.Unlock()

			startTime := time.Now()
			result := pool.processParticleChunkEnhanced(task, id)
			processingTime := float64(time.Since(startTime).Nanoseconds()) / 1e6

			result.ProcessingTime = processingTime
			// Note: WorkerID field doesn't exist in original ParticleResult struct

			select {
			case pool.results <- result:
				// Successfully sent result
			case <-pool.ctx.Done():
				wasmLog("[ENHANCED-WORKER", id, "] Shutting down - context cancelled")
				return
			}

			pool.mutex.Lock()
			pool.activeJobs--
			pool.mutex.Unlock()

		case <-pool.ctx.Done():
			wasmLog("[ENHANCED-WORKER", id, "] Shutting down - context cancelled")
			return
		}
	}
}

// Enhanced particle chunk processing with better algorithms
func (pool *EnhancedParticleWorkerPool) processParticleChunkEnhanced(task ParticleTask, workerID int) ParticleResult {
	// Enhanced processing with better algorithms
	valuesPerParticle := 10
	chunkSize := task.EndIndex - task.StartIndex

	if chunkSize <= 0 || chunkSize%valuesPerParticle != 0 {
		wasmError("[ENHANCED-WORKER] Invalid particle chunk size:", chunkSize)
		return ParticleResult{
			ID:                 task.ID,
			ChunkIndex:         task.ChunkIndex,
			ProcessedPositions: nil,
			StartIndex:         task.StartIndex,
			EndIndex:           task.EndIndex,
			MemoryUsed:         0,
		}
	}

	processedPositions := memoryPools.GetFloat32Buffer(chunkSize)

	// Enhanced particle processing with better algorithms
	for i := task.StartIndex; i+valuesPerParticle <= task.EndIndex && i+9 < len(task.Positions); i += valuesPerParticle {
		particleIndex := (i - task.StartIndex) / valuesPerParticle

		if i < 0 || i+9 >= len(task.Positions) {
			wasmError("[ENHANCED-WORKER] Particle data index out of bounds:", i)
			break
		}

		// Extract particle data (10 values per particle)
		x, y, z := task.Positions[i], task.Positions[i+1], task.Positions[i+2]
		vx, vy, vz := task.Positions[i+3], task.Positions[i+4], task.Positions[i+5]
		phase := task.Positions[i+6]
		intensity := task.Positions[i+7]
		ptype := task.Positions[i+8]
		pid := task.Positions[i+9]

		// Enhanced animation logic with better algorithms
		var newX, newY, newZ float32 = x, y, z
		var newVx, newVy, newVz float32 = vx, vy, vz

		globalTime := float32(time.Now().UnixNano())/1e9 + float32(particleIndex)*0.001
		animationMode := int(task.AnimationMode)

		// Enhanced animation patterns
		switch animationMode {
		case 1: // Enhanced galaxy rotation
			angle := float32(globalTime) * 0.1
			radius := float32(math.Sqrt(float64(x*x + z*z)))
			if radius > 0.001 {
				// Add spiral motion
				spiralFactor := float32(1.0 + intensity*0.5)
				newX = x*float32(math.Cos(float64(angle*spiralFactor))) - z*float32(math.Sin(float64(angle*spiralFactor)))
				newZ = x*float32(math.Sin(float64(angle*spiralFactor))) + z*float32(math.Cos(float64(angle*spiralFactor)))
				newY = y + float32(math.Sin(float64(globalTime*2.0)+float64(phase)))*0.1*intensity
				newVx = (newX - x) / 0.016
				newVz = (newZ - z) / 0.016
			}
		case 2: // Enhanced wave motion
			wave := float32(math.Sin(float64(x)*2.0+float64(globalTime)*5.0+float64(phase))) * 0.3
			// Add secondary wave for complexity
			secondaryWave := float32(math.Sin(float64(z)*1.5+float64(globalTime)*3.0+float64(phase))) * 0.1
			newY = y + (wave+secondaryWave)*intensity*(1.0+ptype*0.2)
			newVx = vx
			newVy = (newY - y) / 0.016
			newVz = vz
		case 3: // Enhanced spiral motion
			radius := float32(math.Sqrt(float64(x*x + z*z)))
			if radius > 0.001 {
				angle := float32(globalTime) * 0.2
				// Add vertical spiral component
				verticalSpiral := float32(math.Sin(float64(globalTime*0.5)+float64(phase))) * 0.2
				newX = x*float32(math.Cos(float64(angle))) - z*float32(math.Sin(float64(angle)))
				newZ = x*float32(math.Sin(float64(angle))) + z*float32(math.Cos(float64(angle)))
				newY = y + verticalSpiral*intensity
				newVx = (newX - x) / 0.016
				newVy = (newY - y) / 0.016
				newVz = (newZ - z) / 0.016
			}
		default: // Enhanced default motion
			// Add subtle random motion
			randomFactor := float32(math.Sin(float64(globalTime*0.1)+float64(phase))) * 0.05
			newX = x + randomFactor*intensity
			newY = y + float32(math.Sin(float64(globalTime)+float64(phase)))*0.1*intensity
			newZ = z + randomFactor*intensity
			newVx = (newX - x) / 0.016
			newVy = (newY - y) / 0.016
			newVz = (newZ - z) / 0.016
		}

		// Store updated particle data (10 values per particle)
		resultIndex := i - task.StartIndex
		processedPositions[resultIndex] = newX
		processedPositions[resultIndex+1] = newY
		processedPositions[resultIndex+2] = newZ
		processedPositions[resultIndex+3] = newVx
		processedPositions[resultIndex+4] = newVy
		processedPositions[resultIndex+5] = newVz
		processedPositions[resultIndex+6] = phase
		processedPositions[resultIndex+7] = intensity
		processedPositions[resultIndex+8] = ptype
		processedPositions[resultIndex+9] = pid
	}

	return ParticleResult{
		ID:                 task.ID,
		ChunkIndex:         task.ChunkIndex,
		ProcessedPositions: processedPositions,
		StartIndex:         task.StartIndex,
		EndIndex:           task.EndIndex,
		MemoryUsed:         chunkSize * 4, // bytes
	}
}

// Enhanced concurrent particle processing
func (pool *EnhancedParticleWorkerPool) ProcessParticlesConcurrentlyEnhanced(positions []float32, deltaTime float64, animationMode float64) []float32 {
	if len(positions) == 0 {
		return positions
	}

	valuesPerParticle := 10
	particleCount := len(positions) / valuesPerParticle
	chunkSize := particleCount / pool.workers
	if chunkSize < 1 {
		chunkSize = 1
	}

	// Create tasks
	tasks := make([]ParticleTask, 0)
	for i := 0; i < particleCount; i += chunkSize {
		endIndex := i + chunkSize
		if endIndex > particleCount {
			endIndex = particleCount
		}

		task := ParticleTask{
			ID:            fmt.Sprintf("task_%d_%d", i, time.Now().UnixNano()),
			ChunkIndex:    len(tasks),
			StartIndex:    i * valuesPerParticle,
			EndIndex:      endIndex * valuesPerParticle,
			Positions:     positions,
			DeltaTime:     deltaTime,
			AnimationMode: animationMode,
			Priority:      1,
		}
		tasks = append(tasks, task)
	}

	// Submit tasks
	for _, task := range tasks {
		select {
		case pool.tasks <- task:
			// Task submitted successfully
		case <-pool.ctx.Done():
			wasmLog("[ENHANCED-WORKER-POOL] Context cancelled during task submission")
			return positions
		}
	}

	// Collect results
	results := make([]ParticleResult, len(tasks))
	for i := 0; i < len(tasks); i++ {
		select {
		case result := <-pool.results:
			results[result.ChunkIndex] = result
		case <-pool.ctx.Done():
			wasmLog("[ENHANCED-WORKER-POOL] Context cancelled during result collection")
			return positions
		}
	}

	// Merge results
	result := make([]float32, len(positions))
	for _, res := range results {
		if res.ProcessedPositions != nil {
			copy(result[res.StartIndex:res.EndIndex], res.ProcessedPositions)
		}
	}

	return result
}

// Get enhanced worker pool status
func (pool *EnhancedParticleWorkerPool) GetStatus() map[string]interface{} {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	return map[string]interface{}{
		"workers":     pool.workers,
		"activeJobs":  pool.activeJobs,
		"taskQueue":   len(pool.tasks),
		"resultQueue": len(pool.results),
		"initialized": true,
	}
}

// Shutdown enhanced worker pool
func (pool *EnhancedParticleWorkerPool) Shutdown() {
	wasmLog("[ENHANCED-WORKER-POOL] Shutting down...")
	pool.cancel()
	pool.wg.Wait()
	close(pool.tasks)
	close(pool.results)
	wasmLog("[ENHANCED-WORKER-POOL] Shutdown complete")
}

// Initialize enhanced worker integration
func initEnhancedWorkerIntegration() {
	// Create enhanced worker pool
	workerCount := maxWorkers
	if workerCount <= 0 {
		workerCount = 1
	}
	enhancedParticleWorkerPool = NewEnhancedParticleWorkerPool(workerCount)

	// Expose enhanced functions to workers
	js.Global().Set("getEnhancedWorkerPoolStatus", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if enhancedParticleWorkerPool == nil {
			return js.ValueOf(map[string]interface{}{
				"initialized": false,
			})
		}
		return js.ValueOf(enhancedParticleWorkerPool.GetStatus())
	}))

	wasmLog("[ENHANCED-WORKER-INTEGRATION] Initialized with", workerCount, "workers")
}
