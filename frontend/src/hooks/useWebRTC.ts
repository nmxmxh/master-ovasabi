import { useEffect, useRef, useState, useCallback } from 'react';
import { subscribeToMediaSignals, sendMediaSignal } from '../lib/wasmBridge';

// Types for signaling and RTC state
interface MediaSignal {
  type: string;
  payload: any;
  from?: string;
  to?: string;
  [key: string]: any;
}

interface WebRTCState {
  connected: boolean;
  connecting: boolean;
  error?: string | null;
  peerConnection?: RTCPeerConnection | null;
}

const DEFAULT_STATE: WebRTCState = {
  connected: false,
  connecting: false,
  error: null,
  peerConnection: null
};

export function useWebRTC(roomId: string, userId: string, onDataMessage?: (data: any) => void) {
  const [state, setState] = useState<WebRTCState>(DEFAULT_STATE);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const dataChannelRef = useRef<RTCDataChannel | null>(null);

  // Helper: Send a signal via wasmBridge
  const sendSignal = useCallback(
    (signal: MediaSignal) => {
      sendMediaSignal({ ...signal, roomId, from: userId });
    },
    [roomId, userId]
  );

  // Handle incoming signals
  useEffect(() => {
    const unsubscribe = subscribeToMediaSignals(async (signal: MediaSignal) => {
      if (!signal) return;
      const { type, payload, from } = signal;
      let pc = pcRef.current;
      if (!pc) return;

      if (type === 'offer') {
        await pc.setRemoteDescription(new RTCSessionDescription(payload));
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);
        sendSignal({ type: 'answer', payload: answer, to: from });
      } else if (type === 'answer') {
        await pc.setRemoteDescription(new RTCSessionDescription(payload));
      } else if (type === 'ice-candidate') {
        try {
          await pc.addIceCandidate(new RTCIceCandidate(payload));
        } catch (err) {
          setState(s => ({ ...s, error: 'Failed to add ICE candidate' }));
        }
      }
    });
    return () => unsubscribe && unsubscribe();
  }, [sendSignal]);

  // Start and create/join connection
  const start = useCallback(async () => {
    setState(s => ({ ...s, connecting: true, error: null }));
    try {
      const pc = new RTCPeerConnection();
      pcRef.current = pc;

      const dataChannel = pc.createDataChannel("particles");
      dataChannelRef.current = dataChannel;
      if (onDataMessage) {
        dataChannel.onmessage = (event) => {
          onDataMessage(event.data);
        };
      }

      pc.onicecandidate = event => {
        if (event.candidate) {
          sendSignal({ type: 'ice-candidate', payload: event.candidate });
        }
      };

      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      sendSignal({ type: 'offer', payload: offer });

      setState(s => ({ ...s, connecting: false, connected: true, peerConnection: pc }));
    } catch (err: any) {
      setState(s => ({ ...s, error: err.message || 'Failed to start WebRTC' }));
    }
  }, [sendSignal, onDataMessage]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (pcRef.current) {
        pcRef.current.close();
        pcRef.current = null;
      }
      setState(DEFAULT_STATE);
    };
  }, []);

  const stop = useCallback(() => {
    if (pcRef.current) {
      pcRef.current.close();
      pcRef.current = null;
    }
    setState(DEFAULT_STATE);
  }, []);

  const sendData = useCallback((data: string) => {
    if (dataChannelRef.current && dataChannelRef.current.readyState === 'open') {
      dataChannelRef.current.send(data);
    }
  }, []);

  return {
    ...state,
    start,
    stop,
    sendData,
    peerConnection: pcRef.current
  };
}
