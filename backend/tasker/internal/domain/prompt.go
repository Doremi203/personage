package domain

import (
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
)

var ErrPromptNotFound = errors.Error("prompt not found")

type PromptID string

func (id PromptID) String() string {
	return string(id)
}

const (
	PromptIDClassifier    PromptID = "classifier"
	PromptIDTaskGenerator PromptID = "task_generator"
)

type Prompt struct {
	ID             PromptID
	Description    string
	SystemTemplate string
	UserTemplate   string
	UpdatedAt      time.Time
}

type PromptUpdate struct {
	SystemTemplate *string
	UserTemplate   *string
}
