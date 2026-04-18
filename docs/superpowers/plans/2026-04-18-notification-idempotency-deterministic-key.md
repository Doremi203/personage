# Notification Idempotency — Deterministic Key Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the bucket-boundary race in idempotency key computation by using the task's pre-computed `notificationTime` as the anchor instead of `time.Now()`.

**Architecture:** Add `NotificationTime *time.Time` to `domain.Notification`. The upcoming-event notifier sets this field to `task.StartTime.Add(-interval)` — a value deterministic from task data. `notificatorPushService.Send` uses it as the anchor for `IdempotencyKey` when set, falling back to `s.now()` when nil. `scheduling/usecase.go` is untouched; it leaves the field nil and continues using `time.Now()`.

**Tech Stack:** Go 1.x, testify, AWS SQS (`sqs.ClientWriter` interface).

**Spec:** `docs/superpowers/specs/2026-04-18-notification-idempotency-deterministic-key-design.md`

**Working tree:** `.worktrees/notification-idempotency` (branch `feature/notification-idempotency`)

---

## File Map

| Action | File | What changes |
|--------|------|--------------|
| Modify | `backend/tasker/internal/domain/notifications.go` | Add `NotificationTime *time.Time` field |
| Modify | `backend/tasker/internal/services/notifications/service.go` | Use `NotificationTime` when set |
| Modify | `backend/tasker/internal/services/notifications/scenarios.go` | Make `now` injectable; set `NotificationTime` |
| Create | `backend/tasker/internal/services/notifications/service_test.go` | Tests for `service.go` anchor logic |
| Modify | `backend/tasker/internal/services/notifications/scenarios_test.go` | Add test for `NotificationTime` propagation |

---

## Task 1: Test + implement `NotificationTime` anchor in `service.go`

**Files:**
- Create: `backend/tasker/internal/services/notifications/service_test.go`
- Modify: `backend/tasker/internal/domain/notifications.go`
- Modify: `backend/tasker/internal/services/notifications/service.go`

All work is done inside `.worktrees/notification-idempotency/`.

- [ ] **Step 1: Write the failing test**

Create `backend/tasker/internal/services/notifications/service_test.go`:

```go
package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/sqs"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingSQSWriter struct {
	messages []*pushpb.Notification
}

func (w *capturingSQSWriter) SendMessage(_ context.Context, msg *pushpb.Notification, _ ...sqs.SendMessageOption) error {
	w.messages = append(w.messages, msg)
	return nil
}

func TestNotificatorPushService_UsesNotificationTimeAsAnchor(t *testing.T) {
	// notifTime sits exactly on a 5-minute bucket boundary
	notifTime := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	// Two worker ticks that straddle the 09:55/10:00 boundary
	tick1 := notifTime.Add(-30 * time.Second) // 09:59:30 — bucket 09:55
	tick2 := notifTime.Add(30 * time.Second)  // 10:00:30 — bucket 10:00

	// Precondition: the raw ticks land in different buckets and produce different keys
	key1 := IdempotencyKey("user-1", tick1, "upcoming_event", "title")
	key2 := IdempotencyKey("user-1", tick2, "upcoming_event", "title")
	require.NotEqual(t, key1, key2, "precondition: ticks must be in different buckets")

	writer := &capturingSQSWriter{}
	svc := &notificatorPushService{client: writer}

	notif := domain.Notification{
		UserID:           domain.UserID("user-1"),
		Title:            "title",
		Type:             "upcoming_event",
		NotificationTime: &notifTime,
	}

	svc.now = func() time.Time { return tick1 }
	require.NoError(t, svc.Send(context.Background(), notif))

	svc.now = func() time.Time { return tick2 }
	require.NoError(t, svc.Send(context.Background(), notif))

	require.Len(t, writer.messages, 2)
	assert.Equal(t, writer.messages[0].IdempotencyKey, writer.messages[1].IdempotencyKey,
		"both ticks must produce the same key when NotificationTime is set")
}

func TestNotificatorPushService_FallsBackToNowWhenNotificationTimeAbsent(t *testing.T) {
	fixedNow := time.Date(2026, 4, 18, 10, 2, 0, 0, time.UTC)

	writer := &capturingSQSWriter{}
	svc := &notificatorPushService{
		client: writer,
		now:    func() time.Time { return fixedNow },
	}

	notif := domain.Notification{
		UserID: domain.UserID("user-1"),
		Title:  "title",
		Type:   "schedule_change",
		// NotificationTime intentionally absent — should fall back to s.now()
	}

	require.NoError(t, svc.Send(context.Background(), notif))
	require.Len(t, writer.messages, 1)

	expected := IdempotencyKey("user-1", fixedNow, "schedule_change", "title")
	assert.Equal(t, expected, writer.messages[0].IdempotencyKey)
}
```

