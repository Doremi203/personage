import { randomUUID } from 'crypto';
import { TaskerClient } from '../client/tasker';
import type { ListTasksResponse, TaskItem } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.TASKER_URL ?? 'http://localhost:9090';
const USER_ID = randomUUID();
const client = new TaskerClient(BASE_URL, makeToken(USER_ID));
const noAuthClient = new TaskerClient(BASE_URL, '');

const tasks: { id: string; title: string }[] = [];

beforeAll(async () => {
  const todayNoon = new Date();
  todayNoon.setHours(12, 0, 0, 0);
  const todayOnePM = new Date(todayNoon);
  todayOnePM.setHours(13, 0, 0, 0);
  const start = todayNoon.toISOString();
  const end = todayOnePM.toISOString();

  const created = await Promise.all([
    client.createTestTask({ user_id: USER_ID, title: 'Alpha work task',    status: 'unplanned', category: 'work',     priority: 8, start_time: start, end_time: end }),
    client.createTestTask({ user_id: USER_ID, title: 'Beta study task',    status: 'planned',   category: 'study',    priority: 4, start_time: start, end_time: end }),
    client.createTestTask({ user_id: USER_ID, title: 'Gamma personal task', status: 'completed', category: 'personal', priority: 2, start_time: start, end_time: end }),
  ]);
  created.forEach((id, i) => tasks.push({ id, title: ['Alpha work task', 'Beta study task', 'Gamma personal task'][i] }));
});

describe('GET /v1/tasks', () => {
  describe('happy path', () => {
    it('returns all tasks for the authenticated user with correct pagination envelope', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10 });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      expect(body.total).toBe(3);
      expect(body.page).toBe(1);
      expect(body.pageSize).toBe(10);
      expect(body.tasks).toHaveLength(3);
    });

    it('returns tasks with expected shape', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10 });
      const body = (await res.json()) as ListTasksResponse;
      const task = body.tasks[0] as TaskItem;

      expect(task).toMatchObject({
        id:       expect.any(String),
        title:    expect.any(String),
        status:   expect.any(String),
        priority: expect.any(String),
        category: expect.any(String),
        createdAt: expect.any(String),
        updatedAt: expect.any(String),
      });
    });

    it('paginates: pageSize=2 returns 2 tasks with total=3', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 2 });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      expect(body.tasks).toHaveLength(2);
      expect(body.total).toBe(3);
      expect(body.pageSize).toBe(2);
    });

    it('paginates: page=2 with pageSize=2 returns the remaining task', async () => {
      const res = await client.listTasks({ page: 2, pageSize: 2 });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      expect(body.tasks).toHaveLength(1);
      expect(body.page).toBe(2);
    });

    it('filters by status=unplanned', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10, status: 'TASK_STATUS_FILTER_UNPLANNED' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      expect(body.tasks.length).toBeGreaterThanOrEqual(1);
      body.tasks.forEach((t) => expect(t.status).toBe('TASK_STATUS_UNPLANNED'));
    });

    it('filters by status=planned', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10, status: 'TASK_STATUS_FILTER_PLANNED' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      body.tasks.forEach((t) => expect(t.status).toBe('TASK_STATUS_PLANNED'));
    });

    it('filters by status=completed', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10, status: 'TASK_STATUS_FILTER_COMPLETED' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      body.tasks.forEach((t) => expect(t.status).toBe('TASK_STATUS_COMPLETED'));
    });

    it('filters by category=work', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10, category: 'TASK_CATEGORY_FILTER_WORK' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      expect(body.tasks.length).toBeGreaterThanOrEqual(1);
      body.tasks.forEach((t) => expect(t.category).toBe('TASK_CATEGORY_WORK'));
    });

    it('filters by category=study', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10, category: 'TASK_CATEGORY_FILTER_STUDY' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      body.tasks.forEach((t) => expect(t.category).toBe('TASK_CATEGORY_STUDY'));
    });

    it('filters by category=personal', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10, category: 'TASK_CATEGORY_FILTER_PERSONAL' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      body.tasks.forEach((t) => expect(t.category).toBe('TASK_CATEGORY_PERSONAL'));
    });

    it('text search finds tasks whose title contains the query word', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 10, text: 'alpha' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      expect(body.tasks.some((t) => t.title.toLowerCase().includes('alpha'))).toBe(true);
    });

    it('returns empty tasks array for a user with no tasks', async () => {
      const emptyUserClient = new TaskerClient(BASE_URL, makeToken(randomUUID()));
      const res = await emptyUserClient.listTasks({ page: 1, pageSize: 10 });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      expect(body.tasks).toHaveLength(0);
      expect(body.total).toBe(0);
    });

    it('filters tasks within a date range that includes today', async () => {
      const today = new Date();
      const yesterday = new Date(today);
      yesterday.setDate(today.getDate() - 1);
      const tomorrow = new Date(today);
      tomorrow.setDate(today.getDate() + 1);

      const fmt = (d: Date) =>
        `${String(d.getDate()).padStart(2, '0')}-${String(d.getMonth() + 1).padStart(2, '0')}-${d.getFullYear()}`;

      const res = await client.listTasks({ page: 1, pageSize: 10, from: fmt(yesterday), till: fmt(tomorrow) });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListTasksResponse;
      expect(body.total).toBe(3);
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const res = await noAuthClient.listTasks({ page: 1, pageSize: 10 });
      expect(res.status).toBe(401);
    });

    it('returns 400 when pageSize is 0 (below minimum)', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 0 });
      expect(res.status).toBe(400);
    });

    it('returns 400 when pageSize exceeds 150', async () => {
      const res = await client.listTasks({ page: 1, pageSize: 151 });
      expect(res.status).toBe(400);
    });

    it('returns 400 when page is 0 (below minimum)', async () => {
      const res = await client.listTasks({ page: 0, pageSize: 10 });
      expect(res.status).toBe(400);
    });

    it('returns 400 for an invalid from date format', async () => {
      const res = await fetch(`${BASE_URL}/v1/tasks?page=1&pageSize=10&from=2024-01-01`, {
        headers: { 'User-Token': makeToken(USER_ID) },
      });
      expect(res.status).toBe(400);
    });

    it('returns 400 for an invalid till date format', async () => {
      const res = await fetch(`${BASE_URL}/v1/tasks?page=1&pageSize=10&till=2024-01-01`, {
        headers: { 'User-Token': makeToken(USER_ID) },
      });
      expect(res.status).toBe(400);
    });
  });
});
