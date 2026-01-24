package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"gitlab.com/amoguscorp/personage/backend/libs/go/slices"
	eventsPb "gitlab.com/amoguscorp/personage/backend/tasker/gen/api/events"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

type NormalizedEventContext string

func FromPB(e *eventsPb.Event) Event {
	eventModel := Event{
		ID:         EventID(e.GetId()),
		UserID:     UserID(e.GetUserId()),
		Source:     EventSource(e.GetConnectorType()),
		OccurredAt: e.GetOccurredAt().AsTime(),
	}

	participantsStrs := slices.Map(e.GetContext().GetOtherParticipants(), formatParticipant)
	sort.Strings(participantsStrs)

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("SOURCE: %s\n", eventModel.Source.String()))
	builder.WriteString(fmt.Sprintf("SUBJECT: %s\n", e.GetContext().GetSubject().GetName()))
	builder.WriteString(fmt.Sprintf("WHEN: %s\n", eventModel.OccurredAt.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("SENDER: %s\n", formatParticipant(e.GetContext().GetSender())))
	builder.WriteString(fmt.Sprintf("PARTICIPANTS: %s\n", strings.Join(participantsStrs, ", ")))
	builder.WriteString(fmt.Sprintf("START TIME: %s\n", formatTimestamp(e.GetContext().GetTimeFrame().GetStart())))
	builder.WriteString(fmt.Sprintf("END TIME: %s\n", formatTimestamp(e.GetContext().GetTimeFrame().GetEnd())))
	builder.WriteString(fmt.Sprintf("TEXT:\n%s", e.GetContext().GetBody()))

	eventModel.Context = NormalizedEventContext(builder.String())

	return eventModel
}

func formatParticipant(p *eventsPb.Context_Participant) string {
	return fmt.Sprintf("[EMAIL %s NAME %s TELEGRAM_TAG %s]",
		p.GetEmail(),
		p.GetName(),
		p.GetTelegramUser().GetTag(),
	)
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}

	return ts.AsTime().Format(time.RFC3339)
}

type Event struct {
	ID         EventID
	UserID     UserID
	Source     EventSource
	Context    NormalizedEventContext
	OccurredAt time.Time
	ClusterID  ClusterID
}

type EventWithEmbedding struct {
	Event
	Embedding []float32
}
