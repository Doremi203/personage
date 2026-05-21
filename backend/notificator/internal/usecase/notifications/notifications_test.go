package notifications_test

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase/notifications"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	notifUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
)

func TestService_List(t *testing.T) {
	type mocks struct {
		repo *mock_notification.MockRepo
	}
	type args struct {
		params usecase.ListNotificationsParams
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    []notification.Notification
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success first page",
			args: args{
				params: usecase.ListNotificationsParams{
					UserID:   notifUserID,
					Page:     1,
					PageSize: 5,
				},
			},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					ListByUserID(gomock.Any(), notifUserID, 5, 0).
					Return([]notification.Notification{{Title: "n1"}}, nil)
			},
			want:    []notification.Notification{{Title: "n1"}},
			wantErr: require.NoError,
		},
		{
			name: "page size clamped to maxPageSize",
			args: args{
				params: usecase.ListNotificationsParams{
					UserID:   notifUserID,
					Page:     1,
					PageSize: 100,
				},
			},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					ListByUserID(gomock.Any(), notifUserID, 10, 0).
					Return(nil, nil)
			},
			want:    nil,
			wantErr: require.NoError,
		},
		{
			name: "second page offset computed from clamped size",
			args: args{
				params: usecase.ListNotificationsParams{
					UserID:   notifUserID,
					Page:     3,
					PageSize: 4,
				},
			},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					ListByUserID(gomock.Any(), notifUserID, 4, 8).
					Return(nil, nil)
			},
			want:    nil,
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			args: args{
				params: usecase.ListNotificationsParams{
					UserID:   notifUserID,
					Page:     1,
					PageSize: 5,
				},
			},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					ListByUserID(gomock.Any(), notifUserID, 5, 0).
					Return(nil, assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_notification.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			s := notifications.New(m.repo, time.Now)
			got, err := s.List(t.Context(), tt.args.params)

			tt.wantErr(t, err)
			if err == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestService_GetSettings(t *testing.T) {
	type mocks struct {
		repo *mock_notification.MockRepo
	}
	type args struct {
		userID uuid.UUID
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    []notification.Setting
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "missing types are returned as enabled by default",
			args: args{userID: notifUserID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					GetSettings(gomock.Any(), notifUserID).
					Return([]notification.Setting{{
						UserID:  notifUserID,
						Type:    notification.SettingTypeScheduleChange,
						Enabled: false,
					}}, nil)
			},
			want: []notification.Setting{
				{UserID: notifUserID, Type: notification.SettingTypeScheduleChange, Enabled: false},
				{UserID: notifUserID, Type: notification.SettingTypeUpcomingEvent, Enabled: true},
			},
			wantErr: require.NoError,
		},
		{
			name: "empty store returns defaults for every available type",
			args: args{userID: notifUserID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					GetSettings(gomock.Any(), notifUserID).
					Return(nil, nil)
			},
			want: []notification.Setting{
				{UserID: notifUserID, Type: notification.SettingTypeScheduleChange, Enabled: true},
				{UserID: notifUserID, Type: notification.SettingTypeUpcomingEvent, Enabled: true},
			},
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			args: args{userID: notifUserID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					GetSettings(gomock.Any(), notifUserID).
					Return(nil, assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_notification.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			s := notifications.New(m.repo, time.Now)
			got, err := s.GetSettings(t.Context(), tt.args.userID)

			tt.wantErr(t, err)
			if err == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestService_MarkAsRead(t *testing.T) {
	notifID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	type mocks struct {
		repo *mock_notification.MockRepo
	}
	type args struct {
		userID  uuid.UUID
		notifID uuid.UUID
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{userID: notifUserID, notifID: notifID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					MarkAsRead(gomock.Any(), a.notifID, a.userID, now).
					Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "not found passes through",
			args: args{userID: notifUserID, notifID: notifID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					MarkAsRead(gomock.Any(), a.notifID, a.userID, now).
					Return(notification.ErrNotificationNotFound)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, notification.ErrNotificationNotFound)
			},
		},
		{
			name: "repo error wraps",
			args: args{userID: notifUserID, notifID: notifID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					MarkAsRead(gomock.Any(), a.notifID, a.userID, now).
					Return(assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_notification.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			s := notifications.New(m.repo, clock)
			err := s.MarkAsRead(t.Context(), tt.args.userID, tt.args.notifID)

			tt.wantErr(t, err)
		})
	}
}

func TestService_MarkAllAsRead(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	type mocks struct {
		repo *mock_notification.MockRepo
	}
	type args struct {
		userID uuid.UUID
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{userID: notifUserID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					MarkAllAsRead(gomock.Any(), a.userID, now).
					Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			args: args{userID: notifUserID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					MarkAllAsRead(gomock.Any(), a.userID, now).
					Return(assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_notification.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			s := notifications.New(m.repo, clock)
			err := s.MarkAllAsRead(t.Context(), tt.args.userID)

			tt.wantErr(t, err)
		})
	}
}

func TestService_Toggle(t *testing.T) {
	type mocks struct {
		repo *mock_notification.MockRepo
	}
	type args struct {
		userID           uuid.UUID
		notificationType string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    notification.Setting
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				userID:           notifUserID,
				notificationType: string(notification.SettingTypeScheduleChange),
			},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					ToggleSetting(gomock.Any(), notifUserID, string(notification.SettingTypeScheduleChange)).
					Return(notification.Setting{
						UserID:  notifUserID,
						Type:    notification.SettingTypeScheduleChange,
						Enabled: false,
					}, nil)
			},
			want: notification.Setting{
				UserID:  notifUserID,
				Type:    notification.SettingTypeScheduleChange,
				Enabled: false,
			},
			wantErr: require.NoError,
		},
		{
			name: "invalid type returns sentinel without calling repo",
			args: args{
				userID:           notifUserID,
				notificationType: "made_up",
			},
			setup: func(m mocks, a args) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, notification.ErrInvalidSettingType)
			},
		},
		{
			name: "repo error wraps",
			args: args{
				userID:           notifUserID,
				notificationType: string(notification.SettingTypeUpcomingEvent),
			},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					ToggleSetting(gomock.Any(), notifUserID, string(notification.SettingTypeUpcomingEvent)).
					Return(notification.Setting{}, assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_notification.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			s := notifications.New(m.repo, time.Now)
			got, err := s.Toggle(t.Context(), tt.args.userID, tt.args.notificationType)

			tt.wantErr(t, err)
			if err == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
