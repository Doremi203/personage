package domain

import (
	"time"

	"github.com/google/uuid"
)

// TelegramContext represents the structure of Telegram event context
type TelegramContext struct {
	ChatID       string             `json:"chat_id"`
	ChatTitle    string             `json:"chat_title,omitempty"`
	MessageID    string             `json:"message_id"`
	From         TelegramUser       `json:"from"`
	Mentions     []TelegramUser     `json:"mentions,omitempty"`
	Text         string             `json:"text"`
	Participants []TelegramUser     `json:"participants,omitempty"`
	ReplyTo      *TelegramReplyInfo `json:"reply_to,omitempty"`
}

type TelegramUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
}

type TelegramReplyInfo struct {
	MessageID string `json:"message_id"`
	From      string `json:"from"`
}

// GmailContext represents the structure of Gmail event context
type GmailContext struct {
	ThreadID  string   `json:"thread_id"`
	MessageID string   `json:"message_id"`
	From      string   `json:"from"`
	To        []string `json:"to"`
	Cc        []string `json:"cc,omitempty"`
	Subject   string   `json:"subject"`
	Snippet   string   `json:"snippet"`
	Body      string   `json:"body,omitempty"` // May contain HTML
	Labels    []string `json:"labels,omitempty"`
}

// CalendarContext represents the structure of Google Calendar event context
type CalendarContext struct {
	CalendarID  string   `json:"calendar_id"`
	EventID     string   `json:"event_id"`
	Title       string   `json:"title"`
	Location    string   `json:"location,omitempty"`
	Attendees   []string `json:"attendees,omitempty"`
	Description string   `json:"description,omitempty"`
	StartTime   string   `json:"start_time"` // ISO8601
	EndTime     string   `json:"end_time"`   // ISO8601
	Recurring   bool     `json:"recurring,omitempty"`
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

// BatchEmbeddingRequest represents a batch of texts to embed
type BatchEmbeddingRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model,omitempty"` // Optional: specify model variant
}

// BatchEmbeddingResponse represents the result of a batch embedding request
type BatchEmbeddingResponse struct {
	Embeddings [][]float32            `json:"embeddings"`
	Metadata   BatchEmbeddingMetadata `json:"metadata"`
}

type BatchEmbeddingMetadata struct {
	Model       string        `json:"model"`
	TotalTokens int           `json:"total_tokens,omitempty"`
	Duration    time.Duration `json:"duration"`
}
