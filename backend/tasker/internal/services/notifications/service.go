package notifications

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/sqs"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func NewNotificatorPushService(
	client sqs.ClientWriter[*pushpb.Notification],
	clock func() time.Time,
) domain.NotificationsService {
	return &notificatorPushService{
		client: client,
		now:    clock,
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
