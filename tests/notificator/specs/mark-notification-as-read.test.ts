import { randomUUID } from 'crypto';
import { NotificatorClient } from '../client/notificator';
import type { ListNotificationsResponse } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.NOTIFICATOR_URL ?? 'http://localhost:9090';
const noAuthClient = new NotificatorClient(BASE_URL, '');

async function findNotification(client: NotificatorClient, id: string) {
  const res = await client.listNotifications({ page: 1, pageSize: 10 });
  const body = (await res.json()) as ListNotificationsResponse;
  return body.notifications.find((n) => n.id === id);
}

describe('POST /notifications/read/{id}', () => {
  describe('happy path', () => {
    it('marks an unread notification as read and returns an empty body', async () => {
      const userId = randomUUID();
      const client = new NotificatorClient(BASE_URL, makeToken(userId));
      const id = await client.createTestNotification({
        user_id: userId,
        title:   'Mark me read',
        type:    'upcoming_event',
        text:    'unread → read',
      });

      const before = await findNotification(client, id);
      expect(before).toBeDefined();
      expect(before!.readAt).toBeNull();

      const res = await client.markNotificationAsRead(id);
      expect(res.status).toBe(200);
      await expect(res.json()).resolves.toEqual({});

      const after = await findNotification(client, id);
      expect(after).toBeDefined();
      expect(typeof after!.readAt).toBe('string');
      expect(new Date(after!.readAt!).getTime()).not.toBeNaN();
    });

    it('is idempotent: marking the same notification twice keeps the original readAt', async () => {
      const userId = randomUUID();
      const client = new NotificatorClient(BASE_URL, makeToken(userId));
      const id = await client.createTestNotification({
        user_id: userId,
        title:   'Idempotent mark',
        type:    'schedule_change',
        text:    'mark twice',
      });

      const first = await client.markNotificationAsRead(id);
      expect(first.status).toBe(200);
      const firstReadAt = (await findNotification(client, id))!.readAt!;

      const second = await client.markNotificationAsRead(id);
      expect(second.status).toBe(200);
      const secondReadAt = (await findNotification(client, id))!.readAt!;

      expect(secondReadAt).toBe(firstReadAt);
    });

    it('only affects the targeted notification', async () => {
      const userId = randomUUID();
      const client = new NotificatorClient(BASE_URL, makeToken(userId));
      const [targetId, otherId] = await Promise.all([
        client.createTestNotification({ user_id: userId, title: 'Target', type: 'upcoming_event',  text: 't' }),
        client.createTestNotification({ user_id: userId, title: 'Other',  type: 'schedule_change', text: 'o' }),
      ]);

      const res = await client.markNotificationAsRead(targetId);
      expect(res.status).toBe(200);

      const target = await findNotification(client, targetId);
      const other  = await findNotification(client, otherId);
      expect(typeof target!.readAt).toBe('string');
      expect(other!.readAt).toBeNull();
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const res = await noAuthClient.markNotificationAsRead(randomUUID());
      expect(res.status).toBe(401);
    });

    it('returns 400 when id is not a valid UUID', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.markNotificationAsRead('not-a-uuid');
      expect(res.status).toBe(400);
    });

    it('returns 404 when notification does not exist', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.markNotificationAsRead(randomUUID());
      expect(res.status).toBe(404);
    });

    it('returns 404 when notification belongs to another user', async () => {
      const ownerId = randomUUID();
      const owner   = new NotificatorClient(BASE_URL, makeToken(ownerId));
      const intruder = new NotificatorClient(BASE_URL, makeToken(randomUUID()));

      const id = await owner.createTestNotification({
        user_id: ownerId,
        title:   "Someone else's notification",
        type:    'upcoming_event',
        text:    'private',
      });

      const res = await intruder.markNotificationAsRead(id);
      expect(res.status).toBe(404);

      const stillUnread = await findNotification(owner, id);
      expect(stillUnread!.readAt).toBeNull();
    });
  });
});
