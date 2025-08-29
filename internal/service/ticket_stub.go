package service

import (
	"context"
	"fmt"
	"time"

	"github.com/memrook/oneway_subot/internal/models"
	"github.com/memrook/oneway_subot/internal/repository"
	"github.com/memrook/oneway_subot/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ticketService реализация TicketService
type ticketService struct {
	repo   repository.TicketRepository
	logger logger.Logger
}

// NewTicketService создает новый сервис тикетов
func NewTicketService(repo repository.TicketRepository, logger logger.Logger) TicketService {
	return &ticketService{
		repo:   repo,
		logger: logger,
	}
}

// NewTicketServiceStub создает заглушку для TicketService (для обратной совместимости)
func NewTicketServiceStub(repo repository.TicketRepository, logger logger.Logger) TicketService {
	return NewTicketService(repo, logger)
}

func (s *ticketService) CreateTicket(ctx context.Context, req *models.CreateTicketRequest) (*models.Ticket, error) {
	// Валидация
	if req.UserID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if req.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if req.Description == "" {
		return nil, fmt.Errorf("description is required")
	}

	// Проверяем, нет ли уже активного тикета у пользователя
	activeTicket, err := s.repo.GetActiveByUserID(ctx, req.UserID)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to check active ticket: %w", err)
	}
	if activeTicket != nil {
		return nil, fmt.Errorf("user already has an active ticket: %s", activeTicket.TicketNumber)
	}

	// Создаем тикет
	ticket := &models.Ticket{
		ID:          primitive.NewObjectID(),
		UserID:      req.UserID,
		Subject:     req.Subject,
		Description: req.Description,
		Status:      models.TicketStatusOpen,
		Priority:    req.Priority,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Messages:    []models.Message{},
	}

	// Устанавливаем приоритет по умолчанию
	if ticket.Priority == "" {
		ticket.Priority = models.PriorityNormal
	}

	// Сохраняем в базу
	if err := s.repo.Create(ctx, ticket); err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":     ticket.ID.Hex(),
		"ticket_number": ticket.TicketNumber,
		"user_id":       ticket.UserID,
		"subject":       ticket.Subject,
		"priority":      ticket.Priority,
	}).Info("Ticket created successfully")

	return ticket, nil
}

func (s *ticketService) GetTicket(ctx context.Context, ticketID primitive.ObjectID) (*models.Ticket, error) {
	if ticketID.IsZero() {
		return nil, fmt.Errorf("ticket ID is required")
	}

	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) GetTicketByNumber(ctx context.Context, ticketNumber string) (*models.Ticket, error) {
	if ticketNumber == "" {
		return nil, fmt.Errorf("ticket number is required")
	}

	ticket, err := s.repo.GetByTicketNumber(ctx, ticketNumber)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) GetUserTickets(ctx context.Context, userID int64, status models.TicketStatus) ([]*models.Ticket, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}

	filters := repository.TicketFilters{
		UserID: &userID,
	}

	if status != "" {
		filters.Status = &status
	}

	tickets, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tickets: %w", err)
	}

	return tickets, nil
}

func (s *ticketService) GetActiveTicket(ctx context.Context, userID int64) (*models.Ticket, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}

	ticket, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Нет активного тикета
		}
		return nil, fmt.Errorf("failed to get active ticket: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) UpdateTicket(ctx context.Context, ticketID primitive.ObjectID, req *models.UpdateTicketRequest) error {
	if ticketID.IsZero() {
		return fmt.Errorf("ticket ID is required")
	}

	// Проверяем существование тикета
	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("ticket not found")
		}
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	if err := s.repo.Update(ctx, ticketID, req); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":     ticketID.Hex(),
		"ticket_number": ticket.TicketNumber,
	}).Info("Ticket updated successfully")

	return nil
}

func (s *ticketService) CloseTicket(ctx context.Context, ticketID primitive.ObjectID, closedBy int64) error {
	if ticketID.IsZero() {
		return fmt.Errorf("ticket ID is required")
	}
	if closedBy == 0 {
		return fmt.Errorf("closedBy is required")
	}

	// Проверяем существование тикета
	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("ticket not found")
		}
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// Проверяем, что тикет не закрыт
	if ticket.Status == models.TicketStatusClosed || ticket.Status == models.TicketStatusRated {
		return fmt.Errorf("ticket is already closed")
	}

	if err := s.repo.Close(ctx, ticketID, closedBy); err != nil {
		return fmt.Errorf("failed to close ticket: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":     ticketID.Hex(),
		"ticket_number": ticket.TicketNumber,
		"closed_by":     closedBy,
	}).Info("Ticket closed successfully")

	return nil
}

func (s *ticketService) AddMessage(ctx context.Context, ticketID primitive.ObjectID, req *models.AddMessageRequest) error {
	if ticketID.IsZero() {
		return fmt.Errorf("ticket ID is required")
	}
	if req.FromUserID == 0 {
		return fmt.Errorf("from user ID is required")
	}
	if req.Text == "" && req.MessageType != "file" {
		return fmt.Errorf("message text is required")
	}

	// Проверяем существование тикета
	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("ticket not found")
		}
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// Проверяем, что тикет не закрыт
	if ticket.Status == models.TicketStatusClosed || ticket.Status == models.TicketStatusRated {
		return fmt.Errorf("cannot add message to closed ticket")
	}

	// Добавляем сообщение
	message := models.Message{
		ID:          primitive.NewObjectID(),
		FromUserID:  req.FromUserID,
		Text:        req.Text,
		MessageType: req.MessageType,
		FromSupport: req.FromSupport,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.AddMessage(ctx, ticketID, &message); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	// Обновляем статус тикета если это первый ответ от админа
	if req.FromSupport && ticket.Status == models.TicketStatusOpen {
		if err := s.repo.UpdateStatus(ctx, ticketID, models.TicketStatusInProgress); err != nil {
			s.logger.WithError(err).Error("Failed to update ticket status to in_progress")
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":       ticketID.Hex(),
		"ticket_number":   ticket.TicketNumber,
		"from_user_id":    req.FromUserID,
		"is_from_support": req.FromSupport,
		"message_type":    req.MessageType,
	}).Info("Message added to ticket")

	return nil
}

