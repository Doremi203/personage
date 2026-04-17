package notifications

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/sqs"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

// NewNotificatorPushService creates a new push notification service that implements domain.NotificationsService.
func NewNotificatorPushService(
	client sqs.ClientWriter[*pushpb.Notification],
) domain.NotificationsService {
	return &notificatorPushService{
		client: client,
		now:    time.Now,
	}
}

type notificatorPushService struct {
	client sqs.ClientWriter[*pushpb.Notification]
	now    func() time.Time
}

func (s *notificatorPushService) Send(
	ctx context.Context,
	notification domain.Notification,
) error {
	userID := notification.UserID.String()
	return s.client.SendMessage(ctx, &pushpb.Notification{
		RecipientId:    userID,
		Title:          notification.Title,
		Body:           notification.Body,
		Icon:           "/icon-72x72.png",
		Url:            "/",
		Type:           notification.Type,
		IdempotencyKey: IdempotencyKey(userID, s.now(), notification.Type, notification.Title),
	}, sqs.WithGroupID("tasker"))
}
