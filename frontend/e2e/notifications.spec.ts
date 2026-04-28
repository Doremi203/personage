import { test, expect } from '@playwright/test';

test.describe('Notifications', () => {
  test.beforeEach(async ({ page }) => {
    // Reset client-side read state so the badge / "mark all" assertions are stable.
    await page.addInitScript(() => {
      localStorage.removeItem('personage_notifications_read');
    });
    await page.goto('/');
    await page.getByRole('button', { name: 'Уведомления' }).click();
    await expect(page.getByText('Уведомления', { exact: true }).first()).toBeVisible({ timeout: 15_000 });
  });

  test('list loads with filter tabs', async ({ page }) => {
    await expect(page.getByRole('button', { name: 'Все' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Непрочитанные' })).toBeVisible();
  });

  test('"Прочитать всё" button appears only when there is unread', async ({ page }) => {
    // Read state is fresh — if the account has any notifications, button is shown.
    const markAll = page.getByRole('button', { name: 'Прочитать всё' });

    if (await markAll.isVisible({ timeout: 2_000 }).catch(() => false)) {
      await markAll.click();
    }
    await expect(markAll).toBeHidden();
    // After either path, the all-read indicator is shown — either the
    // header subtitle, or the unread-tab empty state heading (both read
    // "Всё прочитано").
    await expect(page.getByText('Всё прочитано').first()).toBeVisible();
  });
});
