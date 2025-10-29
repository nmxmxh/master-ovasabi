import { useState, useEffect, useRef } from 'react';

export function useWebRTCStats(peerConnection: RTCPeerConnection | null) {
  const [stats, setStats] = useState<RTCStatsReport | null>(null);
  const intervalRef = useRef<number | null>(null);

  useEffect(() => {
    if (peerConnection) {
      intervalRef.current = window.setInterval(async () => {
        const statsReport = await peerConnection.getStats();
        setStats(statsReport);
      }, 1000);
    } else {
      if (intervalRef.current) {
        window.clearInterval(intervalRef.current);
      }
    }

    return () => {
      if (intervalRef.current) {
        window.clearInterval(intervalRef.current);
      }
    };
  }, [peerConnection]);

  return stats;
}
