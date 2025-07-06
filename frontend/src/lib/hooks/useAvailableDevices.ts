import { useEffect, useState } from 'react';

export interface MediaDeviceInfoLite {
  deviceId: string;
  kind: MediaDeviceKind;
  label: string;
  groupId: string;
}

export function useAvailableDevices() {
  const [devices, setDevices] = useState<MediaDeviceInfoLite[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
      navigator.mediaDevices
        .enumerateDevices()
        .then(mediaDevices => {
          if (!mounted) return;
          setDevices(
            mediaDevices.map(d => ({
              deviceId: d.deviceId,
              kind: d.kind,
              label: d.label,
              groupId: d.groupId
            }))
          );
          setLoading(false);
        })
        .catch(err => {
          if (!mounted) return;
          setError(err.message || 'Failed to enumerate devices');
          setLoading(false);
        });
    } else {
      setError('Media devices API not supported');
      setLoading(false);
    }
    return () => {
      mounted = false;
    };
  }, []);

  return { devices, loading, error };
}
