import { fetchWithAuth } from './fetchWithAuth';

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
  readAt?: string;
}

export interface ListNotificationsResponse {
  notifications: ApiNotificationItem[];
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

export async function markNotificationRead(id: string): Promise<void> {
  await fetchWithAuth<Record<string, never>>(
    `${NOTIFICATOR_API_URL}/notifications/read/${encodeURIComponent(id)}`,
    { method: 'POST' },
  );
}

export async function markAllNotificationsRead(): Promise<void> {
  await fetchWithAuth<Record<string, never>>(
    `${NOTIFICATOR_API_URL}/notifications/read`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    },
  );
}
