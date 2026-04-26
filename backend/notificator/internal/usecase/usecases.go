package usecase

import (
	"context"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/google/uuid"
)

//go:generate mockgen -source=usecases.go -destination=mock/usecases_mock.go -typed

type PushSender interface {
	Send(ctx context.Context, r push.Recipient, p push.Push) error
}

type RecipientGetter interface {
	GetRecipient(ctx context.Context, id push.RecipientID) (push.Recipient, error)
}

type PushSubscription interface {
	Subscribe(ctx context.Context, subscription push.Subscription) error
	Unsubscribe(ctx context.Context, subscription push.Subscription) error
	GetRecipient(ctx context.Context, id push.RecipientID) (push.Recipient, error)
}

type ListNotificationsParams struct {
	UserID   uuid.UUID
	Page     int32
	PageSize int32
}

type Notifications interface {
	List(ctx context.Context, params ListNotificationsParams) ([]notification.Notification, error)
	GetSettings(ctx context.Context, userID uuid.UUID) ([]notification.Setting, error)
	Toggle(ctx context.Context, userID uuid.UUID, notificationType string) (notification.Setting, error)
	MarkAsRead(ctx context.Context, userID, notificationID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}
