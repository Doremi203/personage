package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func NewClusterActionabilityService(
	models ChatModelProvider,
	logger log.Logger,
	prompts PromptProvider,
) *clusterActionabilityService {
	return &clusterActionabilityService{
		models:  models,
		logger:  logger,
		prompts: prompts,
	}
}

type clusterActionabilityService struct {
	models  ChatModelProvider
	logger  log.Logger
	prompts PromptProvider
}

func (s *clusterActionabilityService) GetTaskGenerationDecision(
	ctx context.Context,
	events []domain.Event,
	profile domain.UserProfile,
) (domain.TaskGenerationDecision, error) {
	eventsText, err := formatEvents(events)
	if err != nil {
		return domain.TaskGenerationDecision{}, err
	}

	promptCfg, err := s.prompts.Get(ctx, domain.PromptIDClassifier)
	if err != nil {
		return domain.TaskGenerationDecision{}, errors.WrapFail(err, "load classifier prompt")
	}

	tpl := prompt.FromMessages(schema.GoTemplate,
		schema.SystemMessage(promptCfg.SystemTemplate),
		schema.UserMessage(promptCfg.UserTemplate),
	)

	messages, err := tpl.Format(ctx, map[string]any{
		"events":       eventsText,
		"owner_emails": formatOwnerEmails(profile),
		"user_name":    profile.Name,
	})
	if err != nil {
		return domain.TaskGenerationDecision{}, errors.WrapFail(err, "format messages for actionability llm")
	}

	chatModel, err := s.models.ChatModel(ctx)
	if err != nil {
		return domain.TaskGenerationDecision{}, errors.WrapFail(err, "resolve chat model for actionability llm")
	}

	return generateAndParseWithRetry(ctx, s.logger, chatModel, messages, userIDFromEvents(events), parseActionabilityResponse)
}

func parseActionabilityResponse(responseText string) (domain.TaskGenerationDecision, error) {
	jsonText := extractJSON(responseText)
	if jsonText == "" {
		return domain.TaskGenerationDecision{}, errors.Errorf("no valid JSON found in response %v", errors.Token("response", responseText))
	}

	var llmResp llmActionabilityResponse
	if err := json.Unmarshal([]byte(jsonText), &llmResp); err != nil {
		return domain.TaskGenerationDecision{}, errors.WrapFailf(err, "unmarshal actionability response: %s", jsonText)
	}

	llmResp.Reason = strings.TrimSpace(llmResp.Reason)
	if llmResp.Reason == "" {
		return domain.TaskGenerationDecision{}, errors.Errorf("reason is required for every classification decision")
	}

	reason := llmResp.Reason
	return domain.TaskGenerationDecision{
		ShouldGenerate: llmResp.Actionable,
		Reason:         &reason,
	}, nil
}

type llmActionabilityResponse struct {
	Actionable bool   `json:"actionable"`
	Reason     string `json:"reason"`
}

func formatOwnerEmails(profile domain.UserProfile) string {
	seen := make(map[string]struct{}, 1+len(profile.ConnectedEmails))
	emails := make([]string, 0, 1+len(profile.ConnectedEmails))
	add := func(raw string) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		emails = append(emails, trimmed)
	}
	add(profile.Email)
	for _, email := range profile.ConnectedEmails {
		add(email)
	}
	if len(emails) == 0 {
		return "(none)"
	}
	return strings.Join(emails, ", ")
}
