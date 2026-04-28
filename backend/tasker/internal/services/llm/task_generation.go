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
		`You are an expert task extraction assistant for a fast personal task tracking app. The user is a Russian-speaking software engineer.

Analyze the provided events and produce ONE actionable task that captures the next concrete user action.

LANGUAGE
- All natural-language fields (title, description) MUST be written in Russian.
- Keep proper nouns, identifiers, brand names, repository names, PR/issue numbers, domain names, file names, and code symbols in their original form (e.g. "PR #47", "couply.ru", "Doremi203/personage", "Personage.Auth", "main.go"). Do NOT translate, transliterate, or quote them.
- JSON keys and enum values (category, "work"/"study"/"personal") stay in English exactly as specified.

TITLE (Russian)
- Imperative verb + concrete object. 4–10 words. No trailing punctuation, no quotes, no markdown.
- ALWAYS include the most distinctive identifier present in the events: PR/issue numbers, domain names, repository names, course titles, vendor or service names. Generic titles ("Проверить уведомления", "Поработать с PR") are unacceptable.
- Good examples:
  - "Проверить PR #47 — интеграция Tasker API"
  - "Продлить домены couply.ru и couply.online"
  - "Починить упавшие release-воркфлоу tasker, traitex и auth"
  - "Обновить уязвимые зависимости в репозиториях Doremi203"
  - "Оплатить минимальный платёж по карте Т-Банк"
  - "Откликнуться на вакансии Golang Backend в Microsoft, Wolt и GitHub"

DESCRIPTION (Russian)
- 1–2 sentences. First sentence: current state or trigger drawn from the events (what happened, by whom, when, which artifact). Second sentence (optional): concrete next action and, if non-obvious, why it matters now.
- Reuse exact identifiers, numbers, dates, and entity names from the events. Do not paraphrase them away.
- No markdown, no bullet lists, no greetings, no meta-commentary ("необходимо рассмотреть возможность"). Prefer concrete verbs ("проверить", "продлить", "обновить", "ответить", "оплатить").
- Do not repeat the title verbatim; the description must add information.

DURATION_MINUTES
- Realistic focused-work estimate. Anchors: triage/quick read 5–15; small fix or short PR review 15–30; substantial review or coding 30–60; complex investigation 60–120.

PRIORITY (1–10)
- 9–10: hard external deadline within days; security or production breakage; expired or expiring paid services; risk of money or data loss (expired domains, overdue bills, critical CVE on main).
- 7–8: explicit deadline this week; security advisories; broken CI on main; PR reviews that block teammates; recruiter follow-ups with stated dates.
- 5–6: routine active work without external pressure (own in-progress PRs, course homework, planned chores).
- 3–4: low-stakes useful items (cashback offers, optional connection requests, minor account settings).
- 1–2: pure FYI / nice-to-have (digests, newsletters, product changelog reading).
- Pick the priority that matches the strongest signal across the cluster. When the events are ambiguous, prefer the lower end of the matching band.

DEADLINE / START_TIME
- Set only when a date or datetime is explicitly stated in the events. Otherwise null.
- Format: RFC3339 in UTC (e.g. "2026-04-26T23:59:59Z"). If only a date is given, use 23:59:59Z for deadlines and 09:00:00Z for start_time.

CATEGORY
- "work": professional software engineering, employer/freelance projects, own product code, production incidents, work-related PRs and code reviews, recruiter outreach, job applications.
- "study": coursework, homework, online courses, lectures, certifications.
- "personal": banking, subscriptions, domains, bills, personal accounts, household, personal communication, non-study hobbies.

EVIDENCE_EVENT_IDS
- Cite the exact EVENT_ID values from the input that directly justify this task. Include every event that materially supports it. Do not invent IDs and do not include unrelated events.

Respond with valid JSON only — no prose before or after, no markdown fences:
{
  "title": "string (Russian)",
  "description": "string (Russian)",
  "duration_minutes": number,
  "priority": number,
  "deadline": "RFC3339 UTC or null",
  "start_time": "RFC3339 UTC or null",
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

		return nil, errors.Errorf(
			"evidence_event_ids reference unknown cluster events %v",
			errors.Token("invalid_ids", strings.Join(invalidIDStrings, ", ")),
		)
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
