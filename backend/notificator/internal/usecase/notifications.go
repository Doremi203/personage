package usecase

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
)

const maxPageSize = 10

// NewNotifications creates a new Notifications use case.
func NewNotifications(repo notification.Repo) Notifications {
	return Notifications{repo: repo}
}

// Notifications encapsulates the business logic for listing notifications
// and toggling notification settings.
type Notifications struct {
	repo notification.Repo
}

// ListNotificationsParams holds the validated pagination parameters.
type ListNotificationsParams struct {
	UserID   uuid.UUID
	Page     int32
	PageSize int32
}

// List returns a paginated list of notifications for the given user.
func (u Notifications) List(ctx context.Context, params ListNotificationsParams) ([]notification.Notification, error) {
	pageSize := params.PageSize
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	offset := (params.Page - 1) * pageSize

	notifications, err := u.repo.ListByUserID(ctx, params.UserID, int(pageSize), int(offset))
	if err != nil {
		return nil, errors.WrapFailf(
			err,
			"list notifications for user %v",
			errors.Token("user_id", params.UserID),
		)
	}

	return notifications, nil
}

// Toggle flips the enabled state for the given notification type and user.
func (u Notifications) Toggle(ctx context.Context, userID uuid.UUID, notificationType string) (notification.Setting, error) {
	setting, err := u.repo.ToggleSetting(ctx, userID, notificationType)
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
