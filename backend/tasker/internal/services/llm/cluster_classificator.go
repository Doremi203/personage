package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var actionabilityTemplate = prompt.FromMessages(schema.GoTemplate,
	schema.SystemMessage(
		`You are a cluster actionability classifier for a fast task tracking app.

Decide whether the provided event cluster should create a user task.

Mark actionable=false for clusters that are informational only, spam, receipts, promos, newsletters, passive notifications, or weak signals without a clear next action.
Mark actionable=true only when the cluster implies a concrete user action.

Respond with valid JSON only:
{
  "actionable": true,
  "reason": "short explanation"
}`),
	schema.UserMessage("Events:\n{{.events}}"),
)

func NewClusterActionabilityService(
	model model.BaseChatModel,
	logger log.Logger,
) *clusterActionabilityService {
	return &clusterActionabilityService{
		model:  model,
		logger: logger,
	}
}

type clusterActionabilityService struct {
	model  model.BaseChatModel
	logger log.Logger
}

func (s *clusterActionabilityService) GetTaskGenerationDecision(
	ctx context.Context,
	events []domain.Event,
) (domain.TaskGenerationDecision, error) {
	eventsText, err := formatEvents(events)
	if err != nil {
		return domain.TaskGenerationDecision{}, err
	}

	messages, err := actionabilityTemplate.Format(ctx, map[string]any{"events": eventsText})
	if err != nil {
		return domain.TaskGenerationDecision{}, errors.WrapFail(err, "format messages for actionability llm")
	}

	return generateAndParseWithRetry(ctx, s.logger, s.model, messages, parseActionabilityResponse)
}

func parseActionabilityResponse(responseText string) (domain.TaskGenerationDecision, error) {
	jsonText := extractJSON(responseText)
	if jsonText == "" {
		return domain.TaskGenerationDecision{}, errors.Errorf("no valid JSON found in response: %s", responseText)
	}

	var llmResp llmActionabilityResponse
	if err := json.Unmarshal([]byte(jsonText), &llmResp); err != nil {
		return domain.TaskGenerationDecision{}, errors.WrapFailf(err, "unmarshal actionability response: %s", jsonText)
	}

	llmResp.Reason = strings.TrimSpace(llmResp.Reason)
	if !llmResp.Actionable && llmResp.Reason == "" {
		return domain.TaskGenerationDecision{}, errors.Errorf("reason is required for non-actionable clusters")
	}
	var optReason *string
	if llmResp.Reason != "" {
		optReason = &llmResp.Reason
	}

	return domain.TaskGenerationDecision{
		ShouldGenerate: llmResp.Actionable,
		Reason:         optReason,
	}, nil
}

type llmActionabilityResponse struct {
	Actionable bool   `json:"actionable"`
	Reason     string `json:"reason"`
}
