import { throwIfError } from './apiError';
import type { AdminUpdateTaskPatch } from './taskerService';

const AUTH_API_URL =
  (import.meta.env.VITE_AUTH_API_URL as string | undefined) ??
  'https://auth.persomanage.ru';

const TASKER_API_URL =
  (import.meta.env.VITE_TASKER_API_URL as string | undefined) ??
  'https://tasker.persomanage.ru';

export const ADMIN_KEY_STORAGE_KEY = 'personage_admin_key';
export const ADMIN_KEY_CHANGE_EVENT = 'personage-admin-key-change';

export function getAdminKey(): string | null {
  return localStorage.getItem(ADMIN_KEY_STORAGE_KEY);
}

export function setAdminKey(key: string): void {
  localStorage.setItem(ADMIN_KEY_STORAGE_KEY, key);
  window.dispatchEvent(new Event(ADMIN_KEY_CHANGE_EVENT));
}

export function clearAdminKey(): void {
  localStorage.removeItem(ADMIN_KEY_STORAGE_KEY);
  window.dispatchEvent(new Event(ADMIN_KEY_CHANGE_EVENT));
}

async function fetchAdmin<T>(url: string, options: RequestInit = {}): Promise<T> {
  const key = getAdminKey();
  if (!key) {
    throw new Error('Admin API key not set');
  }
  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'X-Admin-Key': key,
    },
  });
  if (response.status === 401) {
    clearAdminKey();
  }
  await throwIfError(response);
  const text = await response.text();
  return text ? (JSON.parse(text) as T) : ({} as T);
}

export interface AdminUserSummary {
  id: string;
  email: string;
  name: string | null;
}

interface AdminUsersResponse {
  users: AdminUserSummary[];
}

export async function listAdminUsers(): Promise<AdminUserSummary[]> {
  const data = await fetchAdmin<AdminUsersResponse>(`${AUTH_API_URL}/admin/users`);
  return data.users;
}

export interface AdminTaskItem {
  id: string;
  userId: string;
  clusterId?: string;
  title: string;
  description: string;
  durationMinutes: number;
  priority: number;
  deadline?: string;
  startTime?: string;
  endTime?: string;
  status: string;
  category: string;
  evidenceEventIds?: string[];
  isApproved: boolean;
  createdAt: string;
  updatedAt: string;
}

interface AdminTaskListResponse {
  tasks: AdminTaskItem[];
}

interface AdminTaskResponse {
  task: AdminTaskItem;
}

export async function listAdminTasks(userId: string): Promise<AdminTaskItem[]> {
  const data = await fetchAdmin<AdminTaskListResponse>(
    `${TASKER_API_URL}/admin/users/${encodeURIComponent(userId)}/tasks`,
  );
  return data.tasks ?? [];
}

export async function getAdminTask(userId: string, taskId: string): Promise<AdminTaskItem> {
  const data = await fetchAdmin<AdminTaskResponse>(
    `${TASKER_API_URL}/admin/users/${encodeURIComponent(userId)}/tasks/${encodeURIComponent(taskId)}`,
  );
  return data.task;
}

export async function updateAdminTask(
  userId: string,
  taskId: string,
  patch: AdminUpdateTaskPatch,
): Promise<AdminTaskItem> {
  const data = await fetchAdmin<AdminTaskResponse>(
    `${TASKER_API_URL}/admin/users/${encodeURIComponent(userId)}/tasks/${encodeURIComponent(taskId)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(patch),
    },
  );
  return data.task;
}

export async function approveAdminTask(userId: string, taskId: string): Promise<AdminTaskItem> {
  const data = await fetchAdmin<AdminTaskResponse>(
    `${TASKER_API_URL}/admin/users/${encodeURIComponent(userId)}/tasks/${encodeURIComponent(taskId)}/approve`,
    { method: 'POST' },
  );
  return data.task;
}

interface AdminModerationResponse {
  userIds: string[];
}

export async function listModeratedUserIds(): Promise<string[]> {
  const data = await fetchAdmin<AdminModerationResponse>(`${TASKER_API_URL}/admin/moderation`);
  return data.userIds ?? [];
}

export async function setUserModeration(userId: string, enabled: boolean): Promise<void> {
  await fetchAdmin<Record<string, never>>(
    `${TASKER_API_URL}/admin/moderation/${encodeURIComponent(userId)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    },
  );
}
