import { test as setup, expect } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { AUTH_URL, requireCreds } from './_helpers/env';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const authFile = path.join(__dirname, '..', 'playwright', '.auth', 'user.json');

setup('authenticate via API and persist storage state', async ({ page, request }) => {
  const { email, password } = requireCreds();

  const response = await request.post(`${AUTH_URL}/auth/personage/login/password`, {
    data: { email, password },
  });
  expect(response.ok(), `Login HTTP ${response.status()}`).toBeTruthy();
  const tokens = await response.json() as {
    accessToken: string;
    refreshToken?: string | null;
  };

  await page.goto('/');
  await page.evaluate((t) => {
    localStorage.setItem('personage_auth_tokens', JSON.stringify(t));
    localStorage.setItem('personage_consent_accepted', 'true');
    localStorage.setItem('personage_ios_install_dismissed', 'true');
    localStorage.setItem('personage_push_dismissed', 'true');
  }, tokens);

  await page.context().storageState({ path: authFile });
});
