import { randomUUID } from 'crypto';
import { TaskerClient } from '../client/tasker';
import type { TaskItem } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.TASKER_URL ?? 'http://localhost:9090';
const USER_ID = randomUUID();
const client = new TaskerClient(BASE_URL, makeToken(USER_ID));
const noAuthClient = new TaskerClient(BASE_URL, '');

async function freshTask(status: 'unplanned' | 'planned' | 'completed' = 'unplanned'): Promise<string> {
  return client.createTestTask({ user_id: USER_ID, title: 'Complete test task', status });
}

describe('POST /v1/tasks/:id/complete', () => {
  describe('happy path', () => {
    it('sets status to completed for an unplanned task', async () => {
      const id = await freshTask('unplanned');
      const res = await client.completeTask(id);
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.id).toBe(id);
      expect(body.task.status).toBe('TASK_STATUS_COMPLETED');
    });

    it('sets status to completed for a planned task', async () => {
      const id = await freshTask('planned');
      const res = await client.completeTask(id);
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.status).toBe('TASK_STATUS_COMPLETED');
    });

    it('returns the full task in the response body', async () => {
      const id = await freshTask();
      const res = await client.completeTask(id);
      const body = (await res.json()) as { task: TaskItem };

      expect(body.task).toMatchObject({
        id:    id,
        title: 'Complete test task',
      });
    });

    it('is idempotent: completing an already-completed task succeeds', async () => {
      const id = await freshTask('completed');
      const res = await client.completeTask(id);
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.status).toBe('TASK_STATUS_COMPLETED');
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const id = await freshTask();
      const res = await noAuthClient.completeTask(id);
      expect(res.status).toBe(401);
    });

    it('returns 404 for a non-existent task id', async () => {
      const res = await client.completeTask(randomUUID());
      expect(res.status).toBe(404);
    });

    it('returns 404 for a task belonging to a different user', async () => {
      const id = await freshTask();
      const otherClient = new TaskerClient(BASE_URL, makeToken(randomUUID()));
      const res = await otherClient.completeTask(id);
      expect(res.status).toBe(404);
    });
  });
});
