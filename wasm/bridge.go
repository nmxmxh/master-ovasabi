//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"syscall/js"
	"time"
	"unsafe"
)

// Import shared state from main.go and gpu.go
// These variables/functions must be declared in main.go/gpu.go and available here
var sharedBuffer = make([]float32, 8192) // Shared animation/state buffer (32KB)
var lastFrameTime float64
var perfLogger *PerformanceLogger
var pendingRequests sync.Map
var particleWorkerPool *ParticleWorkerPool
var memoryPools *MemoryPoolManager
var computeTaskQueue chan ComputeTask

// --- WebGPU persistent buffers and state ---
// Global GPU backend status for detection/fallback
var gpuBackend string = "unknown"
var bindGroupLayout js.Value // Store bind group layout for ping-pong updates

// --- WebGPU persistent buffers and state ---
// --- WebGPU persistent buffers and state ---

// Handles JS/WASM bridge functions and related logic.
// getGPUBackend returns the current GPU backend and capabilities for frontend integration
func getGPUBackend(this js.Value, args []js.Value) interface{} {
	caps := js.Global().Get("__WASM_GPU_CAPABILITIES")
	return js.ValueOf(map[string]interface{}{
		"backend":      gpuBackend,
		"capabilities": caps,
	})
}

// getSharedBuffer returns a JS ArrayBuffer view of the shared buffer
func getSharedBuffer(this js.Value, args []js.Value) interface{} {
	// Guard against zero-length slice
	if len(sharedBuffer) == 0 {
		return js.Null()
	}
	// Convert []float32 to []byte without allocation
	hdr := (*[1 << 30]byte)(unsafe.Pointer(&sharedBuffer[0]))[: len(sharedBuffer)*4 : len(sharedBuffer)*4]
	uint8Array := js.Global().Get("Uint8Array").New(len(hdr))
	js.CopyBytesToJS(uint8Array, hdr)
	return uint8Array.Get("buffer")
}

// getGPUMetricsBuffer returns dedicated GPU metrics buffer
func getGPUMetricsBuffer(this js.Value, args []js.Value) interface{} {
	if len(gpuMetricsBuffer) == 0 {
		return js.Null()
	}
	hdr := (*[1 << 30]byte)(unsafe.Pointer(&gpuMetricsBuffer[0]))[: len(gpuMetricsBuffer)*4 : len(gpuMetricsBuffer)*4]
	uint8Array := js.Global().Get("Uint8Array").New(len(hdr))
	js.CopyBytesToJS(uint8Array, hdr)
	return uint8Array.Get("buffer")
}

// getGPUComputeBuffer returns dedicated GPU compute buffer
func getGPUComputeBuffer(this js.Value, args []js.Value) interface{} {
	if len(gpuComputeBuffer) == 0 {
		return js.Null()
	}
	hdr := (*[1 << 30]byte)(unsafe.Pointer(&gpuComputeBuffer[0]))[: len(gpuComputeBuffer)*4 : len(gpuComputeBuffer)*4]
	uint8Array := js.Global().Get("Uint8Array").New(len(hdr))
	js.CopyBytesToJS(uint8Array, hdr)
	return uint8Array.Get("buffer")
}

// initWebGPU initializes WebGPU access centrally in WASM with concurrent workers
func initWebGPU(this js.Value, args []js.Value) interface{} {
	gpuMutex.Lock()
	defer gpuMutex.Unlock()

	if gpuInitialized {
		wasmLog("[WASM-GPU] WebGPU already initialized")
		return js.ValueOf(true)
	}

	// Robust GPU detection and fallback logic
	navigator := js.Global().Get("navigator")
	gpu := navigator.Get("gpu")
	if !gpu.Truthy() {
		wasmLog("[WASM-GPU] WebGPU not available, falling back to WASM/JS compute")
		gpuBackend = "fallback"
		gpuInitialized = false
		// Push GPU capabilities to frontend global state for UI/metadata integration
		capabilities := map[string]interface{}{
			"backend":  "none",
			"f16":      false,
			"features": []interface{}{},
			"limits":   map[string]interface{}{},
			"fallback": true,
		}
		js.Global().Set("__WASM_GPU_CAPABILITIES", js.ValueOf(capabilities))
		// If global state manager exists, update device metadata
		if js.Global().Get("useGlobalStore").Type() == js.TypeFunction {
			js.Global().Call("useGlobalStore").Call("setMetadata", js.ValueOf(map[string]interface{}{
				"device": map[string]interface{}{
					"gpuCapabilities": capabilities,
					"gpuBackend":      gpuBackend,
				},
			}))
		}
		return js.ValueOf(false)
	}

	// Initialize WebGPU asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				wasmLog("[WASM-GPU] WebGPU initialization panic:", r)
				gpuMutex.Lock()
				gpuInitialized = false
				gpuMutex.Unlock()
			}
		}()

		// Request adapter
		adapterPromise := gpu.Call("requestAdapter", js.ValueOf(map[string]interface{}{
			"powerPreference": "high-performance",
		}))

		adapterPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			adapter := args[0]
			if adapter.IsNull() {
				wasmLog("[WASM-GPU] WebGPU adapter not available, falling back to WASM/JS compute")
				gpuBackend = "fallback"
				gpuInitialized = false
				capabilities := map[string]interface{}{
					"backend":  "none",
					"f16":      false,
					"features": []interface{}{},
					"limits":   map[string]interface{}{},
					"fallback": true,
				}
				js.Global().Set("__WASM_GPU_CAPABILITIES", js.ValueOf(capabilities))
				if js.Global().Get("useGlobalStore").Type() == js.TypeFunction {
					js.Global().Call("useGlobalStore").Call("setMetadata", js.ValueOf(map[string]interface{}{
						"device": map[string]interface{}{
							"gpuCapabilities": capabilities,
							"gpuBackend":      gpuBackend,
						},
					}))
				}
				return nil
			}

			gpuAdapter = adapter
			wasmLog("[WASM-GPU] WebGPU adapter acquired")

			// Detect features and limits
			features := adapter.Get("features")
			hasF16 := false
			if features.Truthy() && features.Call("has", js.ValueOf("shader-f16")).Truthy() {
				hasF16 = true
			}
			limits := adapter.Get("limits")
			backendName := "unknown"
			if adapter.Get("name").Truthy() {
				backendName = adapter.Get("name").String()
			}

			// Request device with required features and higher buffer limit if needed
			deviceDesc := map[string]interface{}{}
			if hasF16 {
				deviceDesc["requiredFeatures"] = []interface{}{"shader-f16"}
			}
			// --- Buffer size logic ---
			// Use the largest buffer size needed in your pipeline
			desiredBufferSize := 320000000 // bytes (example, match your largest buffer)
			maxBufferSupported := limits.Get("maxBufferSize")
			maxStorageSupported := limits.Get("maxStorageBufferBindingSize")
			requiredLimits := map[string]interface{}{}
			if maxBufferSupported.Type() == js.TypeNumber && maxBufferSupported.Int() < desiredBufferSize {
				requiredLimits["maxBufferSize"] = desiredBufferSize
			} else if maxBufferSupported.Type() == js.TypeNumber {
				requiredLimits["maxBufferSize"] = maxBufferSupported.Int()
			}
			// Always request the highest supported storage buffer binding size
			if maxStorageSupported.Type() == js.TypeNumber {
				requiredLimits["maxStorageBufferBindingSize"] = maxStorageSupported.Int()
			} else {
				// Fallback to a safe default if not available
				requiredLimits["maxStorageBufferBindingSize"] = 134217728
			}
			if len(requiredLimits) > 0 {
				deviceDesc["requiredLimits"] = requiredLimits
			}
			devicePromise := adapter.Call("requestDevice", js.ValueOf(deviceDesc))

			devicePromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				device := args[0]
				if device.IsNull() {
					wasmLog("[WASM-GPU] WebGPU device not available, falling back to WASM/JS compute")
					gpuBackend = "fallback"
					gpuInitialized = false
					capabilities := map[string]interface{}{
						"backend":  backendName,
						"f16":      hasF16,
						"features": features,
						"limits":   limits,
						"fallback": true,
					}
					js.Global().Set("__WASM_GPU_CAPABILITIES", js.ValueOf(capabilities))
					if js.Global().Get("useGlobalStore").Type() == js.TypeFunction {
						js.Global().Call("useGlobalStore").Call("setMetadata", js.ValueOf(map[string]interface{}{
							"device": map[string]interface{}{
								"gpuCapabilities": capabilities,
								"gpuBackend":      gpuBackend,
							},
						}))
					}
					return nil
				}

				gpuDevice = device
				wasmLog("[WASM-GPU] WebGPU device acquired, backend:", backendName, "f16 support:", hasF16)
				gpuBackend = backendName
				gpuInitialized = true

				// Store capabilities for frontend metadata
				capabilities := map[string]interface{}{
					"backend":  backendName,
					"f16":      hasF16,
					"features": features,
					"limits":   limits,
					"fallback": false,
				}
				js.Global().Set("__WASM_GPU_CAPABILITIES", js.ValueOf(capabilities))
				if js.Global().Get("useGlobalStore").Type() == js.TypeFunction {
					js.Global().Call("useGlobalStore").Call("setMetadata", js.ValueOf(map[string]interface{}{
						"device": map[string]interface{}{
							"gpuCapabilities": capabilities,
							"gpuBackend":      gpuBackend,
						},
					}))
				}

				// Initialize compute pipeline
				initComputePipeline()

				// Notify frontend
				notifyGPUReady()

				return nil
			}))

			return nil
		}))
	}()

	return js.ValueOf(true)
}

