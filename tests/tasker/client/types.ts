export type TaskStatus =
  | 'TASK_STATUS_UNSPECIFIED'
  | 'TASK_STATUS_UNPLANNED'
  | 'TASK_STATUS_PLANNED'
  | 'TASK_STATUS_COMPLETED';

export type TaskPriority =
  | 'TASK_PRIORITY_UNSPECIFIED'
  | 'TASK_PRIORITY_LOW'
  | 'TASK_PRIORITY_MID'
  | 'TASK_PRIORITY_HIGH';

export type TaskCategory =
  | 'TASK_CATEGORY_UNSPECIFIED'
  | 'TASK_CATEGORY_WORK'
  | 'TASK_CATEGORY_STUDY'
  | 'TASK_CATEGORY_PERSONAL';

export type TaskStatusFilter =
  | 'TASK_STATUS_FILTER_UNSPECIFIED'
  | 'TASK_STATUS_FILTER_UNPLANNED'
  | 'TASK_STATUS_FILTER_PLANNED'
  | 'TASK_STATUS_FILTER_COMPLETED';

export type TaskCategoryFilter =
  | 'TASK_CATEGORY_FILTER_UNSPECIFIED'
  | 'TASK_CATEGORY_FILTER_WORK'
  | 'TASK_CATEGORY_FILTER_STUDY'
  | 'TASK_CATEGORY_FILTER_PERSONAL';

export interface TaskItem {
  id: string;
  title: string;
  description: string;
  startTime?: string;
  endTime?: string;
  deadline?: string;
  priority: TaskPriority;
  status: TaskStatus;
  category: TaskCategory;
  updatedAt: string;
  createdAt: string;
}

export interface ListTasksResponse {
  tasks: TaskItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ListTasksParams {
  status?: TaskStatusFilter;
  category?: TaskCategoryFilter;
  text?: string;
  from?: string; // DD-MM-YYYY
  till?: string; // DD-MM-YYYY
  pageSize?: number;
  page?: number;
}

export interface UpdateTaskRequest {
  title?: string;
  description?: string;
  startTime?: string; // RFC3339
  endTime?: string;   // RFC3339
  category?: 'TASK_CATEGORY_WORK' | 'TASK_CATEGORY_STUDY' | 'TASK_CATEGORY_PERSONAL';
}

export interface CreateTestTaskRequest {
  user_id: string;
  title: string;
  description?: string;
  status?: 'unplanned' | 'planned' | 'completed';
  priority?: number; // 1–10
  category?: 'work' | 'study' | 'personal';
  start_time?: string; // RFC3339
  end_time?: string;   // RFC3339
  deadline?: string;   // RFC3339
}

export interface CreateTestTaskResponse {
  id: string;
}
