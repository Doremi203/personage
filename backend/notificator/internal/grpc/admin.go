package grpc

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// adminNotificationType is the default in-app notification type assigned to
// admin-broadcast pushes when the caller omits Notification.type.
const adminNotificationType = "admin"

// adminListNotificationsLimit caps how many notifications the per-user admin
// list endpoint returns. The admin UI does not paginate yet, so we surface
// recent history without unbounded scans.
const adminListNotificationsLimit = 200

func NewAdminService(
	pushRepo push.Repo,
	pushSender usecase.PushSender,
	notificationRepo notification.Repo,
	clock func() time.Time,
	logger log.Logger,
) *adminService {
	return &adminService{
		pushRepo:         pushRepo,
		pushSender:       pushSender,
		notificationRepo: notificationRepo,
		clock:            clock,
		logger:           logger,
	}
}

type adminService struct {
	pushRepo         push.Repo
	pushSender       usecase.PushSender
	notificationRepo notification.Repo
	clock            func() time.Time

	logger log.Logger
	pushpb.UnimplementedAdminServer
}

func (s *adminService) RegisterToGateway(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) error {
	return pushpb.RegisterAdminHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (s *adminService) RegisterToServer(gRPC *grpc.Server) {
	pushpb.RegisterAdminServer(gRPC, s)
}

func (s *adminService) SendPushV1(
	ctx context.Context,
	req *pushpb.SendPushV1Request,
) (*pushpb.SendPushV1Response, error) {
	n := req.GetNotification()
	if n.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "notification.title is required")
	}

	typ := n.GetType()
	if typ == "" {
		typ = adminNotificationType
	}

	recipients, err := s.resolveRecipients(ctx, n.GetRecipientId())
	if err != nil {
		return nil, err
	}

	text := n.GetDetailedText()
	if text == "" {
		text = n.GetBody()
	}

	for _, recipient := range recipients {
		if err := s.deliver(ctx, recipient, n, typ, text); err != nil {
			s.logger.Error(errors.WrapFailf(
				err,
				"deliver admin push to %v",
				errors.Token("recipient_id", recipient.ID),
			))
			continue
		}
	}

	return &pushpb.SendPushV1Response{}, nil
}

func (s *adminService) ListUserNotificationsV1(
	ctx context.Context,
	req *pushpb.ListUserNotificationsV1Request,
) (*pushpb.ListUserNotificationsV1Response, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id: must be a UUID")
	}

	notifications, err := s.notificationRepo.ListByUserID(ctx, userID, adminListNotificationsLimit, 0)
	if err != nil {
		return nil, errors.WrapFailf(
			err,
			"list notifications for user",
			errors.Token("user_id", userID.String()),
		)
	}

	protoNotifications := make([]*pushpb.NotificationItem, 0, len(notifications))
	for _, n := range notifications {
		item := &pushpb.NotificationItem{
			Id:    n.ID.String(),
			Title: n.Title,
			Type:  n.Type,
			Text:  n.Text,
		}
		if n.SentAt != nil {
			item.SentAt = timestamppb.New(*n.SentAt)
		}
		if n.ReadAt != nil {
			item.ReadAt = timestamppb.New(*n.ReadAt)
		}
		protoNotifications = append(protoNotifications, item)
	}

	return &pushpb.ListUserNotificationsV1Response{
		Notifications: protoNotifications,
	}, nil
}

func (s *adminService) deliver(
	ctx context.Context,
	recipient push.Recipient,
	n *pushpb.Notification,
	typ, text string,
) error {
	scopedKey := scopedIdempotencyKey(n.GetIdempotencyKey(), recipient.ID)
	sent, err := notification.NewSent(
		uuid.UUID(recipient.ID),
		n.GetTitle(),
		typ,
		text,
		s.clock(),
		scopedKey,
	)
	if err != nil {
		return errors.WrapFail(err, "build notification")
	}

	inserted, err := s.notificationRepo.CreateIfAbsent(ctx, sent)
	if err != nil {
		return errors.WrapFail(err, "persist notification")
	}
	if !inserted {
		s.logger.Infof(
			"duplicate admin push skipped for recipient %v key %v",
			errors.Token("recipient_id", recipient.ID),
			errors.Token("idempotency_key", scopedKey),
		)
		return nil
	}

	if len(recipient.Subscriptions) == 0 {
		return nil
	}

	return s.pushSender.Send(ctx, recipient, push.Push{
		Title: n.GetTitle(),
		Body:  n.GetBody(),
		Url:   n.GetUrl(),
		Icon:  n.GetIcon(),
	})
}

// scopedIdempotencyKey scopes a caller-supplied key by recipient so that a
// single broadcast call inserts one in-app notification per recipient (the
// idempotency_key column is globally unique). Empty in stays empty out.
func scopedIdempotencyKey(key string, recipientID push.RecipientID) string {
	if key == "" {
		return ""
	}
	return key + ":" + recipientID.String()
}

func (s *adminService) resolveRecipients(ctx context.Context, recipientIDStr string) ([]push.Recipient, error) {
	if recipientIDStr == "" {
		all, err := s.pushRepo.GetAllRecipients(ctx)
		if err != nil {
			return nil, errors.WrapFail(err, "get all recipients")
		}
		return all, nil
	}

	recipientUUID, err := uuid.Parse(recipientIDStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid recipient_id: must be a UUID")
	}
	recipientID := push.RecipientID(recipientUUID)

	subs, err := s.pushRepo.GetSubscriptionsByRecipientID(ctx, recipientID)
	if err != nil {
		return nil, errors.WrapFailf(
			err,
			"get subscriptions by recipient id",
			errors.Token("recipient_id", recipientID),
		)
	}
	if len(subs) == 0 {
		s.logger.Infof("no push subscriptions for recipient %s, persisting in-app only", recipientID)
	}
	return []push.Recipient{{ID: recipientID, Subscriptions: subs}}, nil
}