// initComputePipeline creates the standardized GPU compute pipeline
func initComputePipeline() {
	if gpuDevice.IsNull() {
		wasmLog("[WASM-GPU] Cannot initialize compute pipeline: device not available")
		return
	}

	// WGSL shader for 10-value-per-particle format
	computeShaderCode := `
struct Params {
	time: f32,
	animationMode: f32,
	globalOffset: f32,
	intensityScale: f32,
	particleCount: u32,
}
@group(0) @binding(0) var<storage, read> inputData: array<f32>;
@group(0) @binding(1) var<storage, read_write> outputData: array<f32>;
@group(0) @binding(2) var<storage, read> originalPositions: array<f32>;
@group(0) @binding(3) var<uniform> params: Params;

@compute @workgroup_size(256)
fn shade(@builtin(global_invocation_id) global_id: vec3<u32>) {
	let particleIndex = global_id.x;
	if (particleIndex >= params.particleCount) {
		return;
	}
	let baseIndex = particleIndex * 10u;
	let origBaseIndex = particleIndex * 3u;
	// Load particle data
	let pos = vec3<f32>(inputData[baseIndex], inputData[baseIndex+1u], inputData[baseIndex+2u]);
	let vel = vec3<f32>(inputData[baseIndex+3u], inputData[baseIndex+4u], inputData[baseIndex+5u]);
	let phase = inputData[baseIndex+6u];
	let intensity = inputData[baseIndex+7u] * params.intensityScale;
	let ptype = inputData[baseIndex+8u];
	let pid = inputData[baseIndex+9u];
	let origPos = vec3<f32>(originalPositions[origBaseIndex], originalPositions[origBaseIndex+1u], originalPositions[origBaseIndex+2u]);
	// Animation parameters
	let globalTime = params.time + f32(particleIndex) * 0.001;
	let animationMode = params.animationMode;
	// Example: wave motion with phase/type
	let wave = sin(origPos.x * 2.0 + globalTime * 5.0 + phase) * 0.3;
	let newPos = vec3<f32>(origPos.x, origPos.y + wave * intensity * (1.0 + ptype * 0.2), origPos.z);
	let newVel = (newPos - pos) / max(params.time, 0.0001);
	// Write results
	outputData[baseIndex] = newPos.x;
	outputData[baseIndex+1u] = newPos.y;
	outputData[baseIndex+2u] = newPos.z;
	outputData[baseIndex+3u] = newVel.x;
	outputData[baseIndex+4u] = newVel.y;
	outputData[baseIndex+5u] = newVel.z;
	outputData[baseIndex+6u] = phase;
	outputData[baseIndex+7u] = intensity;
	outputData[baseIndex+8u] = ptype;
	outputData[baseIndex+9u] = pid;
}
	`

	// Create shader module
	shaderModule := gpuDevice.Call("createShaderModule", js.ValueOf(map[string]interface{}{
		"code": computeShaderCode,
	}))

	// --- Persistent buffer sizing for up to 10 million particles ---
	constMaxParticles := 10000000                   // 10 million
	bufferSize := constMaxParticles * 10 * 4        // bytes for all particles (10 floats per particle)
	originalBufferSize := constMaxParticles * 3 * 4 // bytes for original positions (3 floats per particle)

	// Input buffer (current 8-value particle data)
	gpuInputBuffer = gpuDevice.Call("createBuffer", js.ValueOf(map[string]interface{}{
		"size":  bufferSize,
		"usage": js.Global().Get("GPUBufferUsage").Get("STORAGE").Int() | js.Global().Get("GPUBufferUsage").Get("COPY_DST").Int(),
		"label": "GPU Input Buffer - 8-value particles (pos+vel+time+intensity)",
	}))

	// Output buffers (ping-pong)
	gpuOutputBufferA = gpuDevice.Call("createBuffer", js.ValueOf(map[string]interface{}{
		"size": bufferSize,
		"usage": js.Global().Get("GPUBufferUsage").Get("STORAGE").Int() |
			js.Global().Get("GPUBufferUsage").Get("COPY_SRC").Int() |
			js.Global().Get("GPUBufferUsage").Get("VERTEX").Int(),
		"label": "GPU Output Buffer A - 10-value particles",
	}))
	gpuOutputBufferB = gpuDevice.Call("createBuffer", js.ValueOf(map[string]interface{}{
		"size": bufferSize,
		"usage": js.Global().Get("GPUBufferUsage").Get("STORAGE").Int() |
			js.Global().Get("GPUBufferUsage").Get("COPY_SRC").Int() |
			js.Global().Get("GPUBufferUsage").Get("VERTEX").Int(),
		"label": "GPU Output Buffer B - 10-value particles",
	}))
	gpuOutputPingPong = false          // Start with A
	gpuOutputBuffer = gpuOutputBufferA // Ensure bind group uses a valid buffer

	// Original positions buffer (3-value reference positions)
	gpuOriginalBuffer = gpuDevice.Call("createBuffer", js.ValueOf(map[string]interface{}{
		"size":  originalBufferSize,
		"usage": js.Global().Get("GPUBufferUsage").Get("STORAGE").Int() | js.Global().Get("GPUBufferUsage").Get("COPY_DST").Int(),
		"label": "GPU Original Positions Buffer - 3-value reference positions",
	}))

	// Persistent staging buffer (for readback)
	gpuStagingBuffer = gpuDevice.Call("createBuffer", js.ValueOf(map[string]interface{}{
		"size":  bufferSize,
		"usage": js.Global().Get("GPUBufferUsage").Get("MAP_READ").Int() | js.Global().Get("GPUBufferUsage").Get("COPY_DST").Int(),
		"label": "GPU Staging Buffer - 8-value particle readback",
	}))

	// Params buffer: 5 floats (time, mode, offset, intensityScale, particleCount)
	gpuParamsBuffer = gpuDevice.Call("createBuffer", js.ValueOf(map[string]interface{}{
		"size":  20, // 5 floats * 4 bytes
		"usage": js.Global().Get("GPUBufferUsage").Get("UNIFORM").Int() | js.Global().Get("GPUBufferUsage").Get("COPY_DST").Int(),
	}))

	// Create bind group layout
	bindGroupLayout = gpuDevice.Call("createBindGroupLayout", js.ValueOf(map[string]interface{}{
		"entries": []interface{}{
			map[string]interface{}{
				"binding":    0,
				"visibility": js.Global().Get("GPUShaderStage").Get("COMPUTE").Int(),
				"buffer": map[string]interface{}{
					"type": "read-only-storage", // Input data (current positions)
				},
			},
			map[string]interface{}{
				"binding":    1,
				"visibility": js.Global().Get("GPUShaderStage").Get("COMPUTE").Int(),
				"buffer": map[string]interface{}{
					"type": "storage", // Output data (computed positions)
				},
			},
			map[string]interface{}{
				"binding":    2,
				"visibility": js.Global().Get("GPUShaderStage").Get("COMPUTE").Int(),
				"buffer": map[string]interface{}{
					"type": "read-only-storage", // Original positions (immutable reference)
				},
			},
			map[string]interface{}{
				"binding":    3,
				"visibility": js.Global().Get("GPUShaderStage").Get("COMPUTE").Int(),
				"buffer": map[string]interface{}{
					"type": "uniform", // Animation parameters
				},
			},
		},
	}))

	// Create pipeline layout
	pipelineLayout := gpuDevice.Call("createPipelineLayout", js.ValueOf(map[string]interface{}{
		"bindGroupLayouts": []interface{}{bindGroupLayout},
	}))

	// Create compute pipeline
	gpuPipeline = gpuDevice.Call("createComputePipeline", js.ValueOf(map[string]interface{}{
		"layout": pipelineLayout,
		"compute": map[string]interface{}{
			"module":     shaderModule,
			"entryPoint": "shade",
		},
	}))

	// Validate buffers before creating bind group
	if gpuInputBuffer.IsNull() || gpuOutputBuffer.IsNull() || gpuOriginalBuffer.IsNull() || gpuParamsBuffer.IsNull() {
		wasmLog("[WASM-GPU] ERROR: One or more GPU buffers are undefined/null before bind group creation!")
		if gpuInputBuffer.IsNull() {
			wasmLog("[WASM-GPU] gpuInputBuffer is NULL!")
		}
		if gpuOutputBuffer.IsNull() {
			wasmLog("[WASM-GPU] gpuOutputBuffer is NULL!")
		}
		if gpuOriginalBuffer.IsNull() {
			wasmLog("[WASM-GPU] gpuOriginalBuffer is NULL!")
		}
		if gpuParamsBuffer.IsNull() {
			wasmLog("[WASM-GPU] gpuParamsBuffer is NULL!")
		}
		return
	}

	// Create bind group with validated buffers
	gpuBindGroup = gpuDevice.Call("createBindGroup", js.ValueOf(map[string]interface{}{
		"layout": bindGroupLayout,
		"entries": []interface{}{
			map[string]interface{}{
				"binding":  0,
				"resource": map[string]interface{}{"buffer": gpuInputBuffer},
			},
			map[string]interface{}{
				"binding":  1,
				"resource": map[string]interface{}{"buffer": gpuOutputBuffer},
			},
			map[string]interface{}{
				"binding":  2,
				"resource": map[string]interface{}{"buffer": gpuOriginalBuffer},
			},
			map[string]interface{}{
				"binding":  3,
				"resource": map[string]interface{}{"buffer": gpuParamsBuffer},
			},
		},
	}))

	wasmLog("[WASM-GPU] Compute pipeline initialized successfully")

	// Add GPU cleanup handler for memory management
	js.Global().Set("cleanupGPU", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wasmLog("[WASM-GPU] Cleaning up GPU resources...")
		if !gpuInputBuffer.IsNull() {
			gpuInputBuffer.Call("destroy")
		}
		if !gpuOutputBuffer.IsNull() {
			gpuOutputBuffer.Call("destroy")
		}
		if !gpuOriginalBuffer.IsNull() {
			gpuOriginalBuffer.Call("destroy")
		}
		if !gpuStagingBuffer.IsNull() {
			gpuStagingBuffer.Call("destroy")
		}
		if !gpuParamsBuffer.IsNull() {
			gpuParamsBuffer.Call("destroy")
		}
		gpuInitialized = false
		wasmLog("[WASM-GPU] GPU resources cleaned up successfully")
		return nil
	}))

	// Expose compute state streaming for backend access
	js.Global().Set("getComputeState", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if !gpuInitialized {
			return js.Null()
		}

		// Return comprehensive compute state for backend analysis
		bufferSize := len(gpuComputeBuffer) * 4
		state := map[string]interface{}{
			"gpuInitialized": gpuInitialized,
			"bufferSize":     bufferSize,                 // bytes
			"particleCount":  len(gpuComputeBuffer) / 10, // 10 values per particle
			"timestamp":      float64(time.Now().UnixMilli()),
			"lastFrameTime":  lastFrameTime,
			"metrics": map[string]interface{}{
				"gpuMemoryUsage":   bufferSize * 4, // All GPU buffers
				"wasmMemoryUsage":  len(sharedBuffer) + len(gpuMetricsBuffer) + len(gpuComputeBuffer),
				"activeOperations": "particle_compute", // Could track active operations
			},
		}

		return js.ValueOf(state)
	}))

	// Expose backend-controlled parameter updates
	js.Global().Set("setComputeParams", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 || !gpuInitialized {
			return js.ValueOf(false)
		}

		params := args[0]
		wasmLog("[WASM-GPU] Backend updating compute parameters...")

		// Backend can control animation parameters
		if !params.Get("animationStrength").IsUndefined() {
			strength := params.Get("animationStrength").Float()
			wasmLog("[WASM-GPU] Backend set animation strength:", strength)
			// Could store this in a global variable for use in compute
		}

		if !params.Get("animationMode").IsUndefined() {
			mode := params.Get("animationMode").Float()
			wasmLog("[WASM-GPU] Backend set animation mode:", mode)
			// Could override the time-based mode cycling
		}

		return js.ValueOf(true)
	}))

	// Add device lost error handling
	if !gpuDevice.IsNull() {
		gpuDevice.Set("onuncapturederror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			wasmLog("[WASM-GPU] GPU device error detected, attempting recovery...")
			// Attempt to reinitialize the pipeline
			go func() {
				time.Sleep(100 * time.Millisecond) // Brief delay
				if gpuDevice.IsNull() {
					wasmLog("[WASM-GPU] Device lost, skipping recovery")
					return
				}
				initComputePipeline()
			}()
			return nil
		}))
	}
}

