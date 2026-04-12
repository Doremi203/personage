import { randomUUID } from 'crypto';
import { TaskerClient } from '../client/tasker';
import type { TaskItem } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.TASKER_URL ?? 'http://localhost:9090';
const USER_ID = randomUUID();
const client = new TaskerClient(BASE_URL, makeToken(USER_ID));
const noAuthClient = new TaskerClient(BASE_URL, '');

let taskId: string;

beforeAll(async () => {
  taskId = await client.createTestTask({
    user_id:     USER_ID,
    title:       'Get task test',
    description: 'Created for get-task spec',
    status:      'unplanned',
    category:    'work',
    priority:    7,
  });
});

describe('GET /v1/tasks/:id', () => {
  describe('happy path', () => {
    it('returns the task with correct fields', async () => {
      const res = await client.getTask(taskId);
      expect(res.status).toBe(200);

      const body = (await res.json()) as { task: TaskItem };
      expect(body.task.id).toBe(taskId);
      expect(body.task.title).toBe('Get task test');
      expect(body.task.description).toBe('Created for get-task spec');
      expect(body.task.status).toBe('TASK_STATUS_UNPLANNED');
      expect(body.task.category).toBe('TASK_CATEGORY_WORK');
      expect(body.task.priority).toBe('TASK_PRIORITY_MID'); // priority 7 → mid (range 4–7)
    });

    it('returns the task with a valid ISO timestamp for createdAt', async () => {
      const res = await client.getTask(taskId);
      const body = (await res.json()) as { task: TaskItem };
      expect(new Date(body.task.createdAt).getTime()).not.toBeNaN();
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const res = await noAuthClient.getTask(taskId);
      expect(res.status).toBe(401);
    });

    it('returns 404 for a non-existent task id', async () => {
      const res = await client.getTask(randomUUID());
      expect(res.status).toBe(404);
    });

    it('returns 404 for a task belonging to a different user', async () => {
      const otherClient = new TaskerClient(BASE_URL, makeToken(randomUUID()));
      const res = await otherClient.getTask(taskId);
      expect(res.status).toBe(404);
    });
  });
});
