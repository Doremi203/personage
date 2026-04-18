package sqspush

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/google/uuid"
)

func NewNotificationHandler(
	logger log.Logger,
	senderUseCase usecase.PushSender,
	subscriptionUseCase usecase.PushSubscription,
	notificationRepo notification.Repo,
) *notificationHandler {
	return &notificationHandler{
		logger:              logger,
		senderUseCase:       senderUseCase,
		subscriptionUseCase: subscriptionUseCase,
		notificationRepo:    notificationRepo,
	}
}

type notificationHandler struct {
	senderUseCase       usecase.PushSender
	subscriptionUseCase usecase.PushSubscription
	notificationRepo    notification.Repo
	logger              log.Logger
}

func (p *notificationHandler) Process(
	ctx context.Context,
	data *pushpb.Notification,
) error {
	recipientUUID, err := uuid.Parse(data.GetRecipientId())
	if err != nil {
		return errors.WrapFailf(
			err,
			"parse recipient id %v",
			errors.Token("recipient_id", data.GetRecipientId()),
		)
	}

	pushRecipientID := push.RecipientID(recipientUUID)

	pushRecipient, err := p.subscriptionUseCase.GetRecipient(ctx, pushRecipientID)
	if err != nil {
		return errors.WrapFailf(
			err,
			"get push recipient with id %v",
			errors.Token("id", pushRecipientID),
		)
	}
	if len(pushRecipient.Subscriptions) == 0 {
		p.logger.Infof(
			"no subscriptions for recipient %v, skipping push",
			errors.Token("id", pushRecipientID),
		)
		return nil
	}

	inserted, err := p.notificationRepo.CreateIfAbsent(ctx, notification.Notification{
		UserID:         recipientUUID,
		Title:          data.GetTitle(),
		Type:           data.GetType(),
		Text:           data.GetDetailedText(),
		IdempotencyKey: data.GetIdempotencyKey(),
	})
	if err != nil {
		return errors.WrapFailf(
			err,
			"persist notification for recipient %v",
			errors.Token("id", pushRecipientID),
		)
	}
	if !inserted {
		p.logger.Infof(
			"duplicate notification skipped for recipient %v key %v",
			errors.Token("id", pushRecipientID),
			errors.Token("idempotency_key", data.GetIdempotencyKey()),
		)
		return nil
	}

	err = p.senderUseCase.Send(ctx, pushRecipient, push.Push{
		Title: data.GetTitle(),
		Body:  data.GetBody(),
		Icon:  data.GetIcon(),
		Url:   data.GetUrl(),
	})
	if err != nil {
		return errors.WrapFailf(
			err,
			"send push to recipient with id %v",
			errors.Token("id", pushRecipientID),
		)
	}

	return nil
}
