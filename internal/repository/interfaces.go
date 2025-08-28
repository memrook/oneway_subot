package repository

import (
	"context"
	"time"

	"github.com/memrook/oneway_subot/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRepository интерфейс для работы с пользователями
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByUserID(ctx context.Context, userID int64) (*models.User, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	Update(ctx context.Context, userID int64, update *models.UpdateUserRequest) error
	Delete(ctx context.Context, userID int64) error
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	GetStats(ctx context.Context, userID int64) (*models.UserStats, error)
	Block(ctx context.Context, userID int64, reason string) error
	Unblock(ctx context.Context, userID int64) error
	GetActiveUsers(ctx context.Context, since time.Time) ([]*models.User, error)
}

// TicketRepository интерфейс для работы с тикетами
type TicketRepository interface {
	Create(ctx context.Context, ticket *models.Ticket) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Ticket, error)
	GetByTicketNumber(ctx context.Context, ticketNumber string) (*models.Ticket, error)
	GetByUserID(ctx context.Context, userID int64, status models.TicketStatus) ([]*models.Ticket, error)
	GetActiveByUserID(ctx context.Context, userID int64) (*models.Ticket, error)
	GetByChannelMessageID(ctx context.Context, messageID int) (*models.Ticket, error)
	GetByGroupMessageID(ctx context.Context, messageID int) (*models.Ticket, error)
	GetByThreadID(ctx context.Context, threadID int) (*models.Ticket, error)
	Update(ctx context.Context, id primitive.ObjectID, update *models.UpdateTicketRequest) error
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.TicketStatus) error
	Close(ctx context.Context, id primitive.ObjectID, closedBy int64) error
	AddMessage(ctx context.Context, ticketID primitive.ObjectID, message *models.Message) error
	UpdateMessage(ctx context.Context, ticketID primitive.ObjectID, messageID int, text string) error
	SetChannelMessageID(ctx context.Context, id primitive.ObjectID, messageID int) error
	SetGroupMessageID(ctx context.Context, id primitive.ObjectID, messageID int, threadID int) error
	SetFirstResponseTime(ctx context.Context, id primitive.ObjectID, responseTime time.Time) error
	SetSurvey(ctx context.Context, id primitive.ObjectID, survey *models.Survey) error
	List(ctx context.Context, filters TicketFilters) ([]*models.Ticket, error)
	Count(ctx context.Context, filters TicketFilters) (int64, error)
	GetOpenTickets(ctx context.Context) ([]*models.Ticket, error)
	GetTicketsForSurvey(ctx context.Context, since time.Time) ([]*models.Ticket, error)
	GetAnalytics(ctx context.Context, from, to time.Time) (*TicketAnalytics, error)
}

// TicketFilters фильтры для поиска тикетов
type TicketFilters struct {
	UserID     *int64                 `json:"user_id,omitempty"`
	Status     *models.TicketStatus   `json:"status,omitempty"`
	Priority   *models.TicketPriority `json:"priority,omitempty"`
	AssignedTo *int64                 `json:"assigned_to,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	FromDate   *time.Time             `json:"from_date,omitempty"`
	ToDate     *time.Time             `json:"to_date,omitempty"`
	Search     string                 `json:"search,omitempty"`
	Limit      int                    `json:"limit,omitempty"`
	Offset     int                    `json:"offset,omitempty"`
	SortBy     string                 `json:"sort_by,omitempty"`
	SortOrder  string                 `json:"sort_order,omitempty"`
}

// TicketAnalytics аналитика по тикетам
type TicketAnalytics struct {
	TotalTickets          int64            `json:"total_tickets"`
	OpenTickets           int64            `json:"open_tickets"`
	ClosedTickets         int64            `json:"closed_tickets"`
	InProgressTickets     int64            `json:"in_progress_tickets"`
	TicketsByPriority     map[string]int64 `json:"tickets_by_priority"`
	TicketsByStatus       map[string]int64 `json:"tickets_by_status"`
	AverageResponseTime   time.Duration    `json:"average_response_time"`
	AverageResolutionTime time.Duration    `json:"average_resolution_time"`
	AverageRating         float64          `json:"average_rating"`
	TotalRatings          int64            `json:"total_ratings"`
	RatingDistribution    map[int]int64    `json:"rating_distribution"`
	TopAgents             []AgentStats     `json:"top_agents"`
	DailyStats            []DailyStats     `json:"daily_stats"`
}

// AgentStats статистика по агентам поддержки
type AgentStats struct {
	UserID              int64         `json:"user_id"`
	Name                string        `json:"name"`
	TicketsHandled      int64         `json:"tickets_handled"`
	TicketsResolved     int64         `json:"tickets_resolved"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	AverageRating       float64       `json:"average_rating"`
	TotalRatings        int64         `json:"total_ratings"`
}

// DailyStats ежедневная статистика
type DailyStats struct {
	Date            time.Time `json:"date"`
	TicketsCreated  int64     `json:"tickets_created"`
	TicketsResolved int64     `json:"tickets_resolved"`
	ActiveTickets   int64     `json:"active_tickets"`
	AverageRating   float64   `json:"average_rating"`
}

// Repositories контейнер для всех репозиториев
type Repositories struct {
	User   UserRepository
	Ticket TicketRepository
}

// New создает новый контейнер репозиториев
func New(user UserRepository, ticket TicketRepository) *Repositories {
	return &Repositories{
		User:   user,
		Ticket: ticket,
	}
}
