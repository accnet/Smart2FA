// Smart2FA Service Worker — caches static assets for fast loading
const CACHE_NAME = 'smart2fa-v1';

const STATIC_ASSETS = [
  '/static/css/app.css',
  '/static/js/app.js',
  '/static/icons/icon-192.png',
  '/static/icons/icon-512.png',
  '/static/manifest.json',
];

// Install: cache static assets
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME).then(cache => cache.addAll(STATIC_ASSETS))
  );
  self.skipWaiting();
});

// Activate: clean up old caches
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(k => k !== CACHE_NAME).map(k => caches.delete(k)))
    )
  );
  self.clients.claim();
});

// Fetch: serve static assets from cache; let API/HTML go through network
self.addEventListener('fetch', event => {
  const url = new URL(event.request.url);

  // Only cache GET requests for static assets
  if (event.request.method === 'GET' && url.pathname.startsWith('/static/')) {
    event.respondWith(
      caches.match(event.request).then(cached => cached || fetch(event.request))
    );
    return;
  }

  // Everything else (HTML, API) → always fetch from network
  // This ensures TOTP codes are always live
});
