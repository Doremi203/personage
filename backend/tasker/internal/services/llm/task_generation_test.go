package llm

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			"priority": 42,
			"deadline": null,
			"start_time": null,
			"category": "work"
		}`}},
		{message: &schema.Message{Content: `{
			"title": "Review PR #47",
			"description": "Review the proposed API integration changes.",
			"duration_minutes": 30,
			"priority": 7,
			"deadline": null,
			"start_time": null,
			"category": "work"
		}`}},
	}}

	service := NewTaskGenerationService(chatModel, log.Stub{}, stubPromptProvider{}, time.UTC)
	task, err := service.GenerateTask(t.Context(), []domain.Event{{ID: "event-1", Context: "Please review PR #47."}}, domain.UserProfile{})
	if err != nil {
		t.Fatalf("GenerateTask returned error: %v", err)
	}

	if chatModel.calls != 2 {
		t.Fatalf("expected 2 model calls, got %d", chatModel.calls)
	}

	if task.Priority != 7 {
		t.Fatalf("unexpected priority: %d", task.Priority)
	}
}

func TestParseOptionalTimestamp(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	tests := []struct {
		name    string
		value   *string
		want    string
		wantNil bool
		wantErr require.ErrorAssertionFunc
	}{
		{name: "nil pointer", value: nil, wantNil: true, wantErr: require.NoError},
		{name: "empty string", value: new(""), wantNil: true, wantErr: require.NoError},
		{name: "literal null", value: new("null"), wantNil: true, wantErr: require.NoError},
		{name: "whitespace", value: new("   "), wantNil: true, wantErr: require.NoError},
		{name: "naive moscow summer", value: new("2026-05-23T15:00:00"), want: "2026-05-23T12:00:00Z", wantErr: require.NoError},
		{name: "naive moscow winter", value: new("2026-01-23T15:00:00"), want: "2026-01-23T12:00:00Z", wantErr: require.NoError},
		{name: "naive without seconds", value: new("2026-05-23T15:00"), want: "2026-05-23T12:00:00Z", wantErr: require.NoError},
		{name: "tolerates trailing Z", value: new("2026-05-23T15:00:00Z"), want: "2026-05-23T15:00:00Z", wantErr: require.NoError},
		{name: "tolerates positive offset", value: new("2026-05-23T15:00:00+03:00"), want: "2026-05-23T12:00:00Z", wantErr: require.NoError},
		{name: "tolerates negative offset", value: new("2026-05-23T15:00:00-05:00"), want: "2026-05-23T20:00:00Z", wantErr: require.NoError},
		{name: "invalid format", value: new("not a date"), wantErr: require.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOptionalTimestamp("start_time", tt.value, moscow)
			tt.wantErr(t, err)
			if err != nil {
				return
			}
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.UTC().Format(time.RFC3339))
		})
	}
}
