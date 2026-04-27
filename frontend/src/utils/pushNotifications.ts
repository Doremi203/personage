import { fetchWithTokenRefresh } from './authService';

const SUBSCRIBE_ENDPOINT = 'https://notificator.persomanage.ru/v1/push/subscribe';
const UNSUBSCRIBE_ENDPOINT = 'https://notificator.persomanage.ru/v1/push/unsubscribe';
const VAPID_PUBLIC_KEY = 'BKu7S4HSkG6p8oPbjTB8p1H5PyUCyc4qPzY1FmtOL1eKyV6hXvcGJ99kIfHW88Atd7n2Co4RMOFtR70fD8CLFHI'

export function isIos(): boolean {
  return /iPad|iPhone|iPod/.test(navigator.userAgent);
}

export function isStandalonePWA(): boolean {
  if ('standalone' in navigator && (navigator as unknown as { standalone: boolean }).standalone
  ) {
    return true;
  }
  return window.matchMedia('(display-mode: standalone)').matches;
}

export function isPushSupported(): boolean {
  return (
    'Notification' in window &&
    'PushManager' in window &&
    'serviceWorker' in navigator
  );
}

export function getNotificationPermission():
  | NotificationPermission
  | 'unsupported' {
  if (!('Notification' in window)) return 'unsupported';
  return Notification.permission;
}


function withTimeout<T>(
  promise: Promise<T>,
  ms: number,
  message: string,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(
      () => reject(new Error(message)),
      ms,
    );
    promise.then(
      (value) => {
        clearTimeout(timer);
        resolve(value);
      },
      (err) => {
        clearTimeout(timer);
        reject(err instanceof Error ? err : new Error(String(err)));
      },
    );
  });
}

// iOS Safari (especially in standalone PWA mode) sometimes never resolves the
// Promise returned by Notification.requestPermission() after the user dismisses
// the system dialog, even though Notification.permission does flip to its new
// value. Without a fallback the calling UI stays stuck on a loading spinner.
// We race the native Promise with a poll on Notification.permission and a
// hard timeout so the caller always gets an answer.
export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (!('Notification' in window)) {
    throw new Error('Notifications are not supported in this browser');
  }

  return new Promise<NotificationPermission>((resolve, reject) => {
    let settled = false;
    let pollId: ReturnType<typeof setInterval> | null = null;
    let hardTimeoutId: ReturnType<typeof setTimeout> | null = null;

    const cleanup = () => {
      if (pollId !== null) {
        clearInterval(pollId);
        pollId = null;
      }
      if (hardTimeoutId !== null) {
        clearTimeout(hardTimeoutId);
        hardTimeoutId = null;
      }
    };
    const settle = (value: NotificationPermission) => {
      if (settled) return;
      settled = true;
      cleanup();
      resolve(value);
    };
    const fail = (err: Error) => {
      if (settled) return;
      settled = true;
      cleanup();
      reject(err);
    };

    pollId = setInterval(() => {
      const perm = Notification.permission;
      if (perm === 'granted' || perm === 'denied') {
        settle(perm);
      }
    }, 500);

    hardTimeoutId = setTimeout(() => {
      const perm = Notification.permission;
      if (perm === 'granted' || perm === 'denied') {
        settle(perm);
      } else {
        fail(new Error('Permission request timed out'));
      }
    }, 15_000);

    try {
      Notification.requestPermission().then(
        (value) => settle(value),
        (err) => fail(err instanceof Error ? err : new Error(String(err))),
      );
    } catch (err) {
      fail(err instanceof Error ? err : new Error(String(err)));
    }
  });
}

function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replace(/-/g, '+')
    .replace(/_/g, '/');
  const rawData = atob(base64);
  const buffer = new ArrayBuffer(rawData.length);
  const outputArray = new Uint8Array(buffer);
  for (let i = 0; i < rawData.length; i++) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

// navigator.serviceWorker.ready can hang on iOS PWA when a SW is installing
// for the first time or recently updated. Prefer a registration with an
// already-active worker before falling back to .ready, and give .ready a
// generous deadline so slow mobile networks don't trip the timeout.
async function getReadyRegistration(): Promise<ServiceWorkerRegistration> {
  const existing = await navigator.serviceWorker.getRegistration();
  if (existing?.active) return existing;

  return withTimeout(
    navigator.serviceWorker.ready,
    20_000,
    'Service worker is not ready',
  );
}

export async function subscribeToPush(
  vapidPublicKey: string,
): Promise<PushSubscription> {
  const registration = await getReadyRegistration();

  const existingSubscription =
    await registration.pushManager.getSubscription();
  if (existingSubscription) {
    return existingSubscription;
  }

  return withTimeout(
    registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
    }),
    15_000,
    'Push subscription request timed out',
  );
}

export async function sendSubscriptionToBackend(
  subscription: PushSubscription,
): Promise<void> {
  const subscriptionJSON = subscription.toJSON();
  const keys = subscriptionJSON.keys;

  if (!keys?.p256dh || !keys?.auth) {
    throw new Error('Push subscription is missing encryption keys');
  }

  const body = {
    endpoint: subscriptionJSON.endpoint,
    p256dh: keys.p256dh,
    auth_key: keys.auth,
  };

  const response = await withTimeout(
    fetchWithTokenRefresh((accessToken) =>
      fetch(SUBSCRIBE_ENDPOINT, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'User-Token': accessToken,
        },
        body: JSON.stringify(body),
      }),
    ),
    15_000,
    'Saving push subscription on the server timed out',
  );

  if (!response.ok) {
    throw new Error(
      `Failed to register push subscription: ${response.status} ${response.statusText}`,
    );
  }
}

export async function unsubscribeFromPush(): Promise<void> {
  const registration = await getReadyRegistration();
  const subscription = await registration.pushManager.getSubscription();
  if (!subscription) return;

  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(UNSUBSCRIBE_ENDPOINT, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'User-Token': accessToken,
      },
      body: JSON.stringify({ endpoint: subscription.endpoint }),
    }),
  );

  if (!response.ok) {
    throw new Error(
      `Failed to unregister push subscription: ${response.status} ${response.statusText}`,
    );
  }

  await subscription.unsubscribe();
}

export type PushSetupResult =
  | { status: 'subscribed' }
  | { status: 'denied' }
  | { status: 'unsupported' }
  | { status: 'error'; error: Error };

export async function setupPushNotifications(): Promise<PushSetupResult> {
  if (!isPushSupported()) {
    return { status: 'unsupported' };
  }

  try {
    const permission = await requestNotificationPermission();

    if (permission !== 'granted') {
      return { status: 'denied' };
    }

    const subscription = await subscribeToPush(VAPID_PUBLIC_KEY);
    await sendSubscriptionToBackend(subscription);

    return { status: 'subscribed' };
  } catch (error) {
    console.error('Push notification setup failed:', error);
    return {
      status: 'error',
      error: error instanceof Error ? error : new Error(String(error)),
    };
  }
}
