// Service Worker for OVASABI Platform PWA
// Provides offline capabilities, caching, and background sync with enhanced error handling

const CACHE_NAME = 'ovasabi-v1.0.1'; // Increment for cache invalidation
const STATIC_CACHE = `${CACHE_NAME}-static`;
const DYNAMIC_CACHE = `${CACHE_NAME}-dynamic`;
const WASM_CACHE = `${CACHE_NAME}-wasm`;

// Development mode detection
const isDevelopment =
  location.hostname === 'localhost' ||
  location.hostname === '127.0.0.1' ||
  location.hostname.includes('dev');

// Error recovery tracking
let errorCount = 0;
let lastErrorTime = 0;
const MAX_ERRORS = 10;
const ERROR_RESET_TIME = 300000; // 5 minutes

// Resource cleanup tracking
const openCacheHandles = new Set();
const activeRequests = new Map();

// Enhanced error handling with graceful degradation
function handleServiceWorkerError(error, context = 'unknown') {
  const now = Date.now();

  // Reset error count if enough time has passed
  if (now - lastErrorTime > ERROR_RESET_TIME) {
    errorCount = 0;
  }

  errorCount++;
  lastErrorTime = now;

  // Only log critical errors
  if (errorCount >= MAX_ERRORS || context.includes('critical')) {
    console.error(`[SW] Error in ${context} (${errorCount}/${MAX_ERRORS}):`, error);
  }

  // If too many errors, enter degraded mode
  if (errorCount >= MAX_ERRORS) {
    console.warn('[SW] Too many errors, entering degraded mode');
    return false; // Signal degraded mode
  }

  return true; // Continue normal operation
}

// Safe cache operations with cleanup
async function safeOpenCache(cacheName) {
  try {
    const cache = await caches.open(cacheName);
    openCacheHandles.add(cache);
    return cache;
  } catch (error) {
    handleServiceWorkerError(error, 'cache open');
    throw error;
  }
}

// Clean up cache handles
function cleanupCacheHandles() {
  openCacheHandles.clear();
}

// Assets to cache immediately on install
const STATIC_ASSETS = [
  '/',
  '/offline.html',
  '/main.wasm',
  '/main.threads.wasm',
  '/wasm_exec.js',
  '/workers/compute-worker.js',
  '/sounds/theme.aac',
  '/sounds/tick.aac',
  '/fonts/Geist[wght].woff2',
  '/fonts/GeistMono[wght].woff2',
  '/fonts/Gordita-Bold.otf',
  '/fonts/Gordita-Medium.otf',
  '/android-chrome-192x192.png',
  '/android-chrome-512x512.png',
  '/apple-touch-icon.png',
  '/favicon-16x16.png',
  '/favicon-32x32.png',
  '/favicon.ico'
];

// Cache strategies for different asset types
const CACHE_STRATEGIES = {
  wasm: isDevelopment ? 'network-first' : 'cache-first', // Network first in dev
  static: isDevelopment ? 'network-first' : 'stale-while-revalidate',
  api: 'network-first',
  media: 'cache-first',
  workers: isDevelopment ? 'network-first' : 'cache-first'
};

// Install event - cache static assets with error handling
self.addEventListener('install', event => {
  event.waitUntil(
    Promise.all([
      // Cache static assets with individual error handling
      caches.open(STATIC_CACHE).then(cache => {
        return cache.addAll(STATIC_ASSETS).catch(error => {
          // Cache individual assets that exist
          return Promise.allSettled(
            STATIC_ASSETS.map(asset =>
              cache.add(asset).catch(err => {
                return null;
              })
            )
          );
        });
      }),

      // Cache WASM files separately for better performance
      caches.open(WASM_CACHE).then(cache => {
        return cache.addAll(['/main.wasm', '/main.threads.wasm', '/wasm_exec.js']).catch(error => {
          // Cache individual WASM assets that exist
          return Promise.allSettled(
            ['/main.wasm', '/main.threads.wasm', '/wasm_exec.js'].map(asset =>
              cache.add(asset).catch(err => {
                return null;
              })
            )
          );
        });
      })
    ])
      .then(() => {
        // Skip waiting to activate immediately
        return self.skipWaiting();
      })
      .catch(error => {
        console.error('[SW] Installation failed:', error);
        // Still skip waiting even if caching failed
        return self.skipWaiting();
      })
  );
});

