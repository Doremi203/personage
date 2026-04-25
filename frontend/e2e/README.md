# Frontend e2e (Playwright)

End-to-end tests run the real frontend (Vite dev server on `localhost:5173`)
against the **production** backend (`*.persomanage.ru`). No mocks.

## Setup

1. Install dependencies (one-time):
   ```bash
   cd frontend
   npm install
   npx playwright install chromium
   ```

2. Provide credentials for a test account in env:
   ```bash
   export TEST_USER_EMAIL='you+test@example.com'
   export TEST_USER_PASSWORD='your-test-password'
   ```

   Optionally override the auth host (defaults to `https://auth.persomanage.ru`):
   ```bash
   export VITE_AUTH_API_URL='https://auth.persomanage.ru'
   ```

## Running

```bash
npm run test:e2e           # headless
npm run test:e2e:ui        # Playwright UI mode
npx playwright test auth   # filter by file/name
```

The `setup` project authenticates once via the auth API and persists the
session to `playwright/.auth/user.json`. The `authenticated` project reuses
that storage state for every spec except `auth.spec.ts`, which exercises the
login UI from a clean session.

## Coverage

| Spec                  | Flow                                                   |
| --------------------- | ------------------------------------------------------ |
| `auth.spec.ts`        | login + consent, forgot password, register tab swap    |
| `tasks.spec.ts`       | filter tabs, search, navigation                        |
| `schedule.spec.ts`    | week strip, prev/next navigation, agenda/empty state   |
| `notifications.spec.ts` | list + filter tabs, "Прочитать всё"                  |
| `settings.spec.ts`    | profile load, notification toggle (idempotent), logout |

## What is **not** covered (by design)

These flows are skipped because they would either pollute production data,
require external systems, or destroy state that can't be cleanly restored:

- **Register** — creates real users
- **Reset password** — needs to read an email
- **Gmail OAuth** — redirects to Google
- **Push notifications** — service worker + browser permission
- **Task complete / postpone / delete** — destructive on real tasks

Test these manually against a sandbox environment.
