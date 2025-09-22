// TypeScript global declarations for WASM bridge globals

declare global {
  // Sends a message from JS/React to Go WASM (string or object)
  var wasmSendMessage: (msg: any) => void;
  // Exported by Go WASM, called by JS to send a message to Go
  var sendWasmMessage: ((msg: any) => void) | undefined;

  // Called by Go WASM to deliver a message to JS/React (string or object)
  var onWasmMessage: (msg: any) => void;

  // Called by Go WASM when WASM is fully ready (e.g., after WebSocket is connected)
  var onWasmReady: () => void;

  // Used by Go/JS for versioning/logging
  var __WASM_VERSION: string;

  // Used as a message queue for WASM concurrency interop
  var __WASM_EVENT_QUEUE: any[];

  interface Window {
    sendWasmMessage?: (msg: any) => void;
    wasmSendMessage?: (msg: any) => void;
    onWasmMessage?: (msg: any) => void;
    onWasmReady?: () => void;
    __WASM_EVENT_QUEUE?: any[];
    __WASM_VERSION?: string;
    getSharedBuffer?: () => ArrayBuffer;
    __WASM_GLOBAL_METADATA?: any;
    userID?: string;
    wasmReady?: boolean;
    // WebGPU functions
    initWebGPU?: () => boolean;
    getWebGPUDevice?: () => any;
    checkWebGPUAvailability?: () => any;
    getWasmWebGPUStatus?: () => any;
    checkWebGPUDeviceValidity?: () => boolean;
    getGPUBackend?: () => any;
    getGPUMetricsBuffer?: () => ArrayBuffer;
    getGPUComputeBuffer?: () => ArrayBuffer;
    runGPUCompute?: (inputData: Float32Array, operation: number, callback: Function) => boolean;
    runGPUComputeWithOffset?: (
      inputData: Float32Array,
      elapsedTime: number,
      globalParticleOffset: number,
      callback: Function
    ) => boolean;
    runConcurrentCompute?: (
      inputData: Float32Array,
      deltaTime: number,
      animationMode: number,
      callback: Function
    ) => boolean;
    // Campaign switch success handler
    handleCampaignSwitchSuccess?: (campaignId: string, reason: string) => void;
  }

  interface GlobalState {
    switchCampaign: (
      campaignId: number | string,
      slug?: string,
      onResponse?: (event: any) => void
    ) => void;
    // ...existing properties...
  }
}

export {}; // Ensures this file is treated as a module