// runGPUCompute executes GPU computation directly without worker pool overhead
func runGPUCompute(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		wasmLog("[WASM-GPU] runGPUCompute requires: inputData, operation, callback")
		return js.ValueOf(false)
	}

	inputData := args[0]         // Float32Array
	operation := args[1].Float() // 0=perf, 1=particle, 2=ai
	callback := args[2]          // Function

	// Process directly for much better performance
	go func() {
		resultPromise := processGPUTaskDirect(inputData, operation)
		if !resultPromise.IsNull() && callback.Type() == js.TypeFunction {
			// Handle the Promise properly
			resultPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				result := args[0] // The actual result array
				callback.Invoke(result)
				return nil
			})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				wasmLog("[WASM-GPU] Error processing GPU task:", args[0].String())
				return nil
			}))
		}
	}()

	return js.ValueOf(true)
}

// runGPUComputeWithOffset - synchronized GPU compute with global particle offset
func runGPUComputeWithOffset(this js.Value, args []js.Value) interface{} {
	if len(args) < 4 {
		wasmLog("[WASM-GPU] runGPUComputeWithOffset requires: inputData, elapsedTime, globalParticleOffset, callback")
		return js.ValueOf(false)
	}

	inputData := args[0]                    // Float32Array
	elapsedTime := args[1].Float()          // Elapsed time in seconds
	globalParticleOffset := args[2].Float() // Global particle offset for synchronization
	callback := args[3]                     // Function

	// Process with synchronized timing and offset
	go func() {
		resultPromise := processGPUTaskDirectWithOffset(inputData, elapsedTime, globalParticleOffset)
		if !resultPromise.IsNull() && callback.Type() == js.TypeFunction {
			// Handle the Promise properly
			resultPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				result := args[0] // The actual result array
				callback.Invoke(result)
				return nil
			})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				wasmLog("[WASM-GPU] Error processing synchronized GPU task:", args[0].String())
				return nil
			}))
		}
	}()

	return js.ValueOf(true)
}

