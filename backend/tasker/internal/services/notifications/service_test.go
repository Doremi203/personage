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
	assert.Equal(t, writer.messages[0].GetIdempotencyKey(), writer.messages[1].GetIdempotencyKey(),
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
	assert.Equal(t, expected, writer.messages[0].GetIdempotencyKey())
}
