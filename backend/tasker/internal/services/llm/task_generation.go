package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var taskGenerationTemplate = prompt.FromMessages(schema.GoTemplate,
	schema.SystemMessage(
		`You are a task extraction assistant for a fast task tracking app.

Analyze the provided events and create ONE actionable task that captures the main action needed.

Guidelines:
- Title: Short, clear action verb + specific object. Include specific identifiers when present: PR numbers (e.g. "Review PR #47"), domain names, service names, repository names, course names. Examples: "Review PR #47 tasker API integration", "Renew couply.ru and couply.online domains", "Fix CI/CD release workflows"
- Description: Brief 1-2 sentence summary of what needs to be done and why (NOT detailed context)
- Duration: Realistic time estimate in minutes based on task complexity
- Priority: Base on urgency signals — deadlines, security vulnerabilities, production impact → 8-10; active work items and PR reviews → 6-8; routine tasks → 3-6; nice-to-haves → 1-2
- Deadline/StartTime: Extract from events if explicitly mentioned, otherwise null
- Category: "work" for professional/coding/career tasks; "study" for coursework, homework, educational content, online courses; "personal" for everything else (banking, personal accounts, subscriptions, personal errands)
- Evidence: cite supporting events by their exact EVENT_ID values from the input. Use only IDs that directly justify the task. Do not invent IDs.

Keep it concise - this is for quick task tracking, not project management.

Respond with valid JSON only:
{
  "title": "string",
  "description": "string",
  "duration_minutes": number,
  "priority": number,
  "deadline": "ISO8601 or null",
  "start_time": "ISO8601 or null",
  "category": "work|study|personal",
  "evidence_event_ids": ["uuid"]
}`),
	schema.UserMessage("Events:\n{{.events}}"),
)

func NewTaskGenerationService(
	model model.BaseChatModel,
	logger log.Logger,
) *taskGenerationService {
	return &taskGenerationService{
		model:  model,
		logger: logger,
	}
}

type taskGenerationService struct {
	model  model.BaseChatModel
	logger log.Logger
}

func (s *taskGenerationService) GenerateTask(ctx context.Context, events []domain.Event) (domain.GeneratedTask, error) {
	eventsText, err := formatEvents(events)
	if err != nil {
		return domain.GeneratedTask{}, err
	}

	messages, err := taskGenerationTemplate.Format(ctx, map[string]any{
		"events": eventsText,
	})
	if err != nil {
		return domain.GeneratedTask{}, errors.WrapFail(err, "format messages for llm")
	}

	return generateAndParseWithRetry(ctx, s.logger, s.model, messages, func(responseText string) (domain.GeneratedTask, error) {
		return s.parseResponse(responseText, events)
	})
}

type llmTaskResponse struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	DurationMinutes  int      `json:"duration_minutes"`
	Priority         int      `json:"priority"`
	Deadline         *string  `json:"deadline"`
	StartTime        *string  `json:"start_time"`
	Category         string   `json:"category"`
	EvidenceEventIDs []string `json:"evidence_event_ids"`
}

func (s *taskGenerationService) parseResponse(responseText string, events []domain.Event) (domain.GeneratedTask, error) {
	jsonText := extractJSON(responseText)
	if jsonText == "" {
		return domain.GeneratedTask{}, errors.Errorf("no valid JSON found in response: %s", responseText)
	}

	var llmResp llmTaskResponse
	if err := json.Unmarshal([]byte(jsonText), &llmResp); err != nil {
		return domain.GeneratedTask{}, errors.WrapFailf(err, "unmarshal llm response: %s", jsonText)
	}

	category, err := validateCategory(llmResp.Category)
	if err != nil {
		return domain.GeneratedTask{}, err
	}

	if llmResp.DurationMinutes <= 0 {
		return domain.GeneratedTask{}, errors.Errorf("duration_minutes must be > 0")
	}

	if llmResp.Priority < 1 || llmResp.Priority > 10 {
		return domain.GeneratedTask{}, errors.Errorf("priority must be between 1 and 10")
	}

	evidenceEventIDs, err := parseEvidenceEventIDs(llmResp.EvidenceEventIDs, events)
	if err != nil {
		return domain.GeneratedTask{}, err
	}

	task := domain.GeneratedTask{
		Title:            llmResp.Title,
		Description:      llmResp.Description,
		DurationMinutes:  llmResp.DurationMinutes,
		Priority:         llmResp.Priority,
		Category:         category,
		EvidenceEventIDs: evidenceEventIDs,
	}

	deadline, err := parseOptionalTimestamp("deadline", llmResp.Deadline)
	if err != nil {
		return domain.GeneratedTask{}, err
	}
	task.Deadline = deadline

	startTime, err := parseOptionalTimestamp("start_time", llmResp.StartTime)
	if err != nil {
		return domain.GeneratedTask{}, err
	}
	task.StartTime = startTime

	return task, nil
}

func formatEvents(events []domain.Event) (string, error) {
	var builder strings.Builder

	for i, event := range events {
		_, err := fmt.Fprintf(
			&builder,
			"--- Event %d ---\nEVENT_ID: %s\n%s\n",
			i+1,
			event.ID,
			event.Context,
		)
		if err != nil {
			return "", errors.WrapFail(err, "format event")
		}
	}

	return builder.String(), nil
}

func parseEvidenceEventIDs(ids []string, events []domain.Event) ([]domain.EventID, error) {
	if len(ids) == 0 {
		return nil, errors.Errorf("evidence_event_ids must contain at least one event id")
	}

	requested := make(map[domain.EventID]struct{}, len(ids))
	for _, id := range ids {
		trimmedID := strings.TrimSpace(id)
		if trimmedID == "" {
			return nil, errors.Errorf("evidence_event_ids must not contain blank values")
		}

		requested[domain.EventID(trimmedID)] = struct{}{}
	}

	validated := make([]domain.EventID, 0, len(requested))
	for _, event := range events {
		if _, ok := requested[event.ID]; !ok {
			continue
		}

		validated = append(validated, event.ID)
		delete(requested, event.ID)
	}

	if len(requested) > 0 {
		invalidIDs := slices.Sorted(maps.Keys(requested))
		invalidIDStrings := make([]string, len(invalidIDs))
		for i, id := range invalidIDs {
			invalidIDStrings[i] = id.String()
		}

		return nil, errors.Errorf("evidence_event_ids reference unknown cluster events: %s", strings.Join(invalidIDStrings, ", "))
	}

	return validated, nil
}

func parseOptionalTimestamp(name string, value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	trimmedValue := strings.TrimSpace(*value)
	if trimmedValue == "" || trimmedValue == "null" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmedValue)
	if err != nil {
		return nil, errors.WrapFailf(err, "parse %s", name)
	}

	return &parsed, nil
}

func validateCategory(category string) (string, error) {
	trimmedCategory := strings.TrimSpace(category)
	switch trimmedCategory {
	case string(domain.TaskCategoryWork), string(domain.TaskCategoryStudy), string(domain.TaskCategoryPersonal):
		return trimmedCategory, nil
	default:
		return "", errors.Errorf("invalid category: %s", category)
	}
}

func extractJSON(text string) string {
	text = strings.TrimSpace(text)

	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		if idx := strings.Index(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		if idx := strings.Index(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start > end {
		return ""
	}

	return text[start : end+1]
}