// processGPUTaskDirect handles GPU computation directly without worker pool overhead
func processGPUTaskDirect(inputData js.Value, operation float64) js.Value {
	gpuMutex.Lock()
	defer gpuMutex.Unlock()

	if !gpuInitialized || gpuDevice.IsNull() {
		wasmLog("[WASM-GPU] GPU not initialized")
		return js.Null()
	}

	// Validate input size
	inputLength := inputData.Get("length").Int()
	if inputLength == 0 {
		wasmLog("[WASM-GPU] Empty input data")
		return js.Null()
	}

	// Debug: Log for large operations using aggregated logging
	if inputLength > 200000 {
		perfLogger.LogSuccess("gpu_direct_processing", int64(inputLength))
	}

	queue := gpuDevice.Get("queue")
	queue.Call("writeBuffer", gpuInputBuffer, 0, inputData)

	// Prepare params buffer: [time, animationMode, globalOffset, intensityScale, particleCount]
	paramsArray := js.Global().Get("Float32Array").New(5)
	var frameTime float64
	if operation > 100 {
		frameTime = operation / 1000.0
	} else {
		frameTime = operation
	}
	var animationMode float64 = 1.2 // Default to wave
	paramsArray.SetIndex(0, frameTime)
	paramsArray.SetIndex(1, animationMode)
	paramsArray.SetIndex(2, 0)                       // globalOffset
	paramsArray.SetIndex(3, 1.0)                     // intensityScale
	paramsArray.SetIndex(4, float64(inputLength/10)) // particleCount
	queue.Call("writeBuffer", gpuParamsBuffer, 0, paramsArray)

	// Create command encoder and dispatch
	commandEncoder := gpuDevice.Call("createCommandEncoder")
	computePass := commandEncoder.Call("beginComputePass")
	computePass.Call("setPipeline", gpuPipeline)
	computePass.Call("setBindGroup", 0, gpuBindGroup)

	// --- Workgroup dispatch based on particle count ---
	particleCount := inputLength / 10
	workgroupSize := 256
	workgroups := (particleCount + workgroupSize - 1) / workgroupSize
	computePass.Call("dispatchWorkgroups", workgroups)
	computePass.Call("end")

	// Submit and read results directly
	commandBuffer := commandEncoder.Call("finish")
	queue.Call("submit", []interface{}{commandBuffer})

	// --- Ping-pong buffer swap logic ---
	// Swap output buffer for next pass
	gpuOutputPingPong = !gpuOutputPingPong
	if gpuOutputPingPong {
		gpuOutputBuffer = gpuOutputBufferB
	} else {
		gpuOutputBuffer = gpuOutputBufferA
	}
	// Update bind group to use the new output buffer
	gpuBindGroup = gpuDevice.Call("createBindGroup", js.ValueOf(map[string]interface{}{
		"layout": bindGroupLayout,
		"entries": []interface{}{
			map[string]interface{}{
				"binding":  0,
				"resource": map[string]interface{}{"buffer": gpuInputBuffer},
			},
			map[string]interface{}{
				"binding":  1,
				"resource": map[string]interface{}{"buffer": gpuOutputBuffer},
			},
			map[string]interface{}{
				"binding":  2,
				"resource": map[string]interface{}{"buffer": gpuOriginalBuffer},
			},
			map[string]interface{}{
				"binding":  3,
				"resource": map[string]interface{}{"buffer": gpuParamsBuffer},
			},
		},
	}))

	// Create staging buffer for results
	stagingSize := inputLength * 4 // 4 bytes per float32
	stagingBuffer := gpuDevice.Call("createBuffer", js.ValueOf(map[string]interface{}{
		"size":  stagingSize,
		"usage": js.Global().Get("GPUBufferUsage").Get("COPY_DST").Int() | js.Global().Get("GPUBufferUsage").Get("MAP_READ").Int(),
	}))

	// Copy output to staging
	copyEncoder := gpuDevice.Call("createCommandEncoder")
	copyEncoder.Call("copyBufferToBuffer", gpuOutputBuffer, 0, stagingBuffer, 0, stagingSize)
	copyCommandBuffer := copyEncoder.Call("finish")
	queue.Call("submit", []interface{}{copyCommandBuffer})

	// Return a promise-like structure - we'll use async reading
	resultPromise := js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]

		// Map and read results asynchronously
		mapPromise := stagingBuffer.Call("mapAsync", js.Global().Get("GPUMapMode").Get("READ").Int())
		mapPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			var resultCopy js.Value
			defer func() {
				stagingBuffer.Call("unmap")
			}()
			mappedRange := stagingBuffer.Call("getMappedRange")
			resultArray := js.Global().Get("Float32Array").New(mappedRange)

			// Debug: Sample some values to verify GPU computation
			resultLength := resultArray.Get("length").Int()
			if resultLength >= 6 && inputLength >= 6 {
				input0 := inputData.Index(0).Float()
				input1 := inputData.Index(1).Float()
				input2 := inputData.Index(2).Float()
				result0 := resultArray.Index(0).Float()
				result1 := resultArray.Index(1).Float()
				result2 := resultArray.Index(2).Float()

				// Only sample animation effectiveness every 30 seconds via aggregated logging
				if time.Now().UnixMilli()%30000 < 100 {
					// Calculate animation effectiveness for sampling
					diff0 := result0 - input0
					diff1 := result1 - input1
					diff2 := result2 - input2
					_ = diff0 // Suppress unused variable warnings - these are for debugging
					_ = diff1
					_ = diff2
					perfLogger.LogSuccess("gpu_animation_debug_sample", 1)
				}
			}

			// Create a copy for the callback
			resultCopy = js.Global().Get("Float32Array").New(resultArray.Get("length").Int())
			for i := 0; i < resultArray.Get("length").Int(); i++ {
				resultCopy.SetIndex(i, resultArray.Index(i))
			}
			resolve.Invoke(resultCopy)
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// If mapAsync or result copy fails, reject the promise
			if len(args) > 0 {
				reject.Invoke(args[0])
			} else {
				reject.Invoke(js.ValueOf("GPU mapAsync or result copy failed"))
			}
			return nil
		}))

		return nil
	}))

	return resultPromise
}

