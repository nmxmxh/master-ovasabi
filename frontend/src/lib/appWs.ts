// Centralized builder for all app WebSocket paths and URLs

export interface AppWebSocketOpts {
  campaignId: string | number;
  userId: string | number;
  type?: string; // e.g. 'search', 'profile', etc.
  wsOrigin?: string; // Optional override for ws(s)://host
}

/**
 * Build the canonical WebSocket path for the app.
 * Example: /ws/search/123/456
 */
export function buildAppWebSocketPath({
  campaignId,
  userId,
  type = 'default'
}: AppWebSocketOpts): string {
  // Default campaign is 'ovasabi_website'
  const cid = campaignId || 'ovasabi_website';
  // Use guest_... for missing/anonymous user
  let uid = userId;
  if (!uid || uid === 'anonymous' || uid === 'undefined' || uid === undefined || uid === null) {
    // Use a deterministic guest id if possible, else random
    uid =
      typeof window !== 'undefined' && window.localStorage
        ? window.localStorage.getItem('guest_id') ||
          `guest_${Math.random().toString(36).slice(2, 10)}`
        : `guest_${Math.random().toString(36).slice(2, 10)}`;
    if (typeof window !== 'undefined' && window.localStorage) {
      window.localStorage.setItem('guest_id', uid);
    }
  }
  return `/ws/${type}/${cid}/${uid}`;
}

/**
 * Build the full WebSocket URL for the app, using getWebSocketUrl.
 */
import { getWebSocketUrl } from './wsUrl';
export function getAppWebSocketUrl(opts: AppWebSocketOpts): string {
  const path = buildAppWebSocketPath(opts);
  return getWebSocketUrl(path, opts.wsOrigin);
}
