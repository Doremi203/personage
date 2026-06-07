import { test, expect, type Page } from '@playwright/test';

// Self-contained tests for task time-field display: the UI must clearly
// separate a task's DEADLINE (крайний срок) from its scheduled START/END slot,
// driven by status — not by the mere presence of a startTime. All backend
// traffic is mocked via page.route so these tests are deterministic and need
// no live credentials. Assertions use exact text matches so they don't depend
// on the date formatting (which is timezone-dependent).

interface FulfillBody {
  status: number;
  contentType?: string;
  body: string;
}

function jsonFulfill(status: number, body: unknown): FulfillBody {
  return { status, contentType: 'application/json', body: JSON.stringify(body) };
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
      JSON.stringify({ accessToken: fakeJwt, refreshToken: 'fake-refresh-token' }),
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

// Three tasks covering every time-display branch:
//  - PLANNED with a real slot → card shows the scheduled time, detail shows Начало/Конец.
//  - UNPLANNED with a deadline → card shows "Дедлайн: …", detail shows a Дедлайн row, NO slot.
//  - UNPLANNED with a stale start/end (the postpone bug) → card shows "Без даты",
//    detail shows no slot and no deadline.
const PLANNED_TITLE = 'Запланированная задача со слотом';
const DEADLINE_TITLE = 'Задача с дедлайном без слота';
const STALE_TITLE = 'Отложенная задача со старым временем';

const TASKS = [
  {
    id: 'planned-1',
    title: PLANNED_TITLE,
    description: 'Описание запланированной задачи',
    status: 'TASK_STATUS_PLANNED',
    priority: 'TASK_PRIORITY_MID',
    category: 'TASK_CATEGORY_WORK',
    startTime: '2030-05-20T11:00:00Z',
    endTime: '2030-05-20T12:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    createdAt: '2026-01-01T00:00:00Z',
  },
  {
    id: 'deadline-1',
    title: DEADLINE_TITLE,
    description: 'Описание задачи с дедлайном',
    status: 'TASK_STATUS_UNPLANNED',
    priority: 'TASK_PRIORITY_MID',
    category: 'TASK_CATEGORY_PERSONAL',
    deadline: '2030-05-20T15:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    createdAt: '2026-01-01T00:00:00Z',
  },
  {
    id: 'stale-1',
    title: STALE_TITLE,
    description: 'Описание отложенной задачи',
    status: 'TASK_STATUS_UNPLANNED',
    priority: 'TASK_PRIORITY_MID',
    category: 'TASK_CATEGORY_STUDY',
    // Stale slot left over from a previous plan — must be ignored by the UI.
    startTime: '2030-05-23T18:00:00Z',
    endTime: '2030-05-23T18:30:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    createdAt: '2026-01-01T00:00:00Z',
  },
];

async function mockBackends(page: Page) {
  await page.route(/\/notifications(\?|$)/, (route) =>
    route.fulfill(jsonFulfill(200, { notifications: [] })),
  );
  await page.route(/\/notifications\/settings/, (route) =>
    route.fulfill(jsonFulfill(200, { settings: [] })),
  );
  await page.route(/\/user(\?|$)/, (route) =>
    route.fulfill(jsonFulfill(200, DEFAULT_USER)),
  );
  // Registered last → matched first: every /v1/tasks call returns our fixture,
  // regardless of the status/from/till query the screen sends.
  await page.route(/\/v1\/tasks(\?|$)/, (route) =>
    route.fulfill(
      jsonFulfill(200, {
        tasks: TASKS,
        total: TASKS.length,
        page: 1,
        pageSize: 50,
      }),
    ),
  );
}

// Open a task's detail sheet by clicking its card title, wait for the sheet.
async function openDetail(page: Page, title: string) {
  await page.getByText(title, { exact: true }).first().click();
  await expect(page.getByRole('button', { name: 'Закрыть' })).toBeVisible();
}

async function closeDetail(page: Page) {
  await page.getByRole('button', { name: 'Закрыть' }).click();
  await expect(page.getByRole('button', { name: 'Закрыть' })).toHaveCount(0);
}

test.describe('Task time-field display', () => {
  test.beforeEach(async ({ page }) => {
    await fakeAuthState(page);
    await mockBackends(page);
    await page.goto('/');
    await expect(page.getByText('Задачи', { exact: true }).first()).toBeVisible({
      timeout: 15_000,
    });
    // All three fixture tasks render in the list.
    await expect(page.getByText(PLANNED_TITLE, { exact: true })).toBeVisible();
    await expect(page.getByText(DEADLINE_TITLE, { exact: true })).toBeVisible();
    await expect(page.getByText(STALE_TITLE, { exact: true })).toBeVisible();
  });

  test('card labels a deadline as "Дедлайн", not as a plain/start date', async ({ page }) => {
    // The unplanned-with-deadline task shows its due date prefixed with "Дедлайн:".
    await expect(page.getByText(/^Дедлайн:/).first()).toBeVisible();
  });

  test('planned task detail shows the scheduled slot (Начало/Конец) and "Запланировано"', async ({ page }) => {
    await openDetail(page, PLANNED_TITLE);

    await expect(page.getByText('Запланировано', { exact: true })).toBeVisible();
    // Slot rows are present (exact label — distinct from the "Дедлайн: …" card text).
    await expect(page.getByText('Начало', { exact: true })).toBeVisible();
    await expect(page.getByText('Конец', { exact: true })).toBeVisible();
    // No deadline row for this task.
    await expect(page.getByText('Дедлайн', { exact: true })).toHaveCount(0);

    await closeDetail(page);
  });

  test('unplanned-with-deadline detail shows a Дедлайн row and NO slot', async ({ page }) => {
    await openDetail(page, DEADLINE_TITLE);

    // Status chip reads "Не запланировано", not "Без даты"/"Запланировано".
    await expect(page.getByText('Не запланировано', { exact: true })).toBeVisible();
    // A dedicated deadline row (exact label).
    await expect(page.getByText('Дедлайн', { exact: true })).toBeVisible();
    // The contradiction is gone: no Начало/Конец slot rows for an unplanned task.
    await expect(page.getByText('Начало', { exact: true })).toHaveCount(0);
    await expect(page.getByText('Конец', { exact: true })).toHaveCount(0);
    // The "no slot" hint is shown instead.
    await expect(page.getByText('В расписании', { exact: true })).toBeVisible();

    await closeDetail(page);
  });

  test('postponed task with a stale slot shows no slot and no deadline', async ({ page }) => {
    await openDetail(page, STALE_TITLE);

    await expect(page.getByText('Не запланировано', { exact: true })).toBeVisible();
    // The stale start/end from a previous plan must NOT surface as a slot…
    await expect(page.getByText('Начало', { exact: true })).toHaveCount(0);
    await expect(page.getByText('Конец', { exact: true })).toHaveCount(0);
    // …and there is no deadline on this task.
    await expect(page.getByText('Дедлайн', { exact: true })).toHaveCount(0);
    await expect(page.getByText('В расписании', { exact: true })).toBeVisible();

    await closeDetail(page);
  });
});
