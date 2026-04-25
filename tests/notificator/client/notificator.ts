import type {
  ListNotificationsParams,
  CreateTestNotificationRequest,
  CreateTestNotificationResponse,
} from './types';

export class NotificatorClient {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl;
    this.token = token;
  }

  private headers(): Record<string, string> {
    const h: Record<string, string> = { 'Content-Type': 'application/json' };
    if (this.token) {
      h['User-Token'] = this.token;
    }
    return h;
  }

  listNotifications(params: ListNotificationsParams = {}): Promise<Response> {
    const query = new URLSearchParams();
    if (params.page !== undefined)     query.set('page',     String(params.page));
    if (params.pageSize !== undefined) query.set('pageSize', String(params.pageSize));

    const qs = query.toString();
    return fetch(`${this.baseUrl}/notifications${qs ? `?${qs}` : ''}`, {
      headers: this.headers(),
    });
  }

  getNotificationSettings(): Promise<Response> {
    return fetch(`${this.baseUrl}/notifications/settings`, {
      headers: this.headers(),
    });
  }

  toggleNotification(type: string): Promise<Response> {
    return fetch(`${this.baseUrl}/notifications/${encodeURIComponent(type)}/toggle`, {
      method: 'POST',
      headers: this.headers(),
    });
  }

  subscribe(endpoint: string, p256dh: string, authKey: string): Promise<Response> {
    return fetch(`${this.baseUrl}/v1/push/subscribe`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify({ endpoint, p256dh, authKey }),
    });
  }

  unsubscribe(endpoint: string): Promise<Response> {
    return fetch(`${this.baseUrl}/v1/push/unsubscribe`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify({ endpoint }),
    });
  }

  /** Creates a notification via the test-only endpoint. Throws if the request fails. */
  async createTestNotification(req: CreateTestNotificationRequest): Promise<string> {
    const res = await fetch(`${this.baseUrl}/v1/test/notifications`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`createTestNotification failed (${res.status}): ${text}`);
    }
    const body = (await res.json()) as CreateTestNotificationResponse;
    return body.id;
  }
}
