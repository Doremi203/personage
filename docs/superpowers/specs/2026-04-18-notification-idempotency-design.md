# Notification Idempotency Design

**Date:** 2026-04-18
**Scope:** `backend/tasker`, `backend/notificator`

## Background

Tasker's `upcomingEventNotifier.NotifyUpcomingEvents` currently decides whether
to send a notification by checking if `now` falls within a wall-clock window
around `notificationTime`. With the worker running every 1 minute and the
window originally set to ±1 minute (2 minutes wide), two consecutive worker
ticks fell inside the same window and the user received the same notification
twice. An earlier fix narrowed the window to a half-open 1-minute slot, which
works only while `notifications-worker.interval` and the window stay in
lockstep — a brittle coupling that also loses notifications if a worker tick
is dropped.

This design replaces the window gymnastics with a producer-generated
idempotency key enforced on the consumer side.

## Goals

- Duplicate upcoming-event / schedule-change notifications are dropped
  regardless of whether the duplicate originates in tasker's timing logic or in
  SQS at-least-once redelivery.
- Distinct notifications (different users, different tasks, different advance
  intervals) are never collapsed.
- The fix is tolerant of worker tick drift, allowing the detection window to
  be widened back to its original 2 minutes without re-introducing duplicates.

## Non-goals

- SQS-level FIFO/MessageDeduplicationId. Idempotency is enforced in the
  notificator DB only.
- Retry-with-backoff semantics for failed pushes.
- A separate "notifications sent" ledger on the tasker side.

## Architecture

```
 tasker                                    notificator
 ------                                    -----------
 scenarios.go  --domain.Notification-->
                                           (SQS)
 notificatorPushService.Send
   computes idempotency_key
   publishes pushpb.Notification ------->  worker/matcher.Process
                                             notificationRepo.CreateIfAbsent
                                               inserted == false → ack & skip
                                               inserted == true  → pushSender.Send
```

- **Tasker** attaches `idempotency_key` to every `pushpb.Notification` it
  publishes. The key is computed at the SQS boundary
  (`notificatorPushService.Send`) so both `upcoming_event` and
  `schedule_change` producers get it for free.
- **Notificator** persists the notification row with a unique constraint on
  `idempotency_key`. If the insert hits the conflict, the handler acknowledges
  the SQS message and does nothing else. Only a successful insert proceeds to
  `pushSender.Send`.
- `scenarios.go:89` is reverted to the original 2-minute window. Duplicate
  firings caused by the wider window are now absorbed by the idempotency
  check.

## Key scheme

Implemented in a new helper in `backend/tasker/internal/services/notifications`:

```go
// IdempotencyKey returns a deterministic key for the given notification.
// All calls within the same 5-minute wall-clock bucket for the same
// (user_id, type, title) tuple produce the same key.
func IdempotencyKey(userID uuid.UUID, now time.Time, typ, title string) string {
    bucket := now.UTC().Truncate(5 * time.Minute).Unix()
    h := sha256.New()
    fmt.Fprintf(h, "%s|%d|%s|%s", userID, bucket, typ, title)
    return hex.EncodeToString(h.Sum(nil))
}
```

Components:

- `user_id` — scopes the key to a recipient.
- `bucket` — `now.Truncate(5m)` in UTC. Five minutes comfortably covers the
  2-minute detection window and absorbs SQS redelivery (10 s visibility
  timeout) without being so coarse that unrelated notifications collide.
- `type` — `"upcoming_event"` / `"schedule_change"`. Different categories for
  the same task in the same bucket remain distinct.
- `title` — already encodes task identity and advance interval for upcoming
  events (e.g. `"⏰ [Task] через 15 минут"`) and the change kind for schedule
  changes (e.g. `"📅 Задача перенесена"`). Two different tasks produce
  different titles; two different intervals for the same task produce
  different titles.

Collision envelope: two notifications collide only if the same user receives
an identical (type, title) pair within the same 5-minute bucket. That matches
the duplicate we want to suppress.

## Proto change

Add one field to `backend/notificator/api/push/push.proto`:

```proto
message Notification {
  // existing fields ...
  // idempotency_key dedupes delivery when the producer retries or fires
  // twice. Optional; when empty, notificator treats each message as unique.
  string idempotency_key = 8;
}
```

Regenerate with `make generate` on tasker and notificator. The field is
optional so any in-flight messages produced before rollout still succeed.

## Notificator changes

### Repository

`notification.Repo` grows one method and `notification.Notification` grows one
field:

```go
type Notification struct {
    ID             uuid.UUID
    UserID         uuid.UUID
    Title          string
    Type           string
    Text           string
    SentAt         time.Time
    IdempotencyKey string // empty string means "no dedup"
}

type Repo interface {
    // Create inserts a new notification and always succeeds.
    Create(ctx context.Context, n Notification) error

    // CreateIfAbsent inserts a new notification. If a row with the same
    // non-empty idempotency_key already exists, returns inserted=false and
    // no error. An empty idempotency_key behaves like Create.
    CreateIfAbsent(ctx context.Context, n Notification) (inserted bool, err error)

    // ... existing methods unchanged
}
```

