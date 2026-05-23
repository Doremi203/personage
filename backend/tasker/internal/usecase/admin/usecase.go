package admin

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

const (
	adminListPageSize    = 500
	adminClusterListSize = 50
)

type PromptCacheInvalidator interface {
	Invalidate(id domain.PromptID)
}

func NewUseCase(
	taskRepo domain.TaskRepo,
	moderationRepo domain.ManualModerationRepo,
	clusterRepo domain.ClusterRepo,
	eventRepo domain.EventRepo,
	promptRepo domain.PromptRepo,
	promptCache PromptCacheInvalidator,
) *UseCase {
	return &UseCase{
		taskRepo:       taskRepo,
		moderationRepo: moderationRepo,
		clusterRepo:    clusterRepo,
		eventRepo:      eventRepo,
		promptRepo:     promptRepo,
		promptCache:    promptCache,
	}
}

type UseCase struct {
	taskRepo       domain.TaskRepo
	moderationRepo domain.ManualModerationRepo
	clusterRepo    domain.ClusterRepo
	eventRepo      domain.EventRepo
	promptRepo     domain.PromptRepo
	promptCache    PromptCacheInvalidator
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

func (uc *UseCase) ListClustersForUser(ctx context.Context, userID domain.UserID) ([]domain.AdminClusterListItem, error) {
	clusters, err := uc.clusterRepo.ListAdminClustersByUserID(ctx, userID, adminClusterListSize)
	if err != nil {
		return nil, errors.WrapFailf(
			err,
			"list admin clusters for user %s",
			errors.Token("user_id", userID.String()),
		)
	}
	return clusters, nil
}

func (uc *UseCase) ListClusterEvents(ctx context.Context, clusterID domain.ClusterID) ([]domain.Event, error) {
	events, err := uc.eventRepo.GetEventsByClusterID(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapFailf(
			err,
			"list events for cluster %s",
			errors.Token("cluster_id", clusterID.String()),
		)
	}
	return events, nil
}

func (uc *UseCase) ListPrompts(ctx context.Context) ([]domain.Prompt, error) {
	prompts, err := uc.promptRepo.ListPrompts(ctx)
	if err != nil {
		return nil, errors.WrapFail(err, "list prompts")
	}
	return prompts, nil
}

func (uc *UseCase) GetPrompt(ctx context.Context, id domain.PromptID) (domain.Prompt, error) {
	prompt, err := uc.promptRepo.GetPrompt(ctx, id)
	if err != nil {
		return domain.Prompt{}, errors.WrapFailf(
			err,
			"get prompt %s",
			errors.Token("prompt_id", id.String()),
		)
	}
	return prompt, nil
}

func (uc *UseCase) UpdatePrompt(
	ctx context.Context,
	id domain.PromptID,
	update domain.PromptUpdate,
) (domain.Prompt, error) {
	prompt, err := uc.promptRepo.UpdatePrompt(ctx, id, update)
	if err != nil {
		return domain.Prompt{}, errors.WrapFailf(
			err,
			"update prompt %s",
			errors.Token("prompt_id", id.String()),
		)
	}
	uc.promptCache.Invalidate(id)
	return prompt, nil
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