// Activate event - cleanup old caches
self.addEventListener('activate', event => {
  event.waitUntil(
    caches
      .keys()
      .then(cacheNames => {
        return Promise.all(
          cacheNames.map(cacheName => {
            // Delete old caches that don't match current version
            if (cacheName.startsWith('ovasabi-') && cacheName !== CACHE_NAME) {
              return caches.delete(cacheName);
            }
          })
        );
      })
      .then(() => {
        // Take control of all clients immediately
        return self.clients.claim();
      })
      .then(() => {
        // Notify all clients that SW is ready and assets are cached
        return self.clients.matchAll().then(clients => {
          clients.forEach(client => {
            client.postMessage({ type: 'sw-ready' });
          });
        });
      })
  );
});

// Fetch event - implement caching strategies
self.addEventListener('fetch', event => {
  const { request } = event;
  const url = new URL(request.url);

  // Skip non-HTTP requests
  if (!url.protocol.startsWith('http')) {
    return;
  }

  // Handle different request types with appropriate strategies
  if (isWASMRequest(request)) {
    event.respondWith(handleWASMRequest(request));
  } else if (isWorkerRequest(request)) {
    event.respondWith(handleWorkerRequest(request));
  } else if (isAPIRequest(request)) {
    event.respondWith(handleAPIRequest(request));
  } else if (isMediaRequest(request)) {
    event.respondWith(handleMediaRequest(request));
  } else if (isStaticAsset(request)) {
    event.respondWith(handleStaticRequest(request));
  } else {
    event.respondWith(handleNavigationRequest(request));
  }
});

// WASM requests - cache first (WASM files rarely change) but network first in dev
async function handleWASMRequest(request) {
  try {
    // Ignore search parameters for caching to avoid issues with cache-busting strings
    const cacheKey = new URL(request.url).pathname;

    const cache = await caches.open(WASM_CACHE);

    let response;
    if (isDevelopment) {
      // In development, always try network first to get latest WASM
      try {
        const networkResponse = await fetch(request);
        if (networkResponse.ok) {
          try {
            await cache.put(cacheKey, networkResponse.clone());
            // Updated WASM cache in development
          } catch (cacheError) {
            console.warn('[SW] Failed to cache WASM response:', cacheError);
            // Continue without caching - the response is still valid
          }
        }
        response = networkResponse;
      } catch (networkError) {
        console.log('[SW] Network failed for WASM, using cache:', request.url);
        const cachedResponse = await cache.match(cacheKey);
        if (cachedResponse) response = cachedResponse;
        else throw networkError;
      }
    } else {
      // Production: cache first
      const cachedResponse = await cache.match(cacheKey);
      if (cachedResponse) {
        console.log('[SW] Serving WASM from cache:', request.url);
        response = cachedResponse;
      } else {
        const networkResponse = await fetch(request);
        if (networkResponse.ok) {
          try {
            await cache.put(cacheKey, networkResponse.clone());
          } catch (cacheError) {
            console.warn('[SW] Failed to cache WASM response:', cacheError);
            // Continue without caching - the response is still valid
          }
        }
        response = networkResponse;
      }
    }
    // Ensure correct MIME type for .wasm files
    if (request.url.endsWith('.wasm') && response && response.ok) {
      try {
        const newHeaders = new Headers(response.headers);
        newHeaders.set('Content-Type', 'application/wasm');
        const body = await response.arrayBuffer();
        return new Response(body, {
          status: response.status,
          statusText: response.statusText,
          headers: newHeaders
        });
      } catch (error) {
        console.warn('[SW] Failed to process WASM response:', error);
        return response; // Return original response if processing fails
      }
    }
    return response;
  } catch (error) {
    console.error('[SW] WASM request failed:', error);
    throw error;
  }
}

