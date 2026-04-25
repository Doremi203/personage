package notifications_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/sqs"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/Doremi203/personage/backend/tasker/internal/services/notifications"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSQSWriter struct {
	err         error
	gotMsg      *pushpb.Notification
	gotOptCount int
}

func (s *stubSQSWriter) SendMessage(_ context.Context, msg *pushpb.Notification, opts ...sqs.SendMessageOption) error {
	s.gotMsg = msg
	s.gotOptCount = len(opts)
	return s.err
}

func TestNotificatorPushService_Send(t *testing.T) {
	notifTime := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	tickEarly := notifTime.Add(-30 * time.Second)
	tickLate := notifTime.Add(30 * time.Second)
	fixedNow := time.Date(2026, 4, 18, 10, 2, 0, 0, time.UTC)
	userID := domain.UserID("user-1")
	const (
		notifType = "upcoming_event"
		title     = "title"
		body      = "body"
	)

	type mocks struct {
		writer *stubSQSWriter
	}
	type args struct {
		notification domain.Notification
		clock        func() time.Time
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		assert  func(t *testing.T, m mocks)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "NotificationTime set populates fields and idempotency key",
			args: args{
				notification: domain.Notification{
					UserID:           userID,
					Title:            title,
					Body:             body,
					Type:             notifType,
					NotificationTime: &notifTime,
				},
				clock: func() time.Time { return tickEarly },
			},
			setup: func(_ mocks, _ args) {},
			assert: func(t *testing.T, m mocks) {
				require.NotNil(t, m.writer.gotMsg)
				expectedKey := notifications.IdempotencyKey(userID.String(), notifTime, notifType, title)
				assert.Equal(t, userID.String(), m.writer.gotMsg.GetRecipientId())
				assert.Equal(t, title, m.writer.gotMsg.GetTitle())
				assert.Equal(t, body, m.writer.gotMsg.GetBody())
				assert.Equal(t, "/icon-72x72.png", m.writer.gotMsg.GetIcon())
				assert.Equal(t, "/", m.writer.gotMsg.GetUrl())
				assert.Equal(t, notifType, m.writer.gotMsg.GetType())
				assert.Equal(t, expectedKey, m.writer.gotMsg.GetIdempotencyKey())
				assert.Equal(t, 1, m.writer.gotOptCount, "expected single group-id option to be passed")
			},
			wantErr: require.NoError,
		},
		{
			name: "NotificationTime overrides clock so the key is bucket-stable",
			args: args{
				notification: domain.Notification{
					UserID:           userID,
					Title:            title,
					Type:             notifType,
					NotificationTime: &notifTime,
				},
				clock: func() time.Time { return tickLate },
			},
			setup: func(_ mocks, _ args) {},
			assert: func(t *testing.T, m mocks) {
				expectedKey := notifications.IdempotencyKey(userID.String(), notifTime, notifType, title)
				assert.Equal(t, expectedKey, m.writer.gotMsg.GetIdempotencyKey())
			},
			wantErr: require.NoError,
		},
		{
			name: "absent NotificationTime falls back to clock",
			args: args{
				notification: domain.Notification{
					UserID: userID,
					Title:  title,
					Type:   "schedule_change",
				},
				clock: func() time.Time { return fixedNow },
			},
			setup: func(_ mocks, _ args) {},
			assert: func(t *testing.T, m mocks) {
				expectedKey := notifications.IdempotencyKey(userID.String(), fixedNow, "schedule_change", title)
				assert.Equal(t, expectedKey, m.writer.gotMsg.GetIdempotencyKey())
			},
			wantErr: require.NoError,
		},
		{
			name: "SendMessage error returned as-is",
			args: args{
				notification: domain.Notification{
					UserID:           userID,
					Title:            title,
					Type:             notifType,
					NotificationTime: &notifTime,
				},
				clock: func() time.Time { return tickEarly },
			},
			setup: func(m mocks, _ args) {
				m.writer.err = assert.AnError
			},
			assert: func(_ *testing.T, _ mocks) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks{writer: &stubSQSWriter{}}
			tt.setup(m, tt.args)

			svc := notifications.NewNotificatorPushService(m.writer, tt.args.clock)
			err := svc.Send(t.Context(), tt.args.notification)

			tt.wantErr(t, err)
			tt.assert(t, m)
		})
	}
}
