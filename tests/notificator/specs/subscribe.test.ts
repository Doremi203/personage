import { randomUUID } from 'crypto';
import { NotificatorClient } from '../client/notificator';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.NOTIFICATOR_URL ?? 'http://localhost:9090';
const noAuthClient = new NotificatorClient(BASE_URL, '');

function fakeSubscription(suffix: string = '') {
  return {
    endpoint: `https://push.example.com/test-endpoint-${suffix || randomUUID()}`,
    p256dh:   'BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPiA48yEimKqYVFggHQG0_gVQ=',
    authKey:  'GFqE4J9Oc4KDSQJ',
  };
}

describe('POST /v1/push/subscribe', () => {
  describe('happy path', () => {
    it('subscribes successfully and returns 200', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const sub = fakeSubscription();
      const res = await client.subscribe(sub.endpoint, sub.p256dh, sub.authKey);
      expect(res.status).toBe(200);
    });

    it('subscribing the same endpoint twice (upsert) returns 200 both times', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const sub = fakeSubscription();

      const res1 = await client.subscribe(sub.endpoint, sub.p256dh, sub.authKey);
      expect(res1.status).toBe(200);

      const res2 = await client.subscribe(sub.endpoint, sub.p256dh, sub.authKey);
      expect(res2.status).toBe(200);
    });

    it('different users can subscribe with different endpoints', async () => {
      const client1 = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const client2 = new NotificatorClient(BASE_URL, makeToken(randomUUID()));

      const sub1 = fakeSubscription();
      const sub2 = fakeSubscription();

      const res1 = await client1.subscribe(sub1.endpoint, sub1.p256dh, sub1.authKey);
      const res2 = await client2.subscribe(sub2.endpoint, sub2.p256dh, sub2.authKey);

      expect(res1.status).toBe(200);
      expect(res2.status).toBe(200);
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const sub = fakeSubscription();
      const res = await noAuthClient.subscribe(sub.endpoint, sub.p256dh, sub.authKey);
      expect(res.status).toBe(401);
    });

    it('returns 400 when endpoint is empty', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.subscribe('', 'p256dh-value', 'auth-key-value');
      expect(res.status).toBe(400);
    });

    it('returns 400 when p256dh is empty', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.subscribe('https://push.example.com/endpoint', '', 'auth-key-value');
      expect(res.status).toBe(400);
    });

    it('returns 400 when authKey is empty', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.subscribe('https://push.example.com/endpoint', 'p256dh-value', '');
      expect(res.status).toBe(400);
    });
  });
});
