import { useWasmBridge } from '../hooks/useWasmBridge';

export type DescriptorDiscoveryOptions = {
  httpUrl?: string; // e.g. "/api/proto/descriptors"
  wsUrl?: string; // e.g. "ws://localhost:8081/ws/proto/descriptors"
  onBinary?: (data: ArrayBuffer) => void;
  onError?: (err: any) => void;
};

/**
 * Fetches the FileDescriptorSet (all registered proto descriptors) via HTTP.
 * Returns a Promise that resolves to the binary ArrayBuffer.
 */
export async function fetchProtoDescriptorsHTTP(url: string): Promise<ArrayBuffer> {
  const res = await fetch(url, {
    method: 'GET',
    headers: { Accept: 'application/x-protobuf' }
  });
  if (!res.ok) throw new Error(`Failed to fetch proto descriptors: ${res.status}`);
  return await res.arrayBuffer();
}

/**
 * Connects to the WebSocket endpoint and requests proto descriptors.
 * Calls onBinary with the binary FileDescriptorSet when received.
 * Returns the WebSocket instance for advanced use (e.g., manual close, event listeners).
 */
// WASM bridge version: sends a message and expects binary response via callback
export function fetchProtoDescriptorsWASM(
  onBinary: (data: ArrayBuffer) => void,
  onError?: (err: any) => void
) {
  // Use the WASM bridge to send a request for proto descriptors
  const { send } = useWasmBridge({
    onMessage: msg => {
      if (msg.type === 'get_proto_descriptors' && msg.payload instanceof ArrayBuffer) {
        onBinary(msg.payload);
      } else if (msg.type === 'error' && onError) {
        onError(msg.payload);
      }
    },
    onError
  });
  send({ type: 'get_proto_descriptors' });
}

/**
 * High-level utility for dynamic contract discovery (HTTP or WS)
 * Returns the WebSocket instance if using WS, otherwise void.
 */
export function discoverProtoDescriptors(options: DescriptorDiscoveryOptions) {
  if (options.httpUrl) {
    if (!options.onBinary) throw new Error('onBinary callback required for HTTP discovery');
    fetchProtoDescriptorsHTTP(options.httpUrl).then(options.onBinary).catch(options.onError);
  } else if (options.wsUrl) {
    if (!options.onBinary) throw new Error('onBinary callback required for WASM discovery');
    // Use WASM bridge for all real-time proto descriptor discovery
    return fetchProtoDescriptorsWASM(options.onBinary, options.onError);
  } else {
    throw new Error('No httpUrl or wsUrl provided');
  }
}
