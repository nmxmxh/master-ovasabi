/**
 * Centralized WebSocket URL builder for browser and WASM clients.
 * Always use this function to construct WebSocket URLs in the frontend.
 *
 * @param path - The WebSocket path (e.g. '/ws/campaign/123/user/456')
 * @param baseOverride - Optional override for ws(s)://host (for tests/dev)
 * @returns The full WebSocket URL
 */
export function getWebSocketUrl(path: string, baseOverride?: string): string {
  if (baseOverride) {
    return `${baseOverride.replace(/\/$/, '')}${path}`;
  }
  const loc = window.location;
  const protocol = loc.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${loc.host}${path}`;
}
