package service

import (
	"context"
	"time"

	"github.com/memrook/oneway_subot/internal/repository"
	"github.com/memrook/oneway_subot/pkg/logger"
)

// analyticsService реализация AnalyticsService
type analyticsService struct {
	ticketRepo repository.TicketRepository
	userRepo   repository.UserRepository
	logger     logger.Logger
}

// NewAnalyticsService создает новый сервис аналитики
func NewAnalyticsService(ticketRepo repository.TicketRepository, userRepo repository.UserRepository, logger logger.Logger) AnalyticsService {
	return &analyticsService{
		ticketRepo: ticketRepo,
		userRepo:   userRepo,
		logger:     logger,
	}
}

// analyticsServiceStub временная заглушка для AnalyticsService
type analyticsServiceStub struct {
	logger logger.Logger
}

// NewAnalyticsServiceStub создает заглушку для AnalyticsService
func NewAnalyticsServiceStub(logger logger.Logger) AnalyticsService {
	return &analyticsServiceStub{
		logger: logger,
	}
}

// Реализация для полного AnalyticsService
func (s *analyticsService) GetTicketAnalytics(ctx context.Context, from, to time.Time) (*repository.TicketAnalytics, error) {
	// Базовая реализация - получаем аналитику из репозитория
	analytics, err := s.ticketRepo.GetAnalytics(ctx, from, to)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get ticket analytics")
		return nil, err
	}

	return analytics, nil
}

func (s *analyticsService) GetDailyReport(ctx context.Context, date time.Time) (*DailyReport, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	analytics, err := s.GetTicketAnalytics(ctx, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	return &DailyReport{
		Date:                date,
		TicketsCreated:      analytics.TotalTickets,
		TicketsResolved:     analytics.ClosedTickets,
		TicketsInProgress:   analytics.InProgressTickets,
		AverageResponseTime: analytics.AverageResponseTime,
		AverageRating:       analytics.AverageRating,
		TopAgents:           analytics.TopAgents,
	}, nil
}

func (s *analyticsService) GetWeeklyReport(ctx context.Context, startDate time.Time) (*WeeklyReport, error) {
	endDate := startDate.Add(7 * 24 * time.Hour)

	analytics, err := s.GetTicketAnalytics(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &WeeklyReport{
		StartDate:           startDate,
		EndDate:             endDate,
		TotalTickets:        analytics.TotalTickets,
		ResolvedTickets:     analytics.ClosedTickets,
		AverageResponseTime: analytics.AverageResponseTime,
		AverageRating:       analytics.AverageRating,
		TopAgents:           analytics.TopAgents,
	}, nil
}

func (s *analyticsService) GetMonthlyReport(ctx context.Context, year int, month int) (*MonthlyReport, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	analytics, err := s.GetTicketAnalytics(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &MonthlyReport{
		Year:                year,
		Month:               month,
		TotalTickets:        analytics.TotalTickets,
		ResolvedTickets:     analytics.ClosedTickets,
		AverageResponseTime: analytics.AverageResponseTime,
		AverageRating:       analytics.AverageRating,
		TopAgents:           analytics.TopAgents,
	}, nil
}

func (s *analyticsService) GetAgentPerformance(ctx context.Context, agentID int64, from, to time.Time) (*AgentPerformance, error) {
	// Базовая реализация
	return &AgentPerformance{
		AgentID: agentID,
		Period:  DateRange{From: from, To: to},
	}, nil
}

func (s *analyticsService) GetTopAgents(ctx context.Context, from, to time.Time, limit int) ([]repository.AgentStats, error) {
	analytics, err := s.GetTicketAnalytics(ctx, from, to)
	if err != nil {
		return nil, err
	}

	if len(analytics.TopAgents) > limit {
		return analytics.TopAgents[:limit], nil
	}

	return analytics.TopAgents, nil
}

// Заглушки для обратной совместимости
func (s *analyticsServiceStub) GetTicketAnalytics(ctx context.Context, from, to time.Time) (*repository.TicketAnalytics, error) {
	s.logger.Info("GetTicketAnalytics called (stub)")
	return &repository.TicketAnalytics{}, nil
}

func (s *analyticsServiceStub) GetDailyReport(ctx context.Context, date time.Time) (*DailyReport, error) {
	s.logger.Info("GetDailyReport called (stub)")
	return &DailyReport{}, nil
}

func (s *analyticsServiceStub) GetWeeklyReport(ctx context.Context, startDate time.Time) (*WeeklyReport, error) {
	s.logger.Info("GetWeeklyReport called (stub)")
	return &WeeklyReport{}, nil
}

func (s *analyticsServiceStub) GetMonthlyReport(ctx context.Context, year int, month int) (*MonthlyReport, error) {
	s.logger.Info("GetMonthlyReport called (stub)")
	return &MonthlyReport{}, nil
}

func (s *analyticsServiceStub) GetAgentPerformance(ctx context.Context, agentID int64, from, to time.Time) (*AgentPerformance, error) {
	s.logger.Info("GetAgentPerformance called (stub)")
	return &AgentPerformance{}, nil
}

func (s *analyticsServiceStub) GetTopAgents(ctx context.Context, from, to time.Time, limit int) ([]repository.AgentStats, error) {
	s.logger.Info("GetTopAgents called (stub)")
	return []repository.AgentStats{}, nil
}
