package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TicketStatus статус тикета
type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "open"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusWaiting    TicketStatus = "waiting_user"
	TicketStatusClosed     TicketStatus = "closed"
	TicketStatusRated      TicketStatus = "rated"
)

// TicketPriority приоритет тикета
type TicketPriority string

const (
	PriorityLow      TicketPriority = "low"
	PriorityNormal   TicketPriority = "normal"
	PriorityHigh     TicketPriority = "high"
	PriorityCritical TicketPriority = "critical"
)

// Ticket модель тикета
type Ticket struct {
	ID               primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	TicketNumber     string                 `bson:"ticket_number" json:"ticket_number"`
	UserID           int64                  `bson:"user_id" json:"user_id"`
	ChannelMessageID int                    `bson:"channel_message_id" json:"channel_message_id"`
	GroupMessageID   int                    `bson:"group_message_id" json:"group_message_id"`
	ThreadID         int                    `bson:"thread_id" json:"thread_id"`
	Status           TicketStatus           `bson:"status" json:"status"`
	Priority         TicketPriority         `bson:"priority" json:"priority"`
	Subject          string                 `bson:"subject" json:"subject"`
	Description      string                 `bson:"description" json:"description"`
	AssignedTo       *int64                 `bson:"assigned_to,omitempty" json:"assigned_to,omitempty"`
	Tags             []string               `bson:"tags" json:"tags"`
	Messages         []Message              `bson:"messages" json:"messages"`
	Survey           *Survey                `bson:"survey,omitempty" json:"survey,omitempty"`
	Metadata         map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt        time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time              `bson:"updated_at" json:"updated_at"`
	ClosedAt         *time.Time             `bson:"closed_at,omitempty" json:"closed_at,omitempty"`
	FirstResponseAt  *time.Time             `bson:"first_response_at,omitempty" json:"first_response_at,omitempty"`
	LastResponseAt   *time.Time             `bson:"last_response_at,omitempty" json:"last_response_at,omitempty"`
	ResolutionTime   *time.Duration         `bson:"resolution_time,omitempty" json:"resolution_time,omitempty"`
}

// Message сообщение в тикете
type Message struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MessageID   int                `bson:"message_id" json:"message_id"`
	FromUserID  int64              `bson:"from_user_id" json:"from_user_id"`
	FromSupport bool               `bson:"from_support" json:"from_support"`
	Text        string             `bson:"text" json:"text"`
	MessageType string             `bson:"message_type" json:"message_type"` // text, photo, document, etc.
	FileID      string             `bson:"file_id,omitempty" json:"file_id,omitempty"`
	ReplyTo     *int               `bson:"reply_to,omitempty" json:"reply_to,omitempty"`
	Edited      bool               `bson:"edited" json:"edited"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// Survey опрос по завершении тикета
type Survey struct {
	Rating         int        `bson:"rating" json:"rating"` // 1-5
	Comment        string     `bson:"comment" json:"comment"`
	Submitted      bool       `bson:"submitted" json:"submitted"`
	SubmittedAt    *time.Time `bson:"submitted_at,omitempty" json:"submitted_at,omitempty"`
	ReminderSent   bool       `bson:"reminder_sent" json:"reminder_sent"`
	ReminderSentAt *time.Time `bson:"reminder_sent_at,omitempty" json:"reminder_sent_at,omitempty"`
}

// CreateTicketRequest запрос на создание тикета
type CreateTicketRequest struct {
	UserID      int64          `json:"user_id" validate:"required"`
	Subject     string         `json:"subject"`
	Description string         `json:"description" validate:"required"`
	Priority    TicketPriority `json:"priority"`
	Tags        []string       `json:"tags"`
}

// UpdateTicketRequest запрос на обновление тикета
type UpdateTicketRequest struct {
	Status     *TicketStatus          `json:"status,omitempty"`
	Priority   *TicketPriority        `json:"priority,omitempty"`
	Subject    *string                `json:"subject,omitempty"`
	AssignedTo *int64                 `json:"assigned_to,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AddMessageRequest запрос на добавление сообщения
type AddMessageRequest struct {
	MessageID   int    `json:"message_id" validate:"required"`
	FromUserID  int64  `json:"from_user_id" validate:"required"`
	FromSupport bool   `json:"from_support"`
	Text        string `json:"text" validate:"required"`
	MessageType string `json:"message_type"`
	FileID      string `json:"file_id,omitempty"`
	ReplyTo     *int   `json:"reply_to,omitempty"`
}

// SurveyRequest запрос на отправку опроса
type SurveyRequest struct {
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment"`
}

// ToTicket конвертирует CreateTicketRequest в Ticket
func (r *CreateTicketRequest) ToTicket(ticketNumber string) *Ticket {
	now := time.Now()

	priority := r.Priority
	if priority == "" {
		priority = PriorityNormal
	}

	return &Ticket{
		ID:           primitive.NewObjectID(),
		TicketNumber: ticketNumber,
		UserID:       r.UserID,
		Status:       TicketStatusOpen,
		Priority:     priority,
		Subject:      r.Subject,
		Description:  r.Description,
		Tags:         r.Tags,
		Messages:     []Message{},
		Metadata:     make(map[string]interface{}),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// ToMessage конвертирует AddMessageRequest в Message
func (r *AddMessageRequest) ToMessage() *Message {
	now := time.Now()

	messageType := r.MessageType
	if messageType == "" {
		messageType = "text"
	}

	return &Message{
		ID:          primitive.NewObjectID(),
		MessageID:   r.MessageID,
		FromUserID:  r.FromUserID,
		FromSupport: r.FromSupport,
		Text:        r.Text,
		MessageType: messageType,
		FileID:      r.FileID,
		ReplyTo:     r.ReplyTo,
		Edited:      false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsOpen проверяет, открыт ли тикет
func (t *Ticket) IsOpen() bool {
	return t.Status == TicketStatusOpen ||
		t.Status == TicketStatusInProgress ||
		t.Status == TicketStatusWaiting
}

// IsClosed проверяет, закрыт ли тикет
func (t *Ticket) IsClosed() bool {
	return t.Status == TicketStatusClosed || t.Status == TicketStatusRated
}

// GetLastMessage возвращает последнее сообщение в тикете
func (t *Ticket) GetLastMessage() *Message {
	if len(t.Messages) == 0 {
		return nil
	}
	return &t.Messages[len(t.Messages)-1]
}

// GetMessagesFromSupport возвращает сообщения от поддержки
func (t *Ticket) GetMessagesFromSupport() []Message {
	var supportMessages []Message
	for _, msg := range t.Messages {
		if msg.FromSupport {
			supportMessages = append(supportMessages, msg)
		}
	}
	return supportMessages
}

// GetMessagesFromUser возвращает сообщения от пользователя
func (t *Ticket) GetMessagesFromUser() []Message {
	var userMessages []Message
	for _, msg := range t.Messages {
		if !msg.FromSupport {
			userMessages = append(userMessages, msg)
		}
	}
	return userMessages
}

// CalculateResponseTime вычисляет время первого ответа
func (t *Ticket) CalculateResponseTime() *time.Duration {
	if t.FirstResponseAt == nil {
		return nil
	}

	duration := t.FirstResponseAt.Sub(t.CreatedAt)
	return &duration
}

// CalculateResolutionTime вычисляет время решения
func (t *Ticket) CalculateResolutionTime() *time.Duration {
	if t.ClosedAt == nil {
		return nil
	}

	duration := t.ClosedAt.Sub(t.CreatedAt)
	return &duration
}
