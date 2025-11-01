//go:build js && wasm
// +build js,wasm

package main

// ExecuteCPU provides a generic CPU execution path and routes known ops to worker pool.
func ExecuteCPU(op string, inputs []any, params map[string]string) (any, bool) {
    switch op {
    case "particles", "runParticlePhysics":
        positions := extractPositions(inputs)
        delta := parseFloat(params["deltaTime"], 0.016)
        mode := parseFloat(params["animationMode"], 0)
        if particleWorkerPool == nil {
            particleWorkerPool = NewParticleWorkerPool(0)
        }
        out := particleWorkerPool.ProcessParticlesConcurrently(positions, delta, mode)
        return map[string]any{
            "total":  len(out) / 3,
            "sample": samplePositions(out, 500),
        }, true
    default:
        // Unknown op: return passthrough status for now
        return map[string]any{"status": "cpu_executed", "op": op}, true
    }
}
