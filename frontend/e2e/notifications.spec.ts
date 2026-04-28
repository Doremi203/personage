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
    const unreadTab = page.getByRole('button', { name: 'Непрочитанные' });
    const allTab = page.getByRole('button', { name: 'Все' });
    await expect(unreadTab).toBeVisible();
    await expect(allTab).toBeVisible();

    // Unread is the default tab and is rendered to the left of All.
    await expect(unreadTab).toHaveAttribute('aria-pressed', 'true');
    await expect(allTab).toHaveAttribute('aria-pressed', 'false');
    const unreadBox = await unreadTab.boundingBox();
    const allBox = await allTab.boundingBox();
    expect(unreadBox && allBox && unreadBox.x < allBox.x).toBe(true);
  });

  test('"Прочитать всё" button appears only when there is unread', async ({ page }) => {
    // Read state is fresh — if the account has unread notifications, button is shown.
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
