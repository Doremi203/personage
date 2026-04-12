import type {
  CreateTestTaskRequest,
  CreateTestTaskResponse,
  ListTasksParams,
  UpdateTaskRequest,
} from './types';

export class TaskerClient {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl;
    this.token = token;
  }

  private headers(): Record<string, string> {
    const h: Record<string, string> = { 'Content-Type': 'application/json' };
    if (this.token) {
      h['User-Token'] = this.token;
    }
    return h;
  }

  listTasks(params: ListTasksParams = {}): Promise<Response> {
    const query = new URLSearchParams();
    if (params.status)   query.set('status',   params.status);
    if (params.category) query.set('category', params.category);
    if (params.text)     query.set('text',     params.text);
    if (params.from)     query.set('from',     params.from);
    if (params.till)     query.set('till',     params.till);
    if (params.pageSize) query.set('pageSize', String(params.pageSize));
    if (params.page)     query.set('page',     String(params.page));

    const qs = query.toString();
    return fetch(`${this.baseUrl}/v1/tasks${qs ? `?${qs}` : ''}`, {
      headers: this.headers(),
    });
  }

  getTask(id: string): Promise<Response> {
    return fetch(`${this.baseUrl}/v1/tasks/${encodeURIComponent(id)}`, {
      headers: this.headers(),
    });
  }

  updateTask(id: string, body: UpdateTaskRequest): Promise<Response> {
    return fetch(`${this.baseUrl}/v1/tasks/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      headers: this.headers(),
      body: JSON.stringify(body),
    });
  }

  postponeTask(id: string): Promise<Response> {
    return fetch(`${this.baseUrl}/v1/tasks/${encodeURIComponent(id)}/postpone`, {
      method: 'POST',
      headers: this.headers(),
    });
  }

  completeTask(id: string): Promise<Response> {
    return fetch(`${this.baseUrl}/v1/tasks/${encodeURIComponent(id)}/complete`, {
      method: 'POST',
      headers: this.headers(),
    });
  }

  deleteTask(id: string): Promise<Response> {
    return fetch(`${this.baseUrl}/v1/tasks/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: this.headers(),
    });
  }

  /** Creates a task via the test-only endpoint. Throws if the request fails. */
  async createTestTask(req: CreateTestTaskRequest): Promise<string> {
    const res = await fetch(`${this.baseUrl}/v1/test/tasks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`createTestTask failed (${res.status}): ${text}`);
    }
    const body = (await res.json()) as CreateTestTaskResponse;
    return body.id;
  }
}
