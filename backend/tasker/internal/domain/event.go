package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/slices"
	eventsPb "github.com/Doremi203/personage/backend/tasker/gen/api/events"
)

const whenLayout = "2006-01-02T15:04:05 (Mon)"

type EventID string

func (id EventID) String() string {
	return string(id)
}

type UserID string

func (id UserID) String() string {
	return string(id)
}

type EventSource int

const (
	EventSourceUnknown EventSource = iota
	EventSourceGmail
	EventSourceTelegram
	EventSourceGoogleCalendar
)

func (es EventSource) String() string {
	switch es {
	case EventSourceTelegram:
		return "telegram"
	case EventSourceGmail:
		return "gmail"
	case EventSourceGoogleCalendar:
		return "google_calendar"
	default:
		return "unknown"
	}
}

func ParseEventSource(s string) EventSource {
	switch s {
	case "telegram":
		return EventSourceTelegram
	case "gmail":
		return EventSourceGmail
	case "google_calendar":
		return EventSourceGoogleCalendar
	default:
		return EventSourceUnknown
	}
}

func eventSourceFromPB(ct eventsPb.ConnectorType) EventSource {
	switch ct {
	case eventsPb.ConnectorType_CONNECTOR_TYPE_GMAIL:
		return EventSourceGmail
	case eventsPb.ConnectorType_CONNECTOR_TYPE_TELEGRAM:
		return EventSourceTelegram
	case eventsPb.ConnectorType_CONNECTOR_TYPE_GOOGLE_CALENDAR:
		return EventSourceGoogleCalendar
	default:
		return EventSourceUnknown
	}
}

type NormalizedEventContext string

func FromPB(e *eventsPb.Event, loc *time.Location) (Event, error) {
	eventModel := Event{
		ID:         EventID(e.GetId()),
		UserID:     UserID(e.GetUserId()),
		Source:     eventSourceFromPB(e.GetConnectorType()),
		OccurredAt: e.GetOccurredAt().AsTime(),
	}

	participantsStrs := slices.Map(e.GetContext().GetOtherParticipants(), formatParticipant)
	sort.Strings(participantsStrs)

	var builder strings.Builder
	text := []writtenStr{
		{format: "SOURCE: %s\n", args: []any{eventModel.Source.String()}},
		{format: "SUBJECT: %s\n", args: []any{e.GetContext().GetSubject().GetName()}},
		{format: "WHEN: %s\n", args: []any{eventModel.OccurredAt.In(loc).Format(whenLayout)}},
		{format: "SENDER: %s\n", args: []any{formatParticipant(e.GetContext().GetSender())}},
		{format: "PARTICIPANTS: %s\n", args: []any{strings.Join(participantsStrs, ", ")}},
		{format: "TEXT:\n%s", args: []any{e.GetContext().GetBody()}},
	}

	if err := writeToBuilder(&builder, text...); err != nil {
		return Event{}, err
	}

	eventModel.Context = NormalizedEventContext(builder.String())

	return eventModel, nil
}

type writtenStr struct {
	format string
	args   []any
}

func writeToBuilder(b *strings.Builder, strs ...writtenStr) error {
	for _, str := range strs {
		_, err := fmt.Fprintf(b, str.format, str.args...)
		if err != nil {
			return err
		}
	}
	return nil
}

func formatParticipant(p *eventsPb.Context_Participant) string {
	return fmt.Sprintf("[EMAIL %s NAME %s TELEGRAM_TAG %s]",
		p.GetEmail(),
		p.GetName(),
		p.GetTelegramUser().GetTag(),
	)
}

type Event struct {
	ID         EventID
	UserID     UserID
	Source     EventSource
	Context    NormalizedEventContext
	OccurredAt time.Time
	ClusterID  ClusterID
	Similarity float64
}

type EventWithEmbedding struct {
	Event
	Embedding []float32
}
