package sqspush

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/google/uuid"
)

type rateLimiter interface {
	Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error)
}

func NewNotificationHandler(
	logger log.Logger,
	senderUseCase usecase.PushSender,
	subscriptionUseCase usecase.PushSubscription,
	notificationRepo notification.Repo,
	rateLimiter rateLimiter,
	retryInterval time.Duration,
	maxAge time.Duration,
) *notificationHandler {
	return &notificationHandler{
		logger:              logger,
		senderUseCase:       senderUseCase,
		subscriptionUseCase: subscriptionUseCase,
		notificationRepo:    notificationRepo,
		rateLimiter:         rateLimiter,
		retryInterval:       retryInterval,
		maxAge:              maxAge,
	}
}

type notificationHandler struct {
	senderUseCase       usecase.PushSender
	subscriptionUseCase usecase.PushSubscription
	notificationRepo    notification.Repo
	rateLimiter         rateLimiter
	retryInterval       time.Duration
	maxAge              time.Duration
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

	typ := notification.SettingType(data.GetType())
	allowed, err := p.rateLimiter.Allow(ctx, recipientUUID, typ)
	if err != nil {
		p.logger.Error(errors.WrapFailf(err, "check rate limit for recipient %v", errors.Token("id", recipientUUID)))
	}

	if !allowed {
		now := time.Now()
		retryAfter := now.Add(p.retryInterval)
		expiresAt := now.Add(p.maxAge)
		inserted, err := p.notificationRepo.CreateIfAbsent(ctx, notification.Notification{
			UserID:         recipientUUID,
			Title:          data.GetTitle(),
			Type:           data.GetType(),
			Text:           data.GetDetailedText(),
			Status:         notification.StatusPending,
			RetryAfter:     &retryAfter,
			ExpiresAt:      &expiresAt,
			IdempotencyKey: data.GetIdempotencyKey(),
			PushPayload: &notification.PushPayload{
				Body: data.GetBody(),
				Icon: data.GetIcon(),
				URL:  data.GetUrl(),
			},
		})
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

	now := time.Now()
	inserted, err := p.notificationRepo.CreateIfAbsent(ctx, notification.Notification{
		UserID:         recipientUUID,
		Title:          data.GetTitle(),
		Type:           data.GetType(),
		Text:           data.GetDetailedText(),
		Status:         notification.StatusSent,
		SentAt:         &now,
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
