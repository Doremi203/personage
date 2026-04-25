// Tells the service worker to drop its cached `/user` response so the next
// fetch hits the network. Used after logout and after Gmail OAuth so stale
// per-user state can't leak into a fresh session.
export function clearUserCache(): void {
  if (!('serviceWorker' in navigator)) return;
  navigator.serviceWorker.ready
    .then((reg) => reg.active?.postMessage({ type: 'CLEAR_USER_CACHE' }))
    .catch(() => { /* best-effort */ });
}
