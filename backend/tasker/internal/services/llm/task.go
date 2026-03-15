package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
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
- Title: Short, clear action (e.g., "Review Q4 budget proposal", "Book dentist appointment")
- Description: Brief 1-2 sentence summary of what needs to be done and why (NOT detailed context)
- Duration: Realistic time estimate in minutes
- Priority: 1-10 scale (1=low, 10=urgent/critical)
- Deadline/StartTime: Extract from events if explicitly mentioned, otherwise null
- Category: Сhoose the most suitable one: work, study, personal

Keep it concise - this is for quick task tracking, not project management.

Respond with valid JSON only:
{
  "title": "string",
  "description": "string",
  "duration_minutes": number,
  "priority": number,
  "deadline": "ISO8601 or null",
  "start_time": "ISO8601 or null",
  "category": "work|study|personal"
}`),
	schema.UserMessage("Events:\n{{.events}}"),
)

func NewTaskGenerationService(model model.BaseChatModel) *taskGenerationService {
	return &taskGenerationService{model: model}
}

type taskGenerationService struct {
	model model.BaseChatModel
}

func (s *taskGenerationService) GenerateTask(ctx context.Context, events []domain.Event) (domain.GeneratedTask, error) {
	eventsText, err := s.formatEvents(events)
	if err != nil {
		return domain.GeneratedTask{}, err
	}

	messages, err := taskGenerationTemplate.Format(ctx, map[string]any{
		"events": eventsText,
	})
	if err != nil {
		return domain.GeneratedTask{}, errors.WrapFail(err, "format messages for llm")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := s.model.Generate(ctx, messages)
	if err != nil {
		return domain.GeneratedTask{}, errors.WrapFail(err, "generate llm response")
	}

	return s.parseResponse(response.Content)
}

func (s *taskGenerationService) formatEvents(events []domain.Event) (string, error) {
	var builder strings.Builder

	for i, event := range events {
		_, err := fmt.Fprintf(&builder, "--- Event %d ---\n%s\n", i+1, event.Context)
		if err != nil {
			return "", errors.WrapFail(err, "format event")
		}
	}

	return builder.String(), nil
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

func (s *taskGenerationService) parseResponse(responseText string) (domain.GeneratedTask, error) {
	jsonText := extractJSON(responseText)
	if jsonText == "" {
		return domain.GeneratedTask{}, errors.Errorf("no valid JSON found in response: %s", responseText)
	}

	var llmResp llmTaskResponse
	if err := json.Unmarshal([]byte(jsonText), &llmResp); err != nil {
		return domain.GeneratedTask{}, errors.WrapFailf(err, "unmarshal llm response: %s", jsonText)
	}

	task := domain.GeneratedTask{
		Title:           llmResp.Title,
		Description:     llmResp.Description,
		DurationMinutes: llmResp.DurationMinutes,
		Priority:        llmResp.Priority,
		Category:        llmResp.Category,
	}

	if llmResp.Deadline != nil && *llmResp.Deadline != "" && *llmResp.Deadline != "null" {
		deadline, err := time.Parse(time.RFC3339, *llmResp.Deadline)
		if err == nil {
			task.Deadline = &deadline
		}
	}

	if llmResp.StartTime != nil && *llmResp.StartTime != "" && *llmResp.StartTime != "null" {
		startTime, err := time.Parse(time.RFC3339, *llmResp.StartTime)
		if err == nil {
			task.StartTime = &startTime
		}
	}

	return task, nil
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