// processGPUTaskDirectWithOffset handles GPU computation with delta time and original position reference
func processGPUTaskDirectWithOffset(inputData js.Value, elapsedTime float64, globalParticleOffset float64) js.Value {
	gpuMutex.Lock()
	defer gpuMutex.Unlock()

	if !gpuInitialized || gpuDevice.IsNull() {
		wasmLog("[WASM-GPU] GPU not initialized for synchronized compute")
		return js.Null()
	}

	// Validate input size
	inputLength := inputData.Get("length").Int()
	if inputLength == 0 {
		wasmLog("[WASM-GPU] Empty input data for synchronized compute")
		return js.Null()
	}

	// Debug: Log synchronized processing info using aggregated logging
	if inputLength > 200000 {
		perfLogger.LogSuccess("gpu_synchronized_processing", int64(inputLength))
	}

	queue := gpuDevice.Get("queue")

	// Write current positions to input buffer
	queue.Call("writeBuffer", gpuInputBuffer, 0, inputData)

	// Initialize original positions if this is the first frame or new particle data
	// Check if original positions buffer needs initialization
	if gpuOriginalPositions == nil || len(gpuOriginalPositions) != inputLength {
		// Initialize original positions from current input data (bulk copy)
		gpuOriginalPositions = make([]float32, inputLength)
		// Bulk copy from JS to Go
		js.CopyBytesToGo(
			(*[1 << 30]byte)(unsafe.Pointer(&gpuOriginalPositions[0]))[:inputLength*4],
			inputData,
		)

		// Write original positions to GPU buffer (bulk copy)
		originalData := js.Global().Get("Float32Array").New(inputLength)
		js.CopyBytesToJS(
			originalData,
			(*[1 << 30]byte)(unsafe.Pointer(&gpuOriginalPositions[0]))[:inputLength*4],
		)
		queue.Call("writeBuffer", gpuOriginalBuffer, 0, originalData)

		// Use aggregated logging instead of verbose per-operation logging
		perfLogger.LogSuccess("gpu_original_positions_init", int64(inputLength))
	}

	// Calculate delta time (limit to prevent large jumps)
	deltaTime := elapsedTime - lastFrameTime
	if deltaTime > 0.1 { // Cap at 100ms to prevent large jumps
		deltaTime = 0.0167 // Default to 60fps timing
	}
	if deltaTime < 0 { // Handle time resets
		deltaTime = 0.0167
	}
	lastFrameTime = elapsedTime

	// Set pure displacement parameters with delta time and adaptive quality
	paramsArray := js.Global().Get("Float32Array").New(4)

	// Determine animation mode based on time patterns for smooth animation
	// Animation mode mapping:
	// 1 = Galaxy rotation
	// 2 = Yin-Yang flow
	// 3 = Wave motion
	// 4 = Spiral motion
	var animationMode int = 3 // Default to wave mode for smooth demo

	// Cycle through animation modes every 15 seconds for variety
	timeInCycle := elapsedTime - (math.Floor(elapsedTime/15.0) * 15.0)
	if timeInCycle < 3.75 {
		animationMode = 1 // Galaxy rotation
	} else if timeInCycle < 7.5 {
		animationMode = 2 // Yin-Yang flow
	} else if timeInCycle < 11.25 {
		animationMode = 3 // Wave motion
	} else {
		animationMode = 4 // Spiral motion
	}

	// Calculate animation strength based on performance (adaptive quality)
	animationStrength := 1.0
	if deltaTime > 0.025 { // If frame time > 25ms (< 40fps)
		animationStrength = 0.7 // Reduce animation complexity
	} else if deltaTime > 0.0167 { // If frame time > 16.7ms (< 60fps)
		animationStrength = 0.85
	}

	paramsArray.SetIndex(0, deltaTime)              // Delta time for consistent animation
	paramsArray.SetIndex(1, float64(animationMode)) // Animation mode for proper behavior (integer)
	paramsArray.SetIndex(2, globalParticleOffset)   // Global particle offset for sync
	paramsArray.SetIndex(3, animationStrength)      // Animation strength (adaptive quality)

	queue.Call("writeBuffer", gpuParamsBuffer, 0, paramsArray)

	// Create command encoder and dispatch
	commandEncoder := gpuDevice.Call("createCommandEncoder")
	computePass := commandEncoder.Call("beginComputePass")
	computePass.Call("setPipeline", gpuPipeline)
	computePass.Call("setBindGroup", 0, gpuBindGroup)

	// Calculate workgroups
	workgroups := (inputLength + 255) / 256
	computePass.Call("dispatchWorkgroups", workgroups)
	computePass.Call("end")

	// Submit and read results directly
	commandBuffer := commandEncoder.Call("finish")
	queue.Call("submit", []interface{}{commandBuffer})

	// Create staging buffer for results
	stagingSize := inputLength * 4 // 4 bytes per float32
	stagingBuffer := gpuDevice.Call("createBuffer", js.ValueOf(map[string]interface{}{
		"size":  stagingSize,
		"usage": js.Global().Get("GPUBufferUsage").Get("COPY_DST").Int() | js.Global().Get("GPUBufferUsage").Get("MAP_READ").Int(),
	}))

	// Copy output to staging
	copyEncoder := gpuDevice.Call("createCommandEncoder")
	copyEncoder.Call("copyBufferToBuffer", gpuOutputBuffer, 0, stagingBuffer, 0, stagingSize)
	copyCommandBuffer := copyEncoder.Call("finish")
	queue.Call("submit", []interface{}{copyCommandBuffer})

	// Return a promise-like structure for async reading
	resultPromise := js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]

		// Map and read results asynchronously
		mapPromise := stagingBuffer.Call("mapAsync", js.Global().Get("GPUMapMode").Get("READ").Int())
		mapPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			var resultCopy js.Value
			defer func() {
				stagingBuffer.Call("unmap")
			}()
			mappedRange := stagingBuffer.Call("getMappedRange")
			resultArray := js.Global().Get("Float32Array").New(mappedRange)

			// Create a copy for the callback
			resultCopy = js.Global().Get("Float32Array").New(resultArray.Get("length").Int())
			for i := 0; i < resultArray.Get("length").Int(); i++ {
				resultCopy.SetIndex(i, resultArray.Index(i))
			}
			resolve.Invoke(resultCopy)
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// If mapAsync or result copy fails, reject the promise
			if len(args) > 0 {
				reject.Invoke(args[0])
			} else {
				reject.Invoke(js.ValueOf("GPU mapAsync or result copy failed"))
			}
			return nil
		}))

		return nil
	}))

	return resultPromise
}

