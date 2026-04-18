package retrier

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/google/uuid"
)

type rateLimiter interface {
	Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error)
}

type pushSender interface {
	Send(ctx context.Context, r push.Recipient, p push.Push) error
}

type subscriptionGetter interface {
	GetRecipient(ctx context.Context, id push.RecipientID) (push.Recipient, error)
}

type Retrier struct {
	repo          notification.Repo
	rateLimiter   rateLimiter
	sender        pushSender
	subscriptions subscriptionGetter
	retryInterval time.Duration
	logger        log.Logger
}

func New(
	repo notification.Repo,
	rateLimiter rateLimiter,
	sender pushSender,
	subscriptions subscriptionGetter,
	retryInterval time.Duration,
) *Retrier {
	return &Retrier{
		repo:          repo,
		rateLimiter:   rateLimiter,
		sender:        sender,
		subscriptions: subscriptions,
		retryInterval: retryInterval,
	}
}

func NewWithLogger(
	repo notification.Repo,
	rateLimiter rateLimiter,
	sender pushSender,
	subscriptions subscriptionGetter,
	retryInterval time.Duration,
	logger log.Logger,
) *Retrier {
	r := New(repo, rateLimiter, sender, subscriptions, retryInterval)
	r.logger = logger
	return r
}

func (r *Retrier) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			r.ProcessOnce(ctx, t)
		}
	}
}

func (r *Retrier) ProcessOnce(ctx context.Context, now time.Time) {
	pending, err := r.repo.ListPending(ctx)
	if err != nil {
		r.logError(err, "list pending notifications")
		return
	}

	for _, n := range pending {
		r.process(ctx, n, now)
	}
}

func (r *Retrier) process(ctx context.Context, n notification.Notification, now time.Time) {
	if n.ExpiresAt != nil && now.After(*n.ExpiresAt) {
		if err := r.repo.Drop(ctx, n.ID); err != nil {
			r.logError(err, "drop expired notification")
		}
		return
	}

	if n.RetryAfter != nil && now.Before(*n.RetryAfter) {
		return
	}

	allowed, err := r.rateLimiter.Allow(ctx, n.UserID, notification.SettingType(n.Type))
	if err != nil {
		r.logError(err, "check rate limit for pending notification")
		return
	}

	if !allowed {
		if err := r.repo.UpdateRetryAfter(ctx, n.ID, now.Add(r.retryInterval)); err != nil {
			r.logError(err, "update retry_after for pending notification")
		}
		return
	}

	recipient, err := r.subscriptions.GetRecipient(ctx, push.RecipientID(n.UserID))
	if err != nil {
		r.logError(err, "get recipient for pending notification")
		return
	}

	if len(recipient.Subscriptions) == 0 {
		if err := r.repo.Drop(ctx, n.ID); err != nil {
			r.logError(err, "drop notification with no subscriptions")
		}
		return
	}

	p := push.Push{Title: n.Title}
	if n.PushPayload != nil {
		p.Body = n.PushPayload.Body
		p.Icon = n.PushPayload.Icon
		p.Url = n.PushPayload.URL
	}

	if err := r.sender.Send(ctx, recipient, p); err != nil {
		if updateErr := r.repo.UpdateRetryAfter(ctx, n.ID, now.Add(r.retryInterval)); updateErr != nil {
			r.logError(updateErr, "update retry_after after send failure")
		}
		r.logError(err, "send push for pending notification")
		return
	}

	if err := r.repo.MarkSent(ctx, n.ID, now); err != nil {
		r.logError(err, "mark notification sent")
	}
}

func (r *Retrier) logError(err error, _ string) {
	if r.logger != nil {
		r.logger.Error(err)
	}
}
