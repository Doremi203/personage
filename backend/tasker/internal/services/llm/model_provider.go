package llm

import (
	"cmp"
	"context"
	"strings"
	"sync"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/components/model"
)

type ChatModelProvider interface {
	ChatModel(ctx context.Context) (model.BaseChatModel, error)
}

// ChatModelFactory builds a chat model bound to a specific model name. It performs
// no network I/O — the OpenRouter client is constructed lazily, so rebuilding on a
// model change is cheap.
type ChatModelFactory func(ctx context.Context, modelName string) (model.BaseChatModel, error)

func NewSettingsChatModelProvider(
	settings domain.GenerationSettingsProvider,
	factory ChatModelFactory,
	fallbackModel string,
) *settingsChatModelProvider {
	return &settingsChatModelProvider{
		settings:      settings,
		factory:       factory,
		fallbackModel: fallbackModel,
	}
}

type settingsChatModelProvider struct {
	settings      domain.GenerationSettingsProvider
	factory       ChatModelFactory
	fallbackModel string

	mu           sync.Mutex
	current      model.BaseChatModel
	currentModel string
}

func (p *settingsChatModelProvider) ChatModel(ctx context.Context) (model.BaseChatModel, error) {
	settings, err := p.settings.GenerationSettings(ctx)
	if err != nil {
		return nil, errors.WrapFail(err, "load generation settings for llm model")
	}

	name := cmp.Or(strings.TrimSpace(settings.LLMModel), p.fallbackModel)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.current != nil && p.currentModel == name {
		return p.current, nil
	}

	chatModel, err := p.factory(ctx, name)
	if err != nil {
		return nil, errors.WrapFailf(err, "build chat model %s", errors.Token("model", name))
	}

	p.current = chatModel
	p.currentModel = name
	return chatModel, nil
}

func newStaticChatModelProvider(chatModel model.BaseChatModel) staticChatModelProvider {
	return staticChatModelProvider{chatModel: chatModel}
}

type staticChatModelProvider struct {
	chatModel model.BaseChatModel
}

func (p staticChatModelProvider) ChatModel(context.Context) (model.BaseChatModel, error) {
	return p.chatModel, nil
}
