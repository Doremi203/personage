package grpc

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

type adminPromptItem struct {
	ID             string    `json:"id"`
	Description    string    `json:"description"`
	SystemTemplate string    `json:"systemTemplate"`
	UserTemplate   string    `json:"userTemplate"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func promptToAdminItem(p domain.Prompt) adminPromptItem {
	return adminPromptItem{
		ID:             p.ID.String(),
		Description:    p.Description,
		SystemTemplate: p.SystemTemplate,
		UserTemplate:   p.UserTemplate,
		UpdatedAt:      p.UpdatedAt,
	}
}

type adminUpdatePromptRequest struct {
	SystemTemplate *string `json:"systemTemplate,omitempty"`
	UserTemplate   *string `json:"userTemplate,omitempty"`
}

func writePromptError(w http.ResponseWriter, err error) {
	if stderrors.Is(err, domain.ErrPromptNotFound) {
		http.Error(w, "prompt not found", http.StatusNotFound)
		return
	}
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func NewAdminListPromptsHandler(uc adminPromptsUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		prompts, err := uc.ListPrompts(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		items := make([]adminPromptItem, 0, len(prompts))
		for _, p := range prompts {
			items = append(items, promptToAdminItem(p))
		}

		writeJSON(w, http.StatusOK, map[string]any{"prompts": items})
	}
}

func NewAdminGetPromptHandler(uc adminPromptsUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		promptID := r.PathValue("promptId")
		if promptID == "" {
			http.Error(w, "promptId is required", http.StatusBadRequest)
			return
		}

		prompt, err := uc.GetPrompt(r.Context(), domain.PromptID(promptID))
		if err != nil {
			writePromptError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"prompt": promptToAdminItem(prompt)})
	}
}

func NewAdminUpdatePromptHandler(uc adminPromptsUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		promptID := r.PathValue("promptId")
		if promptID == "" {
			http.Error(w, "promptId is required", http.StatusBadRequest)
			return
		}

		var req adminUpdatePromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.SystemTemplate == nil && req.UserTemplate == nil {
			http.Error(w, "at least one of systemTemplate or userTemplate is required", http.StatusBadRequest)
			return
		}

		prompt, err := uc.UpdatePrompt(r.Context(), domain.PromptID(promptID), domain.PromptUpdate{
			SystemTemplate: req.SystemTemplate,
			UserTemplate:   req.UserTemplate,
		})
		if err != nil {
			writePromptError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"prompt": promptToAdminItem(prompt)})
	}
}
