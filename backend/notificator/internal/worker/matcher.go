package sqspush

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/Doremi203/personage/backend/notificator/internal/services/ratelimit"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/google/uuid"
)

func NewNotificationHandler(
	logger log.Logger,
	senderUseCase usecase.PushSender,
	subscriptionUseCase usecase.RecipientGetter,
	notificationRepo notification.Repo,
	rateLimiter ratelimit.Allower,
	retryInterval time.Duration,
	maxAge time.Duration,
	clock func() time.Time,
) *notificationHandler {
	return &notificationHandler{
		logger:              logger,
		senderUseCase:       senderUseCase,
		subscriptionUseCase: subscriptionUseCase,
		notificationRepo:    notificationRepo,
		rateLimiter:         rateLimiter,
		retryInterval:       retryInterval,
		maxAge:              maxAge,
		clock:               clock,
	}
}

type notificationHandler struct {
	senderUseCase       usecase.PushSender
	subscriptionUseCase usecase.RecipientGetter
	notificationRepo    notification.Repo
	rateLimiter         ratelimit.Allower
	retryInterval       time.Duration
	maxAge              time.Duration
	logger              log.Logger
	clock               func() time.Time
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

	typ := notification.SettingType(data.GetType())
	enabled, err := p.notificationRepo.IsSettingEnabled(ctx, recipientUUID, typ)
	if err != nil {
		return errors.WrapFailf(
			err,
			"check notification setting for recipient %v type %v",
			errors.Token("id", recipientUUID),
			errors.Token("type", string(typ)),
		)
	}
	if !enabled {
		p.logger.Infof(
			"notification type %v disabled for recipient %v, skipping",
			errors.Token("type", string(typ)),
			errors.Token("id", recipientUUID),
		)
		return nil
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

	now := p.clock()

	if len(pushRecipient.Subscriptions) == 0 {
		sent, err := notification.NewSent(
			recipientUUID,
			data.GetTitle(),
			data.GetType(),
			data.GetDetailedText(),
			now,
			data.GetIdempotencyKey(),
		)
		if err != nil {
			return errors.WrapFailf(err, "build notification for recipient %v", errors.Token("id", pushRecipientID))
		}
		inserted, err := p.notificationRepo.CreateIfAbsent(ctx, sent)
		if err != nil {
			return errors.WrapFailf(
				err,
				"persist in-app notification for recipient %v",
				errors.Token("id", pushRecipientID),
			)
		}
		if !inserted {
			p.logger.Infof(
				"duplicate notification skipped for recipient %v key %v",
				errors.Token("id", pushRecipientID),
				errors.Token("idempotency_key", data.GetIdempotencyKey()),
			)
		} else {
			p.logger.Infof(
				"no subscriptions for recipient %v, stored for in-app only",
				errors.Token("id", pushRecipientID),
			)
		}
		return nil
	}

	allowed, err := p.rateLimiter.Allow(ctx, recipientUUID, typ)
	if err != nil {
		p.logger.Error(errors.WrapFailf(err, "check rate limit for recipient %v", errors.Token("id", recipientUUID)))
	}

	if !allowed {
		pending, err := notification.NewPending(
			recipientUUID,
			data.GetTitle(),
			data.GetType(),
			data.GetDetailedText(),
			now,
			now.Add(p.retryInterval),
			now.Add(p.maxAge),
			&notification.PushPayload{
				Body: data.GetBody(),
				Icon: data.GetIcon(),
				URL:  data.GetUrl(),
			},
			data.GetIdempotencyKey(),
		)
		if err != nil {
			return errors.WrapFailf(err, "build pending notification for recipient %v", errors.Token("id", pushRecipientID))
		}
		inserted, err := p.notificationRepo.CreateIfAbsent(ctx, pending)
		if err != nil {
			return errors.WrapFailf(err, "persist pending notification for recipient %v", errors.Token("id", pushRecipientID))
		}
		if !inserted {
			p.logger.Infof(
				"duplicate notification skipped for recipient %v key %v",
				errors.Token("id", pushRecipientID),
				errors.Token("idempotency_key", data.GetIdempotencyKey()),
			)
		}
		return nil
	}

	sent, err := notification.NewSent(
		recipientUUID,
		data.GetTitle(),
		data.GetType(),
		data.GetDetailedText(),
		now,
		data.GetIdempotencyKey(),
	)
	if err != nil {
		return errors.WrapFailf(err, "build sent notification for recipient %v", errors.Token("id", pushRecipientID))
	}
	inserted, err := p.notificationRepo.CreateIfAbsent(ctx, sent)
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
