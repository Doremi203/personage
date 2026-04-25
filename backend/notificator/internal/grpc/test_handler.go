package grpc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
)

type testNotificationCreator interface {
	CreateAndReturnID(ctx context.Context, n notification.Notification) (uuid.UUID, error)
}

type testCreateNotificationRequest struct {
	UserID string `json:"user_id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Text   string `json:"text"`
}

type testCreateNotificationResponse struct {
	ID string `json:"id"`
}

func NewTestCreateNotificationHandler(repo testNotificationCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req testCreateNotificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.UserID == "" || req.Title == "" || req.Type == "" {
			http.Error(w, "user_id, title and type are required", http.StatusBadRequest)
			return
		}

		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			http.Error(w, "invalid user_id: must be a valid UUID", http.StatusBadRequest)
			return
		}

		n := notification.Notification{
			UserID: userID,
			Title:  req.Title,
			Type:   req.Type,
			Text:   req.Text,
		}

		id, err := repo.CreateAndReturnID(r.Context(), n)
		if err != nil {
			http.Error(w, "failed to create notification", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(testCreateNotificationResponse{ID: id.String()})
	}
}
