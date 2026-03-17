package grpc

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/libs/go/token"
	pushpb "github.com/Doremi203/personage/backend/notificator/gen/api/push"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewNotificationsService creates a new gRPC service for the Notifications API.
func NewNotificationsService(
	notificationsUseCase usecase.Notifications,
	logger log.Logger,
) *notificationsService {
	return &notificationsService{
		notificationsUseCase: notificationsUseCase,
		logger:               logger,
	}
}

type notificationsService struct {
	notificationsUseCase usecase.Notifications

	logger log.Logger
	pushpb.UnimplementedNotificationsServer
}

func (s *notificationsService) RegisterToGateway(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) error {
	return pushpb.RegisterNotificationsHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (s *notificationsService) RegisterToServer(gRPC *grpc.Server) {
	pushpb.RegisterNotificationsServer(gRPC, s)
}

func (s *notificationsService) ListNotificationsV1(
	ctx context.Context,
	req *pushpb.ListNotificationsV1Request,
) (*pushpb.ListNotificationsV1Response, error) {
	t, ok := token.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	notifications, err := s.notificationsUseCase.List(ctx, usecase.ListNotificationsParams{
		UserID:   t.GetUserID(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, errors.WrapFail(err, "list notifications")
	}

	protoNotifications := make([]*pushpb.NotificationItem, 0, len(notifications))
	for _, n := range notifications {
		protoNotifications = append(protoNotifications, &pushpb.NotificationItem{
			Id:     n.ID.String(),
			Title:  n.Title,
			Type:   n.Type,
			Text:   n.Text,
			SentAt: timestamppb.New(n.SentAt),
		})
	}

	return &pushpb.ListNotificationsV1Response{
		Notifications: protoNotifications,
	}, nil
}

func (s *notificationsService) ToggleNotificationV1(
	ctx context.Context,
	req *pushpb.ToggleNotificationV1Request,
) (*pushpb.ToggleNotificationV1Response, error) {
	t, ok := token.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	setting, err := s.notificationsUseCase.Toggle(ctx, t.GetUserID(), req.GetType())
	if err != nil {
		return nil, errors.WrapFailf(
			err,
			"toggle notification type %v",
			errors.Token("type", req.GetType()),
		)
	}

	return &pushpb.ToggleNotificationV1Response{
		Enabled: setting.Enabled,
	}, nil
}
