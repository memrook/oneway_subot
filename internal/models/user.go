package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User модель пользователя
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       int64              `bson:"user_id" json:"user_id"`
	FirstName    string             `bson:"first_name" json:"first_name"`
	LastName     string             `bson:"last_name" json:"last_name"`
	Username     string             `bson:"username" json:"username"`
	LanguageCode string             `bson:"language_code" json:"language_code"`
	IsPremium    bool               `bson:"is_premium" json:"is_premium"`
	IsBot        bool               `bson:"is_bot" json:"is_bot"`
	IsBlocked    bool               `bson:"is_blocked" json:"is_blocked"`
	BlockReason  string             `bson:"block_reason,omitempty" json:"block_reason,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// UserStats статистика пользователя
type UserStats struct {
	UserID          int64     `bson:"user_id" json:"user_id"`
	TotalTickets    int       `bson:"total_tickets" json:"total_tickets"`
	OpenTickets     int       `bson:"open_tickets" json:"open_tickets"`
	ClosedTickets   int       `bson:"closed_tickets" json:"closed_tickets"`
	AverageRating   float64   `bson:"average_rating" json:"average_rating"`
	TotalMessages   int       `bson:"total_messages" json:"total_messages"`
	LastActivity    time.Time `bson:"last_activity" json:"last_activity"`
	FirstTicketDate time.Time `bson:"first_ticket_date" json:"first_ticket_date"`
}

// CreateUserRequest запрос на создание пользователя
type CreateUserRequest struct {
	UserID       int64  `json:"user_id" validate:"required"`
	FirstName    string `json:"first_name" validate:"required"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	IsPremium    bool   `json:"is_premium"`
	IsBot        bool   `json:"is_bot"`
}

// UpdateUserRequest запрос на обновление пользователя
type UpdateUserRequest struct {
	FirstName    *string `json:"first_name,omitempty"`
	LastName     *string `json:"last_name,omitempty"`
	Username     *string `json:"username,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
	IsPremium    *bool   `json:"is_premium,omitempty"`
	IsBlocked    *bool   `json:"is_blocked,omitempty"`
	BlockReason  *string `json:"block_reason,omitempty"`
}

// ToUser конвертирует CreateUserRequest в User
func (r *CreateUserRequest) ToUser() *User {
	now := time.Now()
	return &User{
		ID:           primitive.NewObjectID(),
		UserID:       r.UserID,
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		Username:     r.Username,
		LanguageCode: r.LanguageCode,
		IsPremium:    r.IsPremium,
		IsBot:        r.IsBot,
		IsBlocked:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// GetDisplayName возвращает отображаемое имя пользователя
func (u *User) GetDisplayName() string {
	if u.Username != "" {
		return "@" + u.Username
	}

	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}

	if name == "" {
		return "User#" + string(rune(u.UserID))
	}

	return name
}

// IsActive проверяет, активен ли пользователь
func (u *User) IsActive() bool {
	return !u.IsBlocked && !u.IsBot
}
