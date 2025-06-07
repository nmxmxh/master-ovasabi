// Robust, concurrent browser-side WASM learner for AI input collection
// Usage: wasmLearner.learn(input: Uint8Array)

const learnedInputs: Uint8Array[] = [];

export async function learn(input: Uint8Array): Promise<void> {
  // Call WASM inference (which just logs/learns for now)
  if (typeof (window as any).infer === 'function') {
    await (window as any).infer(input);
  }
  // Store input for future training
  learnedInputs.push(new Uint8Array(input));
}

export function getLearnedInputs(): Uint8Array[] {
  // Return a copy of all learned inputs
  return learnedInputs.map(arr => new Uint8Array(arr));
}

export function clearLearnedInputs(): void {
  learnedInputs.length = 0;
}
