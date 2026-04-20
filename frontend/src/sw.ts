/// <reference lib="webworker" />

import {cleanupOutdatedCaches, precacheAndRoute} from 'workbox-precaching';

declare let self: ServiceWorkerGlobalScope;

cleanupOutdatedCaches();
precacheAndRoute(self.__WB_MANIFEST);

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