func (s *ticketService) SetChannelMessage(ctx context.Context, ticketID primitive.ObjectID, messageID int) error {
	if ticketID.IsZero() {
		return fmt.Errorf("ticket ID is required")
	}
	if messageID == 0 {
		return fmt.Errorf("message ID is required")
	}

	if err := s.repo.SetChannelMessageID(ctx, ticketID, messageID); err != nil {
		return fmt.Errorf("failed to set channel message: %w", err)
	}

	return nil
}

func (s *ticketService) SetGroupMessage(ctx context.Context, ticketID primitive.ObjectID, messageID int, threadID int) error {
	if ticketID.IsZero() {
		return fmt.Errorf("ticket ID is required")
	}
	if messageID == 0 {
		return fmt.Errorf("message ID is required")
	}

	if err := s.repo.SetGroupMessageID(ctx, ticketID, messageID, threadID); err != nil {
		return fmt.Errorf("failed to set group message: %w", err)
	}

	return nil
}

func (s *ticketService) SubmitSurvey(ctx context.Context, ticketID primitive.ObjectID, req *models.SurveyRequest) error {
	if ticketID.IsZero() {
		return fmt.Errorf("ticket ID is required")
	}
	if req.Rating < 1 || req.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	// Проверяем существование тикета
	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("ticket not found")
		}
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// Проверяем, что тикет закрыт но не оценен
	if ticket.Status != models.TicketStatusClosed {
		return fmt.Errorf("can only rate closed tickets")
	}

	survey := &models.Survey{
		Rating:  req.Rating,
		Comment: req.Comment,
	}
	if err := s.repo.SetSurvey(ctx, ticketID, survey); err != nil {
		return fmt.Errorf("failed to submit survey: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":     ticketID.Hex(),
		"ticket_number": ticket.TicketNumber,
		"rating":        req.Rating,
		"has_comment":   req.Comment != "",
	}).Info("Survey submitted successfully")

	return nil
}

func (s *ticketService) SearchTickets(ctx context.Context, filters repository.TicketFilters) ([]*models.Ticket, int64, error) {
	tickets, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search tickets: %w", err)
	}

	total, err := s.repo.Count(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search tickets: %w", err)
	}

	return tickets, total, nil
}

func (s *ticketService) GetOpenTickets(ctx context.Context) ([]*models.Ticket, error) {
	tickets, err := s.repo.GetOpenTickets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get open tickets: %w", err)
	}

	return tickets, nil
}

func (s *ticketService) GetTicketByChannelMessage(ctx context.Context, messageID int) (*models.Ticket, error) {
	if messageID == 0 {
		return nil, fmt.Errorf("message ID is required")
	}

	ticket, err := s.repo.GetByChannelMessageID(ctx, messageID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Тикет не найден
		}
		return nil, fmt.Errorf("failed to get ticket by channel message: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) GetTicketByGroupMessage(ctx context.Context, messageID int) (*models.Ticket, error) {
	if messageID == 0 {
		return nil, fmt.Errorf("message ID is required")
	}

	ticket, err := s.repo.GetByGroupMessageID(ctx, messageID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Тикет не найден
		}
		return nil, fmt.Errorf("failed to get ticket by group message: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) GetTicketByThread(ctx context.Context, threadID int) (*models.Ticket, error) {
	if threadID == 0 {
		return nil, fmt.Errorf("thread ID is required")
	}

	ticket, err := s.repo.GetByThreadID(ctx, threadID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Тикет не найден
		}
		return nil, fmt.Errorf("failed to get ticket by thread: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) AssignTicket(ctx context.Context, ticketID primitive.ObjectID, assignedTo int64) error {
	if ticketID.IsZero() {
		return fmt.Errorf("ticket ID is required")
	}
	if assignedTo == 0 {
		return fmt.Errorf("assigned to user ID is required")
	}

	// Проверяем существование тикета
	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("ticket not found")
		}
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	updateReq := &models.UpdateTicketRequest{
		AssignedTo: &assignedTo,
	}
	if err := s.repo.Update(ctx, ticketID, updateReq); err != nil {
		return fmt.Errorf("failed to assign ticket: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":     ticketID.Hex(),
		"ticket_number": ticket.TicketNumber,
		"assigned_to":   assignedTo,
	}).Info("Ticket assigned successfully")

	return nil
}

func (s *ticketService) ChangeTicketStatus(ctx context.Context, ticketID primitive.ObjectID, status models.TicketStatus) error {
	if ticketID.IsZero() {
		return fmt.Errorf("ticket ID is required")
	}
	if status == "" {
		return fmt.Errorf("status is required")
	}

	// Проверяем существование тикета
	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("ticket not found")
		}
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, ticketID, status); err != nil {
		return fmt.Errorf("failed to change ticket status: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":     ticketID.Hex(),
		"ticket_number": ticket.TicketNumber,
		"old_status":    ticket.Status,
		"new_status":    status,
	}).Info("Ticket status changed successfully")

	return nil
}
