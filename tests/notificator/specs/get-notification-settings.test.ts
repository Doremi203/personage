import { randomUUID } from 'crypto';
import { NotificatorClient } from '../client/notificator';
import type { GetNotificationSettingsResponse, NotificationSetting } from '../client/types';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.NOTIFICATOR_URL ?? 'http://localhost:9090';
const noAuthClient = new NotificatorClient(BASE_URL, '');

describe('GET /notifications/settings', () => {
  describe('happy path', () => {
    it('returns every available type enabled by default for a fresh user', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.getNotificationSettings();
      expect(res.status).toBe(200);

      const body = (await res.json()) as GetNotificationSettingsResponse;
      expect(Array.isArray(body.settings)).toBe(true);
      expect(body.settings).toEqual(
        expect.arrayContaining([
          { type: 'schedule_change', enabled: true },
          { type: 'upcoming_event', enabled: true },
        ]),
      );
      expect(body.settings).toHaveLength(2);
    });

    it('returns settings with the correct shape after a toggle', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      await client.toggleNotification('upcoming_event');

      const res = await client.getNotificationSettings();
      expect(res.status).toBe(200);

      const body = (await res.json()) as GetNotificationSettingsResponse;
      expect(body.settings.length).toBeGreaterThanOrEqual(1);

      const setting = body.settings[0] as NotificationSetting;
      expect(setting).toMatchObject({
        type:    expect.any(String),
        enabled: expect.any(Boolean),
      });
    });

    it('reflects the toggled type after toggle', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      await client.toggleNotification('upcoming_event');

      const res = await client.getNotificationSettings();
      const body = (await res.json()) as GetNotificationSettingsResponse;

      const setting = body.settings.find((s) => s.type === 'upcoming_event');
      expect(setting).toBeDefined();
      expect(setting!.enabled).toBe(false);
    });

    it('reflects multiple toggled types', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      await client.toggleNotification('upcoming_event');
      await client.toggleNotification('schedule_change');

      const res = await client.getNotificationSettings();
      const body = (await res.json()) as GetNotificationSettingsResponse;

      const types = body.settings.map((s) => s.type);
      expect(types).toContain('upcoming_event');
      expect(types).toContain('schedule_change');
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const res = await noAuthClient.getNotificationSettings();
      expect(res.status).toBe(401);
    });
  });
});