Postgres implementation:

```sql
INSERT INTO notifications (recipient_id, title, type, text, idempotency_key)
VALUES ($1, $2, $3, $4, NULLIF($5, ''))
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING id;
```

`NULLIF('', '')` keeps the partial unique index happy for the empty-key
compatibility path.

### Handler

`internal/worker/matcher.go` reorders persistence and delivery:

```go
inserted, err := p.notificationRepo.CreateIfAbsent(ctx, notification.Notification{
    UserID:         recipientUUID,
    Title:          data.GetTitle(),
    Type:           data.GetType(),
    Text:           data.GetDetailedText(),
    IdempotencyKey: data.GetIdempotencyKey(),
})
if err != nil {
    return errors.WrapFailf(err, "persist notification for recipient %v", ...)
}
if !inserted {
    p.logger.Infof("duplicate notification skipped key=%v", ...)
    return nil
}

if err := p.senderUseCase.Send(ctx, pushRecipient, push.Push{ ... }); err != nil {
    return errors.WrapFailf(err, "send push to recipient with id %v", ...)
}
```

Ordering trade-off: insert-first means a crash between `CreateIfAbsent` and
`Send` leaves a DB row without a push delivered. We accept that — duplicate
pushes are user-visible and frequent under the current design, missed pushes
from mid-handler crashes are rare and the DB row is a reasonable audit trail.

### Migration

`backend/notificator/migrations/00004_add_notifications_idempotency_key.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE notifications ADD COLUMN idempotency_key text;
CREATE UNIQUE INDEX notifications_idempotency_key_uniq
    ON notifications (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS notifications_idempotency_key_uniq;
ALTER TABLE notifications DROP COLUMN IF EXISTS idempotency_key;
-- +goose StatementEnd
```

Partial unique index so the backfill of existing rows with `NULL` keys never
collides.

## Tasker changes

### Key computation at the SQS boundary

`backend/tasker/internal/services/notifications/service.go`:

```go
func (s *notificatorPushService) Send(
    ctx context.Context,
    n domain.Notification,
) error {
    return s.client.SendMessage(ctx, &pushpb.Notification{
        RecipientId:    n.UserID.String(),
        Title:          n.Title,
        Body:           n.Body,
        Icon:           "/icon-72x72.png",
        Url:            "/",
        Type:           n.Type,
        IdempotencyKey: IdempotencyKey(uuid.UUID(n.UserID), time.Now(), n.Type, n.Title),
    }, sqs.WithGroupID("tasker"))
}
```

`time.Now()` is captured at publish time, not at notifier-decision time. The
tasker worker runs every minute so publish-time and decision-time sit in the
same 5-minute bucket with a large margin.

### Revert the scenarios.go window

Restore the original 2-minute window — idempotency replaces the window-based
dedup:

```go
// backend/tasker/internal/services/notifications/scenarios.go
for _, interval := range n.config.UpcomingEventIntervals {
    notificationTime := task.StartTime.Add(-interval)
    if now.After(notificationTime.Add(-time.Minute)) &&
        now.Before(notificationTime.Add(time.Minute)) {
        // ... send as before
    }
}
```

This reintroduces the double-fire, which the notificator now absorbs.

## Testing

### Unit

- `IdempotencyKey` determinism:
  - same inputs → same key
  - same `(userID, type, title)` across timestamps in the same 5-min bucket
    → same key
  - timestamps crossing a 5-min boundary → different keys
  - change in any component (user / type / title) → different key

### Notificator integration (testcontainers)

- Two handler invocations with the same `(user_id, type, title, bucket)`
  produce one DB row and call `pushSender.Send` once.
- Two invocations with different keys produce two DB rows and two
  `pushSender.Send` calls.
- Empty `idempotency_key` preserves today's behaviour (every message
  persisted, every message pushed).

### Tasker

- `NotifyUpcomingEvents` fired twice inside the same 2-minute window produces
  two SQS messages with the same `idempotency_key`. (The dedup itself is
  verified in the notificator integration test.)

## Rollout

1. Deploy the notificator migration + consumer change. With no producers
   setting the key yet, the `NULLIF('', '')` branch keeps traffic working.
2. Deploy tasker with `IdempotencyKey` populated on `notificatorPushService`.
3. Revert the `scenarios.go:89` window to the original 2 minutes in the same
   or a follow-up PR.

## Open risks

- **Clock skew between tasker replicas.** If two tasker replicas run with
  >5-minute skew, they could emit the same notification in different buckets.
  In-cluster NTP keeps skew well under a minute; accepted.
- **Title changes.** Any change to the title format between a duplicate fire
  and its retry would bypass dedup. Titles are currently deterministic
  functions of the task and interval; adding live context (e.g. weather) to
  the title would need a dedicated idempotency field instead.
