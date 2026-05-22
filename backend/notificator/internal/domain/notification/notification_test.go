package notification_test

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testNow    = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

func TestNewPending(t *testing.T) {
	retryAfter := testNow.Add(time.Minute)
	expiresAt := testNow.Add(time.Hour)
	payload := &notification.PushPayload{Body: "b", Icon: "i", URL: "u"}

	type args struct {
		userID         uuid.UUID
		title          string
		typ            string
		text           string
		retryAfter     time.Time
		expiresAt      time.Time
		payload        *notification.PushPayload
		idempotencyKey string
	}
	tests := []struct {
		name    string
		args    args
		want    notification.Notification
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				userID:         testUserID,
				title:          "Title",
				typ:            string(notification.SettingTypeScheduleChange),
				text:           "Text",
				retryAfter:     retryAfter,
				expiresAt:      expiresAt,
				payload:        payload,
				idempotencyKey: "key-1",
			},
			want: notification.Notification{
				UserID:         testUserID,
				Title:          "Title",
				Type:           string(notification.SettingTypeScheduleChange),
				Text:           "Text",
				Status:         notification.StatusPending,
				RetryAfter:     &retryAfter,
				ExpiresAt:      &expiresAt,
				PushPayload:    payload,
				IdempotencyKey: "key-1",
			},
			wantErr: require.NoError,
		},
		{
			name: "empty user id",
			args: args{
				userID: uuid.Nil,
				title:  "Title",
				typ:    string(notification.SettingTypeScheduleChange),
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, notification.ErrEmptyUserID)
			},
		},
		{
			name: "empty title",
			args: args{
				userID: testUserID,
				title:  "",
				typ:    string(notification.SettingTypeScheduleChange),
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, notification.ErrEmptyTitle)
			},
		},
		{
			name: "empty type",
			args: args{
				userID: testUserID,
				title:  "Title",
				typ:    "",
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, notification.ErrEmptyType)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := notification.NewPending(
				tt.args.userID,
				tt.args.title,
				tt.args.typ,
				tt.args.text,
				tt.args.retryAfter,
				tt.args.expiresAt,
				tt.args.payload,
				tt.args.idempotencyKey,
			)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, n)
		})
	}
}

func TestNewSent(t *testing.T) {
	sentAt := testNow

	type args struct {
		userID         uuid.UUID
		title          string
		typ            string
		text           string
		sentAt         time.Time
		idempotencyKey string
	}
	tests := []struct {
		name    string
		args    args
		want    notification.Notification
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				userID:         testUserID,
				title:          "Title",
				typ:            string(notification.SettingTypeUpcomingEvent),
				text:           "Text",
				sentAt:         sentAt,
				idempotencyKey: "key-1",
			},
			want: notification.Notification{
				UserID:         testUserID,
				Title:          "Title",
				Type:           string(notification.SettingTypeUpcomingEvent),
				Text:           "Text",
				Status:         notification.StatusSent,
				SentAt:         &sentAt,
				IdempotencyKey: "key-1",
			},
			wantErr: require.NoError,
		},
		{
			name: "empty user id",
			args: args{
				userID: uuid.Nil,
				title:  "Title",
				typ:    string(notification.SettingTypeUpcomingEvent),
				sentAt: testNow,
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.True(t, errors.Is(err, notification.ErrEmptyUserID))
			},
		},
		{
			name: "empty title",
			args: args{
				userID: testUserID,
				title:  "",
				typ:    string(notification.SettingTypeUpcomingEvent),
				sentAt: testNow,
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.True(t, errors.Is(err, notification.ErrEmptyTitle))
			},
		},
		{
			name: "empty type",
			args: args{
				userID: testUserID,
				title:  "Title",
				typ:    "",
				sentAt: testNow,
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.True(t, errors.Is(err, notification.ErrEmptyType))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := notification.NewSent(
				tt.args.userID,
				tt.args.title,
				tt.args.typ,
				tt.args.text,
				tt.args.sentAt,
				tt.args.idempotencyKey,
			)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, n)
		})
	}
}
