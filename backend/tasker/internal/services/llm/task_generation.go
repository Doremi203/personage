package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func NewTaskGenerationService(
	model model.BaseChatModel,
	logger log.Logger,
	prompts PromptProvider,
	defaultLocation *time.Location,
) *taskGenerationService {
	return &taskGenerationService{
		model:           model,
		logger:          logger,
		prompts:         prompts,
		defaultLocation: defaultLocation,
	}
}

type taskGenerationService struct {
	model           model.BaseChatModel
	logger          log.Logger
	prompts         PromptProvider
	defaultLocation *time.Location
}

func (s *taskGenerationService) GenerateTask(ctx context.Context, events []domain.Event) (domain.GeneratedTask, error) {
	eventsText, err := formatEvents(events)
	if err != nil {
		return domain.GeneratedTask{}, err
	}

	promptCfg, err := s.prompts.Get(ctx, domain.PromptIDTaskGenerator)
	if err != nil {
		return domain.GeneratedTask{}, errors.WrapFail(err, "load task generator prompt")
	}

	tpl := prompt.FromMessages(schema.GoTemplate,
		schema.SystemMessage(promptCfg.SystemTemplate),
		schema.UserMessage(promptCfg.UserTemplate),
	)

	messages, err := tpl.Format(ctx, map[string]any{
		"events": eventsText,
	})
	if err != nil {
		return domain.GeneratedTask{}, errors.WrapFail(err, "format messages for llm")
	}

	return generateAndParseWithRetry(ctx, s.logger, s.model, messages, userIDFromEvents(events), func(responseText string) (domain.GeneratedTask, error) {
		return s.parseResponse(responseText, events)
	})
}

type llmTaskResponse struct {
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	DurationMinutes int     `json:"duration_minutes"`
	Priority        int     `json:"priority"`
	Deadline        *string `json:"deadline"`
	StartTime       *string `json:"start_time"`
	Category        string  `json:"category"`
}

func (s *taskGenerationService) parseResponse(responseText string, _ []domain.Event) (domain.GeneratedTask, error) {
	jsonText := extractJSON(responseText)
	if jsonText == "" {
		return domain.GeneratedTask{}, errors.Errorf("no valid JSON found in response %v", errors.Token("response", responseText))
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

	task := domain.GeneratedTask{
		Title:           llmResp.Title,
		Description:     llmResp.Description,
		DurationMinutes: llmResp.DurationMinutes,
		Priority:        llmResp.Priority,
		Category:        category,
	}

	deadline, err := parseOptionalTimestamp("deadline", llmResp.Deadline, s.defaultLocation)
	if err != nil {
		return domain.GeneratedTask{}, err
	}
	task.Deadline = deadline

	startTime, err := parseOptionalTimestamp("start_time", llmResp.StartTime, s.defaultLocation)
	if err != nil {
		return domain.GeneratedTask{}, err
	}
	task.StartTime = roundToTimeSlot(startTime)

	return task, nil
}

func roundToTimeSlot(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	return new(t.Round(domain.TimeSlotSize))
}

func userIDFromEvents(events []domain.Event) string {
	if len(events) == 0 {
		return ""
	}
	return events[0].UserID.String()
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

func parseOptionalTimestamp(name string, value *string, loc *time.Location) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	trimmedValue := strings.TrimSpace(*value)
	if trimmedValue == "" || trimmedValue == "null" {
		return nil, nil
	}

	if hasExplicitZone(trimmedValue) {
		parsed, err := time.Parse(time.RFC3339, trimmedValue)
		if err != nil {
			return nil, errors.WrapFailf(err, "parse %s", name)
		}
		utc := parsed.UTC()
		return &utc, nil
	}

	parsed, err := time.ParseInLocation("2006-01-02T15:04:05", trimmedValue, loc)
	if err != nil {
		parsed, err = time.ParseInLocation("2006-01-02T15:04", trimmedValue, loc)
		if err != nil {
			return nil, errors.WrapFailf(err, "parse %s", name)
		}
	}
	utc := parsed.UTC()
	return &utc, nil
}

func hasExplicitZone(s string) bool {
	if strings.HasSuffix(s, "Z") {
		return true
	}
	if len(s) < 6 {
		return false
	}
	suffix := s[len(s)-6:]
	if (suffix[0] != '+' && suffix[0] != '-') || suffix[3] != ':' {
		return false
	}
	for _, idx := range []int{1, 2, 4, 5} {
		if suffix[idx] < '0' || suffix[idx] > '9' {
			return false
		}
	}
	return true
}

func validateCategory(category string) (string, error) {
	trimmedCategory := strings.TrimSpace(category)
	switch trimmedCategory {
	case string(domain.TaskCategoryWork), string(domain.TaskCategoryStudy), string(domain.TaskCategoryPersonal):
		return trimmedCategory, nil
	default:
		return "", errors.Errorf("invalid category %v", errors.Token("category", category))
	}
}

func extractJSON(text string) string {
	text = strings.TrimSpace(text)

	if rest, ok := strings.CutPrefix(text, "```json"); ok {
		text = strings.TrimPrefix(rest, "```")
		if idx := strings.Index(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	} else if rest, ok := strings.CutPrefix(text, "```"); ok {
		text = rest
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
