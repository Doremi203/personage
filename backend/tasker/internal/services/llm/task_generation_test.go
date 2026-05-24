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

func TestGenerateTaskRoundsStartTimeToSlot(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	tests := []struct {
		name      string
		startTime string
		want      string
	}{
		{name: "1 min past slot rounds down", startTime: "2026-05-23T19:16:00", want: "2026-05-23T16:15:00Z"},
		{name: "7 min past slot rounds down", startTime: "2026-05-23T19:22:00", want: "2026-05-23T16:15:00Z"},
		{name: "8 min past slot rounds up", startTime: "2026-05-23T19:23:00", want: "2026-05-23T16:30:00Z"},
		{name: "already aligned stays", startTime: "2026-05-23T19:15:00", want: "2026-05-23T16:15:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatModel := &stubChatModel{results: []stubChatModelResult{
				{message: &schema.Message{Content: `{
					"title": "Call back",
					"description": "Return the missed call.",
					"duration_minutes": 30,
					"priority": 5,
					"deadline": null,
					"start_time": "` + tt.startTime + `",
					"category": "personal"
				}`}},
			}}

			service := NewTaskGenerationService(chatModel, log.Stub{}, stubPromptProvider{}, moscow)
			task, err := service.GenerateTask(t.Context(), []domain.Event{{ID: "event-1", Context: "Missed call from a friend."}}, domain.UserProfile{})
			require.NoError(t, err)
			require.NotNil(t, task.StartTime)
			assert.Equal(t, tt.want, task.StartTime.UTC().Format(time.RFC3339))
		})
	}
}

func TestRoundToTimeSlot(t *testing.T) {
	t.Run("nil stays nil", func(t *testing.T) {
		assert.Nil(t, roundToTimeSlot(nil))
	})

	t.Run("rounds to nearest 15 min slot", func(t *testing.T) {
		in := time.Date(2026, 5, 23, 19, 16, 0, 0, time.UTC)
		got := roundToTimeSlot(&in)
		require.NotNil(t, got)
		assert.Equal(t, "2026-05-23T19:15:00Z", got.UTC().Format(time.RFC3339))
	})
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

func TestParseOptionalDate(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	tests := []struct {
		name     string
		value    *string
		wantDay  string // YYYY-MM-DD in Moscow loc
		wantNil  bool
		wantErr  require.ErrorAssertionFunc
	}{
		{name: "nil pointer", value: nil, wantNil: true, wantErr: require.NoError},
		{name: "empty string", value: new(""), wantNil: true, wantErr: require.NoError},
		{name: "literal null", value: new("null"), wantNil: true, wantErr: require.NoError},
		{name: "whitespace", value: new("   "), wantNil: true, wantErr: require.NoError},
		{name: "valid day", value: new("2026-05-24"), wantDay: "2026-05-24", wantErr: require.NoError},
		{name: "leading whitespace", value: new("  2026-05-24  "), wantDay: "2026-05-24", wantErr: require.NoError},
		{name: "invalid format", value: new("24.05.2026"), wantErr: require.Error},
		{name: "datetime not allowed", value: new("2026-05-24T15:00:00"), wantErr: require.Error},
		{name: "garbage", value: new("not a date"), wantErr: require.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOptionalDate("date", tt.value, moscow)
			tt.wantErr(t, err)
			if err != nil {
				return
			}
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			// Y/M/D must match the input day when read in Moscow — pgx encodes DATE from
			// these components, so a UTC-shifted value would land on the wrong calendar day.
			assert.Equal(t, tt.wantDay, got.Format("2006-01-02"))
			assert.Equal(t, moscow, got.Location())
		})
	}
}
