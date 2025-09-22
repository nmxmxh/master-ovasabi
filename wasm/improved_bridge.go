//go:build js && wasm
// +build js,wasm

package main

import (
	"math"
	"sync"
	"syscall/js"
	"time"
)

// Enhanced bridge functions with proper worker support
var (
	workerFunctions = make(map[string]js.Func)
	workerMutex     sync.RWMutex
)

// Expose functions to workers (currently no worker-specific functions needed)
func exposeFunctionsToWorkers() {
	workerMutex.Lock()
	defer workerMutex.Unlock()

	// No worker-specific functions to expose currently
	// Main exports are handled in main.go with improved versions
	// Worker communication is handled through the enhanced worker pool

	wasmLog("[BRIDGE] No worker-specific functions to expose - using main exports")
}

// Note: Worker communication functions removed - not currently used
// Worker communication is handled through the enhanced worker pool

// Enhanced runConcurrentCompute with better worker integration
func runConcurrentComputeImproved(this js.Value, args []js.Value) interface{} {
	if len(args) < 4 {
		wasmLog("[CONCURRENT-COMPUTE-IMPROVED] Requires: inputData, deltaTime, animationMode, callback")
		return js.ValueOf(false)
	}

	inputData := args[0]             // Float32Array
	deltaTime := args[1].Float()     // Time delta
	animationMode := args[2].Float() // Animation mode
	callback := args[3]              // Callback function

	// Validate input - expecting 10 values per particle
	inputLength := inputData.Get("length").Int()
	if inputLength == 0 || inputLength%10 != 0 {
		wasmLog("[CONCURRENT-COMPUTE-IMPROVED] Invalid input data length:", inputLength, "- must be multiple of 10 (10 values per particle)")
		return js.ValueOf(false)
	}

	// Convert JS Float32Array to Go slice
	positions := make([]float32, inputLength)
	for i := 0; i < inputLength; i++ {
		positions[i] = float32(inputData.Index(i).Float())
	}

	// Use enhanced worker pool for better performance
	go func() {
		startTime := time.Now()

		var result []float32
		if enhancedParticleWorkerPool != nil {
			result = enhancedParticleWorkerPool.ProcessParticlesConcurrentlyEnhanced(positions, deltaTime, animationMode)
		} else {
			wasmLog("[CONCURRENT-COMPUTE-IMPROVED] Enhanced worker pool not available, falling back to synchronous")
			// Fallback to synchronous processing with enhanced algorithms
			result = processParticlesSynchronouslyEnhanced(positions, deltaTime, animationMode)
		}

		processingTime := float64(time.Since(startTime).Nanoseconds()) / 1e6

		// Convert result back to JS Float32Array
		resultArray := js.Global().Get("Float32Array").New(len(result))
		for i, val := range result {
			resultArray.SetIndex(i, val)
		}

		// Return memory to pool
		if len(result) > 0 {
			memoryPools.PutFloat32Buffer(result)
		}

		// Invoke callback with result and metrics
		if callback.Type() == js.TypeFunction {
			callback.Invoke(resultArray, js.ValueOf(map[string]interface{}{
				"processingTime": processingTime,
				"method":         "concurrent-improved",
				"particleCount":  inputLength / 10,
				"workerCount": func() int {
					if enhancedParticleWorkerPool != nil {
						return enhancedParticleWorkerPool.workers
					} else if particleWorkerPool != nil {
						return particleWorkerPool.workers
					}
					return 0
				}(),
			}))
		}
	}()

	return js.ValueOf(true)
}

// Initialize the improved bridge system
func initImprovedBridge() {
	// Expose functions to workers
	exposeFunctionsToWorkers()

	// Initialize GPU system using the fixed WebGPU system from bridge.go
	// This ensures consistency with our shader binding fixes
	wasmLog("[BRIDGE] Improved bridge system initialized with fixed WebGPU system")

	// Set up worker communication (using existing functions from bridge.go)
	// Note: getWorkerPoolStatus and benchmarkConcurrentVsGPU already exist in bridge.go
}

