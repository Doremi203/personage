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

| Spec                    | Flow                                                                                |
| ----------------------- | ----------------------------------------------------------------------------------- |
| `auth.spec.ts`          | login + consent, forgot password, register tab swap                                 |
| `tasks.spec.ts`         | filter tabs, search, navigation                                                     |
| `schedule.spec.ts`      | week strip, prev/next navigation, agenda/empty state                                |
| `notifications.spec.ts` | list + filter tabs, "Прочитать всё"                                                 |
| `settings.spec.ts`      | profile load, notification toggle (idempotent), logout                              |
| `push.spec.ts`          | onboarding/push prompt visibility, dismissal, denied path, subscribe + backend POST, error path, iOS install prompt |

### Push notifications

Chromium's `pushManager.subscribe` fails in headless mode (no real FCM
endpoint), so `push.spec.ts` patches `PushManager.prototype.subscribe` to
return a deterministic fake subscription. The rest of the flow runs
against the real wiring: the subscribe POST hits the production
notificator. The resulting subscription is stale (real backend will 410
on the next send, which we don't care about — it's how stale subs are
expected to age out).

`Notification` is wrapped in a Proxy via `addInitScript` so we can drive
the JS-visible permission state (`permission`, `requestPermission()`)
independently from the browser-level permission (controlled via
`context.grantPermissions`).

## What is **not** covered (by design)

These flows are skipped because they would either pollute production data
in a way that can't be cleanly recovered or require external systems we
can't drive from the test:

- **Register** — creates real users
- **Reset password** — needs to read an email
- **Gmail OAuth** — redirects to Google
- **Real push delivery** — would require the backend to send to our fake
  endpoint and the SW's `onpush` to fire (the fake endpoint isn't a real
  FCM target)
- **Task complete / postpone / delete** — destructive on real tasks

Test these manually against a sandbox environment.
