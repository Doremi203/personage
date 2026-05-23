package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	mock_push "github.com/Doremi203/personage/backend/notificator/internal/domain/push/mock"
	notifgrpc "github.com/Doremi203/personage/backend/notificator/internal/grpc"
	mock_usecase "github.com/Doremi203/personage/backend/notificator/internal/usecase/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	adminRecipientUUID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	adminNow           = time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
)

func adminRecipient() push.Recipient {
	return push.Recipient{
		ID: push.RecipientID(adminRecipientUUID),
		Subscriptions: []push.Subscription{{
			RecipientID: push.RecipientID(adminRecipientUUID),
			Endpoint:    "https://example.com/push",
		}},
	}
}

type adminMocks struct {
	pushRepo *mock_push.MockRepo
	notifs   *mock_notification.MockRepo
	sender   *mock_usecase.MockPushSender
}

func newAdminService(t *testing.T) (pushpb.AdminServer, adminMocks) {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := adminMocks{
		pushRepo: mock_push.NewMockRepo(ctrl),
		notifs:   mock_notification.NewMockRepo(ctrl),
		sender:   mock_usecase.NewMockPushSender(ctrl),
	}
	svc := notifgrpc.NewAdminService(
		m.pushRepo,
		m.sender,
		m.notifs,
		func() time.Time { return adminNow },
		log.Stub{},
	)
	return svc, m
}

