import { useReactMediaRecorder } from 'react-media-recorder';

export function useMediaRecorder(options?: Parameters<typeof useReactMediaRecorder>[0]) {
  // options: { video, audio, screen, blobPropertyBag, ... }
  const recorder = useReactMediaRecorder(options || { video: true, audio: true });
  return recorder;
}
