package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/libs/go/token"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	notifgrpc "github.com/Doremi203/personage/backend/notificator/internal/grpc"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	mock_usecase "github.com/Doremi203/personage/backend/notificator/internal/usecase/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var notifGrpcUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func authedNotifCtx(t *testing.T) context.Context {
	t.Helper()
	return token.ContextWithToken(t.Context(), token.Token{UserID: notifGrpcUserID})
}

func newNotificationsService(t *testing.T) (pushpb.NotificationsServer, *mock_usecase.MockNotifications) {
	t.Helper()
	ctrl := gomock.NewController(t)
	uc := mock_usecase.NewMockNotifications(ctrl)
	return notifgrpc.NewNotificationsService(uc, log.Stub{}), uc
}

func TestNotificationsService_ListNotificationsV1(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	read := now.Add(time.Hour)
	notifID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	tests := []struct {
		name     string
		ctxFn    func(t *testing.T) context.Context
		req      *pushpb.ListNotificationsV1Request
		setup    func(uc *mock_usecase.MockNotifications)
		wantCode codes.Code
		wantLen  int
	}{
		{
			name:  "success returns mapped notifications",
			ctxFn: authedNotifCtx,
			req:   &pushpb.ListNotificationsV1Request{Page: 1, PageSize: 5},
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().
					List(gomock.Any(), usecase.ListNotificationsParams{
						UserID:   notifGrpcUserID,
						Page:     1,
						PageSize: 5,
					}).
					Return([]notification.Notification{
						{ID: notifID, Title: "t", Type: "x", Text: "body", SentAt: &now, ReadAt: &read},
						{ID: uuid.New(), Title: "t2", Type: "x", Text: "b2", SentAt: &now},
					}, nil)
			},
			wantCode: codes.OK,
			wantLen:  2,
		},
		{
			name:     "missing token",
			ctxFn:    func(t *testing.T) context.Context { return t.Context() },
			req:      &pushpb.ListNotificationsV1Request{Page: 1, PageSize: 5},
			setup:    func(uc *mock_usecase.MockNotifications) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "invalid request",
			ctxFn:    authedNotifCtx,
			req:      &pushpb.ListNotificationsV1Request{Page: 0, PageSize: 0},
			setup:    func(uc *mock_usecase.MockNotifications) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "usecase error wraps",
			ctxFn: authedNotifCtx,
			req:   &pushpb.ListNotificationsV1Request{Page: 1, PageSize: 5},
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
			},
			wantCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, uc := newNotificationsService(t)
			tt.setup(uc)

			resp, err := svc.ListNotificationsV1(tt.ctxFn(t), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Len(t, resp.GetNotifications(), tt.wantLen)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestNotificationsService_GetNotificationSettingsV1(t *testing.T) {
	tests := []struct {
		name     string
		ctxFn    func(t *testing.T) context.Context
		setup    func(uc *mock_usecase.MockNotifications)
		wantCode codes.Code
		wantLen  int
	}{
		{
			name:  "success",
			ctxFn: authedNotifCtx,
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().GetSettings(gomock.Any(), notifGrpcUserID).
					Return([]notification.Setting{
						{UserID: notifGrpcUserID, Type: notification.SettingTypeScheduleChange, Enabled: true},
					}, nil)
			},
			wantCode: codes.OK,
			wantLen:  1,
		},
		{
			name:     "missing token",
			ctxFn:    func(t *testing.T) context.Context { return t.Context() },
			setup:    func(uc *mock_usecase.MockNotifications) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:  "usecase error",
			ctxFn: authedNotifCtx,
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().GetSettings(gomock.Any(), notifGrpcUserID).Return(nil, assert.AnError)
			},
			wantCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, uc := newNotificationsService(t)
			tt.setup(uc)

			resp, err := svc.GetNotificationSettingsV1(tt.ctxFn(t), &pushpb.GetNotificationSettingsV1Request{})

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Len(t, resp.GetSettings(), tt.wantLen)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestNotificationsService_ToggleNotificationV1(t *testing.T) {
	tests := []struct {
		name        string
		ctxFn       func(t *testing.T) context.Context
		req         *pushpb.ToggleNotificationV1Request
		setup       func(uc *mock_usecase.MockNotifications)
		wantCode    codes.Code
		wantEnabled bool
	}{
		{
			name:  "success",
			ctxFn: authedNotifCtx,
			req:   &pushpb.ToggleNotificationV1Request{Type: string(notification.SettingTypeScheduleChange)},
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().Toggle(gomock.Any(), notifGrpcUserID, string(notification.SettingTypeScheduleChange)).
					Return(notification.Setting{Enabled: true}, nil)
			},
			wantCode:    codes.OK,
			wantEnabled: true,
		},
		{
			name:     "missing token",
			ctxFn:    func(t *testing.T) context.Context { return t.Context() },
			req:      &pushpb.ToggleNotificationV1Request{Type: string(notification.SettingTypeScheduleChange)},
			setup:    func(uc *mock_usecase.MockNotifications) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:  "invalid type maps to InvalidArgument",
			ctxFn: authedNotifCtx,
			req:   &pushpb.ToggleNotificationV1Request{Type: "made_up"},
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().Toggle(gomock.Any(), notifGrpcUserID, "made_up").
					Return(notification.Setting{}, notification.ErrInvalidSettingType)
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "other error wraps",
			ctxFn: authedNotifCtx,
			req:   &pushpb.ToggleNotificationV1Request{Type: string(notification.SettingTypeUpcomingEvent)},
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().Toggle(gomock.Any(), notifGrpcUserID, string(notification.SettingTypeUpcomingEvent)).
					Return(notification.Setting{}, assert.AnError)
			},
			wantCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, uc := newNotificationsService(t)
			tt.setup(uc)

			resp, err := svc.ToggleNotificationV1(tt.ctxFn(t), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, tt.wantEnabled, resp.GetEnabled())
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestNotificationsService_MarkNotificationAsReadV1(t *testing.T) {
	notifID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	tests := []struct {
		name     string
		ctxFn    func(t *testing.T) context.Context
		req      *pushpb.MarkNotificationAsReadV1Request
		setup    func(uc *mock_usecase.MockNotifications)
		wantCode codes.Code
	}{
		{
			name:  "success",
			ctxFn: authedNotifCtx,
			req:   &pushpb.MarkNotificationAsReadV1Request{Id: notifID.String()},
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().MarkAsRead(gomock.Any(), notifGrpcUserID, notifID).Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name:     "missing token",
			ctxFn:    func(t *testing.T) context.Context { return t.Context() },
			req:      &pushpb.MarkNotificationAsReadV1Request{Id: notifID.String()},
			setup:    func(uc *mock_usecase.MockNotifications) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "invalid uuid via validation",
			ctxFn:    authedNotifCtx,
			req:      &pushpb.MarkNotificationAsReadV1Request{Id: "not-a-uuid"},
			setup:    func(uc *mock_usecase.MockNotifications) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "not found maps to NotFound",
			ctxFn: authedNotifCtx,
			req:   &pushpb.MarkNotificationAsReadV1Request{Id: notifID.String()},
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().MarkAsRead(gomock.Any(), notifGrpcUserID, notifID).
					Return(notification.ErrNotificationNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name:  "other error wraps",
			ctxFn: authedNotifCtx,
			req:   &pushpb.MarkNotificationAsReadV1Request{Id: notifID.String()},
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().MarkAsRead(gomock.Any(), notifGrpcUserID, notifID).Return(assert.AnError)
			},
			wantCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, uc := newNotificationsService(t)
			tt.setup(uc)

			_, err := svc.MarkNotificationAsReadV1(tt.ctxFn(t), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestNotificationsService_MarkAllNotificationsAsReadV1(t *testing.T) {
	tests := []struct {
		name     string
		ctxFn    func(t *testing.T) context.Context
		setup    func(uc *mock_usecase.MockNotifications)
		wantCode codes.Code
	}{
		{
			name:  "success",
			ctxFn: authedNotifCtx,
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().MarkAllAsRead(gomock.Any(), notifGrpcUserID).Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name:     "missing token",
			ctxFn:    func(t *testing.T) context.Context { return t.Context() },
			setup:    func(uc *mock_usecase.MockNotifications) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:  "usecase error wraps",
			ctxFn: authedNotifCtx,
			setup: func(uc *mock_usecase.MockNotifications) {
				uc.EXPECT().MarkAllAsRead(gomock.Any(), notifGrpcUserID).Return(assert.AnError)
			},
			wantCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, uc := newNotificationsService(t)
			tt.setup(uc)

			_, err := svc.MarkAllNotificationsAsReadV1(tt.ctxFn(t), &pushpb.MarkAllNotificationsAsReadV1Request{})

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}
