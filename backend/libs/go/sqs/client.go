package sqs

import (
	"context"
	"encoding/json"

	"github.com/Doremi203/couply/backend/auth/pkg/errors"
	"github.com/Doremi203/couply/backend/auth/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type ClientWriter[T any] interface {
	SendMessage(ctx context.Context, groupID string, data T) error
}

type ClientReader[T any] interface {
	ReadMessages(ctx context.Context, logger log.Logger, maxCount int) ([]Message[T], error)
	DeleteMessage(context.Context, Message[T]) error
}

type withID interface {
	GetId() string
}

func New[T withID](ctx context.Context, cfg Config) (*client[T], error) {
	clientAWSCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithBaseEndpoint(cfg.Endpoint),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, errors.WrapFail(err, "load aws config")
	}

	awsSQS := sqs.NewFromConfig(
		clientAWSCfg,
		sqs.WithEndpointResolverV2(sqs.NewDefaultEndpointResolverV2()),
	)

	return &client[T]{
		client: awsSQS,
		config: cfg,
	}, nil
}

type client[T withID] struct {
	client *sqs.Client
	config Config
}

func (c *client[T]) SendMessage(ctx context.Context, groupID string, data T) error {
	messageBody, err := json.Marshal(data)
	if err != nil {
		return errors.WrapFail(err, "marshal message")
	}

	input := &sqs.SendMessageInput{
		MessageGroupId:         aws.String(groupID),
		MessageDeduplicationId: aws.String(data.GetId()),
		MessageBody:            aws.String(string(messageBody)),
		QueueUrl:               aws.String(c.config.QueueURL),
	}

	_, err = c.client.SendMessage(ctx, input)
	if err != nil {
		return errors.WrapFailf(err, "send message to %v", errors.Token("queue_url", c.config.QueueURL))
	}

	return nil
}

func (c *client[T]) ReadMessages(ctx context.Context, logger log.Logger, maxCount int) ([]Message[T], error) {
	ret := make([]Message[T], 0, maxCount)

	resp, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.config.QueueURL),
		MaxNumberOfMessages: int32(maxCount),
	})
	if err != nil {
		return nil, errors.WrapFail(err, "recieve message from queue")
	}

	for _, msg := range resp.Messages {
		var body T
		err = json.Unmarshal([]byte(*msg.Body), &body)
		if err != nil {
			logger.Error(errors.WrapFailf(err, "unmarshal message body %v", errors.Token("message_body", *msg.Body)))
			continue
		}

		ret = append(ret, Message[T]{
			ID:            *msg.MessageId,
			ReceiptHandle: *msg.ReceiptHandle,
			Data:          body,
		})
	}

	return ret, nil
}

func (c *client[T]) DeleteMessage(ctx context.Context, msg Message[T]) error {
	_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.config.QueueURL),
		ReceiptHandle: aws.String(msg.ReceiptHandle),
	})
	if err != nil {
		return errors.WrapFailf(err, "delete message %v", errors.Token("message_id", msg.ID))
	}

	return nil
}

type Message[T any] struct {
	ID            string
	ReceiptHandle string
	Data          T
}
