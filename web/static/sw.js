const CACHE_NAME = 'stuff-v1';
const ASSETS = [
  '/',
  '/static/styles.css',
  '/static/nav-menu.js',
  '/static/confirm-dialog.js',
  '/static/tag-autocomplete.js'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(ASSETS);
    })
  );
});

self.addEventListener('fetch', (event) => {
  event.respondWith(
    caches.match(event.request).then((response) => {
      return response || fetch(event.request);
    })
  );
});
