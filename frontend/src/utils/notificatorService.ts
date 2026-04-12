import { getTokens, refreshAccessToken } from './authService';

const NOTIFICATOR_API_URL =
  (import.meta.env.VITE_NOTIFICATOR_API_URL as string | undefined) ??
  'https://notificator.persomanage.ru';

// API response types (camelCase as returned by gRPC-gateway JSON)
export interface ApiNotificationItem {
  id: string;
  title: string;
  type: string;
  text: string;
  sentAt: string;
}

export interface ListNotificationsResponse {
  notifications: ApiNotificationItem[];
}

async function fetchWithAuth<T>(
  url: string,
  options: RequestInit = {},
): Promise<T> {
  const tokens = getTokens();
  if (!tokens) {
    throw new Error('Не авторизован');
  }

  const doFetch = (accessToken: string) =>
    fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'User-Token': accessToken,
      },
    });

  let response = await doFetch(tokens.accessToken);

  if (response.status === 401) {
    const newTokens = await refreshAccessToken();
    if (!newTokens) {
      throw new Error('Сессия истекла');
    }
    response = await doFetch(newTokens.accessToken);
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ?? `Ошибка: ${response.status}`,
    );
  }

  const text = await response.text();
  return (text ? (JSON.parse(text) as T) : ({} as T));
}

export interface NotificationSettingItem {
  type: string;
  enabled: boolean;
}

export interface GetNotificationSettingsResponse {
  settings: NotificationSettingItem[];
}

export async function listNotifications(
  page: number,
  pageSize: number,
): Promise<ListNotificationsResponse> {
  const query = new URLSearchParams();
  query.set('page', String(page));
  query.set('pageSize', String(pageSize));

  return fetchWithAuth<ListNotificationsResponse>(
    `${NOTIFICATOR_API_URL}/notifications?${query.toString()}`,
  );
}

export async function toggleNotification(type: string): Promise<boolean> {
  const data = await fetchWithAuth<{ enabled: boolean }>(
    `${NOTIFICATOR_API_URL}/notifications/${encodeURIComponent(type)}/toggle`,
    { method: 'POST' },
  );
  return data.enabled;
}

export async function getNotificationSettings(): Promise<GetNotificationSettingsResponse> {
  return fetchWithAuth<GetNotificationSettingsResponse>(
    `${NOTIFICATOR_API_URL}/notifications/settings`,
  );
}
