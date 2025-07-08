import { useCallback, useEffect, useRef } from 'react';
import { useWasmBridge } from './useWasmBridge';

// Optionally support direct WebSocket endpoint for proto descriptors (legacy or ws-gateway)
function fetchProtoDescriptorsWS(
  wsUrl: string,
  onBinary: (data: ArrayBuffer) => void,
  onError?: (err: any) => void
) {
  const ws = new WebSocket(wsUrl);
  ws.binaryType = 'arraybuffer';
  ws.onopen = () => {
    console.log('[ProtoDescriptorWS] WebSocket opened:', wsUrl);
    ws.send(JSON.stringify({ type: 'get_proto_descriptors' }));
  };
  ws.onmessage = event => {
    if (event.data instanceof ArrayBuffer) {
      console.log('[ProtoDescriptorWS] Received binary data');
      onBinary(event.data);
    } else {
      try {
        const err = JSON.parse(event.data);
        console.warn('[ProtoDescriptorWS] Received error:', err);
        onError?.(err);
      } catch (e) {
        console.warn('[ProtoDescriptorWS] Received non-binary, non-JSON data:', event.data);
        onError?.(event.data);
      }
    }
  };
  ws.onerror = event => {
    console.error('[ProtoDescriptorWS] WebSocket error:', event);
    onError?.(event);
  };
  ws.onclose = event => {
    console.log('[ProtoDescriptorWS] WebSocket closed:', event);
  };
  return ws;
}
export interface UseProtoDescriptorDiscoveryOptions {
  httpUrl?: string;
  wsUrl?: string; // e.g. ws://localhost:8081/ws/proto/descriptors or ws-gateway endpoint
  onBinary?: (data: ArrayBuffer) => void;
  onError?: (err: any) => void;
  auto?: boolean; // auto-fetch on mount
}

/**
 * React hook for dynamic proto descriptor discovery (HTTP or WASM bridge)
 * Usage:
 *   const { fetchDescriptors, loading, error } = useProtoDescriptorDiscovery({ ... });
 */
export function useProtoDescriptorDiscovery(options: UseProtoDescriptorDiscoveryOptions) {
  const { httpUrl, wsUrl, onBinary, onError, auto = true } = options;
  const loadingRef = useRef(false);
  const errorRef = useRef<any>(null);

  // WASM bridge for real-time proto descriptor discovery
  const { send } = useWasmBridge({
    onMessage: msg => {
      if (msg.type === 'get_proto_descriptors' && msg.payload instanceof ArrayBuffer) {
        loadingRef.current = false;
        errorRef.current = null;
        onBinary?.(msg.payload);
      } else if (msg.type === 'error') {
        loadingRef.current = false;
        errorRef.current = msg.payload;
        onError?.(msg.payload);
      }
    },
    onError: err => {
      loadingRef.current = false;
      errorRef.current = err;
      onError?.(err);
    }
  });

  // Fetch via HTTP
  const fetchDescriptorsHTTP = useCallback(async () => {
    if (!httpUrl) throw new Error('No httpUrl provided');
    loadingRef.current = true;
    errorRef.current = null;
    try {
      const res = await fetch(httpUrl, {
        method: 'GET',
        headers: { Accept: 'application/x-protobuf' }
      });
      if (!res.ok) throw new Error(`Failed to fetch proto descriptors: ${res.status}`);
      const buffer = await res.arrayBuffer();
      loadingRef.current = false;
      errorRef.current = null;
      onBinary?.(buffer);
    } catch (err) {
      loadingRef.current = false;
      errorRef.current = err;
      onError?.(err);
    }
  }, [httpUrl, onBinary, onError]);

  // Fetch via direct WebSocket endpoint (legacy or ws-gateway)
  const fetchDescriptorsWS = useCallback(() => {
    if (!wsUrl) throw new Error('No wsUrl provided');
    loadingRef.current = true;
    errorRef.current = null;
    fetchProtoDescriptorsWS(
      wsUrl,
      data => {
        loadingRef.current = false;
        errorRef.current = null;
        onBinary?.(data);
      },
      err => {
        loadingRef.current = false;
        errorRef.current = err;
        onError?.(err);
      }
    );
  }, [wsUrl, onBinary, onError]);

  // Fetch via WASM bridge
  const fetchDescriptorsWASM = useCallback(() => {
    loadingRef.current = true;
    errorRef.current = null;
    send({ type: 'get_proto_descriptors' });
  }, [send]);

  // Auto-fetch on mount
  useEffect(() => {
    if (!auto) return;
    if (httpUrl) {
      fetchDescriptorsHTTP();
    } else if (wsUrl) {
      fetchDescriptorsWS();
    } else {
      fetchDescriptorsWASM();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auto, httpUrl, wsUrl]);

  return {
    fetchDescriptors: httpUrl
      ? fetchDescriptorsHTTP
      : wsUrl
        ? fetchDescriptorsWS
        : fetchDescriptorsWASM,
    loading: loadingRef.current,
    error: errorRef.current
  };
}
