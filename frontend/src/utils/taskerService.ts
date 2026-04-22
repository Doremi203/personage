import { fetchWithTokenRefresh } from './authService';

const TASKER_API_URL =
  (import.meta.env.VITE_TASKER_API_URL as string | undefined) ??
  'https://tasker.persomanage.ru';

// String enum values matching task.proto enum names (used in query params and JSON responses)
export const ApiTaskStatusFilter = {
  UNSPECIFIED: 'TASK_STATUS_FILTER_UNSPECIFIED',
  UNPLANNED: 'TASK_STATUS_FILTER_UNPLANNED',
  PLANNED: 'TASK_STATUS_FILTER_PLANNED',
  COMPLETED: 'TASK_STATUS_FILTER_COMPLETED',
} as const;

export const ApiTaskCategoryFilter = {
  UNSPECIFIED: 'TASK_CATEGORY_FILTER_UNSPECIFIED',
  WORK: 'TASK_CATEGORY_FILTER_WORK',
  STUDY: 'TASK_CATEGORY_FILTER_STUDY',
  PERSONAL: 'TASK_CATEGORY_FILTER_PERSONAL',
} as const;

export const ApiTaskPriority = {
  UNSPECIFIED: 'TASK_PRIORITY_UNSPECIFIED',
  LOW: 'TASK_PRIORITY_LOW',
  MID: 'TASK_PRIORITY_MID',
  HIGH: 'TASK_PRIORITY_HIGH',
} as const;

export const ApiTaskStatus = {
  UNSPECIFIED: 'TASK_STATUS_UNSPECIFIED',
  UNPLANNED: 'TASK_STATUS_UNPLANNED',
  PLANNED: 'TASK_STATUS_PLANNED',
  COMPLETED: 'TASK_STATUS_COMPLETED',
} as const;

export const ApiTaskCategory = {
  UNSPECIFIED: 'TASK_CATEGORY_UNSPECIFIED',
  WORK: 'TASK_CATEGORY_WORK',
  STUDY: 'TASK_CATEGORY_STUDY',
  PERSONAL: 'TASK_CATEGORY_PERSONAL',
} as const;

// API response types (camelCase as returned by gRPC-gateway JSON, enums as string names)
export interface ApiTaskItem {
  id: string;
  title: string;
  description: string;
  startTime?: string;
  endTime?: string;
  deadline?: string;
  priority: string;
  status: string;
  category: string;
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
  status?: string;
  category?: string;
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
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'User-Token': accessToken,
      },
    }),
  );

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
  if (params.status) query.set('status', params.status);
  if (params.category) query.set('category', params.category);
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
