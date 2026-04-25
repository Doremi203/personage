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
	"github.com/google/uuid"
)

//go:generate mockgen -source=matcher.go -destination=mock/matcher_mock.go -typed

type Sender interface {
	Send(ctx context.Context, r push.Recipient, p push.Push) error
}

type SubscriptionGetter interface {
	GetRecipient(ctx context.Context, id push.RecipientID) (push.Recipient, error)
}

func NewNotificationHandler(
	logger log.Logger,
	senderUseCase Sender,
	subscriptionUseCase SubscriptionGetter,
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
	senderUseCase       Sender
	subscriptionUseCase SubscriptionGetter
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
		now := p.clock()
		pending, err := notification.NewPending(
			recipientUUID,
			data.GetTitle(),
			data.GetType(),
			data.GetDetailedText(),
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
		p.clock(),
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
