# Notification Rate Limiting Design

## Overview

Add per-user rate limiting to the notificator service to prevent overwhelming users with notifications. `upcoming_event` notifications always send immediately. `schedule_change` notifications are rate-limited per user; when the limit is exceeded they are delayed and retried, then dropped if too old.

## Requirements

- `upcoming_event`: no rate limiting, always deliver immediately
- `schedule_change`: max 2/hour and max 10/day per user (system-wide thresholds, tracked per user)
- On rate limit hit: delay delivery, retry every 15 minutes, drop after 2 hours
- Limits are configurable via app config

## Data Model

Extend the existing `notifications` table rather than adding a separate pending queue table.

### Migration

```sql
ALTER TABLE notifications
    ALTER COLUMN sent_at DROP NOT NULL,
    ADD COLUMN status       TEXT        NOT NULL DEFAULT 'sent',
    ADD COLUMN retry_after  TIMESTAMPTZ,
    ADD COLUMN expires_at   TIMESTAMPTZ,
    ADD COLUMN push_payload JSONB;
```

**`status`** values: `sent` | `pending` | `dropped`

**`sent_at`** becomes nullable — `NULL` while a notification is pending, set when actually delivered.

**`push_payload`** stores `{body, icon, url}` required to re-send a pending notification. `title` is already a top-level column.

**`push_payload`** and scheduling columns (`retry_after`, `expires_at`) are `NULL` for `sent` records.

### Rate Limit Queries

Rate limit checks count only delivered notifications:

```sql
SELECT COUNT(*) FROM notifications
WHERE recipient_id = $1
  AND type = $2
  AND status = 'sent'
  AND sent_at > NOW() - INTERVAL '1 hour'

SELECT COUNT(*) FROM notifications
WHERE recipient_id = $1
  AND type = $2
  AND status = 'sent'
  AND sent_at > NOW() - INTERVAL '1 day'
```

The existing `notifications_recipient_id_sent_at_idx` index supports these queries.

## Domain Changes

### `Notification` struct

```go
type Notification struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Title       string
    Type        SettingType
    Text        string
    Status      NotificationStatus   // Sent | Pending | Dropped
    SentAt      *time.Time           // nil while pending; breaking change from time.Time — existing callers need nil checks
    RetryAfter  *time.Time
    ExpiresAt   *time.Time
    PushPayload *PushPayload         // {Body, Icon, URL}
}

type PushPayload struct {
    Body string
    Icon string
    URL  string
}

type NotificationStatus string

const (
    StatusSent    NotificationStatus = "sent"
    StatusPending NotificationStatus = "pending"
    StatusDropped NotificationStatus = "dropped"
)
```

### `notification.Repo` interface additions

```go
ListPending(ctx context.Context) ([]Notification, error)
Drop(ctx context.Context, id uuid.UUID) error
UpdateRetryAfter(ctx context.Context, id uuid.UUID, retryAfter time.Time) error
CountSentSince(ctx context.Context, userID uuid.UUID, typ SettingType, since time.Time) (int, error)
```

## New Components

### `internal/services/ratelimit/`

```go
type Limits struct {
    Hourly int
    Daily  int
}

type RateLimiter struct {
    repo   notification.Repo
    limits map[notification.SettingType]Limits
}

func (r *RateLimiter) Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error)
```

- Types not present in the `limits` map are always allowed.
- If `CountSentSince` returns an error, `Allow` returns `false` (fail-safe: don't send on DB error).
- Two checks must both pass: hourly count < `Limits.Hourly` AND daily count < `Limits.Daily`.

### `internal/services/retrier/`

```go
type Retrier struct {
    repo          notification.Repo
    rateLimiter   *ratelimit.RateLimiter
    pushSender    usecase.PushSender
    retryInterval time.Duration  // 15m
    maxAge        time.Duration  // 2h
}

func (r *Retrier) Run(ctx context.Context)
```

Ticks every minute. On each tick:

1. `ListPending()` — fetch all pending notifications
2. For each:
   - `now > expires_at` → `Drop(id)` (expired)
   - `now >= retry_after` → `RateLimiter.Allow()`?
     - allowed: send push → mark `status='sent'`, `sent_at=now`
     - denied: `UpdateRetryAfter(id, now + retryInterval)`
3. Errors during push send → `UpdateRetryAfter` (do not drop), log error and continue

## Worker Changes

`internal/worker/matcher.go` — updated handler flow:

```
1. Parse SQS message (recipient_id, title, body, icon, url, type, detailed_text)
2. Fetch push recipient; if no subscriptions → skip (existing behavior)
3. Check notification settings (existing — user enabled/disabled)
4. RateLimiter.Allow(userID, type)?
   - allowed (or type has no limits):
       send push
       Insert Notification{status='sent', sent_at=now, ...}
   - denied:
       Insert Notification{status='pending', retry_after=now+15m, expires_at=now+2h, push_payload={body,icon,url}, ...}
```

## Configuration

```go
type RateLimitConfig struct {
    ScheduleChange struct {
        HourlyLimit int           // default: 2
        DailyLimit  int           // default: 10
    }
    RetryInterval  time.Duration  // default: 15m
    MaxAge         time.Duration  // default: 2h
}
```

## App Wiring

In `cmd/app/`:

1. Construct `RateLimiter` with config limits + notification repo
2. Construct `Retrier` with `RateLimiter` + `PushSender` + notification repo + config intervals
3. Start `retrier.Run(ctx)` as a goroutine alongside the existing SQS worker

Both share the same notification repo instance.

## Error Handling

| Scenario | Behavior |
|---|---|
| DB error in `Allow()` | Treat as rate-limited (fail-safe) |
| Push send fails in retrier | `UpdateRetryAfter`, log, continue |
| Push send fails in worker | Existing behavior (log, continue) |
| Retrier loop error | Log, continue to next record |

## Out of Scope

- Per-user configurable limits (all users share the same system-wide thresholds)
- Rate limiting for `upcoming_event` notifications
- Metrics/observability beyond structured logging