// notifyGPUReady notifies frontend that GPU is ready
func notifyGPUReady() {
	if onMsgHandler := js.Global().Get("onWasmMessage"); onMsgHandler.Type() == js.TypeFunction {
		gpuEvent := js.Global().Get("Object").New()
		gpuEvent.Set("type", "gpu_ready")
		gpuEvent.Set("payload", js.ValueOf(map[string]interface{}{
			"initialized": true,
			"timestamp":   time.Now().Unix(),
		}))
		gpuEvent.Set("metadata", js.ValueOf(map[string]interface{}{
			"source": "wasm_gpu_system",
		}))
		onMsgHandler.Invoke(gpuEvent)
	}
}

// goEventToJSValue converts Go EventEnvelope to proper JavaScript object
func goEventToJSValue(event EventEnvelope) js.Value {
	jsObj := js.Global().Get("Object").New()

	// Set type
	jsObj.Set("type", event.Type)

	// Convert payload - properly handle JSON vs raw bytes
	if len(event.Payload) > 0 {
		var payloadObj interface{}
		if err := json.Unmarshal(event.Payload, &payloadObj); err == nil {
			// It's valid JSON - convert to JS object
			jsObj.Set("payload", goValueToJSValue(payloadObj))
		} else {
			// It's raw bytes - convert to Uint8Array
			uint8Array := js.Global().Get("Uint8Array").New(len(event.Payload))
			js.CopyBytesToJS(uint8Array, event.Payload)
			jsObj.Set("payload", uint8Array)
		}
	}

	// Convert metadata - ensure it's a proper JS object
	if len(event.Metadata) > 0 {
		var metadataObj interface{}
		if err := json.Unmarshal(event.Metadata, &metadataObj); err == nil {
			jsObj.Set("metadata", goValueToJSValue(metadataObj))
		} else {
			// Fallback to empty object
			jsObj.Set("metadata", js.Global().Get("Object").New())
		}
	} else {
		jsObj.Set("metadata", js.Global().Get("Object").New())
	}

	return jsObj
}

// goValueToJSValue recursively converts Go interface{} to JavaScript values
func goValueToJSValue(v interface{}) js.Value {
	switch val := v.(type) {
	case nil:
		return js.Null()
	case bool:
		return js.ValueOf(val)
	case int, int8, int16, int32, int64:
		return js.ValueOf(val)
	case uint, uint8, uint16, uint32, uint64:
		return js.ValueOf(val)
	case float32, float64:
		return js.ValueOf(val)
	case string:
		return js.ValueOf(val)
	case []interface{}:
		// Array
		jsArray := js.Global().Get("Array").New(len(val))
		for i, item := range val {
			jsArray.SetIndex(i, goValueToJSValue(item))
		}
		return jsArray
	case map[string]interface{}:
		// Object
		jsObj := js.Global().Get("Object").New()
		for k, item := range val {
			jsObj.Set(k, goValueToJSValue(item))
		}
		return jsObj
	default:
		// Fallback: convert to JSON string
		if jsonBytes, err := json.Marshal(val); err == nil {
			return js.ValueOf(string(jsonBytes))
		}
		return js.ValueOf("")
	}
}

