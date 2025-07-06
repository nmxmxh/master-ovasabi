import { useEffect, useState } from 'react';
import { usePermission } from './usePermission';
import { useLocationStream } from './useGeolocation';
import { useCamera } from './useCamera';
import { useMediaRecorder } from './useMediaRecorder';
import { useBattery } from './useBattery';

export function useDevice() {
  // Permissions
  const geoPerm = usePermission('geolocation');
  const camPerm = usePermission('camera');
  const micPerm = usePermission('microphone');

  // Geolocation
  const location = useLocationStream();

  // Camera
  const camera = useCamera();

  // Media Recorder
  const mediaRecorder = useMediaRecorder({ video: true, audio: true });

  // Device info
  const battery = useBattery();

  // Network status
  const [online, setOnline] = useState(navigator.onLine);
  useEffect(() => {
    const update = () => setOnline(navigator.onLine);
    window.addEventListener('online', update);
    window.addEventListener('offline', update);
    return () => {
      window.removeEventListener('online', update);
      window.removeEventListener('offline', update);
    };
  }, []);

  return {
    permissions: {
      geolocation: geoPerm,
      camera: camPerm,
      microphone: micPerm
    },
    location,
    camera,
    mediaRecorder,
    battery,
    online
  };
}
