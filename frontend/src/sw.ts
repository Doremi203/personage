/// <reference lib="webworker" />

import {cleanupOutdatedCaches, precacheAndRoute} from 'workbox-precaching';
import {registerRoute} from 'workbox-routing';
import {StaleWhileRevalidate} from 'workbox-strategies';

declare let self: ServiceWorkerGlobalScope;

// With injectManifest + registerType: 'autoUpdate', the custom SW must take
// control on its own — vite-plugin-pwa cannot inject skipWaiting/clientsClaim
// into a user-provided sw.ts. Without these, a freshly-installed SW stays in
// "waiting" forever (until every tab is closed), and navigator.serviceWorker.ready
// hangs — which on iOS PWA surfaces as "Service worker is not ready" when the
// user enables push notifications.
self.skipWaiting();
self.addEventListener('activate', (event) => {
    event.waitUntil(self.clients.claim());
});

cleanupOutdatedCaches();
precacheAndRoute(self.__WB_MANIFEST);

const USER_CACHE = 'personage-user-v1';
const NOTIFICATIONS_CACHE = 'personage-notifications-v1';

registerRoute(
    ({url, request}) => request.method === 'GET' && url.pathname === '/user',
    new StaleWhileRevalidate({cacheName: USER_CACHE}),
);

registerRoute(
    ({url, request}) => request.method === 'GET' && url.pathname === '/notifications',
    new StaleWhileRevalidate({cacheName: NOTIFICATIONS_CACHE}),
);

self.addEventListener('message', (event: ExtendableMessageEvent) => {
    const data = event.data as {type?: string} | undefined;
    if (data?.type === 'CLEAR_USER_CACHE') {
        event.waitUntil(caches.delete(USER_CACHE));
    }
    if (data?.type === 'CLEAR_NOTIFICATIONS_CACHE') {
        event.waitUntil(caches.delete(NOTIFICATIONS_CACHE));
    }
});

interface PushPayload {
    title: string;
    body: string;
    url: string;
    icon: string;
}

self.addEventListener('push', (event: PushEvent) => {
    if (!event.data) return;

    let payload: PushPayload;
    try {
        payload = event.data.json() as PushPayload;
    } catch {
        payload = {
            title: 'Personage',
            body: event.data.text(),
            url: '/',
            icon: '/icon-192x192.png',
        };
    }

    const options: NotificationOptions = {
        body: payload.body,
        icon: payload.icon || '/icon-192x192.png',
        badge: '/icon-96x96.png',
        data: {url: payload.url || '/'},
    };

    event.waitUntil(
        self.registration.showNotification(payload.title, options),
    );
});

self.addEventListener('notificationclick', (event: NotificationEvent) => {
    event.notification.close();

    const targetUrl = (event.notification.data?.url as string) || '/';

    event.waitUntil(
        self.clients.matchAll({
            type: 'window',
            includeUncontrolled: true
        }).then((windowClients) => {
            for (const client of windowClients) {
                if (client.url === targetUrl) {
                    return client.focus();
                }
            }

            return self.clients.openWindow(targetUrl);
        }),
    );
});
