import { randomUUID } from 'crypto';
import { NotificatorClient } from '../client/notificator';
import type { ListNotificationsResponse, NotificationItem } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.NOTIFICATOR_URL ?? 'http://localhost:9090';
const USER_ID = randomUUID();
const client = new NotificatorClient(BASE_URL, makeToken(USER_ID));
const noAuthClient = new NotificatorClient(BASE_URL, '');

const notifications: { id: string; title: string; type: string }[] = [];

beforeAll(async () => {
  const created = await Promise.all([
    client.createTestNotification({ user_id: USER_ID, title: 'First notification',  type: 'upcoming_event',   text: 'Event in 15 min' }),
    client.createTestNotification({ user_id: USER_ID, title: 'Second notification', type: 'schedule_change',  text: 'Schedule updated' }),
    client.createTestNotification({ user_id: USER_ID, title: 'Third notification',  type: 'upcoming_event',   text: 'Event in 5 min' }),
  ]);
  const meta = [
    { title: 'First notification',  type: 'upcoming_event' },
    { title: 'Second notification', type: 'schedule_change' },
    { title: 'Third notification',  type: 'upcoming_event' },
  ];
  created.forEach((id, i) => notifications.push({ id, ...meta[i] }));
});

describe('GET /notifications', () => {
  describe('happy path', () => {
    it('returns notifications for the authenticated user', async () => {
      const res = await client.listNotifications({ page: 1, pageSize: 10 });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListNotificationsResponse;
      expect(body.notifications).toHaveLength(3);
    });

    it('returns notifications with the expected shape', async () => {
      const res = await client.listNotifications({ page: 1, pageSize: 10 });
      const body = (await res.json()) as ListNotificationsResponse;
      const item = body.notifications[0] as NotificationItem;

      expect(item).toMatchObject({
        id:     expect.any(String),
        title:  expect.any(String),
        type:   expect.any(String),
        text:   expect.any(String),
        sentAt: expect.any(String),
      });
    });

    it('returns notifications with a valid ISO timestamp for sentAt', async () => {
      const res = await client.listNotifications({ page: 1, pageSize: 10 });
      const body = (await res.json()) as ListNotificationsResponse;
      body.notifications.forEach((n) => {
        expect(new Date(n.sentAt).getTime()).not.toBeNaN();
      });
    });

    it('paginates: pageSize=2 returns at most 2 items', async () => {
      const res = await client.listNotifications({ page: 1, pageSize: 2 });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListNotificationsResponse;
      expect(body.notifications).toHaveLength(2);
    });

    it('paginates: page=2 with pageSize=2 returns the remaining notification', async () => {
      const res = await client.listNotifications({ page: 2, pageSize: 2 });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListNotificationsResponse;
      expect(body.notifications).toHaveLength(1);
    });

    it('returns empty notifications array for a user with no notifications', async () => {
      const emptyUserClient = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await emptyUserClient.listNotifications({ page: 1, pageSize: 10 });
      expect(res.status).toBe(200);

      const body = (await res.json()) as ListNotificationsResponse;
      expect(body.notifications).toHaveLength(0);
    });

    it('returns notifications ordered by sentAt descending (newest first)', async () => {
      const res = await client.listNotifications({ page: 1, pageSize: 10 });
      const body = (await res.json()) as ListNotificationsResponse;

      const times = body.notifications.map((n) => new Date(n.sentAt).getTime());
      for (let i = 1; i < times.length; i++) {
        expect(times[i - 1]).toBeGreaterThanOrEqual(times[i]);
      }
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const res = await noAuthClient.listNotifications({ page: 1, pageSize: 10 });
      expect(res.status).toBe(401);
    });

    it('returns 400 when page is 0 (below minimum)', async () => {
      const res = await client.listNotifications({ page: 0, pageSize: 10 });
      expect(res.status).toBe(400);
    });

    it('returns 400 when pageSize is 0 (below minimum)', async () => {
      const res = await client.listNotifications({ page: 1, pageSize: 0 });
      expect(res.status).toBe(400);
    });

    it('returns 400 when pageSize exceeds 10 (maximum)', async () => {
      const res = await client.listNotifications({ page: 1, pageSize: 11 });
      expect(res.status).toBe(400);
    });
  });
});
