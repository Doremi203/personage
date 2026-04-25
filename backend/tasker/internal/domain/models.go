package domain

import (
	"time"

	"github.com/google/uuid"
)

type TelegramContext struct {
	ChatID       string             `json:"chat_id"`
	ChatTitle    string             `json:"chat_title,omitzero"`
	MessageID    string             `json:"message_id"`
	From         TelegramUser       `json:"from"`
	Mentions     []TelegramUser     `json:"mentions,omitzero"`
	Text         string             `json:"text"`
	Participants []TelegramUser     `json:"participants,omitzero"`
	ReplyTo      *TelegramReplyInfo `json:"reply_to,omitzero"`
}

type TelegramUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitzero"`
}

type TelegramReplyInfo struct {
	MessageID string `json:"message_id"`
	From      string `json:"from"`
}

type GmailContext struct {
	ThreadID  string   `json:"thread_id"`
	MessageID string   `json:"message_id"`
	From      string   `json:"from"`
	To        []string `json:"to"`
	Cc        []string `json:"cc,omitzero"`
	Subject   string   `json:"subject"`
	Snippet   string   `json:"snippet"`
	Body      string   `json:"body,omitzero"` // May contain HTML
	Labels    []string `json:"labels,omitzero"`
}

type CalendarContext struct {
	CalendarID  string   `json:"calendar_id"`
	EventID     string   `json:"event_id"`
	Title       string   `json:"title"`
	Location    string   `json:"location,omitzero"`
	Attendees   []string `json:"attendees,omitzero"`
	Description string   `json:"description,omitzero"`
	StartTime   string   `json:"start_time"` // ISO8601
	EndTime     string   `json:"end_time"`   // ISO8601
	Recurring   bool     `json:"recurring,omitzero"`
}

type SearchParams struct {
	UserID        uuid.UUID
	QueryVector   []float32
	TopK          int
	MinSimilarity float64
	Sources       []EventSource
	AfterDate     *time.Time
	BeforeDate    *time.Time
}

type BatchEmbeddingRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model,omitzero"` // Optional: specify model variant
}

type BatchEmbeddingResponse struct {
	Embeddings [][]float32            `json:"embeddings"`
	Metadata   BatchEmbeddingMetadata `json:"metadata"`
}

type BatchEmbeddingMetadata struct {
	Model       string        `json:"model"`
	TotalTokens int           `json:"total_tokens,omitzero"`
	Duration    time.Duration `json:"duration"`
}
