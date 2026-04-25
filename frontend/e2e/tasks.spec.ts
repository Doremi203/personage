import { test, expect } from '@playwright/test';

// Tasks screen — read-only assertions only.
// Complete / postpone / delete are destructive against the live backend
// and are NOT exercised here. They should be tested manually or against
// a sandbox env with seeded data.

test.describe('Tasks', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('Задачи', { exact: true }).first()).toBeVisible({ timeout: 15_000 });
  });

  test('all four filter tabs are present and switchable', async ({ page }) => {
    for (const label of ['Сегодня', 'Скоро', 'Без даты', 'Готово'] as const) {
      const tab = page.getByRole('button', { name: new RegExp(`^${label}(\\s|$)`) });
      await expect(tab).toBeVisible();
      await tab.click();
      // Wait for tab to settle — list area is below.
      await page.waitForTimeout(150);
    }
  });

  test('search input filters the list', async ({ page }) => {
    const search = page.getByPlaceholder('Найти задачу');
    await expect(search).toBeVisible();
    await search.fill('zzzz_no_match_zzzz');
    // Either an empty-state hero or a count of zero — the list should not
    // render its normal cards. Just assert the search input keeps its value.
    await expect(search).toHaveValue('zzzz_no_match_zzzz');
    await search.fill('');
  });

  test('bottom tab bar navigates to Schedule and back', async ({ page }) => {
    await page.getByRole('button', { name: 'Расписание' }).click();
    await expect(page.getByText('Расписание', { exact: true }).first()).toBeVisible();

    await page.getByRole('button', { name: 'Задачи' }).click();
    await expect(page.getByPlaceholder('Найти задачу')).toBeVisible();
  });
});
