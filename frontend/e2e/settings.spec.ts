import { test, expect } from '@playwright/test';

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Настройки' }).click();
    await expect(page.getByText('Настройки', { exact: true }).first()).toBeVisible({ timeout: 15_000 });
  });

  test('profile + integrations sections render', async ({ page }) => {
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

  test('toggle a notification setting and revert it (idempotent)', async ({ page }) => {
    const toggles = page.getByRole('switch');
    const count = await toggles.count();
    if (count === 0) test.skip(true, 'Account has no notification toggles.');

    const toggle = toggles.first();
    const initial = await toggle.getAttribute('aria-checked');

    await toggle.click();
    await expect(toggle).not.toHaveAttribute('aria-checked', initial ?? 'true', { timeout: 5_000 });

    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-checked', initial ?? 'true', { timeout: 5_000 });
  });

  test('logout returns to auth screen', async ({ page }) => {
    await page.getByRole('button', { name: 'Выйти из аккаунта' }).click();

    await expect(page.getByText('Personage', { exact: true })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByPlaceholder('Email')).toBeVisible();

    // Re-seed for subsequent tests.
    const tokens = await page.evaluate(() => localStorage.getItem('personage_auth_tokens'));
    expect(tokens).toBeNull();
  });
});
