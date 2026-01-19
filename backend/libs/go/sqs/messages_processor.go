package sqs

import (
	"context"
	"time"

	"gitlab.com/amoguscorp/personage/backend/libs/go/errors"
	"gitlab.com/amoguscorp/personage/backend/libs/go/log"
)

type processor[T withID] interface {
	Process(ctx context.Context, data T) error
}

func NewMessageProcessor[T withID](
	ctx context.Context,
	logger log.Logger,
	cfg Config,
	processor processor[T],
	processingTimeout time.Duration,
	maxMessagesPerBatch int,
) (*MessageProcessor[T], error) {
	client, err := New[T](ctx, cfg)
	if err != nil {
		return nil, errors.WrapFail(err, "create sqs client")
	}
	return &MessageProcessor[T]{
		logger:              logger,
		sqsClient:           client,
		processor:           processor,
		processingTimeout:   processingTimeout,
		maxMessagesPerBatch: maxMessagesPerBatch,
	}, nil
}

type MessageProcessor[T withID] struct {
	sqsClient ClientReader[T]
	logger    log.Logger
	processor processor[T]

	processingTimeout   time.Duration
	maxMessagesPerBatch int
}

func (p *MessageProcessor[T]) ProcessMessages(ctx context.Context) error {
	batch, err := p.sqsClient.ReadMessages(ctx, p.logger, p.maxMessagesPerBatch)
	if err != nil {
		return errors.WrapFail(err, "read messages from sqs")
	}

	for _, msg := range batch {
		err := p.processMessage(ctx, msg)
		if err != nil {
			p.logger.Error(errors.WrapFailf(
				err,
				"process message %v",
				errors.Token("message_id", msg.ID),
			))
		}
	}

	return nil
}

func (p *MessageProcessor[T]) processMessage(
	ctx context.Context,
	msg Message[T],
) error {
	ctx, cancel := context.WithTimeout(ctx, p.processingTimeout)
	defer cancel()
	err := p.processor.Process(ctx, msg.Data)
	if err != nil {
		return errors.WrapFailf(
			err,
			"process message data %v",
			errors.Token("message_id", msg.ID),
		)
	}

	err = p.sqsClient.DeleteMessage(ctx, msg)
	if err != nil {
		return errors.WrapFailf(
			err,
			"delete message %v",
			errors.Token("message_id", msg.ID),
		)
	}

	return nil
}