- [ ] **Step 2: Run the test — confirm compile error**

From `backend/` inside the worktree:

```bash
go test ./tasker/internal/services/notifications/... -run TestNotificatorPushService -count=1
```

Expected: compile error — `domain.Notification{} unknown field NotificationTime`.

- [ ] **Step 3: Add `NotificationTime` to `domain.Notification`**

Replace the `Notification` struct in `backend/tasker/internal/domain/notifications.go`:

```go
type Notification struct {
	UserID UserID
	Title  string
	Body   string
	Type   string
	// NotificationTime, when set, is used as the idempotency time anchor instead of
	// time.Now(). Callers with a stable reference time (e.g. upcoming-event notifier)
	// set this to avoid bucket-boundary races.
	NotificationTime *time.Time
}
```

(`time` is already imported in that file.)

- [ ] **Step 4: Update `service.go` to use `NotificationTime` when set**

Replace the `Send` method in `backend/tasker/internal/services/notifications/service.go`:

```go
func (s *notificatorPushService) Send(
	ctx context.Context,
	notification domain.Notification,
) error {
	t := s.now()
	if notification.NotificationTime != nil {
		t = *notification.NotificationTime
	}
	userID := notification.UserID.String()
	return s.client.SendMessage(ctx, &pushpb.Notification{
		RecipientId:    userID,
		Title:          notification.Title,
		Body:           notification.Body,
		Icon:           "/icon-72x72.png",
		Url:            "/",
		Type:           notification.Type,
		IdempotencyKey: IdempotencyKey(userID, t, notification.Type, notification.Title),
	}, sqs.WithGroupID("tasker"))
}
```

- [ ] **Step 5: Run the tests — confirm they pass**

```bash
go test ./tasker/internal/services/notifications/... -run TestNotificatorPushService -race -count=1
```

Expected: `PASS` on both sub-tests.

- [ ] **Step 6: Run the full tasker suite to catch regressions**

```bash
go test ./tasker/... -race -count=1
```

Expected: `ok` everywhere.

- [ ] **Step 7: Commit**

```bash
git add backend/tasker/internal/domain/notifications.go \
        backend/tasker/internal/services/notifications/service.go \
        backend/tasker/internal/services/notifications/service_test.go
git commit -m "feat(tasker): use NotificationTime as idempotency anchor in push service"
```

---

## Task 2: Test + propagate `NotificationTime` in `scenarios.go`

**Files:**
- Modify: `backend/tasker/internal/services/notifications/scenarios.go`
- Modify: `backend/tasker/internal/services/notifications/scenarios_test.go`

- [ ] **Step 1: Write the failing test**

Add the following to `backend/tasker/internal/services/notifications/scenarios_test.go` (append before the final closing brace — there is none, just append at end of file):

