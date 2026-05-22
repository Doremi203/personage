package llm

import (
	"strings"
	"testing"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/schema"
)

func TestParseActionabilityResponseRequiresReasonRegardlessOfDecision(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{name: "missing reason on reject", response: `{"actionable": false, "reason": "   "}`},
		{name: "missing reason on accept", response: `{"actionable": true, "reason": ""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseActionabilityResponse(tt.response); err == nil {
				t.Fatal("parseActionabilityResponse succeeded without reason")
			}
		})
	}
}

func TestParseActionabilityResponseAcceptsBothBranchesWithReason(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		actionable bool
		reason     string
	}{
		{name: "reject with reason", response: `{"actionable": false, "reason": "group broadcast"}`, actionable: false, reason: "group broadcast"},
		{name: "accept with reason", response: `{"actionable": true, "reason": "direct DM with ask"}`, actionable: true, reason: "direct DM with ask"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := parseActionabilityResponse(tt.response)
			if err != nil {
				t.Fatalf("parseActionabilityResponse returned error: %v", err)
			}
			if decision.ShouldGenerate != tt.actionable {
				t.Fatalf("ShouldGenerate=%v want %v", decision.ShouldGenerate, tt.actionable)
			}
			if decision.Reason == nil || *decision.Reason != tt.reason {
				t.Fatalf("Reason=%v want %s", decision.Reason, tt.reason)
			}
		})
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
	decision, err := service.GetTaskGenerationDecision(
		t.Context(),
		[]domain.Event{{ID: "event-1", Context: "Please review PR #47."}},
		domain.UserProfile{},
	)
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

func TestGetTaskGenerationDecisionInjectsOwnerIdentity(t *testing.T) {
	chatModel := &stubChatModel{results: []stubChatModelResult{
		{message: &schema.Message{Content: `{"actionable": true, "reason": "direct DM with ask"}`}},
	}}

	service := NewClusterActionabilityService(chatModel, log.Stub{})
	_, err := service.GetTaskGenerationDecision(
		t.Context(),
		[]domain.Event{{ID: "event-1", Context: "Hi Owner, please review PR #47."}},
		domain.UserProfile{
			Email:           "owner@example.com",
			Name:            "Owner Smith",
			ConnectedEmails: []string{"owner.gmail@gmail.com"},
		},
	)
	if err != nil {
		t.Fatalf("GetTaskGenerationDecision returned error: %v", err)
	}

	if len(chatModel.lastMessages) == 0 {
		t.Fatal("expected stubChatModel to capture messages")
	}

	rendered := joinMessages(chatModel.lastMessages)
	if !strings.Contains(rendered, "owner@example.com") {
		t.Fatalf("expected rendered prompt to contain owner email, got: %s", rendered)
	}
	if !strings.Contains(rendered, "owner.gmail@gmail.com") {
		t.Fatalf("expected rendered prompt to contain connected email, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Owner Smith") {
		t.Fatalf("expected rendered prompt to contain owner name, got: %s", rendered)
	}
}

func TestFormatOwnerEmailsDedupesAndDropsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		profile domain.UserProfile
		want    string
	}{
		{
			name:    "auth only",
			profile: domain.UserProfile{Email: "a@b.com"},
			want:    "a@b.com",
		},
		{
			name: "auth plus connected",
			profile: domain.UserProfile{
				Email:           "a@b.com",
				ConnectedEmails: []string{"a.b@gmail.com"},
			},
			want: "a@b.com, a.b@gmail.com",
		},
		{
			name: "dedupes case-insensitively and drops blanks",
			profile: domain.UserProfile{
				Email:           "A@B.com",
				ConnectedEmails: []string{"", "  ", "a@b.COM", "a.b@gmail.com"},
			},
			want: "A@B.com, a.b@gmail.com",
		},
		{
			name:    "empty profile",
			profile: domain.UserProfile{},
			want:    "(none)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOwnerEmails(tt.profile)
			if got != tt.want {
				t.Fatalf("formatOwnerEmails=%q want %q", got, tt.want)
			}
		})
	}
}

func joinMessages(messages []*schema.Message) string {
	var b strings.Builder
	for _, m := range messages {
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return b.String()
}
