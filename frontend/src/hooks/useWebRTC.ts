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
  localStream?: MediaStream | null;
  remoteStream?: MediaStream | null;
  peerConnection?: RTCPeerConnection | null;
  // Add more fields as needed
}

const DEFAULT_STATE: WebRTCState = {
  connected: false,
  connecting: false,
  error: null,
  localStream: null,
  remoteStream: null,
  peerConnection: null
};

export function useWebRTC(roomId: string, userId: string) {
  const [state, setState] = useState<WebRTCState>(DEFAULT_STATE);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);
  const remoteStreamRef = useRef<MediaStream | null>(null);

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

  // Start local media and create/join connection
  const start = useCallback(async () => {
    setState(s => ({ ...s, connecting: true, error: null }));
    try {
      const localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true });
      localStreamRef.current = localStream;
      setState(s => ({ ...s, localStream }));

      const pc = new RTCPeerConnection();
      pcRef.current = pc;
      localStream.getTracks().forEach(track => pc.addTrack(track, localStream));

      const remoteStream = new MediaStream();
      remoteStreamRef.current = remoteStream;
      setState(s => ({ ...s, remoteStream }));

      pc.ontrack = event => {
        event.streams[0].getTracks().forEach(track => {
          remoteStream.addTrack(track);
        });
        setState(s => ({ ...s, remoteStream }));
      };

      pc.onicecandidate = event => {
        if (event.candidate) {
          sendSignal({ type: 'ice-candidate', payload: event.candidate });
        }
      };

      // Create offer and send
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      sendSignal({ type: 'offer', payload: offer });

      setState(s => ({ ...s, connecting: false, connected: true, peerConnection: pc }));
    } catch (err: any) {
      setState(s => ({ ...s, error: err.message || 'Failed to start WebRTC' }));
    }
  }, [sendSignal]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (pcRef.current) {
        pcRef.current.close();
        pcRef.current = null;
      }
      if (localStreamRef.current) {
        localStreamRef.current.getTracks().forEach(track => track.stop());
        localStreamRef.current = null;
      }
      setState(DEFAULT_STATE);
    };
  }, []);

  return {
    ...state,
    start,
    sendSignal
    // Expose more helpers as needed
  };
}