// Worker requests - handle compute workers and other workers
async function handleWorkerRequest(request) {
  try {
    const cache = await caches.open(STATIC_CACHE);

    if (isDevelopment) {
      // In development, always get fresh worker code
      try {
        const networkResponse = await fetch(request);
        if (networkResponse.ok) {
          await cache.put(request, networkResponse.clone());
          // Updated worker cache in development
        }
        return networkResponse;
      } catch (networkError) {
        const cachedResponse = await cache.match(request);
        if (cachedResponse) return cachedResponse;
        throw networkError;
      }
    } else {
      // Production: cache first for workers
      const cachedResponse = await cache.match(request);
      if (cachedResponse) return cachedResponse;

      const networkResponse = await fetch(request);
      if (networkResponse.ok) {
        await cache.put(request, networkResponse.clone());
      }
      return networkResponse;
    }
  } catch (error) {
    console.error('[SW] Worker request failed:', error);
    throw error;
  }
}

// API requests - network first with fallback to cache
async function handleAPIRequest(request) {
  try {
    const cache = await caches.open(DYNAMIC_CACHE);

    // Try network first
    try {
      const networkResponse = await fetch(request, {
        timeout: 5000 // 5 second timeout
      });

      if (networkResponse.ok) {
        // Cache successful API responses (except POST/PUT/DELETE)
        if (request.method === 'GET') {
          await cache.put(request, networkResponse.clone());
        }
        return networkResponse;
      }
    } catch (networkError) {
      console.log('[SW] Network failed for API request, trying cache');
    }

    // Fallback to cache
    const cachedResponse = await cache.match(request);
    if (cachedResponse) {
      console.log('[SW] Serving API response from cache:', request.url);
      return cachedResponse;
    }

    // Return offline response for API failures
    return new Response(
      JSON.stringify({
        error: 'Offline',
        message: 'Request failed and no cached response available'
      }),
      {
        status: 503,
        headers: { 'Content-Type': 'application/json' }
      }
    );
  } catch (error) {
    console.error('[SW] API request handler failed:', error);
    throw error;
  }
}

// Media requests - cache first (images, videos, audio)
async function handleMediaRequest(request) {
  try {
    const cache = await caches.open(DYNAMIC_CACHE);
    const cachedResponse = await cache.match(request);

    if (cachedResponse) {
      return cachedResponse;
    }

    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      await cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    console.error('[SW] Media request failed:', error);
    // Return placeholder image for failed media requests
    return new Response('', { status: 404 });
  }
}

// Static assets - stale while revalidate
async function handleStaticRequest(request) {
  try {
    const cache = await caches.open(STATIC_CACHE);
    const cachedResponse = await cache.match(request);

    // Serve from cache immediately if available
    if (cachedResponse) {
      // Update cache in background
      fetch(request)
        .then(networkResponse => {
          if (networkResponse.ok) {
            cache.put(request, networkResponse);
          }
        })
        .catch(() => {
          // Ignore background update failures
        });

      return cachedResponse;
    }

    // No cache, fetch from network
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      await cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    console.error('[SW] Static request failed:', error);
    throw error;
  }
}

// Navigation requests - serve app shell
async function handleNavigationRequest(request) {
  try {
    // Try network first for navigation
    const networkResponse = await fetch(request);
    return networkResponse;
  } catch (error) {
    // Fallback to cached app shell
    const cache = await caches.open(STATIC_CACHE);
    const appShell = await cache.match('/');

    if (appShell) {
      return appShell;
    }

    // Ultimate fallback to offline page
    const offlinePage = await cache.match('/offline.html');
    if (offlinePage) {
      return offlinePage;
    }

    return new Response('<h1>Offline</h1><p>Please check your connection and try again.</p>', {
      headers: { 'Content-Type': 'text/html' }
    });
  }
}

