package llm

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type stubChatModelResult struct {
	message *schema.Message
	err     error
}

type stubChatModel struct {
	results []stubChatModelResult
	calls   int
}

func (s *stubChatModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	if s.calls >= len(s.results) {
		return nil, errors.Error("unexpected Generate call")
	}

	result := s.results[s.calls]
	s.calls++
	return result.message, result.err
}

func (*stubChatModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}