// Synchronous fallback with enhanced algorithms
func processParticlesSynchronouslyEnhanced(positions []float32, deltaTime float64, animationMode float64) []float32 {
	if len(positions) == 0 {
		return positions
	}

	valuesPerParticle := 10
	particleCount := len(positions) / valuesPerParticle
	result := make([]float32, len(positions))

	// Use enhanced algorithms from the improved worker
	for i := 0; i < particleCount; i++ {
		baseIndex := i * valuesPerParticle

		// Extract particle data (10 values per particle)
		x, y, z := positions[baseIndex], positions[baseIndex+1], positions[baseIndex+2]
		vx, vy, vz := positions[baseIndex+3], positions[baseIndex+4], positions[baseIndex+5]
		phase := positions[baseIndex+6]
		intensity := positions[baseIndex+7]
		ptype := positions[baseIndex+8]
		pid := positions[baseIndex+9]

		// Enhanced animation logic (same as in improved worker)
		var newX, newY, newZ float32 = x, y, z
		var newVx, newVy, newVz float32 = vx, vy, vz

		globalTime := float32(time.Now().UnixNano())/1e9 + float32(i)*0.001
		animationModeInt := int(animationMode)

		// Enhanced animation patterns (same as improved worker)
		switch animationModeInt {
		case 1: // Enhanced galaxy rotation
			angle := float32(globalTime) * 0.1
			radius := float32(math.Sqrt(float64(x*x + z*z)))
			if radius > 0.001 {
				spiralFactor := float32(1.0 + intensity*0.5)
				newX = x*float32(math.Cos(float64(angle*spiralFactor))) - z*float32(math.Sin(float64(angle*spiralFactor)))
				newZ = x*float32(math.Sin(float64(angle*spiralFactor))) + z*float32(math.Cos(float64(angle*spiralFactor)))
				newY = y + float32(math.Sin(float64(globalTime*2.0)+float64(phase)))*0.1*intensity
				newVx = (newX - x) / 0.016
				newVz = (newZ - z) / 0.016
			}
		case 2: // Enhanced wave motion
			wave := float32(math.Sin(float64(x)*2.0+float64(globalTime)*5.0+float64(phase))) * 0.3
			secondaryWave := float32(math.Sin(float64(z)*1.5+float64(globalTime)*3.0+float64(phase))) * 0.1
			newY = y + (wave+secondaryWave)*intensity*(1.0+ptype*0.2)
			newVx = vx
			newVy = (newY - y) / 0.016
			newVz = vz
		case 3: // Enhanced spiral motion
			radius := float32(math.Sqrt(float64(x*x + z*z)))
			if radius > 0.001 {
				angle := float32(globalTime) * 0.2
				verticalSpiral := float32(math.Sin(float64(globalTime*0.5)+float64(phase))) * 0.2
				newX = x*float32(math.Cos(float64(angle))) - z*float32(math.Sin(float64(angle)))
				newZ = x*float32(math.Sin(float64(angle))) + z*float32(math.Cos(float64(angle)))
				newY = y + verticalSpiral*intensity
				newVx = (newX - x) / 0.016
				newVy = (newY - y) / 0.016
				newVz = (newZ - z) / 0.016
			}
		default: // Enhanced default motion
			randomFactor := float32(math.Sin(float64(globalTime*0.1)+float64(phase))) * 0.05
			newX = x + randomFactor*intensity
			newY = y + float32(math.Sin(float64(globalTime)+float64(phase)))*0.1*intensity
			newZ = z + randomFactor*intensity
			newVx = (newX - x) / 0.016
			newVy = (newY - y) / 0.016
			newVz = (newZ - z) / 0.016
		}

		// Store updated particle data
		result[baseIndex] = newX
		result[baseIndex+1] = newY
		result[baseIndex+2] = newZ
		result[baseIndex+3] = newVx
		result[baseIndex+4] = newVy
		result[baseIndex+5] = newVz
		result[baseIndex+6] = phase
		result[baseIndex+7] = intensity
		result[baseIndex+8] = ptype
		result[baseIndex+9] = pid
	}

	return result
}
