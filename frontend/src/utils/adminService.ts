import { throwIfError } from './apiError';
import type { AdminUpdateTaskPatch } from './taskerService';

const AUTH_API_URL =
  (import.meta.env.VITE_AUTH_API_URL as string | undefined) ??
  'https://auth.persomanage.ru';

const TASKER_API_URL =
  (import.meta.env.VITE_TASKER_API_URL as string | undefined) ??
  'https://tasker.persomanage.ru';

const NOTIFICATOR_API_URL =
  (import.meta.env.VITE_NOTIFICATOR_API_URL as string | undefined) ??
  'https://notificator.persomanage.ru';

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
      'x-api-key': key,
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

export interface AdminCreateTaskPayload {
  title: string;
  description?: string;
  durationMinutes?: number;
  priority?: number;
  deadline?: string; // ISO 8601
  startTime?: string; // ISO 8601
  endTime?: string; // ISO 8601
  status?: string; // "unplanned" | "planned" | "completed"
  category?: string; // "work" | "study" | "personal"
  isApproved?: boolean;
}

export async function createAdminTask(
  userId: string,
  payload: AdminCreateTaskPayload,
): Promise<AdminTaskItem> {
  const data = await fetchAdmin<AdminTaskResponse>(
    `${TASKER_API_URL}/admin/users/${encodeURIComponent(userId)}/tasks`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    },
  );
  return data.task;
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

export interface AdminClusterListItem {
  id: string;
  userId: string;
  status: string;
  eventCount: number;
  generationOutcome?: string;
  generationReason?: string;
  taskId?: string;
  createdAt: string;
  updatedAt: string;
}

interface AdminClusterListResponse {
  clusters: AdminClusterListItem[];
}

export async function listAdminClusters(userId: string): Promise<AdminClusterListItem[]> {
  const data = await fetchAdmin<AdminClusterListResponse>(
    `${TASKER_API_URL}/admin/users/${encodeURIComponent(userId)}/clusters`,
  );
  return data.clusters ?? [];
}

interface AdminClusterResponse {
  cluster: AdminClusterListItem;
}

export async function getAdminCluster(
  userId: string,
  clusterId: string,
): Promise<AdminClusterListItem> {
  const data = await fetchAdmin<AdminClusterResponse>(
    `${TASKER_API_URL}/admin/users/${encodeURIComponent(userId)}/clusters/${encodeURIComponent(clusterId)}`,
  );
  return data.cluster;
}

export interface AdminClusterEventItem {
  id: string;
  userId: string;
  clusterId: string;
  source: string;
  occurredAt: string;
  context: string;
  similarity: number;
}

interface AdminClusterEventsResponse {
  events: AdminClusterEventItem[];
}

export async function listAdminClusterEvents(
  userId: string,
  clusterId: string,
): Promise<AdminClusterEventItem[]> {
  const data = await fetchAdmin<AdminClusterEventsResponse>(
    `${TASKER_API_URL}/admin/users/${encodeURIComponent(userId)}/clusters/${encodeURIComponent(clusterId)}/events`,
  );
  return data.events ?? [];
}

export interface AdminPromptItem {
  id: string;
  description: string;
  systemTemplate: string;
  userTemplate: string;
  updatedAt: string;
}

interface AdminPromptListResponse {
  prompts: AdminPromptItem[];
}

interface AdminPromptResponse {
  prompt: AdminPromptItem;
}

export async function listAdminPrompts(): Promise<AdminPromptItem[]> {
  const data = await fetchAdmin<AdminPromptListResponse>(`${TASKER_API_URL}/admin/prompts`);
  return data.prompts ?? [];
}

export async function getAdminPrompt(promptId: string): Promise<AdminPromptItem> {
  const data = await fetchAdmin<AdminPromptResponse>(
    `${TASKER_API_URL}/admin/prompts/${encodeURIComponent(promptId)}`,
  );
  return data.prompt;
}

export interface AdminPromptUpdate {
  systemTemplate?: string;
  userTemplate?: string;
}

export async function updateAdminPrompt(
  promptId: string,
  patch: AdminPromptUpdate,
): Promise<AdminPromptItem> {
  const data = await fetchAdmin<AdminPromptResponse>(
    `${TASKER_API_URL}/admin/prompts/${encodeURIComponent(promptId)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(patch),
    },
  );
  return data.prompt;
}

export interface AdminPushPayload {
  title: string;
  body: string;
  type?: string;
  url?: string;
  icon?: string;
}

export async function sendAdminPushToUser(
  userId: string,
  payload: AdminPushPayload,
): Promise<void> {
  await fetchAdmin<Record<string, never>>(`${NOTIFICATOR_API_URL}/v1/push/admin/send`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      notification: {
        recipientId: userId,
        title: payload.title,
        body: payload.body,
        type: payload.type,
        url: payload.url,
        icon: payload.icon,
      },
    }),
  });
}

export interface AdminNotificationItem {
  id: string;
  title: string;
  type: string;
  text: string;
  sentAt?: string;
  readAt?: string;
}

interface AdminNotificationsResponse {
  notifications: AdminNotificationItem[];
}

export async function listAdminUserNotifications(
  userId: string,
): Promise<AdminNotificationItem[]> {
  const data = await fetchAdmin<AdminNotificationsResponse>(
    `${NOTIFICATOR_API_URL}/v1/push/admin/users/${encodeURIComponent(userId)}/notifications`,
  );
  return data.notifications ?? [];
}
