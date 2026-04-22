package llm

import (
	"testing"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/schema"
)

func TestParseActionabilityResponseRequiresReasonForNonActionable(t *testing.T) {
	response := `{"actionable": false, "reason": "   "}`

	if _, err := parseActionabilityResponse(response); err == nil {
		t.Fatal("parseActionabilityResponse succeeded without non-actionable reason")
	}
}

func TestGetTaskGenerationDecisionRetriesInvalidModelOutput(t *testing.T) {
	originalBackoff := llmRetryBaseBackoff
	originalAttempts := llmRetryMaxAttempts
	llmRetryBaseBackoff = 0
	llmRetryMaxAttempts = 2
	t.Cleanup(func() {
		llmRetryBaseBackoff = originalBackoff
		llmRetryMaxAttempts = originalAttempts
	})

	chatModel := &stubChatModel{results: []stubChatModelResult{
		{message: &schema.Message{Content: `{"actionable": false, "reason": "   "}`}},
		{message: &schema.Message{Content: `{"actionable": true, "reason": "work item"}`}},
	}}

	service := NewClusterActionabilityService(chatModel, log.Stub{})
	decision, err := service.GetTaskGenerationDecision(t.Context(), []domain.Event{{ID: "event-1", Context: "Please review PR #47."}})
	if err != nil {
		t.Fatalf("GetTaskGenerationDecision returned error: %v", err)
	}

	if chatModel.calls != 2 {
		t.Fatalf("expected 2 model calls, got %d", chatModel.calls)
	}

	if !decision.ShouldGenerate {
		t.Fatal("expected actionable decision")
	}
}
