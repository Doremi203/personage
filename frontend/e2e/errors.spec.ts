import { test, expect, type Page } from '@playwright/test';

// Self-contained tests for the centralized API error handling in
// `src/utils/apiError.ts`. All backend traffic is mocked via page.route so
// these tests are deterministic and need no live credentials.

interface FulfillBody {
  status: number;
  contentType?: string;
  body: string;
}

function jsonFulfill(status: number, body: unknown): FulfillBody {
  return {
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  };
}

// Auth API error shape: { errorCode, message, statusCode } (camelCase JSON
// from the C# ExceptionHandlingMiddleware).
function authError(status: number, errorCode: string, message = ''): FulfillBody {
  return jsonFulfill(status, { errorCode, message, statusCode: status });
}

// gRPC-Gateway error shape: { code, message, details: [] } — message is the
// raw gRPC status message, almost always English ("task not found").
function grpcError(status: number, message: string, code = 0): FulfillBody {
  return jsonFulfill(status, { code, message, details: [] });
}

async function bypassConsent(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('personage_consent_accepted', 'true');
    localStorage.setItem('personage_ios_install_dismissed', 'true');
    localStorage.setItem('personage_push_dismissed', 'true');
  });
}

async function fakeAuthState(page: Page) {
  await page.addInitScript(() => {
    const enc = (s: string) =>
      btoa(s).replace(/=+$/, '').replace(/\+/g, '-').replace(/\//g, '_');
    const header = enc('{"alg":"none","typ":"JWT"}');
    const payload = enc(
      JSON.stringify({ user_id: 'test-user-id', exp: 9_999_999_999 }),
    );
    const fakeJwt = `${header}.${payload}.sig`;
    localStorage.setItem(
      'personage_auth_tokens',
      JSON.stringify({
        accessToken: fakeJwt,
        refreshToken: 'fake-refresh-token',
      }),
    );
    localStorage.setItem('personage_consent_accepted', 'true');
    localStorage.setItem('personage_ios_install_dismissed', 'true');
    localStorage.setItem('personage_push_dismissed', 'true');
  });
}

const DEFAULT_USER = {
  email: 'test@example.com',
  name: 'Test',
  gmailIntegration: { enabled: false },
  telegramIntegration: { enabled: false },
  googleCalendarIntegration: { enabled: false },
};

async function mockSuccessfulBackends(page: Page) {
  // Default 200 mocks so screens render without hitting the live backend.
  // Per-test overrides registered after these are matched first (LIFO).
  await page.route(/\/v1\/tasks(\?|$)/, (route) =>
    route.fulfill(
      jsonFulfill(200, { tasks: [], total: 0, page: 1, pageSize: 20 }),
    ),
  );
  await page.route(/\/notifications(\?|$)/, (route) =>
    route.fulfill(jsonFulfill(200, { notifications: [] })),
  );
  await page.route(/\/notifications\/settings/, (route) =>
    route.fulfill(jsonFulfill(200, { settings: [] })),
  );
  await page.route(/\/user(\?|$)/, (route) =>
    route.fulfill(jsonFulfill(200, DEFAULT_USER)),
  );
}

// ─── Unauthenticated screens ──────────────────────────────────────────────

test.describe('Error handling — unauthenticated', () => {
  test.beforeEach(async ({ page }) => {
    await bypassConsent(page);
  });

  test('login 401 InvalidCredentials → "Неверный email или пароль."', async ({
    page,
  }) => {
    await page.route(/\/auth\/personage\/login\/password/, (route) =>
      route.fulfill(authError(401, 'InvalidCredentials', 'Authentication exception. Please log in again.')),
    );

    await page.goto('/');
    await page.getByPlaceholder('Email').fill('wrong@example.com');
    await page.getByPlaceholder('Пароль').fill('wrongpassword');
    await page.getByRole('button', { name: 'Войти' }).click();

    await expect(page.getByText('Неверный email или пароль.')).toBeVisible();
  });

  test('register 409 UserAlreadyExists → "Аккаунт с таким email уже существует."', async ({
    page,
  }) => {
    await page.route(/\/auth\/personage\/register/, (route) =>
      route.fulfill(authError(409, 'UserAlreadyExists', 'User already exists')),
    );

    await page.goto('/');
    await page.getByRole('button', { name: 'Регистрация' }).click();
    await page.getByPlaceholder('Ваше имя', { exact: true }).fill('Test');
    await page.getByPlaceholder('Email').fill('exists@example.com');
    await page.getByPlaceholder('Пароль', { exact: true }).fill('password123');
    await page.getByPlaceholder('Повторите пароль').fill('password123');
    await page.getByRole('button', { name: 'Создать аккаунт' }).click();

    await expect(
      page.getByText('Аккаунт с таким email уже существует.'),
    ).toBeVisible();
  });

  test('register 400 EmailValidationFail → "Введите корректный email."', async ({
    page,
  }) => {
    await page.route(/\/auth\/personage\/register/, (route) =>
      route.fulfill(authError(400, 'EmailValidationFail', 'Invalid email')),
    );

    await page.goto('/');
    await page.getByRole('button', { name: 'Регистрация' }).click();
    await page.getByPlaceholder('Ваше имя', { exact: true }).fill('Test');
    await page.getByPlaceholder('Email').fill('not-an-email');
    await page.getByPlaceholder('Пароль', { exact: true }).fill('password123');
    await page.getByPlaceholder('Повторите пароль').fill('password123');
    await page.getByRole('button', { name: 'Создать аккаунт' }).click();

    await expect(page.getByText('Введите корректный email.')).toBeVisible();
  });

  test('register 400 PasswordValidationFail → friendly password message', async ({
    page,
  }) => {
    await page.route(/\/auth\/personage\/register/, (route) =>
      route.fulfill(
        authError(400, 'PasswordValidationFail', 'Password too weak'),
      ),
    );

    await page.goto('/');
    await page.getByRole('button', { name: 'Регистрация' }).click();
    await page.getByPlaceholder('Ваше имя', { exact: true }).fill('Test');
    await page.getByPlaceholder('Email').fill('test@example.com');
    await page.getByPlaceholder('Пароль', { exact: true }).fill('weakpass');
    await page.getByPlaceholder('Повторите пароль').fill('weakpass');
    await page.getByRole('button', { name: 'Создать аккаунт' }).click();

    await expect(
      page.getByText('Пароль не отвечает требованиям безопасности.'),
    ).toBeVisible();
  });

  test('forgot password 429 → "Слишком много запросов..."', async ({ page }) => {
    await page.route(/\/auth\/personage\/forgot-password/, (route) =>
      route.fulfill(grpcError(429, 'rate limit exceeded')),
    );

    await page.goto('/');
    await page.getByRole('button', { name: 'Забыли пароль?' }).click();
    await page.getByPlaceholder('Email').fill('test@example.com');
    await page.getByRole('button', { name: 'Отправить ссылку' }).click();

    await expect(
      page.getByText('Слишком много запросов. Попробуйте позже.'),
    ).toBeVisible();
  });

  test('login 500 → "Сервис временно недоступен..."', async ({ page }) => {
    await page.route(/\/auth\/personage\/login\/password/, (route) =>
      route.fulfill(grpcError(500, 'internal server error')),
    );

    await page.goto('/');
    await page.getByPlaceholder('Email').fill('test@example.com');
    await page.getByPlaceholder('Пароль').fill('password');
    await page.getByRole('button', { name: 'Войти' }).click();

    await expect(
      page.getByText('Сервис временно недоступен. Попробуйте позже.'),
    ).toBeVisible();
  });

  test('login: backend Russian message passes through', async ({ page }) => {
    await page.route(/\/auth\/personage\/login\/password/, (route) =>
      route.fulfill(
        jsonFulfill(400, {
          message: 'Учётная запись заблокирована администратором.',
        }),
      ),
    );

    await page.goto('/');
    await page.getByPlaceholder('Email').fill('blocked@example.com');
    await page.getByPlaceholder('Пароль').fill('password');
    await page.getByRole('button', { name: 'Войти' }).click();

    await expect(
      page.getByText('Учётная запись заблокирована администратором.'),
    ).toBeVisible();
  });

  test('login: unmapped 4xx (418) → generic Russian fallback', async ({
    page,
  }) => {
    await page.route(/\/auth\/personage\/login\/password/, (route) =>
      route.fulfill({
        status: 418,
        contentType: 'application/json',
        body: '{}',
      }),
    );

    await page.goto('/');
    await page.getByPlaceholder('Email').fill('test@example.com');
    await page.getByPlaceholder('Пароль').fill('password');
    await page.getByRole('button', { name: 'Войти' }).click();

    await expect(page.getByText('Некорректный запрос (418).')).toBeVisible();
  });

  test('login: empty error body → status fallback', async ({ page }) => {
    await page.route(/\/auth\/personage\/login\/password/, (route) =>
      route.fulfill({ status: 403, contentType: 'application/json', body: '' }),
    );

    await page.goto('/');
    await page.getByPlaceholder('Email').fill('test@example.com');
    await page.getByPlaceholder('Пароль').fill('password');
    await page.getByRole('button', { name: 'Войти' }).click();

    await expect(
      page.getByText('Недостаточно прав для этого действия.'),
    ).toBeVisible();
  });

  test('login: non-JSON error body still produces friendly message', async ({
    page,
  }) => {
    await page.route(/\/auth\/personage\/login\/password/, (route) =>
      route.fulfill({
        status: 502,
        contentType: 'text/plain',
        body: 'Bad Gateway',
      }),
    );

    await page.goto('/');
    await page.getByPlaceholder('Email').fill('test@example.com');
    await page.getByPlaceholder('Пароль').fill('password');
    await page.getByRole('button', { name: 'Войти' }).click();

    await expect(
      page.getByText('Сервис временно недоступен. Попробуйте позже.'),
    ).toBeVisible();
  });
});

// ─── Authenticated screens ────────────────────────────────────────────────

test.describe('Error handling — authenticated', () => {
  test.beforeEach(async ({ page }) => {
    await fakeAuthState(page);
    await mockSuccessfulBackends(page);
  });

  test('tasks list 500 → ErrorState with Russian fallback + retry button', async ({
    page,
  }) => {
    await page.route(/\/v1\/tasks(\?|$)/, (route) =>
      route.fulfill(grpcError(500, 'database unavailable')),
    );

    await page.goto('/');
    await expect(
      page.getByText('Сервис временно недоступен. Попробуйте позже.'),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole('button', { name: 'Повторить' })).toBeVisible();
  });

  test('tasks list 403 → "Недостаточно прав для этого действия."', async ({
    page,
  }) => {
    await page.route(/\/v1\/tasks(\?|$)/, (route) =>
      route.fulfill(grpcError(403, 'permission denied')),
    );

    await page.goto('/');
    await expect(
      page.getByText('Недостаточно прав для этого действия.'),
    ).toBeVisible({ timeout: 10_000 });
  });

  test('tasks list 429 → "Слишком много запросов..."', async ({ page }) => {
    await page.route(/\/v1\/tasks(\?|$)/, (route) =>
      route.fulfill(grpcError(429, 'too many requests')),
    );

    await page.goto('/');
    await expect(
      page.getByText('Слишком много запросов. Попробуйте позже.'),
    ).toBeVisible({ timeout: 10_000 });
  });

  test('tasks list 404 → "Запрашиваемый объект не найден."', async ({ page }) => {
    await page.route(/\/v1\/tasks(\?|$)/, (route) =>
      route.fulfill(grpcError(404, 'not found')),
    );

    await page.goto('/');
    await expect(
      page.getByText('Запрашиваемый объект не найден.'),
    ).toBeVisible({ timeout: 10_000 });
  });

  test('401 with refresh failure logs the user out back to AuthScreen', async ({
    page,
  }) => {
    await page.route(/\/v1\/tasks(\?|$)/, (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: '{}',
      }),
    );
    await page.route(/\/auth\/personage\/refresh/, (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: '{}',
      }),
    );

    await page.goto('/');

    // Token cleared → AUTH_STATE_CHANGE_EVENT → AuthScreen renders.
    await expect(page.getByPlaceholder('Email')).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.getByPlaceholder('Пароль')).toBeVisible();
  });

  test('notifications list 500 → ErrorState in Notifications tab', async ({
    page,
  }) => {
    // Override the default 200 list mock to fail.
    await page.route(/\/notifications(\?|$)/, (route) =>
      route.fulfill(grpcError(500, 'internal')),
    );

    await page.goto('/');
    await page.getByRole('button', { name: 'Уведомления' }).click();

    await expect(
      page.getByText('Сервис временно недоступен. Попробуйте позже.'),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole('button', { name: 'Повторить' })).toBeVisible();
  });

  test('settings: revoke Gmail 403 → inline gmailError displays Russian', async ({
    page,
  }) => {
    await page.route(/\/user(\?|$)/, (route) =>
      route.fulfill(
        jsonFulfill(200, {
          ...DEFAULT_USER,
          gmailIntegration: { enabled: true, gmail: 'foo@gmail.com' },
        }),
      ),
    );
    await page.route(/\/integrations\/revoke-access/, (route) =>
      route.fulfill(authError(403, 'Unknown', 'forbidden')),
    );

    await page.goto('/');
    await page.getByRole('button', { name: 'Настройки' }).click();
    await expect(page.getByText('foo@gmail.com')).toBeVisible({
      timeout: 10_000,
    });

    await page.getByRole('button', { name: /Отключить/ }).first().click();

    await expect(
      page.getByText('Недостаточно прав для этого действия.'),
    ).toBeVisible();
  });

  test('settings: revoke Gmail 409 → conflict message', async ({ page }) => {
    await page.route(/\/user(\?|$)/, (route) =>
      route.fulfill(
        jsonFulfill(200, {
          ...DEFAULT_USER,
          gmailIntegration: { enabled: true, gmail: 'foo@gmail.com' },
        }),
      ),
    );
    await page.route(/\/integrations\/revoke-access/, (route) =>
      route.fulfill(grpcError(409, 'integration is already being revoked')),
    );

    await page.goto('/');
    await page.getByRole('button', { name: 'Настройки' }).click();
    await expect(page.getByText('foo@gmail.com')).toBeVisible({
      timeout: 10_000,
    });

    await page.getByRole('button', { name: /Отключить/ }).first().click();

    await expect(
      page.getByText('Конфликт данных. Возможно, объект уже изменён.'),
    ).toBeVisible();
  });
});
