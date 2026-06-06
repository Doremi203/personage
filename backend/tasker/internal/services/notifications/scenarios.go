package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const timeLayout = "15:04"

func NewUpcomingEventNotifier(
	logger log.Logger,
	sender domain.NotificationsService,
	config domain.NotificationConfig,
	printer *message.Printer,
) (*upcomingEventNotifier, error) {
	err := errors.Join(
		message.Set(
			language.Russian,
			"через %d час",
			plural.Selectf(1, "%d",
				plural.One, "через %d час",
				plural.Few, "через %d часа",
				plural.Many, "через %d часов",
				plural.Other, "через %d часов",
			),
		),
		message.Set(
			language.Russian,
			"через %d минуту",
			plural.Selectf(1, "%d",
				plural.One, "через %d минуту",
				plural.Few, "через %d минуты",
				plural.Many, "через %d минут",
				plural.Other, "через %d минут",
			),
		),
		message.Set(
			language.Russian,
			"• Длительность: %d минут",
			plural.Selectf(1, "%d",
				plural.One, "• Длительность: %d минута",
				plural.Few, "• Длительность: %d минуты",
				plural.Many, "• Длительность: %d минут",
				plural.Other, "• Длительность: %d минут",
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &upcomingEventNotifier{
		sender:  sender,
		config:  config,
		printer: printer,
		logger:  logger,
		now:     time.Now,
	}, nil
}

type upcomingEventNotifier struct {
	sender  domain.NotificationsService
	config  domain.NotificationConfig
	printer *message.Printer
	logger  log.Logger
	now     func() time.Time
}

func (n *upcomingEventNotifier) NotifyUpcomingEvents(
	ctx context.Context,
	userID domain.UserID,
	tasks []domain.Task,
) error {
	now := n.now()

	n.logger.Infof(
		"NotifyUpcomingEvents start for user %s %s %s",
		errors.Token("user_id", userID.String()),
		errors.Token("task_count", len(tasks)),
		errors.Token("now", now.Format(time.RFC3339)),
	)

	sentCount := 0
	skippedNoStartTime := 0
	skippedBelowMinPriority := 0
	skippedNotPlanned := 0
	skippedUnapproved := 0
	for _, task := range tasks {
		if task.StartTime == nil {
			skippedNoStartTime++
			continue
		}
		if task.Priority < n.config.UpcomingEventMinPriority {
			skippedBelowMinPriority++
			continue
		}

		if task.Status != domain.TaskStatusPlanned {
			skippedNotPlanned++
			continue
		}

		if !task.IsApproved {
			skippedUnapproved++
			continue
		}

		for _, interval := range n.config.UpcomingEventIntervals {
			notificationTime := task.StartTime.Add(-interval)

			if !now.After(notificationTime.Add(-time.Minute)) || !now.Before(notificationTime.Add(time.Minute)) {
				continue
			}

			notification := domain.Notification{
				UserID:           userID,
				Title:            n.formatUpcomingEventTitle(task, interval),
				Body:             n.formatUpcomingEventBody(task),
				Type:             "upcoming_event",
				NotificationTime: &notificationTime,
			}

			n.logger.Infof(
				"sending upcoming event notification %s %s %s %s",
				errors.Token("user_id", userID.String()),
				errors.Token("task_id", task.ID.String()),
				errors.Token("interval", interval.String()),
				errors.Token("notification_time", notificationTime.Format(time.RFC3339)),
			)

			if err := n.sender.Send(ctx, notification); err != nil {
				return errors.WrapFailf(
					err,
					"send upcoming event notification for %s",
					errors.Token("task_id", task.ID),
				)
			}
			sentCount++
		}
	}

	n.logger.Infof(
		"NotifyUpcomingEvents done for user %s %s %s %s %s %s",
		errors.Token("user_id", userID.String()),
		errors.Token("sent_count", sentCount),
		errors.Token("skipped_no_start_time", skippedNoStartTime),
		errors.Token("skipped_below_min_priority", skippedBelowMinPriority),
		errors.Token("skipped_not_planned", skippedNotPlanned),
		errors.Token("skipped_unapproved", skippedUnapproved),
	)

	return nil
}

func (n *upcomingEventNotifier) formatUpcomingEventTitle(task domain.Task, interval time.Duration) string {
	title := fmt.Sprintf("⏰ [%s] ", task.Title)
	switch {
	case interval >= time.Hour:
		hours := int(interval.Hours())
		return title + n.printer.Sprintf("через %d час", hours)
	case interval >= time.Minute:
		minutes := int(interval.Minutes())
		return title + n.printer.Sprintf("через %d минуту", minutes)
	default:
		return title + "скоро начнется"
	}
}

func (n *upcomingEventNotifier) formatUpcomingEventBody(task domain.Task) string {
	timeStr := task.StartTime.Format(timeLayout)
	duration := int(task.Duration.Minutes())

	body := n.printer.Sprintf("Начало в %s ", timeStr)
	if duration > 0 {
		body += n.printer.Sprintf("• Длительность: %d минут", duration)
	}

	if task.Category != "" {
		body += n.printer.Sprintf(" • Категория: %s", task.Category.StringRu())
	}

	return body
}

func NewScheduleChangeNotifier(
	sender domain.NotificationsService,
	config domain.NotificationConfig,
) *scheduleChangeNotifier {
	return &scheduleChangeNotifier{
		sender: sender,
		config: config,
	}
}

type scheduleChangeNotifier struct {
	sender domain.NotificationsService
	config domain.NotificationConfig
}

func (n *scheduleChangeNotifier) NotifyScheduleChanges(
	ctx context.Context,
	userID domain.UserID,
	changes []domain.ScheduleChange,
) error {
	if !n.config.ScheduleChangesEnabled {
		return nil
	}

	for _, change := range changes {
		notification := domain.Notification{
			UserID: userID,
			Title:  n.formatScheduleChangeTitle(change),
			Body:   n.formatScheduleChangeBody(change),
			Type:   "schedule_change",
		}

		if err := n.sender.Send(ctx, notification); err != nil {
			return errors.WrapFailf(
				err,
				"failed to send schedule change notification for %s",
				errors.Token("task_id", change.TaskID),
			)
		}
	}

	return nil
}

func (n *scheduleChangeNotifier) formatScheduleChangeTitle(change domain.ScheduleChange) string {
	switch change.ChangeType {
	case domain.ScheduleChangeTypeNew:
		return "📅 Новая задача запланирована"
	case domain.ScheduleChangeTypeModified:
		return "📅 Задача перенесена"
	case domain.ScheduleChangeTypeDeleted:
		return "📅 Задача удалена из расписания"
	default:
		return "📅 Расписание обновлено"
	}
}

func (n *scheduleChangeNotifier) formatScheduleChangeBody(change domain.ScheduleChange) string {
	var body string

	switch change.ChangeType {
	case domain.ScheduleChangeTypeNew:
		body = fmt.Sprintf("Задача: %s\n", change.TaskTitle)
		if change.NewStart != nil {
			body += fmt.Sprintf("Время: %s - %s",
				change.NewStart.Format(timeLayout),
				change.NewEnd.Format(timeLayout))
		}

	case domain.ScheduleChangeTypeModified:
		body = fmt.Sprintf("Задача: %s\n", change.TaskTitle)
		if change.OldStart != nil && change.NewStart != nil {
			body += fmt.Sprintf("Изменено с %s-%s на %s-%s",
				change.OldStart.Format(timeLayout),
				change.OldEnd.Format(timeLayout),
				change.NewStart.Format(timeLayout),
				change.NewEnd.Format(timeLayout))
		} else if change.OldStart != nil {
			body += fmt.Sprintf("Удалено из %s-%s",
				change.OldStart.Format(timeLayout),
				change.OldEnd.Format(timeLayout))
		} else if change.NewStart != nil {
			body += fmt.Sprintf("Запланировано на %s-%s",
				change.NewStart.Format(timeLayout),
				change.NewEnd.Format(timeLayout))
		}

	case domain.ScheduleChangeTypeDeleted:
		body = fmt.Sprintf("Задача: %s\n", change.TaskTitle)
		if change.OldStart != nil {
			body += fmt.Sprintf("Было запланировано: %s - %s",
				change.OldStart.Format(timeLayout),
				change.OldEnd.Format(timeLayout))
		}
	}

	return body
}