// Background sync for failed API requests
self.addEventListener('sync', event => {
  console.log('[SW] Background sync triggered:', event.tag);

  if (event.tag === 'background-sync-api') {
    event.waitUntil(syncPendingRequests());
  }

  if (event.tag === 'background-sync-state') {
    event.waitUntil(syncStateToIndexedDB());
  }
});

// Background sync for state management
async function syncStateToIndexedDB() {
  try {
    console.log('[SW] Syncing state to IndexedDB...');

    // Get state from WASM if available
    if (typeof self.initializeState === 'function') {
      const state = await self.initializeState();
      if (state) {
        // Store in IndexedDB
        const db = await openIndexedDB();
        if (db) {
          const transaction = db.transaction(['userSessions'], 'readwrite');
          const store = transaction.objectStore('userSessions');
          await store.put({
            userId: state.user_id,
            sessionId: state.session_id,
            deviceId: state.device_id,
            timestamp: state.timestamp,
            sessionType: state.is_temporary ? 'guest' : 'authenticated',
            metadata: state.metadata || {},
            computeStats: {
              totalTasks: 0,
              avgProcessingTime: 0,
              peakThroughput: 0
            }
          });
        }
      }
    }
  } catch (error) {
    console.error('[SW] Failed to sync state to IndexedDB:', error);
  }
}

// Open IndexedDB for state management
async function openIndexedDB() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('OvasabiStateDB', 1);

    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);

    request.onupgradeneeded = event => {
      const db = event.target.result;

      // Create userSessions store if it doesn't exist
      if (!db.objectStoreNames.contains('userSessions')) {
        const store = db.createObjectStore('userSessions', { keyPath: 'userId' });
        store.createIndex('sessionId', 'sessionId');
        store.createIndex('timestamp', 'timestamp');
        store.createIndex('sessionType', 'sessionType');
      }
    };
  });
}

// Push notifications
self.addEventListener('push', event => {
  console.log('[SW] Push notification received');

  const options = {
    body: event.data?.text() || 'New notification from OVASABI',
    icon: '/icons/icon-192x192.png',
    badge: '/icons/badge-72x72.png',
    vibrate: [200, 100, 200],
    actions: [
      {
        action: 'open',
        title: 'Open App'
      },
      {
        action: 'dismiss',
        title: 'Dismiss'
      }
    ]
  };

  event.waitUntil(self.registration.showNotification('OVASABI Platform', options));
});

// Notification click handler
self.addEventListener('notificationclick', event => {
  console.log('[SW] Notification clicked:', event.action);

  event.notification.close();

  if (event.action === 'open' || !event.action) {
    event.waitUntil(clients.openWindow('/'));
  }
});

// Helper functions
function isWASMRequest(request) {
  return request.url.includes('.wasm') || request.url.includes('wasm_exec.js');
}

function isWorkerRequest(request) {
  return request.url.includes('/workers/') || request.url.includes('worker.js');
}

function isAPIRequest(request) {
  return (
    request.url.includes('/api/') ||
    request.url.includes('/ws/') ||
    request.url.includes('event_type=')
  );
}

function isMediaRequest(request) {
  const url = request.url.toLowerCase();
  return (
    url.includes('/media/') ||
    url.includes('/uploads/') ||
    /\.(jpg|jpeg|png|gif|webp|svg|mp4|webm|mp3|wav|ogg)$/i.test(url)
  );
}

function isStaticAsset(request) {
  const url = request.url.toLowerCase();
  return (
    /\.(js|css|ico|json|txt)$/i.test(url) || url.includes('/static/') || url.includes('/assets/')
  );
}

async function syncPendingRequests() {
  // Implement background sync logic for failed API requests
  console.log('[SW] Syncing pending requests...');
  // This would integrate with your event-driven architecture
  // to retry failed events when connectivity is restored
}

console.log('[SW] Service worker script loaded');
