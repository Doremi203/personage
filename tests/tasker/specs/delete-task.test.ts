import { randomUUID } from 'crypto';
import { TaskerClient } from '../client/tasker';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.TASKER_URL ?? 'http://localhost:9090';
const USER_ID = randomUUID();
const client = new TaskerClient(BASE_URL, makeToken(USER_ID));
const noAuthClient = new TaskerClient(BASE_URL, '');

async function freshTask(): Promise<string> {
  return client.createTestTask({ user_id: USER_ID, title: 'Delete test task', status: 'unplanned' });
}

describe('DELETE /v1/tasks/:id', () => {
  describe('happy path', () => {
    it('deletes the task and returns 200 with an empty body', async () => {
      const id = await freshTask();
      const res = await client.deleteTask(id);
      expect(res.status).toBe(200);

      const text = await res.text();
      // gRPC-Gateway serialises DeleteTaskV1Response{} as `{}`
      expect(text.trim()).toBe('{}');
    });

    it('task is no longer retrievable after deletion', async () => {
      const id = await freshTask();
      await client.deleteTask(id);

      const getRes = await client.getTask(id);
      expect(getRes.status).toBe(404);
    });

    it('deleted task no longer appears in list', async () => {
      // Use an isolated user so we can count tasks precisely
      const isolatedUserId = randomUUID();
      const isolatedClient = new TaskerClient(BASE_URL, makeToken(isolatedUserId));

      const id = await isolatedClient.createTestTask({ user_id: isolatedUserId, title: 'To be deleted' });
      await isolatedClient.deleteTask(id);

      const listRes = await isolatedClient.listTasks({ page: 1, pageSize: 10 });
      const body = (await listRes.json()) as { total: number };
      expect(body.total).toBe(0);
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const id = await freshTask();
      const res = await noAuthClient.deleteTask(id);
      expect(res.status).toBe(401);
    });

    it('returns 404 for a non-existent task id', async () => {
      const res = await client.deleteTask(randomUUID());
      expect(res.status).toBe(404);
    });

    it('returns 404 for a task belonging to a different user', async () => {
      const id = await freshTask();
      const otherClient = new TaskerClient(BASE_URL, makeToken(randomUUID()));
      const res = await otherClient.deleteTask(id);
      expect(res.status).toBe(404);
    });
  });
});
