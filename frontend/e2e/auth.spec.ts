import { test, expect } from '@playwright/test';
import { requireCreds } from './_helpers/env';

// Auth flow tests run against the live backend with no pre-seeded session.
// Register / reset-password are intentionally NOT exercised here:
//   - register would create a real user on every run
//   - reset-password requires reading an email
// Both are best validated manually in a sandbox environment.

test.describe('Auth', () => {
  test.beforeEach(async ({ page }) => {
    // Make sure consent / onboarding are unset so we exercise the full flow.
    await page.addInitScript(() => {
      localStorage.removeItem('personage_consent_accepted');
      localStorage.setItem('personage_ios_install_dismissed', 'true');
      localStorage.setItem('personage_push_dismissed', 'true');
    });
  });

  test('login → consent → tasks screen', async ({ page }) => {
    const { email, password } = requireCreds();

    await page.goto('/');

    await expect(page.getByText('Personage', { exact: true })).toBeVisible();

    await page.getByPlaceholder('Email').fill(email);
    await page.getByPlaceholder('Пароль').fill(password);
    await page.getByRole('button', { name: 'Войти' }).click();

    // Consent sheet appears.
    const consentSheet = page.getByText('Обработка персональных данных');
    await expect(consentSheet).toBeVisible();

    // Two checkboxes by their labels.
    await page.getByText('Я прочитал(а) и принимаю').click();
    await page.getByText('Согласен(на) на обработку').click();

    await page.getByRole('button', { name: 'Принять и продолжить' }).click();

    // Lands on Tasks screen.
    await expect(page.getByRole('heading', { name: 'Задачи' }).or(
      page.getByText('Задачи', { exact: true }).first(),
    )).toBeVisible({ timeout: 15_000 });
  });

  test('forgot-password reveals "letter sent" state', async ({ page }) => {
    const { email } = requireCreds();
    await page.goto('/');

    await page.getByRole('button', { name: 'Забыли пароль?' }).click();

    await expect(page.getByText('Восстановление пароля')).toBeVisible();

    await page.getByPlaceholder('Email').fill(email);
    await page.getByRole('button', { name: 'Отправить ссылку' }).click();

    await expect(page.getByText('Письмо отправлено')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole('button', { name: 'Вернуться к входу' })).toBeVisible();
  });

  test('forgot-password "back" returns to login', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Забыли пароль?' }).click();
    await expect(page.getByText('Восстановление пароля')).toBeVisible();

    await page.getByRole('button', { name: 'Назад' }).click();
    await expect(page.getByPlaceholder('Пароль')).toBeVisible();
  });

  test('register tab swap surfaces the name field', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Регистрация' }).click();
    await expect(page.getByPlaceholder('Ваше имя')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Создать аккаунт' })).toBeVisible();
  });
});
