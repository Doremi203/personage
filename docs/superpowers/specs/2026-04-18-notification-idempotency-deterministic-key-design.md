# Notification Idempotency — Deterministic Key Revision

**Date:** 2026-04-18
**Revises:** `docs/superpowers/specs/2026-04-18-notification-idempotency-design.md`
**Scope:** `backend/tasker/internal/domain`, `backend/tasker/internal/services/notifications`, `backend/tasker/internal/usecase/scheduling`

## Problem

The original spec computes the idempotency key using `time.Now()` truncated to a 5-minute UTC bucket:

```go
IdempotencyKey(userID, time.Now(), typ, title)
```

This creates a bucket boundary race for upcoming-event notifications. If two worker ticks
straddle a 5-minute boundary — e.g. `notificationTime = 09:59:59`, tick 1 at `09:59:30`
(bucket `09:55`) and tick 2 at `10:00:30` (bucket `10:00`) — they produce different keys
and both push through. The 2-minute detection window (restored by the original spec) makes
this likely: any notification whose scheduled time sits within 1 minute of a bucket edge is
vulnerable.

## Fix

Use the task's pre-computed `notificationTime` (`task.StartTime.Add(-interval)`) as the
key anchor instead of `time.Now()`. This value is derived from task data and is identical
for every worker tick that fires for the same task and interval, regardless of wall-clock
timing.

## Changes

### `backend/tasker/internal/domain/notifications.go`

Add one optional field to `Notification`:

```go
type Notification struct {
    UserID           UserID
    Title            string
    Body             string
    Type             string
    // NotificationTime, when set, is used as the idempotency time anchor instead of
    // time.Now(). Set by callers that have a stable reference time (e.g. upcoming-event
    // notifier uses task.StartTime.Add(-interval)).
    NotificationTime *time.Time
}
```

### `backend/tasker/internal/services/notifications/scenarios.go`

In `NotifyUpcomingEvents`, set `NotificationTime` on the notification before sending:

```go
notificationTime := task.StartTime.Add(-interval)
// ... window check unchanged ...
notification := domain.Notification{
    UserID:           userID,
    Title:            n.formatUpcomingEventTitle(task, interval),
    Body:             n.formatUpcomingEventBody(task),
    Type:             "upcoming_event",
    NotificationTime: &notificationTime,
}
```

### `backend/tasker/internal/services/notifications/service.go`

Fall back to `s.now()` when `NotificationTime` is nil:

```go
func (s *notificatorPushService) Send(ctx context.Context, n domain.Notification) error {
    t := s.now()
    if n.NotificationTime != nil {
        t = *n.NotificationTime
    }
    return s.client.SendMessage(ctx, &pushpb.Notification{
        RecipientId:    n.UserID.String(),
        Title:          n.Title,
        Body:           n.Body,
        Icon:           "/icon-72x72.png",
        Url:            "/",
        Type:           n.Type,
        IdempotencyKey: IdempotencyKey(n.UserID.String(), t, n.Type, n.Title),
    }, sqs.WithGroupID("tasker"))
}
```

### `backend/tasker/internal/usecase/scheduling/usecase.go`

No change. The scheduling usecase leaves `NotificationTime` nil, so `service.go` falls
back to `time.Now()`. A scheduling run completes well within 5 minutes, so the existing
bucket approach remains correct for that path.

## What stays the same

- `IdempotencyKey` function signature and hash algorithm are unchanged.
- Notificator changes (migration, `CreateIfAbsent`, matcher reorder) are unchanged.
- The 5-minute bucket width is unchanged.
- The title still encodes task identity for upcoming events
  (`"⏰ [Task] через 15 минут"`), so distinct tasks and intervals remain distinct keys.

## Correctness guarantee after fix

Two worker ticks at `09:59:30` and `10:00:30` for a task with
`notificationTime = 09:59:59` both compute:

```
bucket = 09:59:59.Truncate(5min) = 09:55:00
key    = SHA256("user_id|09:55:00.Unix()|upcoming_event|⏰ [Task] через 15 минут")
```

Same key → notificator `INSERT ... ON CONFLICT DO NOTHING` drops the second → one push delivered.

## Testing delta

Add a unit test case to `TestIdempotencyKey` in `idempotency_test.go`:

```
- two calls with times on opposite sides of a 5-minute boundary but the same
  notificationTime anchor → same key (because the anchor is the task time, not wall clock)
```

This case is already covered implicitly by the "stable within the same 5-minute bucket"
test if `notificationTime` is used as input, but an explicit regression test naming the
boundary scenario makes the intent clear.

Add a test to `scenarios_test.go` (or a new `service_test.go`):

```
- NotifyUpcomingEvents called twice with simulated now values straddling a 5-minute
  bucket boundary → both domain.Notification values carry the same NotificationTime →
  IdempotencyKey produces the same result for both
```
