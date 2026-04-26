// Tells the service worker to drop its cached `/notifications` response so the
// next list fetch hits the network. Used after marking notifications read and
// on logout so a switched-user session can't see another user's cached list.
export function clearNotificationsCache(): void {
  if (!('serviceWorker' in navigator)) return;
  navigator.serviceWorker.ready
    .then((reg) => reg.active?.postMessage({ type: 'CLEAR_NOTIFICATIONS_CACHE' }))
    .catch(() => { /* best-effort */ });
}
