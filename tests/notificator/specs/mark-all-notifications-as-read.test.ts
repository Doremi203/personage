import { randomUUID } from 'crypto';
import { NotificatorClient } from '../client/notificator';
import type { ListNotificationsResponse } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.NOTIFICATOR_URL ?? 'http://localhost:9090';
const noAuthClient = new NotificatorClient(BASE_URL, '');

async function listAll(client: NotificatorClient) {
  const res = await client.listNotifications({ page: 1, pageSize: 10 });
  const body = (await res.json()) as ListNotificationsResponse;
  return body.notifications;
}

describe('POST /notifications/read', () => {
  describe('happy path', () => {
    it('marks every unread notification as read', async () => {
      const userId = randomUUID();
      const client = new NotificatorClient(BASE_URL, makeToken(userId));
      await Promise.all([
        client.createTestNotification({ user_id: userId, title: 'A', type: 'upcoming_event',  text: 'a' }),
        client.createTestNotification({ user_id: userId, title: 'B', type: 'schedule_change', text: 'b' }),
        client.createTestNotification({ user_id: userId, title: 'C', type: 'upcoming_event',  text: 'c' }),
      ]);

      const before = await listAll(client);
      expect(before.every((n) => n.readAt === null)).toBe(true);

      const res = await client.markAllNotificationsAsRead();
      expect(res.status).toBe(200);
      await expect(res.json()).resolves.toEqual({});

      const after = await listAll(client);
      expect(after).toHaveLength(3);
      after.forEach((n) => {
        expect(typeof n.readAt).toBe('string');
        expect(new Date(n.readAt!).getTime()).not.toBeNaN();
      });
    });

    it('preserves the original readAt for already-read notifications', async () => {
      const userId = randomUUID();
      const client = new NotificatorClient(BASE_URL, makeToken(userId));
      const [oldId, newId] = await Promise.all([
        client.createTestNotification({ user_id: userId, title: 'Already read', type: 'upcoming_event',  text: 'old' }),
        client.createTestNotification({ user_id: userId, title: 'Still unread', type: 'schedule_change', text: 'new' }),
      ]);

      const single = await client.markNotificationAsRead(oldId);
      expect(single.status).toBe(200);
      const oldReadAtBefore = (await listAll(client)).find((n) => n.id === oldId)!.readAt!;

      const all = await client.markAllNotificationsAsRead();
      expect(all.status).toBe(200);

      const after = await listAll(client);
      const oldAfter = after.find((n) => n.id === oldId)!;
      const newAfter = after.find((n) => n.id === newId)!;

      expect(oldAfter.readAt).toBe(oldReadAtBefore);
      expect(typeof newAfter.readAt).toBe('string');
    });

    it('returns 200 when the user has no notifications (no-op)', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.markAllNotificationsAsRead();
      expect(res.status).toBe(200);
      await expect(res.json()).resolves.toEqual({});
    });

    it('is idempotent: calling twice keeps every readAt stable', async () => {
      const userId = randomUUID();
      const client = new NotificatorClient(BASE_URL, makeToken(userId));
      await Promise.all([
        client.createTestNotification({ user_id: userId, title: 'X', type: 'upcoming_event',  text: 'x' }),
        client.createTestNotification({ user_id: userId, title: 'Y', type: 'schedule_change', text: 'y' }),
      ]);

      const first = await client.markAllNotificationsAsRead();
      expect(first.status).toBe(200);
      const readAtsAfterFirst = (await listAll(client)).map((n) => n.readAt);

      const second = await client.markAllNotificationsAsRead();
      expect(second.status).toBe(200);
      const readAtsAfterSecond = (await listAll(client)).map((n) => n.readAt);

      expect(readAtsAfterSecond).toEqual(readAtsAfterFirst);
    });

    it('does not affect other users notifications', async () => {
      const userA = randomUUID();
      const userB = randomUUID();
      const clientA = new NotificatorClient(BASE_URL, makeToken(userA));
      const clientB = new NotificatorClient(BASE_URL, makeToken(userB));

      await Promise.all([
        clientA.createTestNotification({ user_id: userA, title: 'A1', type: 'upcoming_event',  text: 'a1' }),
        clientB.createTestNotification({ user_id: userB, title: 'B1', type: 'schedule_change', text: 'b1' }),
      ]);

      const res = await clientA.markAllNotificationsAsRead();
      expect(res.status).toBe(200);

      const bAfter = await listAll(clientB);
      expect(bAfter).toHaveLength(1);
      expect(bAfter[0]!.readAt).toBeNull();
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const res = await noAuthClient.markAllNotificationsAsRead();
      expect(res.status).toBe(401);
    });
  });
});
