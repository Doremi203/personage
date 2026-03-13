import { getTokens } from './authService';

const SUBSCRIBE_ENDPOINT = 'https://notificator.persomanage.ru/v1/push/subscribe';
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


export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (!('Notification' in window)) {
    throw new Error('Notifications are not supported in this browser');
  }
  return Notification.requestPermission();
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

export async function subscribeToPush(
  vapidPublicKey: string,
): Promise<PushSubscription> {
  const registration = await navigator.serviceWorker.ready;

  const existingSubscription =
    await registration.pushManager.getSubscription();
  if (existingSubscription) {
    return existingSubscription;
  }

  return await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
  });
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

  const tokens = getTokens();
  const response = await fetch(SUBSCRIBE_ENDPOINT, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(tokens?.accessToken
        ? { 'User-Token': tokens.accessToken }
        : {}),
    },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    throw new Error(
      `Failed to register push subscription: ${response.status} ${response.statusText}`,
    );
  }
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
