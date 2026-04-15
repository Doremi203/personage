package notification

import (
	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/google/uuid"
)

// ErrInvalidSettingType is returned when the caller supplies a notification type
// that is not in the AvailableSettingTypes list.
var ErrInvalidSettingType = errors.Error("invalid notification setting type")

// SettingType represents a notification category.
type SettingType string

const (
	SettingTypeScheduleChange SettingType = "schedule_change"
	SettingTypeUpcomingEvent  SettingType = "upcoming_event"
)

// AvailableSettingTypes is the exhaustive list of notification types the system supports.
var AvailableSettingTypes = []SettingType{
	SettingTypeScheduleChange,
	SettingTypeUpcomingEvent,
}

// IsValidSettingType reports whether typ is a recognised notification setting type.
func IsValidSettingType(typ string) bool {
	for _, t := range AvailableSettingTypes {
		if string(t) == typ {
			return true
		}
	}
	return false
}

// Setting represents a user's notification preference for a specific type.
type Setting struct {
	UserID  uuid.UUID
	Type    SettingType
	Enabled bool
}
