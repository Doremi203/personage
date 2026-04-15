import { randomUUID } from 'crypto';
import { NotificatorClient } from '../client/notificator';
import type { ToggleNotificationResponse } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.NOTIFICATOR_URL ?? 'http://localhost:9090';
const noAuthClient = new NotificatorClient(BASE_URL, '');

describe('POST /notifications/{type}/toggle', () => {
  describe('happy path', () => {
    it('first toggle for a type returns enabled=false (default is true → first explicit override is false)', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.toggleNotification('upcoming_event');
      expect(res.status).toBe(200);

      const body = (await res.json()) as ToggleNotificationResponse;
      expect(body.enabled).toBe(false);
    });

    it('second toggle for the same type returns enabled=true', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      await client.toggleNotification('upcoming_event');
      const res = await client.toggleNotification('upcoming_event');
      expect(res.status).toBe(200);

      const body = (await res.json()) as ToggleNotificationResponse;
      expect(body.enabled).toBe(true);
    });

    it('third toggle returns enabled=false again', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      await client.toggleNotification('upcoming_event');
      await client.toggleNotification('upcoming_event');
      const res = await client.toggleNotification('upcoming_event');
      expect(res.status).toBe(200);

      const body = (await res.json()) as ToggleNotificationResponse;
      expect(body.enabled).toBe(false);
    });

    it('different types are toggled independently', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));

      const res1 = await client.toggleNotification('upcoming_event');
      const body1 = (await res1.json()) as ToggleNotificationResponse;

      const res2 = await client.toggleNotification('schedule_change');
      const body2 = (await res2.json()) as ToggleNotificationResponse;

      expect(body1.enabled).toBe(false);
      expect(body2.enabled).toBe(false);
    });

    it('toggling for one user does not affect another user', async () => {
      const client1 = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const client2 = new NotificatorClient(BASE_URL, makeToken(randomUUID()));

      await client1.toggleNotification('upcoming_event');

      const res2 = await client2.toggleNotification('upcoming_event');
      const body2 = (await res2.json()) as ToggleNotificationResponse;
      // client2 has no prior toggles, so first toggle returns false
      expect(body2.enabled).toBe(false);
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const res = await noAuthClient.toggleNotification('upcoming_event');
      expect(res.status).toBe(401);
    });
  });
});
