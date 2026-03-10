package sqspush

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/google/uuid"
)

func NewNotificationHandler(
	logger log.Logger,
	senderUseCase usecase.PushSender,
	subscriptionUseCase usecase.PushSubscription,
) *notificationHandler {
	return &notificationHandler{
		logger:              logger,
		senderUseCase:       senderUseCase,
		subscriptionUseCase: subscriptionUseCase,
	}
}

type notificationHandler struct {
	senderUseCase       usecase.PushSender
	subscriptionUseCase usecase.PushSubscription
	logger              log.Logger
}

func (p *notificationHandler) Process(
	ctx context.Context,
	data *pushpb.Notification,
) error {
	idStr, err := uuid.Parse(data.GetRecipientId())
	if err != nil {
		return errors.WrapFailf(
			err,
			"parse recipient id %v",
			errors.Token("recipient_id", data.GetRecipientId()),
		)
	}

	pushRecipientID := push.RecipientID(idStr)

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
