import { randomUUID } from 'crypto';
import { TaskerClient } from '../client/tasker';
import type { TaskItem } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.TASKER_URL ?? 'http://localhost:9090';
const USER_ID = randomUUID();
const client = new TaskerClient(BASE_URL, makeToken(USER_ID));
const noAuthClient = new TaskerClient(BASE_URL, '');

async function freshTask(): Promise<string> {
  return client.createTestTask({
    user_id:     USER_ID,
    title:       'Original title',
    description: 'Original description',
    status:      'unplanned',
    category:    'personal',
    priority:    3,
  });
}

describe('PATCH /v1/tasks/:id', () => {
  describe('happy path', () => {
    it('updates title only', async () => {
      const id = await freshTask();
      const res = await client.updateTask(id, { title: 'Updated title' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.title).toBe('Updated title');
      expect(body.task.description).toBe('Original description');
    });

    it('updates description only', async () => {
      const id = await freshTask();
      const res = await client.updateTask(id, { description: 'New description' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.description).toBe('New description');
      expect(body.task.title).toBe('Original title');
    });

    it('updates category only', async () => {
      const id = await freshTask();
      const res = await client.updateTask(id, { category: 'TASK_CATEGORY_WORK' });
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.category).toBe('TASK_CATEGORY_WORK');
    });

    it('updates multiple fields at once', async () => {
      const id = await freshTask();
      const res = await client.updateTask(id, {
        title:    'Multi updated',
        category: 'TASK_CATEGORY_STUDY',
      });
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.title).toBe('Multi updated');
      expect(body.task.category).toBe('TASK_CATEGORY_STUDY');
    });

    it('updates startTime and endTime', async () => {
      const id = await freshTask();
      const start = '2025-06-01T09:00:00Z';
      const end   = '2025-06-01T10:00:00Z';
      const res = await client.updateTask(id, { startTime: start, endTime: end });
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(new Date(body.task.startTime!).toISOString()).toBe(new Date(start).toISOString());
      expect(new Date(body.task.endTime!).toISOString()).toBe(new Date(end).toISOString());
    });

    it('empty body returns the task unchanged', async () => {
      const id = await freshTask();
      const res = await client.updateTask(id, {});
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.title).toBe('Original title');
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const id = await freshTask();
      const res = await noAuthClient.updateTask(id, { title: 'x' });
      expect(res.status).toBe(401);
    });

    it('returns 404 for a non-existent task id', async () => {
      const res = await client.updateTask(randomUUID(), { title: 'x' });
      expect(res.status).toBe(404);
    });

    it('returns 404 for a task belonging to a different user', async () => {
      const id = await freshTask();
      const otherClient = new TaskerClient(BASE_URL, makeToken(randomUUID()));
      const res = await otherClient.updateTask(id, { title: 'x' });
      expect(res.status).toBe(404);
    });
  });
});
