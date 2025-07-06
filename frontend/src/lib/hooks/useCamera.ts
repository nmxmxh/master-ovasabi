import { useState, useRef, useCallback } from 'react';
import Webcam from 'react-webcam';

export function useCamera() {
  const webcamRef = useRef<Webcam>(null);
  const [captured, setCaptured] = useState<string | null>(null);

  const capture = useCallback(() => {
    if (webcamRef.current) {
      const imageSrc = webcamRef.current.getScreenshot();
      setCaptured(imageSrc || null);
    }
  }, []);

  return {
    webcamRef,
    captured,
    capture
  };
}
