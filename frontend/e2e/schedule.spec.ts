import { test, expect } from '@playwright/test';

test.describe('Schedule', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Расписание' }).click();
    await expect(page.getByText('Расписание', { exact: true }).first()).toBeVisible({ timeout: 15_000 });
  });

  test('week strip + prev/next navigation', async ({ page }) => {
    const prev = page.getByRole('button', { name: 'Предыдущая неделя' });
    const next = page.getByRole('button', { name: 'Следующая неделя' });
    await expect(prev).toBeVisible();
    await expect(next).toBeVisible();

    await next.click();
    await page.waitForTimeout(150);
    await prev.click();
    await prev.click();
    await page.waitForTimeout(150);
  });

  test('renders either an agenda or the empty-day message', async ({ page }) => {
    // Whatever the day looks like in prod, one of the two should be visible.
    const empty = page.getByText('На выбранный день событий нет');
    const agendaSubtitle = page.getByText(/событий\b/i);
    await expect(empty.or(agendaSubtitle).first()).toBeVisible({ timeout: 10_000 });
  });
});