// jsValueToEventEnvelope converts JavaScript value to Go EventEnvelope
func jsValueToEventEnvelope(jsVal js.Value) (EventEnvelope, error) {
	var event EventEnvelope

	// Handle string input (JSON)
	if jsVal.Type() == js.TypeString {
		jsonStr := jsVal.String()
		if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
			return event, fmt.Errorf("failed to unmarshal JSON string: %w", err)
		}
		return event, nil
	}

	// Handle object input
	if jsVal.Type() == js.TypeObject {
		// Extract type
		if typeVal := jsVal.Get("type"); typeVal.Type() == js.TypeString {
			event.Type = typeVal.String()
		} else {
			return event, fmt.Errorf("event type is missing or not a string")
		}

		// Extract payload
		if payloadVal := jsVal.Get("payload"); !payloadVal.IsUndefined() {
			payloadGo := jsValueToGoValue(payloadVal)
			if payloadBytes, err := json.Marshal(payloadGo); err == nil {
				event.Payload = payloadBytes
			} else {
				return event, fmt.Errorf("failed to marshal payload: %w", err)
			}
		}

		// Extract metadata
		if metadataVal := jsVal.Get("metadata"); !metadataVal.IsUndefined() {
			metadataGo := jsValueToGoValue(metadataVal)
			if metadataBytes, err := json.Marshal(metadataGo); err == nil {
				event.Metadata = metadataBytes
			} else {
				return event, fmt.Errorf("failed to marshal metadata: %w", err)
			}
		} else {
			event.Metadata = json.RawMessage(`{}`)
		}

		return event, nil
	}

	return event, fmt.Errorf("unsupported JavaScript value type: %s", jsVal.Type().String())
}

// jsValueToGoValue recursively converts a JS value to a Go interface{}
func jsValueToGoValue(v js.Value) interface{} {
	switch v.Type() {
	case js.TypeString:
		return v.String()
	case js.TypeNumber:
		return v.Float()
	case js.TypeBoolean:
		return v.Bool()
	case js.TypeObject:
		if v.InstanceOf(js.Global().Get("Array")) {
			// Handle arrays
			length := v.Get("length").Int()
			result := make([]interface{}, length)
			for i := 0; i < length; i++ {
				result[i] = jsValueToGoValue(v.Index(i))
			}
			return result
		} else {
			// Handle objects
			result := make(map[string]interface{})
			keys := js.Global().Get("Object").Call("keys", v)
			for i := 0; i < keys.Length(); i++ {
				key := keys.Index(i).String()
				result[key] = jsValueToGoValue(v.Get(key))
			}
			return result
		}
	default:
		// null, undefined, etc.
		return nil
	}
}

// jsSendWasmMessage handles Frontend→WASM type conversion at the boundary
func jsSendWasmMessage(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		wasmLog("[WASM] sendWasmMessage called with no arguments")
		return nil
	}

	jsMsg := args[0]
	wasmLog("[WASM] sendWasmMessage received from JS:", jsMsg)

	// Convert JavaScript object to Go EventEnvelope at the boundary
	event, err := jsValueToEventEnvelope(jsMsg)
	if err != nil {
		wasmLog("[WASM] Failed to convert JS message to EventEnvelope:", err)
		return nil
	}

	wasmLog("[WASM] Converted to EventEnvelope:", event.Type)

	// Forward the event to the backend (Nexus)
	emitToNexus(event.Type, event.Payload, event.Metadata)

	// Process the properly typed event internally if a handler exists
	if handler := eventBus.GetHandler(event.Type); handler != nil {
		go handler(event)
	} else {
		wasmLog("[WASM] No internal handler registered for event type from JS:", event.Type)
	}
	return nil
}

// --- AI/ML Inference and Task Submission ---
// jsInfer is exposed to JavaScript for performing inference.
func jsInfer(this js.Value, args []js.Value) interface{} {
	if len(args) == 0 {
		return js.ValueOf([]byte{})
	}
	input := make([]byte, args[0].Get("byteLength").Int())
	js.CopyBytesToGo(input, args[0])
	result := Infer(input)
	out := js.Global().Get("Uint8Array").New(len(result))
	js.CopyBytesToJS(out, result)
	return out
}

// jsMigrateUser is exposed to JavaScript for user session migration.
func jsMigrateUser(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		migrateUserSession(args[0].String())
	}
	return nil
}

// jsSendBinary handles binary data with proper Frontend→WASM type conversion
func jsSendBinary(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 { // Expecting type, payload, and metadata
		wasmLog("[WASM] sendBinary requires type, payload, and metadata arguments")
		return nil
	}

	eventType := args[0].String()
	payloadJS := args[1]
	metadataJS := args[2]

	// Convert payload to proper Go format at the boundary
	var payloadBytes []byte
	if payloadJS.InstanceOf(js.Global().Get("Uint8Array")) || payloadJS.InstanceOf(js.Global().Get("ArrayBuffer")) {
		payloadBytes = make([]byte, payloadJS.Get("byteLength").Int())
		js.CopyBytesToGo(payloadBytes, payloadJS)
	} else if payloadJS.Type() == js.TypeString {
		payloadBytes = []byte(payloadJS.String())
	} else {
		// Convert JS object to JSON at the boundary
		payloadGo := jsValueToGoValue(payloadJS)
		if jsonBytes, err := json.Marshal(payloadGo); err == nil {
			payloadBytes = jsonBytes
		} else {
			wasmLog("[WASM] Failed to marshal payload to JSON:", err)
			return nil
		}
	}

	// Convert metadata to proper Go format at the boundary
	var metadataBytes json.RawMessage
	if metadataJS.Type() == js.TypeString {
		metadataBytes = json.RawMessage(metadataJS.String())
	} else {
		// Convert JS object to JSON at the boundary
		metadataGo := jsValueToGoValue(metadataJS)
		if jsonBytes, err := json.Marshal(metadataGo); err == nil {
			metadataBytes = jsonBytes
		} else {
			wasmLog("[WASM] Failed to marshal metadata to JSON:", err)
			return nil
		}
	}

	// Construct the EventEnvelope with properly converted types
	event := EventEnvelope{
		Type:     eventType,
		Payload:  payloadBytes,
		Metadata: metadataBytes,
	}

	// Marshal and send to backend
	envelopeBytes, err := json.Marshal(event)
	if err != nil {
		wasmLog("[WASM] Failed to marshal EventEnvelope:", err)
		return nil
	}

	// Send as JSON message (dataType 0)
	sendWSMessage(0, envelopeBytes)
	return nil
}

