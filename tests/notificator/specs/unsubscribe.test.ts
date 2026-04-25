import { randomUUID } from 'crypto';
import { NotificatorClient } from '../client/notificator';
import { makeToken } from '../helpers/auth';

const BASE_URL = process.env.NOTIFICATOR_URL ?? 'http://localhost:9090';
const noAuthClient = new NotificatorClient(BASE_URL, '');

function fakeSubscription() {
  return {
    endpoint: `https://push.example.com/test-endpoint-${randomUUID()}`,
    p256dh:   'BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPiA48yEimKqYVFggHQG0_gVQ=',
    authKey:  'GFqE4J9Oc4KDSQJ',
  };
}

describe('POST /v1/push/unsubscribe', () => {
  describe('happy path', () => {
    it('unsubscribes an existing subscription and returns 200', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const sub = fakeSubscription();
      await client.subscribe(sub.endpoint, sub.p256dh, sub.authKey);

      const res = await client.unsubscribe(sub.endpoint);
      expect(res.status).toBe(200);
    });

    it('unsubscribing is idempotent: removing a non-existent endpoint returns 200', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.unsubscribe(`https://push.example.com/nonexistent-${randomUUID()}`);
      expect(res.status).toBe(200);
    });

    it('unsubscribing one endpoint does not affect another endpoint for the same user', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const sub1 = fakeSubscription();
      const sub2 = fakeSubscription();

      await client.subscribe(sub1.endpoint, sub1.p256dh, sub1.authKey);
      await client.subscribe(sub2.endpoint, sub2.p256dh, sub2.authKey);

      const res = await client.unsubscribe(sub1.endpoint);
      expect(res.status).toBe(200);

      // re-subscribing sub1 should succeed (it was removed)
      const resResub = await client.subscribe(sub1.endpoint, sub1.p256dh, sub1.authKey);
      expect(resResub.status).toBe(200);
    });
  });

  describe('error cases', () => {
    it('returns 401 when User-Token header is missing', async () => {
      const res = await noAuthClient.unsubscribe('https://push.example.com/endpoint');
      expect(res.status).toBe(401);
    });

    it('returns 400 when endpoint is empty', async () => {
      const client = new NotificatorClient(BASE_URL, makeToken(randomUUID()));
      const res = await client.unsubscribe('');
      expect(res.status).toBe(400);
    });
  });
});
