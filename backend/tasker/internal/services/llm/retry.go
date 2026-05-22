package llm

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var (
	llmRequestTimeout   = 60 * time.Second
	llmRetryBaseBackoff = 500 * time.Millisecond
	llmRetryMaxAttempts = 3
)

func generateAndParseWithRetry[T any](
	ctx context.Context,
	logger log.Logger,
	chatModel model.BaseChatModel,
	messages []*schema.Message,
	userID string,
	parse func(responseText string) (T, error),
) (T, error) {
	var zero T
	lastErr := errors.Errorf("llm response was not produced")
	backoff := llmRetryBaseBackoff

	for attempt := range llmRetryMaxAttempts {
		var parsed T
		response, err := generateResponse(ctx, chatModel, messages)
		if err == nil {
			parsed, err = parse(response.Content)
			if err == nil {
				return parsed, nil
			}
		}

		lastErr = err
		logger.Warn(errors.Errorf(
			"llm generation of %T failed for user %v %v retrying %v",
			parsed,
			errors.Token("user_id", userID),
			errors.Token("err", lastErr.Error()),
			errors.Token("attempt", attempt),
		))
		if attempt == llmRetryMaxAttempts-1 {
			break
		}

		if err := waitForRetryBackoff(ctx, backoff); err != nil {
			return zero, err
		}

		backoff *= 2
	}

	return zero, errors.WrapFailf(
		lastErr,
		"obtain valid llm response for user %s after %d attempts",
		errors.Token("user_id", userID),
		llmRetryMaxAttempts,
	)
}

func generateResponse(
	ctx context.Context,
	chatModel model.BaseChatModel,
	messages []*schema.Message,
) (*schema.Message, error) {
	requestCtx, cancel := context.WithTimeout(ctx, llmRequestTimeout)
	defer cancel()

	response, err := chatModel.Generate(requestCtx, messages)
	if err != nil {
		return nil, errors.WrapFail(err, "generate llm response")
	}

	if response == nil {
		return nil, errors.Errorf("llm returned nil response")
	}

	return response, nil
}

func waitForRetryBackoff(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return errors.WrapFail(ctx.Err(), "wait for llm retry backoff")
	case <-timer.C:
		return nil
	}
}
