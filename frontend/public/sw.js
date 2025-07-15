// Service Worker for OVASABI Platform PWA
// Provides offline capabilities, caching, and background sync

const CACHE_NAME = 'ovasabi-v1.0.0';
const STATIC_CACHE = `${CACHE_NAME}-static`;
const DYNAMIC_CACHE = `${CACHE_NAME}-dynamic`;
const WASM_CACHE = `${CACHE_NAME}-wasm`;

// Assets to cache immediately on install
const STATIC_ASSETS = [
  '/',
  '/offline.html',
  '/main.wasm',
  '/main.threads.wasm',
  '/wasm_exec.js',
  '/sounds/notification.mp3',
  '/sounds/success.mp3',
  '/sounds/error.mp3'
];

// Cache strategies for different asset types
const CACHE_STRATEGIES = {
  wasm: 'cache-first',
  static: 'stale-while-revalidate',
  api: 'network-first',
  media: 'cache-first'
};

// Install event - cache static assets
self.addEventListener('install', event => {
  console.log('[SW] Installing service worker...');

  event.waitUntil(
    Promise.all([
      // Cache static assets
      caches.open(STATIC_CACHE).then(cache => {
        console.log('[SW] Caching static assets');
        return cache.addAll(STATIC_ASSETS);
      }),

      // Cache WASM files separately for better performance
      caches.open(WASM_CACHE).then(cache => {
        console.log('[SW] Caching WASM assets');
        return cache.addAll(['/main.wasm', '/main.threads.wasm', '/wasm_exec.js']);
      })
    ]).then(() => {
      console.log('[SW] Installation complete');
      // Skip waiting to activate immediately
      return self.skipWaiting();
    })
  );
});

// Activate event - cleanup old caches
self.addEventListener('activate', event => {
  console.log('[SW] Activating service worker...');

  event.waitUntil(
    caches
      .keys()
      .then(cacheNames => {
        return Promise.all(
          cacheNames.map(cacheName => {
            // Delete old caches that don't match current version
            if (cacheName.startsWith('ovasabi-') && cacheName !== CACHE_NAME) {
              console.log(`[SW] Deleting old cache: ${cacheName}`);
              return caches.delete(cacheName);
            }
          })
        );
      })
      .then(() => {
        console.log('[SW] Activation complete');
        // Take control of all clients immediately
        return self.clients.claim();
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

// WASM requests - cache first (WASM files rarely change)
async function handleWASMRequest(request) {
  try {
    const cache = await caches.open(WASM_CACHE);
    const cachedResponse = await cache.match(request);

    if (cachedResponse) {
      console.log('[SW] Serving WASM from cache:', request.url);
      return cachedResponse;
    }

    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      await cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    console.error('[SW] WASM request failed:', error);
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
    return (
      cache.match('/offline.html') ||
      new Response('<h1>Offline</h1><p>Please check your connection and try again.</p>', {
        headers: { 'Content-Type': 'text/html' }
      })
    );
  }
}

// Background sync for failed API requests
self.addEventListener('sync', event => {
  console.log('[SW] Background sync triggered:', event.tag);

  if (event.tag === 'background-sync-api') {
    event.waitUntil(syncPendingRequests());
  }
});

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
