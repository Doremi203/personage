import { test, expect, type Page } from '@playwright/test';

// The Settings → Notifications group renders a master "Push-уведомления"
// switch that drives the browser permission + PushManager subscription.
// Per-type toggles are disabled until that master switch is on. For tests
// that need to flip a per-type toggle, stub Notification + PushManager so
// `getPushSubscriptionStatus` resolves to 'subscribed'.
async function stubSubscribedPushState(page: Page): Promise<void> {
  await page.addInitScript(() => {
    if ('Notification' in window) {
      const Original = window.Notification;
      const Proxied = new Proxy(Original, {
        get(target, prop, receiver) {
          if (prop === 'permission') return 'granted';
          if (prop === 'requestPermission') return async () => 'granted';
          return Reflect.get(target, prop, receiver);
        },
      });
      Object.defineProperty(window, 'Notification', {
        value: Proxied,
        configurable: true,
        writable: true,
      });
    }
    if ('PushManager' in window) {
      const fakeSub = {
        endpoint: 'https://fcm.googleapis.com/fcm/send/playwright-settings-fake',
        expirationTime: null,
        options: { userVisibleOnly: true, applicationServerKey: null },
        toJSON() {
          return {
            endpoint: this.endpoint,
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
      Object.defineProperty(PushManager.prototype, 'getSubscription', {
        value: async function () {
          return fakeSub;
        },
        configurable: true,
      });
    }
  });
}

async function openSettings(page: Page): Promise<void> {
  await page.goto('/');
  await page.getByRole('button', { name: 'Настройки' }).click();
  await expect(page.getByText('Настройки', { exact: true }).first()).toBeVisible({ timeout: 15_000 });
}

test.describe('Settings', () => {
  test('profile + integrations sections render', async ({ page }) => {
    await openSettings(page);
    // Profile card has the user's email — pulled from /user.
    await expect(page.getByText(/@/)).toBeVisible({ timeout: 10_000 });

    // Notification settings group renders if there's at least one toggle.
    const notifGroup = page.getByText('Уведомления', { exact: true });
    const sources = page.getByText('Источники задач', { exact: true });
    await expect(sources).toBeVisible();
    if (await notifGroup.count() > 0) {
      await expect(notifGroup.first()).toBeVisible();
    }

    await expect(page.getByText('Telegram', { exact: true })).toBeVisible();
    await expect(page.getByText('Gmail', { exact: true })).toBeVisible();
  });

  test('logout returns to auth screen', async ({ page }) => {
    await openSettings(page);
    await page.getByRole('button', { name: 'Выйти из аккаунта' }).click();

    await expect(page.getByText('Personage', { exact: true })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByPlaceholder('Email')).toBeVisible();

    // Re-seed for subsequent tests.
    const tokens = await page.evaluate(() => localStorage.getItem('personage_auth_tokens'));
    expect(tokens).toBeNull();
  });
});

// Per-type toggles are disabled until the master push switch is on, so
// stub the push state to 'subscribed' before navigating.
test.describe('Settings — notification toggles', () => {
  test.beforeEach(async ({ page }) => {
    await stubSubscribedPushState(page);
    await openSettings(page);
  });

  test('toggle a per-type notification setting and revert it (idempotent)', async ({ page }) => {
    // First switch is the "Push-уведомления" master toggle. Pick a per-type
    // toggle by aria-label so the test never accidentally drives the browser
    // permission / subscription flow.
    const perTypeToggles = page.locator(
      'button[role="switch"]:not([aria-label="Push-уведомления"])',
    );
    const count = await perTypeToggles.count();
    if (count === 0) test.skip(true, 'Account has no per-type notification toggles.');

    const toggle = perTypeToggles.first();
    const initial = await toggle.getAttribute('aria-checked');

    await toggle.click();
    await expect(toggle).not.toHaveAttribute('aria-checked', initial ?? 'true', { timeout: 5_000 });

    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', initial ?? 'true', { timeout: 5_000 });
  });
});
