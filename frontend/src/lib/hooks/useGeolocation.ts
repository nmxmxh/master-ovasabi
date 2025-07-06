import { useGeolocation } from 'react-use';

export function useLocationStream() {
  // react-use's useGeolocation provides streaming updates
  const state = useGeolocation();
  return {
    latitude: state.latitude,
    longitude: state.longitude,
    accuracy: state.accuracy,
    timestamp: state.timestamp,
    error: state.error,
    loading: state.loading,
    raw: state
  };
}
