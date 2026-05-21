package sqspush_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	mock_ratelimit "github.com/Doremi203/personage/backend/notificator/internal/services/ratelimit/mock"
	mock_usecase "github.com/Doremi203/personage/backend/notificator/internal/usecase/mock"
	sqspush "github.com/Doremi203/personage/backend/notificator/internal/worker"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	matcherUUID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	matcherNow  = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

const (
	matcherRetryInterval = 15 * time.Minute
	matcherMaxAge        = time.Hour
)

func newPushNotif() *pushpb.Notification {
	return &pushpb.Notification{
		RecipientId:    matcherUUID.String(),
		Title:          "Title",
		Type:           string(notification.SettingTypeScheduleChange),
		DetailedText:   "Detailed",
		Body:           "Body",
		Icon:           "Icon",
		Url:            "https://example.com",
		IdempotencyKey: "key-1",
	}
}

func recipientWith(subs ...push.Subscription) push.Recipient {
	return push.Recipient{
		ID:            push.RecipientID(matcherUUID),
		Subscriptions: subs,
	}
}

func TestNotificationHandler_Process(t *testing.T) {
	withSub := recipientWith(push.Subscription{
		RecipientID: push.RecipientID(matcherUUID),
		Endpoint:    "https://example.com/push",
	})
	expectedPush := push.Push{Title: "Title", Body: "Body", Icon: "Icon", Url: "https://example.com"}

	type mocks struct {
		repo          *mock_notification.MockRepo
		rateLimiter   *mock_ratelimit.MockAllower
		sender        *mock_usecase.MockPushSender
		subscriptions *mock_usecase.MockRecipientGetter
	}
	type args struct {
		data *pushpb.Notification
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "invalid recipient id",
			args:    args{data: &pushpb.Notification{RecipientId: "not-a-uuid"}},
			setup:   func(m mocks, a args) {},
			wantErr: require.Error,
		},
		{
			name: "setting disabled skips everything without persist",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(false, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "setting check error wraps",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(false, assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "get recipient error wraps",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(push.Recipient{}, assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "no subscriptions persists as sent for in-app",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(recipientWith(), nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.AssignableToTypeOf(notification.Notification{})).
					DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
						require.Equal(t, notification.StatusSent, n.Status)
						require.Equal(t, "key-1", n.IdempotencyKey)
						require.NotNil(t, n.SentAt)
						require.True(t, n.SentAt.Equal(matcherNow))
						return true, nil
					})
			},
			wantErr: require.NoError,
		},
		{
			name: "no subscriptions duplicate is logged and returns nil",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(recipientWith(), nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "no subscriptions persist error wraps",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(recipientWith(), nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.Any()).
					Return(false, assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "rate limited persists pending and returns nil",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(withSub, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(false, nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.AssignableToTypeOf(notification.Notification{})).
					DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
						require.Equal(t, notification.StatusPending, n.Status)
						require.Equal(t, "key-1", n.IdempotencyKey)
						require.NotNil(t, n.SentAt)
						require.True(t, n.SentAt.Equal(matcherNow))
						require.NotNil(t, n.RetryAfter)
						require.True(t, n.RetryAfter.Equal(matcherNow.Add(matcherRetryInterval)))
						require.NotNil(t, n.ExpiresAt)
						require.True(t, n.ExpiresAt.Equal(matcherNow.Add(matcherMaxAge)))
						return true, nil
					})
			},
			wantErr: require.NoError,
		},
		{
			name: "rate limited duplicate is logged and returns nil",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(withSub, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(false, nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "rate limited persist error wraps",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(withSub, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(false, nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.Any()).
					Return(false, assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "allowed sends and persists sent",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(withSub, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.AssignableToTypeOf(notification.Notification{})).
					DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
						require.Equal(t, notification.StatusSent, n.Status)
						require.NotNil(t, n.SentAt)
						require.True(t, n.SentAt.Equal(matcherNow))
						return true, nil
					})
				m.sender.EXPECT().Send(gomock.Any(), withSub, expectedPush).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "allowed duplicate skips send",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(withSub, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "allowed persist error wraps",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(withSub, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.Any()).
					Return(false, assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "allowed send error wraps",
			args: args{data: newPushNotif()},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(matcherUUID)).
					Return(withSub, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), matcherUUID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.repo.EXPECT().
					CreateIfAbsent(gomock.Any(), gomock.Any()).
					Return(true, nil)
				m.sender.EXPECT().Send(gomock.Any(), withSub, expectedPush).Return(assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				repo:          mock_notification.NewMockRepo(ctrl),
				rateLimiter:   mock_ratelimit.NewMockAllower(ctrl),
				sender:        mock_usecase.NewMockPushSender(ctrl),
				subscriptions: mock_usecase.NewMockRecipientGetter(ctrl),
			}
			tt.setup(m, tt.args)

			h := sqspush.NewNotificationHandler(
				log.Stub{},
				m.sender,
				m.subscriptions,
				m.repo,
				m.rateLimiter,
				matcherRetryInterval,
				matcherMaxAge,
				func() time.Time { return matcherNow },
			)

			err := h.Process(t.Context(), tt.args.data)
			tt.wantErr(t, err)
		})
	}
}
