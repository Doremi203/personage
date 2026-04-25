package pushsubscription

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
)

func New(pushRepo push.Repo) *Service {
	return &Service{pushRepo: pushRepo}
}

type Service struct {
	pushRepo push.Repo
}

func (s *Service) Subscribe(ctx context.Context, subscription push.Subscription) error {
	err := s.pushRepo.UpsertSubscription(ctx, subscription)
	if err != nil {
		return errors.WrapFailf(
			err,
			"upsert push subscription for recipient with %v",
			errors.Token("id", subscription.RecipientID),
		)
	}

	return nil
}

func (s *Service) Unsubscribe(ctx context.Context, subscription push.Subscription) error {
	err := s.pushRepo.DeleteSubscription(ctx, subscription)
	if err != nil {
		return errors.WrapFailf(
			err,
			"delete push subscription for recipient with %v",
			errors.Token("id", subscription.RecipientID),
		)
	}

	return nil
}

func (s *Service) GetRecipient(ctx context.Context, recipientID push.RecipientID) (push.Recipient, error) {
	subs, err := s.pushRepo.GetSubscriptionsByRecipientID(ctx, recipientID)
	if err != nil {
		return push.Recipient{}, errors.WrapFailf(
			err,
			"delete push subscription for recipient with %v",
			errors.Token("id", recipientID),
		)
	}

	return push.Recipient{
		ID:            recipientID,
		Subscriptions: subs,
	}, nil
}
