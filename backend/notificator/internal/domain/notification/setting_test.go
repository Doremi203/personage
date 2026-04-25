package notification_test

import (
	"testing"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/stretchr/testify/assert"
)

func TestIsValidSettingType(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want bool
	}{
		{
			name: "schedule_change is valid",
			typ:  string(notification.SettingTypeScheduleChange),
			want: true,
		},
		{
			name: "upcoming_event is valid",
			typ:  string(notification.SettingTypeUpcomingEvent),
			want: true,
		},
		{
			name: "empty string is invalid",
			typ:  "",
			want: false,
		},
		{
			name: "unknown type is invalid",
			typ:  "made_up_type",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, notification.IsValidSettingType(tt.typ))
		})
	}
}
