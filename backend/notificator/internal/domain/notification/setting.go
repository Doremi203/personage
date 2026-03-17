package notification

import "github.com/google/uuid"

// Setting represents a user's notification preference for a specific type.
type Setting struct {
	UserID  uuid.UUID
	Type    string
	Enabled bool
}
