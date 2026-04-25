package ratelimit_test

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/services/ratelimit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	rlUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	rlNow    = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
)

func TestRateLimiter_Allow(t *testing.T) {
	limits := map[notification.SettingType]ratelimit.Limits{
		notification.SettingTypeScheduleChange: {Hourly: 2, Daily: 10},
	}

	type mocks struct {
		repo *mock_notification.MockRepo
	}
	type args struct {
		typ notification.SettingType
	}
	tests := []struct {
		name        string
		args        args
		setup       func(m mocks, a args)
		wantAllowed bool
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name:        "type not in limits is allowed without repo call",
			args:        args{typ: notification.SettingTypeUpcomingEvent},
			setup:       func(m mocks, a args) {},
			wantAllowed: true,
			wantErr:     require.NoError,
		},
		{
			name: "within both limits is allowed",
			args: args{typ: notification.SettingTypeScheduleChange},
			setup: func(m mocks, a args) {
				gomock.InOrder(
					m.repo.EXPECT().
						CountSentSince(gomock.Any(), rlUserID, a.typ, rlNow.Add(-time.Hour)).
						Return(1, nil),
					m.repo.EXPECT().
						CountSentSince(gomock.Any(), rlUserID, a.typ, rlNow.Add(-24*time.Hour)).
						Return(5, nil),
				)
			},
			wantAllowed: true,
			wantErr:     require.NoError,
		},
		{
			name: "hourly limit exceeded denies and skips daily check",
			args: args{typ: notification.SettingTypeScheduleChange},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					CountSentSince(gomock.Any(), rlUserID, a.typ, rlNow.Add(-time.Hour)).
					Return(2, nil)
			},
			wantAllowed: false,
			wantErr:     require.NoError,
		},
		{
			name: "daily limit exceeded denies",
			args: args{typ: notification.SettingTypeScheduleChange},
			setup: func(m mocks, a args) {
				gomock.InOrder(
					m.repo.EXPECT().
						CountSentSince(gomock.Any(), rlUserID, a.typ, rlNow.Add(-time.Hour)).
						Return(1, nil),
					m.repo.EXPECT().
						CountSentSince(gomock.Any(), rlUserID, a.typ, rlNow.Add(-24*time.Hour)).
						Return(10, nil),
				)
			},
			wantAllowed: false,
			wantErr:     require.NoError,
		},
		{
			name: "hourly db error fails safe — denies without error",
			args: args{typ: notification.SettingTypeScheduleChange},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					CountSentSince(gomock.Any(), rlUserID, a.typ, rlNow.Add(-time.Hour)).
					Return(0, assert.AnError)
			},
			wantAllowed: false,
			wantErr:     require.NoError,
		},
		{
			name: "daily db error fails safe — denies without error",
			args: args{typ: notification.SettingTypeScheduleChange},
			setup: func(m mocks, a args) {
				gomock.InOrder(
					m.repo.EXPECT().
						CountSentSince(gomock.Any(), rlUserID, a.typ, rlNow.Add(-time.Hour)).
						Return(1, nil),
					m.repo.EXPECT().
						CountSentSince(gomock.Any(), rlUserID, a.typ, rlNow.Add(-24*time.Hour)).
						Return(0, assert.AnError),
				)
			},
			wantAllowed: false,
			wantErr:     require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_notification.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			limiter := ratelimit.New(m.repo, limits, func() time.Time { return rlNow })

			allowed, err := limiter.Allow(t.Context(), rlUserID, tt.args.typ)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantAllowed, allowed)
		})
	}
}
