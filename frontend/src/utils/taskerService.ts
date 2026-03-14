import { getTokens, refreshAccessToken } from './authService';

const TASKER_API_URL =
  (import.meta.env.VITE_TASKER_API_URL as string | undefined) ??
  'https://tasker.persomanage.ru';

// Numeric enum values from task.proto
export const ApiTaskStatusFilter = {
  UNSPECIFIED: 0,
  UNPLANNED: 1,
  PLANNED: 2,
  COMPLETED: 3,
} as const;

export const ApiTaskCategoryFilter = {
  UNSPECIFIED: 0,
  WORK: 1,
  STUDY: 2,
  PERSONAL: 3,
} as const;

export const ApiTaskPriority = {
  UNSPECIFIED: 0,
  LOW: 1,
  MID: 2,
  HIGH: 3,
} as const;

export const ApiTaskStatus = {
  UNSPECIFIED: 0,
  UNPLANNED: 1,
  PLANNED: 2,
  COMPLETED: 3,
} as const;

export const ApiTaskCategory = {
  UNSPECIFIED: 0,
  WORK: 1,
  STUDY: 2,
  PERSONAL: 3,
} as const;

// API response types (camelCase as returned by gRPC-gateway JSON)
export interface ApiTaskItem {
  id: string;
  title: string;
  description: string;
  startTime?: string;
  endTime?: string;
  deadline?: string;
  priority: number;
  status: number;
  category: number;
  updatedAt: string;
  createdAt: string;
}

export interface ListTasksResponse {
  tasks: ApiTaskItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ListTasksParams {
  status?: number;
  category?: number;
  text?: string;
  from?: string; // DD-MM-YYYY
  till?: string; // DD-MM-YYYY
  pageSize?: number;
  page?: number;
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

export async function listTasks(
  params: ListTasksParams = {},
): Promise<ListTasksResponse> {
  const query = new URLSearchParams();
  if (params.status) query.set('status', String(params.status));
  if (params.category) query.set('category', String(params.category));
  if (params.text) query.set('text', params.text);
  if (params.from) query.set('from', params.from);
  if (params.till) query.set('till', params.till);
  if (params.pageSize) query.set('pageSize', String(params.pageSize));
  if (params.page) query.set('page', String(params.page));

  const qs = query.toString();
  return fetchWithAuth<ListTasksResponse>(
    `${TASKER_API_URL}/v1/tasks${qs ? `?${qs}` : ''}`,
  );
}

export async function getTask(id: string): Promise<ApiTaskItem> {
  const data = await fetchWithAuth<{ task: ApiTaskItem }>(
    `${TASKER_API_URL}/v1/tasks/${encodeURIComponent(id)}`,
  );
  return data.task;
}

export async function completeTask(id: string): Promise<ApiTaskItem> {
  const data = await fetchWithAuth<{ task: ApiTaskItem }>(
    `${TASKER_API_URL}/v1/tasks/${encodeURIComponent(id)}/complete`,
    { method: 'POST' },
  );
  return data.task;
}

export async function postponeTask(id: string): Promise<ApiTaskItem> {
  const data = await fetchWithAuth<{ task: ApiTaskItem }>(
    `${TASKER_API_URL}/v1/tasks/${encodeURIComponent(id)}/postpone`,
    { method: 'POST' },
  );
  return data.task;
}

export async function deleteTask(id: string): Promise<void> {
  await fetchWithAuth<Record<string, never>>(
    `${TASKER_API_URL}/v1/tasks/${encodeURIComponent(id)}`,
    { method: 'DELETE' },
  );
}