```go
// testNotificationSender captures sent notifications for assertions.
type testNotificationSender struct {
	sent []domain.Notification
}

func (s *testNotificationSender) Send(_ context.Context, n domain.Notification) error {
	s.sent = append(s.sent, n)
	return nil
}

func TestNotifyUpcomingEvents_SetsNotificationTimeAnchor(t *testing.T) {
	startTime := time.Date(2026, 4, 18, 10, 15, 0, 0, time.UTC)
	interval := 15 * time.Minute
	expectedNotifTime := startTime.Add(-interval) // 10:00:00 — on a bucket edge

	// Two worker ticks straddling the 09:55/10:00 boundary
	tick1 := expectedNotifTime.Add(-30 * time.Second) // 09:59:30
	tick2 := expectedNotifTime.Add(30 * time.Second)  // 10:00:30

	task := domain.Task{
		ID:        "task-1",
		Title:     "Test",
		StartTime: &startTime,
		Status:    domain.TaskStatusPlanned,
		Priority:  5,
	}

	sender := &testNotificationSender{}
	notifier, err := NewUpcomingEventNotifier(sender, domain.NotificationConfig{
		UpcomingEventMinPriority: 0,
		UpcomingEventIntervals:   []time.Duration{interval},
	}, message.NewPrinter(language.Russian))
	require.NoError(t, err)

	notifier.now = func() time.Time { return tick1 }
	require.NoError(t, notifier.NotifyUpcomingEvents(context.Background(), "user-1", []domain.Task{task}))

	notifier.now = func() time.Time { return tick2 }
	require.NoError(t, notifier.NotifyUpcomingEvents(context.Background(), "user-1", []domain.Task{task}))

	require.Len(t, sender.sent, 2, "both ticks are within the 2-minute window")
	require.NotNil(t, sender.sent[0].NotificationTime)
	require.NotNil(t, sender.sent[1].NotificationTime)
	assert.Equal(t, expectedNotifTime, *sender.sent[0].NotificationTime)
	assert.Equal(t, *sender.sent[0].NotificationTime, *sender.sent[1].NotificationTime,
		"both ticks must carry the same NotificationTime anchor")
}
```

Also add `"context"` to the import block at the top of `scenarios_test.go` if it is not already present.

- [ ] **Step 2: Run the test — confirm it fails**

```bash
go test ./tasker/internal/services/notifications/... -run TestNotifyUpcomingEvents_SetsNotificationTimeAnchor -count=1
```

Expected: compile error — `notifier.now undefined` (the field does not exist yet).

- [ ] **Step 3: Make `now` injectable in `upcomingEventNotifier`**

In `backend/tasker/internal/services/notifications/scenarios.go`:

Add `now func() time.Time` to the struct:

```go
type upcomingEventNotifier struct {
	sender  domain.NotificationsService
	config  domain.NotificationConfig
	printer *message.Printer
	now     func() time.Time
}
```

Wire it in `NewUpcomingEventNotifier` (replace the return statement):

```go
return &upcomingEventNotifier{
	sender:  sender,
	config:  config,
	printer: printer,
	now:     time.Now,
}, nil
```

Replace the `now := time.Now()` line at the top of `NotifyUpcomingEvents`:

```go
now := n.now()
```

- [ ] **Step 4: Set `NotificationTime` on the outgoing notification**

In the same `NotifyUpcomingEvents` method, replace the `domain.Notification` literal:

```go
notification := domain.Notification{
	UserID:           userID,
	Title:            n.formatUpcomingEventTitle(task, interval),
	Body:             n.formatUpcomingEventBody(task),
	Type:             "upcoming_event",
	NotificationTime: &notificationTime,
}
```

`notificationTime` is already in scope: `notificationTime := task.StartTime.Add(-interval)`.

- [ ] **Step 5: Run the new test — confirm it passes**

```bash
go test ./tasker/internal/services/notifications/... -run TestNotifyUpcomingEvents_SetsNotificationTimeAnchor -race -count=1
```

Expected: `PASS`.

- [ ] **Step 6: Run the full scenarios test suite**

```bash
go test ./tasker/internal/services/notifications/... -race -count=1
```

Expected: `ok` — all existing tests still pass.

- [ ] **Step 7: Run the full tasker suite**

```bash
go test ./tasker/... -race -count=1
```

Expected: `ok` everywhere.

- [ ] **Step 8: Lint**

From `backend/`:

```bash
make lint
```

Expected: no findings.

- [ ] **Step 9: Commit**

```bash
git add backend/tasker/internal/services/notifications/scenarios.go \
        backend/tasker/internal/services/notifications/scenarios_test.go
git commit -m "feat(tasker): propagate NotificationTime anchor from upcoming-event notifier"
```

---

## Done criteria

- `make lint` is clean.
- `go test ./tasker/... -race -count=1` passes under `backend/`.
- `TestNotificatorPushService_UsesNotificationTimeAsAnchor` confirms two ticks straddling a 5-minute boundary produce the same `idempotency_key` when `NotificationTime` is set.
- `TestNotifyUpcomingEvents_SetsNotificationTimeAnchor` confirms `NotifyUpcomingEvents` populates `NotificationTime` with the stable task-derived anchor.
- `scheduling/usecase.go` is unchanged; its schedule-change notification continues to use `time.Now()` via the nil-fallback path.
