package retrier_test

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	mock_ratelimit "github.com/Doremi203/personage/backend/notificator/internal/services/ratelimit/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/services/retrier"
	mock_usecase "github.com/Doremi203/personage/backend/notificator/internal/usecase/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	retrierUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	retrierNotif  = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	retrierNow    = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

const retrierInterval = 15 * time.Minute

func notif(retryAfter, expiresAt *time.Time) notification.Notification {
	return notification.Notification{
		ID:          retrierNotif,
		UserID:      retrierUserID,
		Title:       "T",
		Type:        string(notification.SettingTypeScheduleChange),
		Status:      notification.StatusPending,
		PushPayload: &notification.PushPayload{Body: "b", Icon: "i", URL: "u"},
		RetryAfter:  retryAfter,
		ExpiresAt:   expiresAt,
	}
}

func recipientWithSub() push.Recipient {
	return push.Recipient{
		ID: push.RecipientID(retrierUserID),
		Subscriptions: []push.Subscription{{
			RecipientID: push.RecipientID(retrierUserID),
			Endpoint:    "https://example.com/push",
		}},
	}
}

func TestRetrier_ProcessOnce(t *testing.T) {
	expired := retrierNow.Add(-3 * time.Hour)
	pastRetry := retrierNow.Add(-time.Minute)
	futureRetry := retrierNow.Add(time.Minute)
	futureExpiry := retrierNow.Add(time.Hour)

	type mocks struct {
		repo          *mock_notification.MockRepo
		rateLimiter   *mock_ratelimit.MockAllower
		sender        *mock_usecase.MockPushSender
		subscriptions *mock_usecase.MockRecipientGetter
	}
	tests := []struct {
		name  string
		setup func(m mocks)
	}{
		{
			name: "list pending error logged and returns",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).Return(nil, assert.AnError)
			},
		},
		{
			name: "empty pending list is a no-op",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).Return(nil, nil)
			},
		},
		{
			name: "missing expiry skipped without further calls",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, nil)}, nil)
			},
		},
		{
			name: "expired notification is dropped",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &expired)}, nil)
				m.repo.EXPECT().Drop(gomock.Any(), retrierNotif).Return(nil)
			},
		},
		{
			name: "retry_after in the future is skipped",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&futureRetry, &futureExpiry)}, nil)
			},
		},
		{
			name: "setting disabled drops pending notification",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &futureExpiry)}, nil)
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(false, nil)
				m.repo.EXPECT().Drop(gomock.Any(), retrierNotif).Return(nil)
			},
		},
		{
			name: "setting check error skips entry",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &futureExpiry)}, nil)
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(false, assert.AnError)
			},
		},
		{
			name: "rate limit error skips entry",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &futureExpiry)}, nil)
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(false, assert.AnError)
			},
		},
		{
			name: "rate limited updates retry_after",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &futureExpiry)}, nil)
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(false, nil)
				m.repo.EXPECT().
					UpdateRetryAfter(gomock.Any(), retrierNotif, retrierNow.Add(retrierInterval)).
					Return(nil)
			},
		},
		{
			name: "get recipient error skips entry",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &futureExpiry)}, nil)
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(retrierUserID)).
					Return(push.Recipient{}, assert.AnError)
			},
		},
		{
			name: "no subscriptions causes drop",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &futureExpiry)}, nil)
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(retrierUserID)).
					Return(push.Recipient{ID: push.RecipientID(retrierUserID)}, nil)
				m.repo.EXPECT().Drop(gomock.Any(), retrierNotif).Return(nil)
			},
		},
		{
			name: "send failure updates retry_after and skips MarkSent",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &futureExpiry)}, nil)
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(retrierUserID)).
					Return(recipientWithSub(), nil)
				m.sender.EXPECT().
					Send(gomock.Any(), recipientWithSub(), push.Push{Title: "T", Body: "b", Icon: "i", Url: "u"}).
					Return(assert.AnError)
				m.repo.EXPECT().
					UpdateRetryAfter(gomock.Any(), retrierNotif, retrierNow.Add(retrierInterval)).
					Return(nil)
			},
		},
		{
			name: "successful send marks as sent",
			setup: func(m mocks) {
				m.repo.EXPECT().ListPending(gomock.Any()).
					Return([]notification.Notification{notif(&pastRetry, &futureExpiry)}, nil)
				m.repo.EXPECT().
					IsSettingEnabled(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.rateLimiter.EXPECT().
					Allow(gomock.Any(), retrierUserID, notification.SettingTypeScheduleChange).
					Return(true, nil)
				m.subscriptions.EXPECT().
					GetRecipient(gomock.Any(), push.RecipientID(retrierUserID)).
					Return(recipientWithSub(), nil)
				m.sender.EXPECT().
					Send(gomock.Any(), recipientWithSub(), push.Push{Title: "T", Body: "b", Icon: "i", Url: "u"}).
					Return(nil)
				m.repo.EXPECT().MarkSent(gomock.Any(), retrierNotif, retrierNow).Return(nil)
			},
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
			tt.setup(m)

			r := retrier.New(m.repo, m.rateLimiter, m.sender, m.subscriptions, retrierInterval, log.Stub{})
			r.ProcessOnce(t.Context(), retrierNow)
		})
	}
}
