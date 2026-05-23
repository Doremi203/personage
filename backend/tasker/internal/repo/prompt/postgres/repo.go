package promptpostgres

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/slices"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/jackc/pgx/v5"
)

func NewRepo(client postgres.Client, clock func() time.Time) *repo {
	return &repo{
		client: client,
		clock:  clock,
	}
}

type repo struct {
	client postgres.Client
	clock  func() time.Time
}

func (r *repo) GetPrompt(ctx context.Context, id domain.PromptID) (domain.Prompt, error) {
	query := `
		SELECT prompt_id, description, system_template, user_template, updated_at
		FROM prompts
		WHERE prompt_id = $1
	`

	rows, err := r.client.Query(ctx, query, id)
	if err != nil {
		return domain.Prompt{}, errors.WrapFail(err, "query prompt")
	}
	defer rows.Close()

	entity, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[promptEntity])
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return domain.Prompt{}, domain.ErrPromptNotFound
		}
		return domain.Prompt{}, errors.WrapFail(err, "collect prompt row")
	}

	return entity.ToDomain(), nil
}

func (r *repo) ListPrompts(ctx context.Context) ([]domain.Prompt, error) {
	query := `
		SELECT prompt_id, description, system_template, user_template, updated_at
		FROM prompts
		ORDER BY prompt_id ASC
	`

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, errors.WrapFail(err, "query prompts")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[promptEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect prompt rows")
	}

	return slices.Map(entities, promptEntity.ToDomain), nil
}

func (r *repo) UpdatePrompt(
	ctx context.Context,
	id domain.PromptID,
	update domain.PromptUpdate,
) (domain.Prompt, error) {
	query := `
		UPDATE prompts
		SET system_template = COALESCE($2, system_template),
		    user_template   = COALESCE($3, user_template),
		    updated_at      = $4
		WHERE prompt_id = $1
		RETURNING prompt_id, description, system_template, user_template, updated_at
	`

	rows, err := r.client.Query(ctx, query, id, update.SystemTemplate, update.UserTemplate, r.clock())
	if err != nil {
		return domain.Prompt{}, errors.WrapFail(err, "update prompt")
	}
	defer rows.Close()

	entity, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[promptEntity])
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return domain.Prompt{}, domain.ErrPromptNotFound
		}
		return domain.Prompt{}, errors.WrapFail(err, "collect updated prompt row")
	}

	return entity.ToDomain(), nil
}

type promptEntity struct {
	PromptID       string    `db:"prompt_id"`
	Description    string    `db:"description"`
	SystemTemplate string    `db:"system_template"`
	UserTemplate   string    `db:"user_template"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func (e promptEntity) ToDomain() domain.Prompt {
	return domain.Prompt{
		ID:             domain.PromptID(e.PromptID),
		Description:    e.Description,
		SystemTemplate: e.SystemTemplate,
		UserTemplate:   e.UserTemplate,
		UpdatedAt:      e.UpdatedAt,
	}
}
