package llm

import (
	"testing"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/schema"
)

func TestParseResponseIncludesNormalizedEvidenceEventIDs(t *testing.T) {
	service := &taskGenerationService{}
	events := []domain.Event{{ID: "event-1"}, {ID: "event-2"}, {ID: "event-3"}}

	response := `{
		"title": "Review PR #47",
		"description": "Review the proposed API integration changes.",
		"duration_minutes": 30,
		"priority": 7,
		"deadline": null,
		"start_time": null,
		"category": "work",
		"evidence_event_ids": [" event-3 ", "event-1", "event-1"]
	}`

	task, err := service.parseResponse(response, events)
	if err != nil {
		t.Fatalf("parseResponse returned error: %v", err)
	}

	if len(task.EvidenceEventIDs) != 2 {
		t.Fatalf("expected 2 evidence ids, got %d", len(task.EvidenceEventIDs))
	}

	if task.EvidenceEventIDs[0] != domain.EventID("event-1") || task.EvidenceEventIDs[1] != domain.EventID("event-3") {
		t.Fatalf("unexpected evidence ids: %#v", task.EvidenceEventIDs)
	}
}

func TestParseResponseRejectsMissingEvidenceEventIDs(t *testing.T) {
	service := &taskGenerationService{}

	response := `{
		"title": "Review PR #47",
		"description": "Review the proposed API integration changes.",
		"duration_minutes": 30,
		"priority": 7,
		"deadline": null,
		"start_time": null,
		"category": "work",
		"evidence_event_ids": []
	}`

	if _, err := service.parseResponse(response, []domain.Event{{ID: "event-1"}}); err == nil {
		t.Fatal("parseResponse succeeded without evidence_event_ids")
	}
}

func TestParseResponseRejectsUnknownEvidenceEventIDs(t *testing.T) {
	service := &taskGenerationService{}

	response := `{
		"title": "Review PR #47",
		"description": "Review the proposed API integration changes.",
		"duration_minutes": 30,
		"priority": 7,
		"deadline": null,
		"start_time": null,
		"category": "work",
		"evidence_event_ids": ["missing-event"]
	}`

	if _, err := service.parseResponse(response, []domain.Event{{ID: "event-1"}}); err == nil {
		t.Fatal("parseResponse succeeded with unknown evidence_event_ids")
	}
}

func TestGenerateTaskRetriesInvalidModelOutput(t *testing.T) {
	originalBackoff := llmRetryBaseBackoff
	originalAttempts := llmRetryMaxAttempts
	llmRetryBaseBackoff = 0
	llmRetryMaxAttempts = 2
	t.Cleanup(func() {
		llmRetryBaseBackoff = originalBackoff
		llmRetryMaxAttempts = originalAttempts
	})

	chatModel := &stubChatModel{results: []stubChatModelResult{
		{message: &schema.Message{Content: `{
			"title": "Review PR #47",
			"description": "Review the proposed API integration changes.",
			"duration_minutes": 30,
			"priority": 7,
			"deadline": null,
			"start_time": null,
			"category": "work",
			"evidence_event_ids": ["missing-event"]
		}`}},
		{message: &schema.Message{Content: `{
			"title": "Review PR #47",
			"description": "Review the proposed API integration changes.",
			"duration_minutes": 30,
			"priority": 7,
			"deadline": null,
			"start_time": null,
			"category": "work",
			"evidence_event_ids": ["event-1"]
		}`}},
	}}

	service := NewTaskGenerationService(chatModel)
	task, err := service.GenerateTask(t.Context(), []domain.Event{{ID: "event-1", Context: "Please review PR #47."}})
	if err != nil {
		t.Fatalf("GenerateTask returned error: %v", err)
	}

	if chatModel.calls != 2 {
		t.Fatalf("expected 2 model calls, got %d", chatModel.calls)
	}

	if got := len(task.EvidenceEventIDs); got != 1 || task.EvidenceEventIDs[0] != "event-1" {
		t.Fatalf("unexpected evidence ids: %#v", task.EvidenceEventIDs)
	}
}
