package domain_test

import (
	"strings"
	"testing"
	"time"

	eventsPb "github.com/Doremi203/personage/backend/tasker/gen/api/events"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEventSource_String(t *testing.T) {
	assert.Equal(t, "telegram", domain.EventSourceTelegram.String())
	assert.Equal(t, "gmail", domain.EventSourceGmail.String())
	assert.Equal(t, "google_calendar", domain.EventSourceGoogleCalendar.String())
	assert.Equal(t, "unknown", domain.EventSourceUnknown.String())
	assert.Equal(t, "unknown", domain.EventSource(99).String())
}

func TestParseEventSource(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want domain.EventSource
	}{
		{name: "telegram", in: "telegram", want: domain.EventSourceTelegram},
		{name: "gmail", in: "gmail", want: domain.EventSourceGmail},
		{name: "google_calendar", in: "google_calendar", want: domain.EventSourceGoogleCalendar},
		{name: "garbage", in: "wat", want: domain.EventSourceUnknown},
		{name: "empty", in: "", want: domain.EventSourceUnknown},
		{name: "case-sensitive: GMAIL is unknown", in: "GMAIL", want: domain.EventSourceUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, domain.ParseEventSource(tt.in))
		})
	}
}

func TestFromPB_PopulatesAllSections(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	occurred := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)

	pb := &eventsPb.Event{
		Id:            "event-1",
		UserId:        "user-1",
		ConnectorType: eventsPb.ConnectorType_CONNECTOR_TYPE_GMAIL,
		OccurredAt:    timestamppb.New(occurred),
		Context: &eventsPb.Context{
			Body:    "hello world",
			Subject: &eventsPb.Context_Subject{Name: "important"},
			Sender:  &eventsPb.Context_Participant{Email: new("alice@example.com")},
			OtherParticipants: []*eventsPb.Context_Participant{
				{Email: new("zach@example.com")},
				{Email: new("bob@example.com")},
			},
		},
	}

	got, err := domain.FromPB(pb, moscow)
	require.NoError(t, err)

	assert.Equal(t, domain.EventID("event-1"), got.ID)
	assert.Equal(t, domain.UserID("user-1"), got.UserID)
	assert.Equal(t, domain.EventSourceGmail, got.Source)
	assert.True(t, got.OccurredAt.Equal(occurred))

	ctx := string(got.Context)
	assert.Contains(t, ctx, "SOURCE: gmail")
	assert.Contains(t, ctx, "SUBJECT: important")
	assert.Contains(t, ctx, "WHEN: 2026-04-01T12:30:00 (Wed)")
	assert.Contains(t, ctx, "SENDER: [EMAIL alice@example.com NAME  TELEGRAM_TAG ]")
	assert.NotContains(t, ctx, "START TIME:")
	assert.NotContains(t, ctx, "END TIME:")
	assert.Contains(t, ctx, "TEXT:\nhello world")

	bobIdx := strings.Index(ctx, "[EMAIL bob@example.com")
	zachIdx := strings.Index(ctx, "[EMAIL zach@example.com")
	require.NotEqual(t, -1, bobIdx)
	require.NotEqual(t, -1, zachIdx)
	assert.Less(t, bobIdx, zachIdx, "participants should be sorted alphabetically")
}

func TestFromPB_WhenFormatRendersDefaultLocation(t *testing.T) {
	occurred := time.Date(2026, 5, 21, 7, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		locName  string
		expected string
	}{
		{name: "moscow", locName: "Europe/Moscow", expected: "WHEN: 2026-05-21T10:00:00 (Thu)"},
		{name: "utc", locName: "UTC", expected: "WHEN: 2026-05-21T07:00:00 (Thu)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := time.LoadLocation(tt.locName)
			require.NoError(t, err)

			got, err := domain.FromPB(&eventsPb.Event{
				Id:         "event-1",
				UserId:     "user-1",
				OccurredAt: timestamppb.New(occurred),
			}, loc)
			require.NoError(t, err)

			assert.Contains(t, string(got.Context), tt.expected)
		})
	}
}

func TestFromPB_ConnectorTypeMapping(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	tests := []struct {
		name string
		in   eventsPb.ConnectorType
		want domain.EventSource
	}{
		{name: "gmail", in: eventsPb.ConnectorType_CONNECTOR_TYPE_GMAIL, want: domain.EventSourceGmail},
		{name: "telegram", in: eventsPb.ConnectorType_CONNECTOR_TYPE_TELEGRAM, want: domain.EventSourceTelegram},
		{name: "google_calendar", in: eventsPb.ConnectorType_CONNECTOR_TYPE_GOOGLE_CALENDAR, want: domain.EventSourceGoogleCalendar},
		{name: "unknown", in: eventsPb.ConnectorType_CONNECTOR_TYPE_UNKNOWN, want: domain.EventSourceUnknown},
		{name: "out_of_range", in: eventsPb.ConnectorType(99), want: domain.EventSourceUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.FromPB(&eventsPb.Event{
				Id:            "event-1",
				UserId:        "user-1",
				ConnectorType: tt.in,
			}, moscow)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Source)
		})
	}
}

func TestFromPB_EmptyContextOmitsTimeFrame(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	pb := &eventsPb.Event{
		Id:     "event-1",
		UserId: "user-1",
	}

	got, err := domain.FromPB(pb, moscow)
	require.NoError(t, err)

	ctx := string(got.Context)
	assert.NotContains(t, ctx, "START TIME:")
	assert.NotContains(t, ctx, "END TIME:")
	assert.Contains(t, ctx, "PARTICIPANTS: \n")
	assert.Contains(t, ctx, "SUBJECT: \n")
	assert.Equal(t, domain.EventSourceUnknown, got.Source)
}