func TestAdminService_SendPushV1(t *testing.T) {
	expectedPush := push.Push{Title: "T", Body: "B", Url: "https://u", Icon: "i"}

	t.Run("missing title returns InvalidArgument", func(t *testing.T) {
		svc, _ := newAdminService(t)
		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{},
		})
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid recipient uuid returns InvalidArgument", func(t *testing.T) {
		svc, _ := newAdminService(t)
		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{
				Title:       "T",
				RecipientId: "not-a-uuid",
			},
		})
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("targeted send persists notification then sends push", func(t *testing.T) {
		svc, m := newAdminService(t)
		m.pushRepo.EXPECT().
			GetSubscriptionsByRecipientID(gomock.Any(), push.RecipientID(adminRecipientUUID)).
			Return(adminRecipient().Subscriptions, nil)
		m.notifs.EXPECT().
			CreateIfAbsent(gomock.Any(), gomock.AssignableToTypeOf(notification.Notification{})).
			DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
				require.Equal(t, adminRecipientUUID, n.UserID)
				require.Equal(t, "T", n.Title)
				require.Equal(t, "admin", n.Type)
				require.Equal(t, "Detailed", n.Text)
				require.Equal(t, notification.StatusSent, n.Status)
				require.NotNil(t, n.SentAt)
				require.True(t, n.SentAt.Equal(adminNow))
				require.Empty(t, n.IdempotencyKey)
				return true, nil
			})
		m.sender.EXPECT().Send(gomock.Any(), adminRecipient(), expectedPush).Return(nil)

		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{
				RecipientId:  adminRecipientUUID.String(),
				Title:        "T",
				Body:         "B",
				Url:          "https://u",
				Icon:         "i",
				DetailedText: "Detailed",
			},
		})
		require.NoError(t, err)
	})

	t.Run("custom type is preserved on persisted row", func(t *testing.T) {
		svc, m := newAdminService(t)
		m.pushRepo.EXPECT().
			GetSubscriptionsByRecipientID(gomock.Any(), push.RecipientID(adminRecipientUUID)).
			Return(adminRecipient().Subscriptions, nil)
		m.notifs.EXPECT().
			CreateIfAbsent(gomock.Any(), gomock.AssignableToTypeOf(notification.Notification{})).
			DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
				require.Equal(t, "schedule_change", n.Type)
				return true, nil
			})
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{
				RecipientId: adminRecipientUUID.String(),
				Title:       "T",
				Type:        "schedule_change",
			},
		})
		require.NoError(t, err)
	})

	t.Run("empty detailed_text falls back to body", func(t *testing.T) {
		svc, m := newAdminService(t)
		m.pushRepo.EXPECT().
			GetSubscriptionsByRecipientID(gomock.Any(), push.RecipientID(adminRecipientUUID)).
			Return(adminRecipient().Subscriptions, nil)
		m.notifs.EXPECT().
			CreateIfAbsent(gomock.Any(), gomock.AssignableToTypeOf(notification.Notification{})).
			DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
				require.Equal(t, "B", n.Text)
				return true, nil
			})
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{
				RecipientId: adminRecipientUUID.String(),
				Title:       "T",
				Body:        "B",
			},
		})
		require.NoError(t, err)
	})

	t.Run("idempotency_key is scoped per recipient", func(t *testing.T) {
		svc, m := newAdminService(t)
		other := uuid.MustParse("33333333-3333-3333-3333-333333333333")
		recipients := []push.Recipient{
			adminRecipient(),
			{ID: push.RecipientID(other), Subscriptions: []push.Subscription{{RecipientID: push.RecipientID(other), Endpoint: "https://e2"}}},
		}
		m.pushRepo.EXPECT().GetAllRecipients(gomock.Any()).Return(recipients, nil)

		gotKeys := make([]string, 0, 2)
		m.notifs.EXPECT().
			CreateIfAbsent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
				gotKeys = append(gotKeys, n.IdempotencyKey)
				return true, nil
			}).
			Times(2)
		m.sender.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)

		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{
				Title:          "T",
				IdempotencyKey: "broadcast-1",
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{
			"broadcast-1:" + adminRecipientUUID.String(),
			"broadcast-1:" + other.String(),
		}, gotKeys)
	})

	t.Run("duplicate idempotency skips push send", func(t *testing.T) {
		svc, m := newAdminService(t)
		m.pushRepo.EXPECT().
			GetSubscriptionsByRecipientID(gomock.Any(), push.RecipientID(adminRecipientUUID)).
			Return(adminRecipient().Subscriptions, nil)
		m.notifs.EXPECT().
			CreateIfAbsent(gomock.Any(), gomock.Any()).
			Return(false, nil)
		// sender.Send must NOT be called.

		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{
				RecipientId:    adminRecipientUUID.String(),
				Title:          "T",
				IdempotencyKey: "k",
			},
		})
		require.NoError(t, err)
	})

	t.Run("persist error swallowed continues to next recipient", func(t *testing.T) {
		svc, m := newAdminService(t)
		other := uuid.MustParse("44444444-4444-4444-4444-444444444444")
		recipients := []push.Recipient{
			adminRecipient(),
			{ID: push.RecipientID(other), Subscriptions: []push.Subscription{{RecipientID: push.RecipientID(other), Endpoint: "https://e2"}}},
		}
		m.pushRepo.EXPECT().GetAllRecipients(gomock.Any()).Return(recipients, nil)

		gomock.InOrder(
			m.notifs.EXPECT().
				CreateIfAbsent(gomock.Any(), gomock.Any()).
				Return(false, assert.AnError),
			m.notifs.EXPECT().
				CreateIfAbsent(gomock.Any(), gomock.Any()).
				Return(true, nil),
		)
		m.sender.EXPECT().Send(gomock.Any(), recipients[1], gomock.Any()).Return(nil)

		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{Title: "T"},
		})
		require.NoError(t, err)
	})

	t.Run("no subscriptions for targeted recipient persists in-app and skips push", func(t *testing.T) {
		svc, m := newAdminService(t)
		m.pushRepo.EXPECT().
			GetSubscriptionsByRecipientID(gomock.Any(), push.RecipientID(adminRecipientUUID)).
			Return(nil, nil)
		m.notifs.EXPECT().
			CreateIfAbsent(gomock.Any(), gomock.AssignableToTypeOf(notification.Notification{})).
			DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
				require.Equal(t, adminRecipientUUID, n.UserID)
				require.Equal(t, notification.StatusSent, n.Status)
				return true, nil
			})
		// sender.Send must NOT be called when there are no subscriptions.

		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{
				RecipientId: adminRecipientUUID.String(),
				Title:       "T",
			},
		})
		require.NoError(t, err)
	})

	t.Run("duplicate idempotency persists scoped key", func(t *testing.T) {
		svc, m := newAdminService(t)
		m.pushRepo.EXPECT().
			GetSubscriptionsByRecipientID(gomock.Any(), push.RecipientID(adminRecipientUUID)).
			Return(adminRecipient().Subscriptions, nil)
		m.notifs.EXPECT().
			CreateIfAbsent(gomock.Any(), gomock.AssignableToTypeOf(notification.Notification{})).
			DoAndReturn(func(_ context.Context, n notification.Notification) (bool, error) {
				require.Equal(t, "broadcast-1:"+adminRecipientUUID.String(), n.IdempotencyKey)
				return false, nil
			})
		// sender.Send must NOT be called on duplicate.

		_, err := svc.SendPushV1(context.Background(), &pushpb.SendPushV1Request{
			Notification: &pushpb.Notification{
				RecipientId:    adminRecipientUUID.String(),
				Title:          "T",
				IdempotencyKey: "broadcast-1",
			},
		})
		require.NoError(t, err)
	})
}

