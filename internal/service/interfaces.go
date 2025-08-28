package service

import (
	"context"
	"time"

	"github.com/memrook/oneway_subot/internal/models"
	"github.com/memrook/oneway_subot/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserService интерфейс для работы с пользователями
type UserService interface {
	CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error)
	GetUser(ctx context.Context, userID int64) (*models.User, error)
	UpdateUser(ctx context.Context, userID int64, req *models.UpdateUserRequest) error
	BlockUser(ctx context.Context, userID int64, reason string) error
	UnblockUser(ctx context.Context, userID int64) error
	GetUserStats(ctx context.Context, userID int64) (*models.UserStats, error)
	IsUserBlocked(ctx context.Context, userID int64) (bool, error)
	GetOrCreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error)
}

// TicketService интерфейс для работы с тикетами
type TicketService interface {
	CreateTicket(ctx context.Context, req *models.CreateTicketRequest) (*models.Ticket, error)
	GetTicket(ctx context.Context, ticketID primitive.ObjectID) (*models.Ticket, error)
	GetTicketByNumber(ctx context.Context, ticketNumber string) (*models.Ticket, error)
	GetUserTickets(ctx context.Context, userID int64, status models.TicketStatus) ([]*models.Ticket, error)
	GetActiveTicket(ctx context.Context, userID int64) (*models.Ticket, error)
	UpdateTicket(ctx context.Context, ticketID primitive.ObjectID, req *models.UpdateTicketRequest) error
	CloseTicket(ctx context.Context, ticketID primitive.ObjectID, closedBy int64) error
	AddMessage(ctx context.Context, ticketID primitive.ObjectID, req *models.AddMessageRequest) error
	SetChannelMessage(ctx context.Context, ticketID primitive.ObjectID, messageID int) error
	SetGroupMessage(ctx context.Context, ticketID primitive.ObjectID, messageID int, threadID int) error
	SubmitSurvey(ctx context.Context, ticketID primitive.ObjectID, req *models.SurveyRequest) error
	SearchTickets(ctx context.Context, filters repository.TicketFilters) ([]*models.Ticket, int64, error)
	GetOpenTickets(ctx context.Context) ([]*models.Ticket, error)
	GetTicketByChannelMessage(ctx context.Context, messageID int) (*models.Ticket, error)
	GetTicketByGroupMessage(ctx context.Context, messageID int) (*models.Ticket, error)
	GetTicketByThread(ctx context.Context, threadID int) (*models.Ticket, error)
	AssignTicket(ctx context.Context, ticketID primitive.ObjectID, assignedTo int64) error
	ChangeTicketStatus(ctx context.Context, ticketID primitive.ObjectID, status models.TicketStatus) error
}

// NotificationService интерфейс для уведомлений
type NotificationService interface {
	SendTicketCreated(ctx context.Context, ticket *models.Ticket, user *models.User) error
	SendTicketClosed(ctx context.Context, ticket *models.Ticket, user *models.User) error
	SendSurveyRequest(ctx context.Context, ticket *models.Ticket, user *models.User) error
	SendSurveyReminder(ctx context.Context, ticket *models.Ticket, user *models.User) error
	SendAdminNotification(ctx context.Context, message string, data map[string]interface{}) error
	SendUserMessage(ctx context.Context, userID int64, message string) error
	SendSupportMessage(ctx context.Context, chatID int64, message string, threadID int) error
}

// AnalyticsService интерфейс для аналитики
type AnalyticsService interface {
	GetTicketAnalytics(ctx context.Context, from, to time.Time) (*repository.TicketAnalytics, error)
	GetDailyReport(ctx context.Context, date time.Time) (*DailyReport, error)
	GetWeeklyReport(ctx context.Context, startDate time.Time) (*WeeklyReport, error)
	GetMonthlyReport(ctx context.Context, year int, month int) (*MonthlyReport, error)
	GetAgentPerformance(ctx context.Context, agentID int64, from, to time.Time) (*AgentPerformance, error)
	GetTopAgents(ctx context.Context, from, to time.Time, limit int) ([]repository.AgentStats, error)
}

// DailyReport ежедневный отчет
type DailyReport struct {
	Date                time.Time               `json:"date"`
	TicketsCreated      int64                   `json:"tickets_created"`
	TicketsResolved     int64                   `json:"tickets_resolved"`
	TicketsInProgress   int64                   `json:"tickets_in_progress"`
	AverageResponseTime time.Duration           `json:"average_response_time"`
	AverageRating       float64                 `json:"average_rating"`
	TopAgents           []repository.AgentStats `json:"top_agents"`
}

// WeeklyReport недельный отчет
type WeeklyReport struct {
	StartDate           time.Time               `json:"start_date"`
	EndDate             time.Time               `json:"end_date"`
	TotalTickets        int64                   `json:"total_tickets"`
	ResolvedTickets     int64                   `json:"resolved_tickets"`
	AverageResponseTime time.Duration           `json:"average_response_time"`
	AverageRating       float64                 `json:"average_rating"`
	DailyStats          []repository.DailyStats `json:"daily_stats"`
	TopAgents           []repository.AgentStats `json:"top_agents"`
}

// MonthlyReport месячный отчет
type MonthlyReport struct {
	Year                int                     `json:"year"`
	Month               int                     `json:"month"`
	TotalTickets        int64                   `json:"total_tickets"`
	ResolvedTickets     int64                   `json:"resolved_tickets"`
	AverageResponseTime time.Duration           `json:"average_response_time"`
	AverageRating       float64                 `json:"average_rating"`
	WeeklyStats         []WeeklyStats           `json:"weekly_stats"`
	TopAgents           []repository.AgentStats `json:"top_agents"`
	Trends              TrendAnalysis           `json:"trends"`
}

// WeeklyStats статистика по неделям
type WeeklyStats struct {
	Week            int     `json:"week"`
	TicketsCreated  int64   `json:"tickets_created"`
	TicketsResolved int64   `json:"tickets_resolved"`
	AverageRating   float64 `json:"average_rating"`
}

// TrendAnalysis анализ трендов
type TrendAnalysis struct {
	TicketVolumeChange   float64 `json:"ticket_volume_change"`   // Изменение объема тикетов в %
	RatingChange         float64 `json:"rating_change"`          // Изменение рейтинга в %
	ResponseTimeChange   float64 `json:"response_time_change"`   // Изменение времени ответа в %
	ResolutionTimeChange float64 `json:"resolution_time_change"` // Изменение времени решения в %
}

// AgentPerformance производительность агента
type AgentPerformance struct {
	AgentID               int64         `json:"agent_id"`
	Name                  string        `json:"name"`
	Period                DateRange     `json:"period"`
	TicketsHandled        int64         `json:"tickets_handled"`
	TicketsResolved       int64         `json:"tickets_resolved"`
	AverageResponseTime   time.Duration `json:"average_response_time"`
	AverageResolutionTime time.Duration `json:"average_resolution_time"`
	AverageRating         float64       `json:"average_rating"`
	TotalRatings          int64         `json:"total_ratings"`
	ProductivityScore     float64       `json:"productivity_score"`
	QualityScore          float64       `json:"quality_score"`
}

// DateRange диапазон дат
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// Services контейнер для всех сервисов
type Services struct {
	User         UserService
	Ticket       TicketService
	Notification NotificationService
	Analytics    AnalyticsService
}

// New создает новый контейнер сервисов
func New(
	user UserService,
	ticket TicketService,
	notification NotificationService,
	analytics AnalyticsService,
) *Services {
	return &Services{
		User:         user,
		Ticket:       ticket,
		Notification: notification,
		Analytics:    analytics,
	}
}
