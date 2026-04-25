package pushsender

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/SherClockHolmes/webpush-go"
)

func New(
	webPushOptions *webpush.Options,
	pushRepo push.Repo,
	logger log.Logger,
) *Service {
	return &Service{
		webPushOptions: webPushOptions,
		pushRepo:       pushRepo,
		logger:         logger,
	}
}

type Service struct {
	webPushOptions *webpush.Options
	pushRepo       push.Repo
	logger         log.Logger
}

func (s *Service) Send(ctx context.Context, r push.Recipient, p push.Push) error {
	pushJson, err := json.Marshal(p)
	if err != nil {
		return errors.WrapFailf(
			err,
			"marshal %v",
			errors.Token("push", p),
		)
	}

	for _, sub := range r.Subscriptions {
		var body []byte
		resp, err := webpush.SendNotificationWithContext(ctx, pushJson, &webpush.Subscription{
			Endpoint: string(sub.Endpoint),
			Keys: webpush.Keys{
				Auth:   sub.Credentials.AuthKey,
				P256dh: sub.Credentials.P256dh,
			},
		}, s.webPushOptions)
		if err != nil {
			s.logger.Error(errors.WrapFailf(
				err,
				"send push to %v",
				errors.Token("endpoint", sub.Endpoint),
			))
		} else {
			body, err = io.ReadAll(resp.Body)
			if err != nil {
				s.logger.Error(errors.WrapFail(err, "read response body"))
			}
			err = resp.Body.Close()
			if err != nil {
				s.logger.Warn(errors.WrapFailf(err, "close response body"))
			}
		}

		switch resp.StatusCode {
		case 201:
		case 410, 404:
			err = s.pushRepo.DeleteSubscription(ctx, sub)
			if err != nil {
				s.logger.Error(errors.WrapFailf(
					err,
					"delete stale push subscription for %v",
					errors.Token("endpoint", sub.Endpoint),
				))
			}
		default:
			s.logger.Error(errors.Errorf(
				"got unexpected %v sending push for %v %v",
				errors.Token("status_code", resp.StatusCode),
				errors.Token("endpoint", sub.Endpoint),
				errors.Token("body", string(body)),
			))
		}
	}

	return nil
}
