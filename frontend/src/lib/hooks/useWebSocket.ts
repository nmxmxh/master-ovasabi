import { useEffect, useState } from 'react';
import useWebSocket from 'react-use-websocket';

export interface WebSocketMessage {
  type: string;
  payload?: any;
  [key: string]: any;
}

export interface UseWebSocketOptions {
  url: string;
  onMessage?: (msg: WebSocketMessage) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (err: Event) => void;
  shouldReconnect?: boolean;
}

// ws-gateway aware WebSocket hook with ingress/egress message handling
export function useWebSocketConnection({
  url,
  onMessage,
  onOpen,
  onClose,
  onError,
  shouldReconnect = true
}: UseWebSocketOptions) {
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);
  const [connected, setConnected] = useState(false);
  const [egressQueue, setEgressQueue] = useState<any[]>([]); // Outbound messages
  const [ingressQueue, setIngressQueue] = useState<any[]>([]); // Inbound messages

  const { sendMessage, lastJsonMessage, readyState } = useWebSocket(url, {
    onOpen: () => {
      setConnected(true);
      onOpen?.();
      // Flush egress queue on connect
      egressQueue.forEach(msg => sendMessage(msg));
      setEgressQueue([]);
    },
    onClose: () => {
      setConnected(false);
      onClose?.();
    },
    onError: e => {
      setConnected(false);
      onError?.(e);
    },
    shouldReconnect: () => shouldReconnect
  });

  // Type guard for WebSocketMessage
  function isWebSocketMessage(msg: any): msg is WebSocketMessage {
    return msg && typeof msg === 'object' && typeof msg.type === 'string';
  }

  // Handle ingress (inbound) messages
  useEffect(() => {
    if (lastJsonMessage && isWebSocketMessage(lastJsonMessage)) {
      setLastMessage(lastJsonMessage);
      setIngressQueue(q => [...q, lastJsonMessage]);
      onMessage?.(lastJsonMessage);
    }
  }, [lastJsonMessage, onMessage]);

  // Egress (outbound) message sender
  const send = (msg: any) => {
    if (connected && readyState === 1) {
      sendMessage(msg);
    } else {
      setEgressQueue(q => [...q, msg]);
    }
  };

  // Utility: clear ingress/egress queues
  const clearQueues = () => {
    setEgressQueue([]);
    setIngressQueue([]);
  };

  return {
    connected,
    sendMessage: send,
    lastMessage,
    readyState,
    ingressQueue,
    egressQueue,
    clearQueues
  };
}