func TestAdminService_ListUserNotificationsV1(t *testing.T) {
	t.Run("invalid uuid returns InvalidArgument", func(t *testing.T) {
		svc, _ := newAdminService(t)
		_, err := svc.ListUserNotificationsV1(t.Context(), &pushpb.ListUserNotificationsV1Request{
			UserId: "not-a-uuid",
		})
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("empty user_id returns InvalidArgument", func(t *testing.T) {
		svc, _ := newAdminService(t)
		_, err := svc.ListUserNotificationsV1(t.Context(), &pushpb.ListUserNotificationsV1Request{})
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("returns notifications ordered by sent_at desc", func(t *testing.T) {
		svc, m := newAdminService(t)
		id1 := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		id2 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		sentAt1 := adminNow
		sentAt2 := adminNow.Add(-time.Hour)
		readAt := adminNow.Add(-30 * time.Minute)

		m.notifs.EXPECT().
			ListByUserID(gomock.Any(), adminRecipientUUID, 200, 0).
			Return([]notification.Notification{
				{
					ID:     id1,
					UserID: adminRecipientUUID,
					Title:  "Hello",
					Type:   "admin",
					Text:   "Body 1",
					Status: notification.StatusSent,
					SentAt: &sentAt1,
				},
				{
					ID:     id2,
					UserID: adminRecipientUUID,
					Title:  "Earlier",
					Type:   "upcoming_event",
					Text:   "Body 2",
					Status: notification.StatusSent,
					SentAt: &sentAt2,
					ReadAt: &readAt,
				},
			}, nil)

		resp, err := svc.ListUserNotificationsV1(t.Context(), &pushpb.ListUserNotificationsV1Request{
			UserId: adminRecipientUUID.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.GetNotifications(), 2)

		first := resp.GetNotifications()[0]
		require.Equal(t, id1.String(), first.GetId())
		require.Equal(t, "Hello", first.GetTitle())
		require.Equal(t, "admin", first.GetType())
		require.Equal(t, "Body 1", first.GetText())
		require.True(t, first.GetSentAt().AsTime().Equal(sentAt1))
		require.Nil(t, first.GetReadAt())

		second := resp.GetNotifications()[1]
		require.Equal(t, id2.String(), second.GetId())
		require.Equal(t, "upcoming_event", second.GetType())
		require.True(t, second.GetSentAt().AsTime().Equal(sentAt2))
		require.True(t, second.GetReadAt().AsTime().Equal(readAt))
	})

	t.Run("empty list returns empty response", func(t *testing.T) {
		svc, m := newAdminService(t)
		m.notifs.EXPECT().
			ListByUserID(gomock.Any(), adminRecipientUUID, 200, 0).
			Return(nil, nil)

		resp, err := svc.ListUserNotificationsV1(t.Context(), &pushpb.ListUserNotificationsV1Request{
			UserId: adminRecipientUUID.String(),
		})
		require.NoError(t, err)
		require.Empty(t, resp.GetNotifications())
	})

	t.Run("repo error is wrapped", func(t *testing.T) {
		svc, m := newAdminService(t)
		m.notifs.EXPECT().
			ListByUserID(gomock.Any(), adminRecipientUUID, 200, 0).
			Return(nil, assert.AnError)

		_, err := svc.ListUserNotificationsV1(t.Context(), &pushpb.ListUserNotificationsV1Request{
			UserId: adminRecipientUUID.String(),
		})
		require.Error(t, err)
	})
}
