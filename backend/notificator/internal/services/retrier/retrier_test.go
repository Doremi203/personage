package retrier_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/Doremi203/personage/backend/notificator/internal/services/retrier"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	userID   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	notifID  = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	now      = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	retryInt = 15 * time.Minute
)

// mockRateLimiter is a simple test double — no gomock needed.
type mockRateLimiter struct{ allow bool }

func (m *mockRateLimiter) Allow(_ context.Context, _ uuid.UUID, _ notification.SettingType) (bool, error) {
	return m.allow, nil
}

// mockSender records the last Send call.
type mockSender struct{ called bool }

func (m *mockSender) Send(_ context.Context, _ push.Recipient, _ push.Push) error {
	m.called = true
	return nil
}

// mockSubscriptions always returns a single-subscription recipient.
type mockSubscriptions struct{}

func (m *mockSubscriptions) GetRecipient(_ context.Context, id push.RecipientID) (push.Recipient, error) {
	return push.Recipient{
		ID: id,
		Subscriptions: []push.Subscription{{
			RecipientID: id,
			Endpoint:    "https://example.com/push",
		}},
	}, nil
}

type errorSubscriptions struct{}

func (m *errorSubscriptions) GetRecipient(_ context.Context, id push.RecipientID) (push.Recipient, error) {
	return push.Recipient{}, assert.AnError
}

func expiredNotif() notification.Notification {
	expired := now.Add(-3 * time.Hour)
	retryT := now.Add(-time.Minute)
	return notification.Notification{
		ID:          notifID,
		UserID:      userID,
		Type:        string(notification.SettingTypeScheduleChange),
		Status:      notification.StatusPending,
		PushPayload: &notification.PushPayload{Body: "b", Icon: "i", URL: "u"},
		RetryAfter:  &retryT,
		ExpiresAt:   &expired,
	}
}

func readyNotif() notification.Notification {
	expires := now.Add(time.Hour)
	retryT := now.Add(-time.Minute)
	return notification.Notification{
		ID:          notifID,
		UserID:      userID,
		Type:        string(notification.SettingTypeScheduleChange),
		Status:      notification.StatusPending,
		PushPayload: &notification.PushPayload{Body: "b", Icon: "i", URL: "u"},
		RetryAfter:  &retryT,
		ExpiresAt:   &expires,
	}
}

func notReadyNotif() notification.Notification {
	expires := now.Add(time.Hour)
	retryT := now.Add(time.Minute) // retry in the future
	return notification.Notification{
		ID:          notifID,
		UserID:      userID,
		Type:        string(notification.SettingTypeScheduleChange),
		Status:      notification.StatusPending,
		PushPayload: &notification.PushPayload{Body: "b", Icon: "i", URL: "u"},
		RetryAfter:  &retryT,
		ExpiresAt:   &expires,
	}
}

func TestRetrier_ProcessOnce_ExpiredNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{expiredNotif()}, nil)
	repo.EXPECT().Drop(gomock.Any(), notifID).Return(nil)

	r := retrier.New(repo, &mockRateLimiter{allow: true}, &mockSender{}, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}

func TestRetrier_ProcessOnce_RateLimited(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{readyNotif()}, nil)
	repo.EXPECT().UpdateRetryAfter(gomock.Any(), notifID, now.Add(retryInt)).Return(nil)

	r := retrier.New(repo, &mockRateLimiter{allow: false}, &mockSender{}, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}

func TestRetrier_ProcessOnce_SendsWhenAllowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	sender := &mockSender{}

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{readyNotif()}, nil)
	repo.EXPECT().MarkSent(gomock.Any(), notifID, now).Return(nil)

	r := retrier.New(repo, &mockRateLimiter{allow: true}, sender, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)

	assert.True(t, sender.called)
}

func TestRetrier_ProcessOnce_NotYetReady(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{notReadyNotif()}, nil)
	// No Drop, MarkSent, or UpdateRetryAfter — notification is not due yet.

	r := retrier.New(repo, &mockRateLimiter{allow: true}, &mockSender{}, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}

func TestRetrier_ProcessOnce_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return(nil, nil)

	r := retrier.New(repo, &mockRateLimiter{allow: true}, &mockSender{}, &mockSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}

func TestRetrier_ProcessOnce_GetRecipientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)

	repo.EXPECT().ListPending(gomock.Any()).Return([]notification.Notification{readyNotif()}, nil)
	// No Drop, MarkSent, or UpdateRetryAfter — error getting recipient should just log and skip

	r := retrier.New(repo, &mockRateLimiter{allow: true}, &mockSender{}, &errorSubscriptions{}, retryInt)
	r.ProcessOnce(context.Background(), now)
}
