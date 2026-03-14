package domain

import (
	"context"
	"time"
)

type Notification struct {
	UserID UserID
	Title  string
	Body   string
}

//go:generate mockgen -source=notifications.go -destination=mock/notifications_mock.go -typed

type NotificationsService interface {
	Send(ctx context.Context, notification Notification) error
}

type NotificationConfig struct {
	UpcomingEventMinPriority int
	UpcomingEventIntervals   []time.Duration
	ScheduleChangesEnabled   bool
}

type UpcomingEventNotifier interface {
	NotifyUpcomingEvents(ctx context.Context, userID UserID, tasks []Task) error
}

type ScheduleChangeNotifier interface {
	NotifyScheduleChanges(ctx context.Context, userID UserID, changes []ScheduleChange) error
}

type ScheduleChange struct {
	TaskID     TaskID
	TaskTitle  string
	OldStart   *time.Time
	OldEnd     *time.Time
	NewStart   *time.Time
	NewEnd     *time.Time
	ChangeType ScheduleChangeType
	ChangedAt  time.Time
}

type ScheduleChangeType string

const (
	ScheduleChangeTypeNew      ScheduleChangeType = "new"
	ScheduleChangeTypeModified ScheduleChangeType = "modified"
	ScheduleChangeTypeDeleted  ScheduleChangeType = "deleted"
)
