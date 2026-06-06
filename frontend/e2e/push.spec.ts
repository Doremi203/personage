import { test, expect, type Page } from '@playwright/test';

// Onboarding / push prompt coverage. Real backend is hit for the happy
// path (subscription will end up stale on the next CI run; the backend
// 410s on send and we don't care).

const PUSH_TITLE = 'Включить уведомления';
const GRANTED_TITLE = 'Уведомления включены';
const DENIED_TITLE = 'Уведомления отключены';
const ERROR_TITLE = 'Не удалось включить';
const IOS_TITLE = 'Установить Personage';

const ENABLE_BTN = { name: 'Включить' } as const;
const DISMISS_BTN = { name: 'Не сейчас' } as const;

const ORIGIN = 'http://localhost:3000';

const IOS_INSTALL_DISMISSED_KEY = 'personage_ios_install_dismissed';
const PUSH_DISMISSED_KEY = 'personage_push_dismissed';

// Replace `window.Notification` with a Proxy. The app's pickMode reads
// `Notification.permission` and the click handler calls
// `Notification.requestPermission()` — overriding both lets us drive the
// JS-visible permission state independently from the browser-level
// permission (which is controlled via `context.grantPermissions`).
async function stubNotification(
  page: Page,
  permissionForReads: NotificationPermission,
  permissionForRequest: NotificationPermission,
): Promise<void> {
  await page.addInitScript(({ readPerm, reqPerm }) => {
    if (!('Notification' in window)) return;
    const Original = window.Notification;
    const Proxied = new Proxy(Original, {
      get(target, prop, receiver) {
        if (prop === 'permission') return readPerm;
        if (prop === 'requestPermission') return async () => reqPerm;
        return Reflect.get(target, prop, receiver);
      },
    });
    Object.defineProperty(window, 'Notification', {
      value: Proxied,
      configurable: true,
      writable: true,
    });
  }, { readPerm: permissionForReads, reqPerm: permissionForRequest });
}

async function clearOnboardingFlag(page: Page): Promise<void> {
  await page.addInitScript(() => {
    localStorage.removeItem('personage_ios_install_dismissed');
    localStorage.removeItem('personage_push_dismissed');
  });
}

// Chromium's headless `pushManager.subscribe` always rejects with
// "permission denied" because there's no real FCM endpoint to talk to.
// Patch the prototype so the SW path resolves with a deterministic fake
// subscription; the rest of the flow (encryption keys, backend POST) runs
// against the real wiring.
async function stubPushManager(page: Page): Promise<void> {
  await page.addInitScript(() => {
    if (!('PushManager' in window)) return;
    let counter = 0;
    const makeFakeSub = () => {
      const id = ++counter;
      const endpoint = `https://fcm.googleapis.com/fcm/send/playwright-fake-${String(id)}-${String(Date.now())}`;
      return {
        endpoint,
        expirationTime: null,
        options: { userVisibleOnly: true, applicationServerKey: null },
        toJSON() {
          return {
            endpoint,
            expirationTime: null,
            keys: {
              p256dh:
                'BJfQq2bRwKJrCDqzZJpGr3PvWcLqVrn7HMxVxXOEBZAr1tJg9XPgxGw3OKWQXhQE5x9XpKGqyLg5HOH2L7E0OxA',
              auth: 'XKLPgJ2qF9rHMxJpEvKGrQ',
            },
          };
        },
        unsubscribe: async () => true,
        getKey: () => null,
      };
    };
    Object.defineProperty(PushManager.prototype, 'subscribe', {
      value: async function () {
        return makeFakeSub();
      },
      configurable: true,
    });
    Object.defineProperty(PushManager.prototype, 'getSubscription', {
      value: async function () {
        return null;
      },
      configurable: true,
    });
  });
}

test.describe('Onboarding prompt — visibility logic', () => {
  test('hidden when both dismissal flags are set', async ({ page }) => {
    // auth.setup.ts already set the flags; reuse them as-is.
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText(PUSH_TITLE)).toBeHidden();
    await expect(page.getByText(IOS_TITLE)).toBeHidden();
  });

  test('hidden when notification permission is already granted', async ({ page }) => {
    await clearOnboardingFlag(page);
    await stubNotification(page, 'granted', 'granted');
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText(PUSH_TITLE)).toBeHidden();
  });

  test('hidden when notification permission is already denied', async ({ page }) => {
    await clearOnboardingFlag(page);
    await stubNotification(page, 'denied', 'denied');
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText(PUSH_TITLE)).toBeHidden();
  });

  test('shown when push supported and permission is default', async ({ page }) => {
    await clearOnboardingFlag(page);
    await stubNotification(page, 'default', 'default');
    await page.goto('/');
    await expect(page.getByText(PUSH_TITLE)).toBeVisible();
    await expect(page.getByRole('button', ENABLE_BTN)).toBeVisible();
    await expect(page.getByRole('button', DISMISS_BTN)).toBeVisible();
  });
});

