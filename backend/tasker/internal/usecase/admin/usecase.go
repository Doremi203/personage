package admin

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

const adminListPageSize = 500

func NewUseCase(taskRepo domain.TaskRepo, moderationRepo domain.ManualModerationRepo) *UseCase {
	return &UseCase{
		taskRepo:       taskRepo,
		moderationRepo: moderationRepo,
	}
}

type UseCase struct {
	taskRepo       domain.TaskRepo
	moderationRepo domain.ManualModerationRepo
}

func (uc *UseCase) ListTasks(ctx context.Context, userID domain.UserID) ([]domain.Task, error) {
	tasks, _, err := uc.taskRepo.ListTasks(
		ctx,
		domain.TaskFilter{UserID: userID, IncludeUnapproved: true},
		domain.Pagination{Page: 1, PageSize: adminListPageSize},
	)
	if err != nil {
		return nil, errors.WrapFail(err, "list tasks for admin")
	}
	return tasks, nil
}

func (uc *UseCase) GetTask(ctx context.Context, taskID domain.TaskID, userID domain.UserID) (domain.Task, error) {
	task, err := uc.taskRepo.GetTaskByID(ctx, taskID, userID)
	if err != nil {
		return domain.Task{}, errors.WrapFail(err, "get task for admin")
	}
	return task, nil
}

func (uc *UseCase) UpdateTask(
	ctx context.Context,
	taskID domain.TaskID,
	userID domain.UserID,
	update domain.TaskUpdate,
) (domain.Task, error) {
	task, err := uc.taskRepo.UpdateTask(ctx, taskID, userID, update)
	if err != nil {
		return domain.Task{}, errors.WrapFail(err, "update task for admin")
	}
	return task, nil
}

func (uc *UseCase) Approve(ctx context.Context, taskID domain.TaskID, userID domain.UserID) (domain.Task, error) {
	approved := true
	return uc.UpdateTask(ctx, taskID, userID, domain.TaskUpdate{IsApproved: &approved})
}

func (uc *UseCase) ListModeratedUsers(ctx context.Context) ([]domain.UserID, error) {
	userIDs, err := uc.moderationRepo.ListUsers(ctx)
	if err != nil {
		return nil, errors.WrapFail(err, "list moderated users")
	}
	return userIDs, nil
}

func (uc *UseCase) SetUserModeration(ctx context.Context, userID domain.UserID, enabled bool) error {
	if enabled {
		if err := uc.moderationRepo.AddUser(ctx, userID); err != nil {
			return errors.WrapFailf(
				err,
				"enable manual moderation for user %s",
				errors.Token("user_id", userID.String()),
			)
		}
		return nil
	}
	if err := uc.moderationRepo.RemoveUser(ctx, userID); err != nil {
		return errors.WrapFailf(
			err,
			"disable manual moderation for user %s",
			errors.Token("user_id", userID.String()),
		)
	}
	return nil
}
