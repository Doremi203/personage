package notifications

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase"
	"github.com/google/uuid"
)

const maxPageSize = 10

func New(repo notification.Repo, clock func() time.Time) *Service {
	return &Service{repo: repo, clock: clock}
}

type Service struct {
	repo  notification.Repo
	clock func() time.Time
}

func (s *Service) List(ctx context.Context, params usecase.ListNotificationsParams) ([]notification.Notification, error) {
	pageSize := min(params.PageSize, maxPageSize)

	offset := (params.Page - 1) * pageSize

	notifications, err := s.repo.ListByUserID(ctx, params.UserID, int(pageSize), int(offset))
	if err != nil {
		return nil, errors.WrapFailf(
			err,
			"list notifications for user %v",
			errors.Token("user_id", params.UserID),
		)
	}

	return notifications, nil
}

func (s *Service) GetSettings(ctx context.Context, userID uuid.UUID) ([]notification.Setting, error) {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, errors.WrapFailf(
			err,
			"get notification settings for user %v",
			errors.Token("user_id", userID),
		)
	}

	return settings, nil
}

func (s *Service) MarkAsRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	if err := s.repo.MarkAsRead(ctx, notificationID, userID, s.clock().UTC()); err != nil {
		return errors.WrapFailf(
			err,
			"mark notification as read",
			errors.Token("user_id", userID),
			errors.Token("notification_id", notificationID),
		)
	}
	return nil
}

func (s *Service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.MarkAllAsRead(ctx, userID, s.clock().UTC()); err != nil {
		return errors.WrapFailf(
			err,
			"mark all notifications as read",
			errors.Token("user_id", userID),
		)
	}
	return nil
}

func (s *Service) Toggle(ctx context.Context, userID uuid.UUID, notificationType string) (notification.Setting, error) {
	if !notification.IsValidSettingType(notificationType) {
		return notification.Setting{}, notification.ErrInvalidSettingType
	}

	setting, err := s.repo.ToggleSetting(ctx, userID, notificationType)
	if err != nil {
		return notification.Setting{}, errors.WrapFailf(
			err,
			"toggle notification setting for user %v type %v",
			errors.Token("user_id", userID),
			errors.Token("type", notificationType),
		)
	}

	return setting, nil
}