// --- Register JS callback for a pending request (exposed to JS) ---
// window.registerWasmPendingRequest(correlationId, callback)
func jsRegisterPendingRequest(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		wasmLog("[WASM] registerWasmPendingRequest requires correlationId and callback")
		return nil
	}
	correlationId := args[0].String()
	callback := args[1]
	if correlationId == "" || callback.Type() != js.TypeFunction {
		wasmLog("[WASM] registerWasmPendingRequest: invalid arguments")
		return nil
	}
	pendingRequests.Store(correlationId, callback)
	return nil
}

// --- Handler Examples ---
func handleGPUFrame(event EventEnvelope) {
	// Process WebGPU frame data from event.Payload
	wasmLog("[WASM] Received gpu_frame event. Metadata:", string(event.Metadata))

	// The contract for binary data is crucial. This implementation handles two possibilities:
	// 1. A raw binary payload (from the binary WebSocket message path).
	// 2. A JSON payload with a base64-encoded "data" field.
	var frameData []byte
	var payloadMap map[string]interface{}

	// Attempt to unmarshal as JSON to see if it's a structured payload.
	if err := json.Unmarshal(event.Payload, &payloadMap); err == nil {
		// It's JSON. Check for a base64-encoded data field.
		if data, ok := payloadMap["data"].(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				wasmLog("[WASM] Error decoding base64 gpu_frame payload:", err)
				return
			}
			frameData = decoded
		} else {
			wasmLog("[WASM] gpu_frame JSON payload does not contain a 'data' field.")
			return
		}
	} else {
		// It's not valid JSON, so we assume it's a raw binary payload.
		frameData = event.Payload
	}

	if len(frameData) == 0 {
		wasmLog("[WASM] Received gpu_frame event with no data.")
		return
	}

	// Pass the raw frame data to a JavaScript function that can interact with the WebGPU API.
	// We assume a global JS function `window.processGPUFrame` exists for this purpose.
	jsBuf := js.Global().Get("Uint8Array").New(len(frameData))
	js.CopyBytesToJS(jsBuf, frameData)

	// Invoke the JS function in a goroutine to avoid blocking the message loop.
	// This function would be responsible for submitting the data to the GPU.
	go js.Global().Get("processGPUFrame").Invoke(jsBuf)
}

func handleStateUpdate(event EventEnvelope) {
	// Process game state update from event.Payload
	var state struct { // Example structure for state update
		Players []struct {
			ID       string     `json:"id"`
			Position [3]float32 `json:"position"`
		} `json:"players"`
	}

	if err := json.Unmarshal(event.Payload, &state); err == nil {
		wasmLog("[WASM] Received state_update event. Players:", len(state.Players), "Metadata:", string(event.Metadata))
		// Process state update
	} else {
		wasmLog("[WASM] Error unmarshaling state_update payload:", err)
	}
}

// --- Enhanced Concurrent Processing JavaScript API ---

// runConcurrentCompute processes particles using Go's concurrent worker pool
func runConcurrentCompute(this js.Value, args []js.Value) interface{} {
	if len(args) < 4 {
		wasmLog("[CONCURRENT-COMPUTE] Requires: inputData, deltaTime, animationMode, callback")
		return js.ValueOf(false)
	}

	inputData := args[0]             // Float32Array
	deltaTime := args[1].Float()     // Time delta
	animationMode := args[2].Float() // Animation mode
	callback := args[3]              // Callback function

	// Validate input - expecting 10 values per particle: position(3) + velocity(3) + phase(1) + intensity(1) + type(1) + id(1)
	inputLength := inputData.Get("length").Int()
	if inputLength == 0 || inputLength%10 != 0 {
		wasmLog("[CONCURRENT-COMPUTE] Invalid input data length:", inputLength, "- must be multiple of 10 (10 values per particle)")
		return js.ValueOf(false)
	}

	// Convert JS Float32Array to Go slice
	positions := make([]float32, inputLength)
	for i := 0; i < inputLength; i++ {
		positions[i] = float32(inputData.Index(i).Float())
	}

	// Process concurrently in background
	go func() {
		startTime := time.Now()

		var result []float32
		if particleWorkerPool != nil {
			result = particleWorkerPool.ProcessParticlesConcurrently(positions, deltaTime, animationMode)
		} else {
			wasmLog("[CONCURRENT-COMPUTE] Worker pool not initialized, falling back to synchronous")
			result = make([]float32, len(positions))
			copy(result, positions)
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
				"method":         "concurrent",
				"particleCount":  inputLength / 10, // 10 values per particle
				"workerCount":    particleWorkerPool.workers,
			}))
		}
	}()

	return js.ValueOf(true)
}

// submitComputeTask adds a general compute task to the queue
func submitComputeTask(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		wasmLog("[SUBMIT-TASK] Requires: taskType, data, callback")
		return js.ValueOf(false)
	}

	taskType := args[0].String()
	data := args[1]
	callback := args[2]

	// Convert data to Go slice
	var dataSlice []float32
	if data.Type() == js.TypeObject && data.Get("length").Type() == js.TypeNumber {
		length := data.Get("length").Int()
		dataSlice = make([]float32, length)
		for i := 0; i < length; i++ {
			dataSlice[i] = float32(data.Index(i).Float())
		}
	}

	// Create task
	task := ComputeTask{
		ID:        fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Type:      taskType,
		Data:      dataSlice,
		Params:    make(map[string]float64),
		Callback:  callback,
		Priority:  1, // Normal priority
		Timestamp: time.Now(),
	}

	// Add optional parameters
	if len(args) > 3 && args[3].Type() == js.TypeObject {
		params := args[3]
		if !params.Get("deltaTime").IsUndefined() {
			task.Params["deltaTime"] = params.Get("deltaTime").Float()
		}
		if !params.Get("animationMode").IsUndefined() {
			task.Params["animationMode"] = params.Get("animationMode").Float()
		}
		if !params.Get("priority").IsUndefined() {
			task.Priority = int(params.Get("priority").Float())
		}
	}

	// Submit to queue
	select {
	case computeTaskQueue <- task:
		wasmLog("[SUBMIT-TASK] Task submitted:", task.ID, task.Type)
		return js.ValueOf(task.ID)
	default:
		wasmLog("[SUBMIT-TASK] Task queue full, rejecting task")
		return js.ValueOf(false)
	}
}