test.describe('Push prompt — dismissal', () => {
  test.beforeEach(async ({ page }) => {
    await clearOnboardingFlag(page);
    await stubNotification(page, 'default', 'default');
  });

  test('"Не сейчас" closes prompt and persists dismissal', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText(PUSH_TITLE)).toBeVisible();
    await page.getByRole('button', DISMISS_BTN).click();
    await expect(page.getByText(PUSH_TITLE)).toBeHidden();
    const flag = await page.evaluate(
      (key) => localStorage.getItem(key),
      PUSH_DISMISSED_KEY,
    );
    expect(flag).toBe('true');
  });
});

test.describe('Push prompt — denied path', () => {
  test.beforeEach(async ({ page }) => {
    await clearOnboardingFlag(page);
    // Reads default → prompt shows; request resolves denied → click → denied UI.
    await stubNotification(page, 'default', 'denied');
  });

  test('clicking "Включить" → user denies → denied UI', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText(PUSH_TITLE)).toBeVisible();
    await page.getByRole('button', ENABLE_BTN).click();

    await expect(page.getByText(DENIED_TITLE)).toBeVisible({ timeout: 5_000 });
    await expect(
      page.getByText('Включите уведомления для Personage в настройках браузера'),
    ).toBeVisible();
    // Enable button gone, dismiss stays.
    await expect(page.getByRole('button', ENABLE_BTN)).toBeHidden();
    await expect(page.getByRole('button', DISMISS_BTN)).toBeVisible();
  });
});

test.describe('Push prompt — full subscribe flow', () => {
  test.use({ serviceWorkers: 'allow' });

  test.beforeEach(async ({ page, context }) => {
    await context.grantPermissions(['notifications'], { origin: ORIGIN });
    await clearOnboardingFlag(page);
    // Notification.permission reads 'default' so the prompt shows;
    // requestPermission resolves 'granted' so the flow proceeds.
    await stubNotification(page, 'default', 'granted');
    await stubPushManager(page);
  });

  test('grants → subscribes → backend POST → success UI auto-dismisses', async ({ page }) => {
    const subscribeRequest = page.waitForRequest(
      (req) => req.url().includes('/v1/push/subscribe') && req.method() === 'POST',
      { timeout: 15_000 },
    );

    await page.goto('/');
    await expect(page.getByText(PUSH_TITLE)).toBeVisible();
    await page.getByRole('button', ENABLE_BTN).click();

    const req = await subscribeRequest;
    const body = JSON.parse(req.postData() ?? '{}') as {
      endpoint?: string;
      p256dh?: string;
      auth_key?: string;
    };
    expect(body.endpoint).toBeTruthy();
    expect(body.p256dh).toBeTruthy();
    expect(body.auth_key).toBeTruthy();

    await expect(page.getByText(GRANTED_TITLE)).toBeVisible({ timeout: 10_000 });
    // OnboardingPrompt schedules dismiss at 1.2 s after success.
    await expect(page.getByText(GRANTED_TITLE)).toBeHidden({ timeout: 5_000 });

    const flag = await page.evaluate(
      (key) => localStorage.getItem(key),
      PUSH_DISMISSED_KEY,
    );
    expect(flag).toBe('true');
  });
});

test.describe('Push prompt — backend errors', () => {
  test.use({ serviceWorkers: 'allow' });

  test.beforeEach(async ({ page, context }) => {
    await context.grantPermissions(['notifications'], { origin: ORIGIN });
    await clearOnboardingFlag(page);
    await stubNotification(page, 'default', 'granted');
    await stubPushManager(page);
  });

  test('subscribe POST aborted → error UI shown, retry stays available', async ({ page }) => {
    await page.route('**/v1/push/subscribe', (route) => route.abort('failed'));

    await page.goto('/');
    await page.getByRole('button', ENABLE_BTN).click();

    await expect(page.getByText(ERROR_TITLE)).toBeVisible({ timeout: 10_000 });
    // Error path leaves the enable button visible so the user can retry.
    await expect(page.getByRole('button', ENABLE_BTN)).toBeVisible();
  });
});

test.describe('iOS install prompt', () => {
  // Pretend to be Mobile Safari on iPhone (and not standalone).
  test.use({
    userAgent:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
  });

  test.beforeEach(async ({ page }) => {
    await clearOnboardingFlag(page);
  });

  test('renders install steps with the share + add labels', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText(IOS_TITLE)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Поделиться')).toBeVisible();
  });

  test('"Не сейчас" dismisses and persists', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText(IOS_TITLE)).toBeVisible();
    await page.getByRole('button', DISMISS_BTN).click();
    await expect(page.getByText(IOS_TITLE)).toBeHidden();
    const flag = await page.evaluate(
      (key) => localStorage.getItem(key),
      IOS_INSTALL_DISMISSED_KEY,
    );
    expect(flag).toBe('true');
  });

  test('"Понятно" dismisses and persists', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText(IOS_TITLE)).toBeVisible();
    await page.getByRole('button', { name: 'Понятно' }).click();
    await expect(page.getByText(IOS_TITLE)).toBeHidden();
    const flag = await page.evaluate(
      (key) => localStorage.getItem(key),
      IOS_INSTALL_DISMISSED_KEY,
    );
    expect(flag).toBe('true');
  });
});
