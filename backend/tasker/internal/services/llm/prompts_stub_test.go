package llm

import (
	"context"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

const (
	stubClassifierSystemTemplate = `You are a classifier.
Owner emails: {{.owner_emails}}
Owner name: {{.user_name}}
Respond as JSON with actionable and reason.`

	stubClassifierUserTemplate = "Events:\n{{.events}}"

	stubTaskGeneratorSystemTemplate = `You generate a task from events. Reply with JSON.`

	stubTaskGeneratorUserTemplate = "Events:\n{{.events}}"
)

type stubPromptProvider struct{}

func (stubPromptProvider) Get(_ context.Context, id domain.PromptID) (domain.Prompt, error) {
	switch id {
	case domain.PromptIDClassifier:
		return domain.Prompt{
			ID:             domain.PromptIDClassifier,
			SystemTemplate: stubClassifierSystemTemplate,
			UserTemplate:   stubClassifierUserTemplate,
		}, nil
	case domain.PromptIDTaskGenerator:
		return domain.Prompt{
			ID:             domain.PromptIDTaskGenerator,
			SystemTemplate: stubTaskGeneratorSystemTemplate,
			UserTemplate:   stubTaskGeneratorUserTemplate,
		}, nil
	}
	return domain.Prompt{}, domain.ErrPromptNotFound
}
