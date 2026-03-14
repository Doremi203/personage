package notifications

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/sqs"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

// NewNotificatorPushService creates a new push notification service that implements domain.NotificationsService
func NewNotificatorPushService(
	client sqs.ClientWriter[*pushpb.Notification],
) domain.NotificationsService {
	return &notificatorPushService{
		client: client,
	}
}

type notificatorPushService struct {
	client sqs.ClientWriter[*pushpb.Notification]
}

func (s *notificatorPushService) Send(
	ctx context.Context,
	notification domain.Notification,
) error {
	return s.client.SendMessage(ctx, &pushpb.Notification{
		RecipientId: notification.UserID.String(),
		Title:       notification.Title,
		Body:        notification.Body,
		Icon:        "/icon-72x72.png",
		Url:         "/",
	}, sqs.WithGroupID("tasker"))
}
