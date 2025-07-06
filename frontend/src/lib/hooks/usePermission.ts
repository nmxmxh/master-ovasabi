import { usePermission as useBrowserPermission } from 'react-use';

type PermissionName =
  | 'geolocation'
  | 'notifications'
  | 'camera'
  | 'microphone'
  | 'midi'
  | 'clipboard-read'
  | 'clipboard-write'
  | 'persistent-storage'
  | 'push';

export function usePermission(name: PermissionName) {
  const result = useBrowserPermission({ name } as PermissionDescriptor);
  return {
    state: result as PermissionState | undefined, // 'granted' | 'prompt' | 'denied' | undefined
    loading: result === '',
    error: undefined // react-use does not provide error info
  };
}
