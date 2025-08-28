package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/memrook/oneway_subot/internal/models"
	"github.com/memrook/oneway_subot/internal/repository"
	"github.com/memrook/oneway_subot/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ticketRepository реализация TicketRepository для MongoDB
type ticketRepository struct {
	collection *mongo.Collection
	logger     logger.Logger
}

// NewTicketRepository создает новый репозиторий тикетов
func NewTicketRepository(db *Database, log logger.Logger) repository.TicketRepository {
	return &ticketRepository{
		collection: db.GetCollection("tickets"),
		logger:     log,
	}
}

// Create создает новый тикет
func (r *ticketRepository) Create(ctx context.Context, ticket *models.Ticket) error {
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()

	if ticket.ID.IsZero() {
		ticket.ID = primitive.NewObjectID()
	}

	// Генерация номера тикета если не указан
	if ticket.TicketNumber == "" {
		ticketNumber, err := r.generateTicketNumber(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate ticket number: %w", err)
		}
		ticket.TicketNumber = ticketNumber
	}

	_, err := r.collection.InsertOne(ctx, ticket)
	if err != nil {
		return fmt.Errorf("failed to create ticket: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"ticket_id":     ticket.ID.Hex(),
		"ticket_number": ticket.TicketNumber,
		"user_id":       ticket.UserID,
		"status":        ticket.Status,
	}).Info("Ticket created successfully")

	return nil
}

// generateTicketNumber генерирует уникальный номер тикета
func (r *ticketRepository) generateTicketNumber(ctx context.Context) (string, error) {
	// Получаем количество тикетов + 1
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return "", err
	}

	// Формат: YYYYMMDD-NNNN
	today := time.Now().Format("20060102")
	sequence := fmt.Sprintf("%04d", count+1)

	return fmt.Sprintf("%s-%s", today, sequence), nil
}

// GetByID получает тикет по ID
func (r *ticketRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Ticket, error) {
	filter := bson.M{"_id": id}

	var ticket models.Ticket
	err := r.collection.FindOne(ctx, filter).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket with ID %s not found", id.Hex())
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return &ticket, nil
}

// GetByTicketNumber получает тикет по номеру
func (r *ticketRepository) GetByTicketNumber(ctx context.Context, ticketNumber string) (*models.Ticket, error) {
	filter := bson.M{"ticket_number": ticketNumber}

	var ticket models.Ticket
	err := r.collection.FindOne(ctx, filter).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket with number %s not found", ticketNumber)
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return &ticket, nil
}

// GetByUserID получает тикеты пользователя по статусу
func (r *ticketRepository) GetByUserID(ctx context.Context, userID int64, status models.TicketStatus) ([]*models.Ticket, error) {
	filter := bson.M{"user_id": userID}
	if status != "" {
		filter["status"] = status
	}

	opts := options.Find().SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find tickets: %w", err)
	}
	defer cursor.Close(ctx)

	var tickets []*models.Ticket
	for cursor.Next(ctx) {
		var ticket models.Ticket
		if err := cursor.Decode(&ticket); err != nil {
			r.logger.WithError(err).Error("Failed to decode ticket")
			continue
		}
		tickets = append(tickets, &ticket)
	}

	return tickets, cursor.Err()
}

// GetActiveByUserID получает активный тикет пользователя
func (r *ticketRepository) GetActiveByUserID(ctx context.Context, userID int64) (*models.Ticket, error) {
	filter := bson.M{
		"user_id": userID,
		"status": bson.M{
			"$in": []models.TicketStatus{
				models.TicketStatusOpen,
				models.TicketStatusInProgress,
				models.TicketStatusWaiting,
			},
		},
	}

	opts := options.FindOne().SetSort(bson.M{"created_at": -1})

	var ticket models.Ticket
	err := r.collection.FindOne(ctx, filter, opts).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Нет активного тикета
		}
		return nil, fmt.Errorf("failed to get active ticket: %w", err)
	}

	return &ticket, nil
}

// GetByChannelMessageID получает тикет по ID сообщения в канале
func (r *ticketRepository) GetByChannelMessageID(ctx context.Context, messageID int) (*models.Ticket, error) {
	filter := bson.M{"channel_message_id": messageID}

	var ticket models.Ticket
	err := r.collection.FindOne(ctx, filter).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket with channel message ID %d not found", messageID)
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return &ticket, nil
}

// GetByGroupMessageID получает тикет по ID сообщения в группе
func (r *ticketRepository) GetByGroupMessageID(ctx context.Context, messageID int) (*models.Ticket, error) {
	filter := bson.M{"group_message_id": messageID}

	var ticket models.Ticket
	err := r.collection.FindOne(ctx, filter).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket with group message ID %d not found", messageID)
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return &ticket, nil
}

// GetByThreadID получает тикет по ID треда
func (r *ticketRepository) GetByThreadID(ctx context.Context, threadID int) (*models.Ticket, error) {
	filter := bson.M{"thread_id": threadID}

	var ticket models.Ticket
	err := r.collection.FindOne(ctx, filter).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket with thread ID %d not found", threadID)
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return &ticket, nil
}

// Update обновляет тикет
func (r *ticketRepository) Update(ctx context.Context, id primitive.ObjectID, update *models.UpdateTicketRequest) error {
	filter := bson.M{"_id": id}
	updateDoc := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	set := updateDoc["$set"].(bson.M)
	if update.Status != nil {
		set["status"] = *update.Status
	}
	if update.Priority != nil {
		set["priority"] = *update.Priority
	}
	if update.Subject != nil {
		set["subject"] = *update.Subject
	}
	if update.AssignedTo != nil {
		set["assigned_to"] = *update.AssignedTo
	}
	if update.Tags != nil {
		set["tags"] = update.Tags
	}
	if update.Metadata != nil {
		set["metadata"] = update.Metadata
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("ticket with ID %s not found", id.Hex())
	}

	r.logger.WithField("ticket_id", id.Hex()).Info("Ticket updated successfully")
	return nil
}

// UpdateStatus обновляет статус тикета
func (r *ticketRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.TicketStatus) error {
	filter := bson.M{"_id": id}
	updateDoc := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update ticket status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("ticket with ID %s not found", id.Hex())
	}

	r.logger.WithFields(map[string]interface{}{
		"ticket_id": id.Hex(),
		"status":    status,
	}).Info("Ticket status updated")

	return nil
}

// Close закрывает тикет
func (r *ticketRepository) Close(ctx context.Context, id primitive.ObjectID, closedBy int64) error {
	now := time.Now()

	filter := bson.M{"_id": id}
	updateDoc := bson.M{
		"$set": bson.M{
			"status":     models.TicketStatusClosed,
			"closed_at":  &now,
			"updated_at": now,
		},
	}

	// Добавляем информацию о том, кто закрыл тикет
	if closedBy != 0 {
		updateDoc["$set"].(bson.M)["metadata.closed_by"] = closedBy
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to close ticket: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("ticket with ID %s not found", id.Hex())
	}

	r.logger.WithFields(map[string]interface{}{
		"ticket_id": id.Hex(),
		"closed_by": closedBy,
	}).Info("Ticket closed")

	return nil
}

// AddMessage добавляет сообщение в тикет
func (r *ticketRepository) AddMessage(ctx context.Context, ticketID primitive.ObjectID, message *models.Message) error {
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()

	if message.ID.IsZero() {
		message.ID = primitive.NewObjectID()
	}

	filter := bson.M{"_id": ticketID}
	updateDoc := bson.M{
		"$push": bson.M{"messages": message},
		"$set": bson.M{
			"updated_at":       time.Now(),
			"last_response_at": time.Now(),
		},
	}

	// Устанавливаем время первого ответа если это сообщение от поддержки
	if message.FromSupport {
		updateDoc["$setOnInsert"] = bson.M{
			"first_response_at": time.Now(),
		}
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("ticket with ID %s not found", ticketID.Hex())
	}

	r.logger.WithFields(map[string]interface{}{
		"ticket_id":    ticketID.Hex(),
		"message_id":   message.MessageID,
		"from_support": message.FromSupport,
	}).Debug("Message added to ticket")

	return nil
}

// UpdateMessage обновляет сообщение в тикете
func (r *ticketRepository) UpdateMessage(ctx context.Context, ticketID primitive.ObjectID, messageID int, text string) error {
	filter := bson.M{
		"_id":                 ticketID,
		"messages.message_id": messageID,
	}

	updateDoc := bson.M{
		"$set": bson.M{
			"messages.$.text":       text,
			"messages.$.edited":     true,
			"messages.$.updated_at": time.Now(),
			"updated_at":            time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("message with ID %d not found in ticket %s", messageID, ticketID.Hex())
	}

	return nil
}

// SetChannelMessageID устанавливает ID сообщения в канале
func (r *ticketRepository) SetChannelMessageID(ctx context.Context, id primitive.ObjectID, messageID int) error {
	filter := bson.M{"_id": id}
	updateDoc := bson.M{
		"$set": bson.M{
			"channel_message_id": messageID,
			"updated_at":         time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to set channel message ID: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("ticket with ID %s not found", id.Hex())
	}

	return nil
}

// SetGroupMessageID устанавливает ID сообщения в группе и thread ID
func (r *ticketRepository) SetGroupMessageID(ctx context.Context, id primitive.ObjectID, messageID int, threadID int) error {
	filter := bson.M{"_id": id}
	updateDoc := bson.M{
		"$set": bson.M{
			"group_message_id": messageID,
			"thread_id":        threadID,
			"updated_at":       time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to set group message ID: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("ticket with ID %s not found", id.Hex())
	}

	return nil
}

// SetFirstResponseTime устанавливает время первого ответа
func (r *ticketRepository) SetFirstResponseTime(ctx context.Context, id primitive.ObjectID, responseTime time.Time) error {
	filter := bson.M{
		"_id":               id,
		"first_response_at": bson.M{"$exists": false},
	}

	updateDoc := bson.M{
		"$set": bson.M{
			"first_response_at": responseTime,
			"updated_at":        time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to set first response time: %w", err)
	}

	// Не возвращаем ошибку если документ не найден - возможно время уже установлено

	return nil
}

// SetSurvey устанавливает результат опроса
func (r *ticketRepository) SetSurvey(ctx context.Context, id primitive.ObjectID, survey *models.Survey) error {
	filter := bson.M{"_id": id}
	updateDoc := bson.M{
		"$set": bson.M{
			"survey":     survey,
			"status":     models.TicketStatusRated,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to set survey: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("ticket with ID %s not found", id.Hex())
	}

	r.logger.WithFields(map[string]interface{}{
		"ticket_id": id.Hex(),
		"rating":    survey.Rating,
	}).Info("Survey submitted for ticket")

	return nil
}

// List получает список тикетов с фильтрами
func (r *ticketRepository) List(ctx context.Context, filters repository.TicketFilters) ([]*models.Ticket, error) {
	filter := bson.M{}

	// Применяем фильтры
	if filters.UserID != nil {
		filter["user_id"] = *filters.UserID
	}
	if filters.Status != nil {
		filter["status"] = *filters.Status
	}
	if filters.Priority != nil {
		filter["priority"] = *filters.Priority
	}
	if filters.AssignedTo != nil {
		filter["assigned_to"] = *filters.AssignedTo
	}
	if len(filters.Tags) > 0 {
		filter["tags"] = bson.M{"$in": filters.Tags}
	}
	if filters.FromDate != nil || filters.ToDate != nil {
		dateFilter := bson.M{}
		if filters.FromDate != nil {
			dateFilter["$gte"] = *filters.FromDate
		}
		if filters.ToDate != nil {
			dateFilter["$lte"] = *filters.ToDate
		}
		filter["created_at"] = dateFilter
	}
	if filters.Search != "" {
		filter["$text"] = bson.M{"$search": filters.Search}
	}

	// Настройка опций
	opts := options.Find()
	if filters.Limit > 0 {
		opts.SetLimit(int64(filters.Limit))
	}
	if filters.Offset > 0 {
		opts.SetSkip(int64(filters.Offset))
	}

	// Сортировка
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	sortOrder := -1
	if filters.SortOrder == "asc" {
		sortOrder = 1
	}
	opts.SetSort(bson.M{sortBy: sortOrder})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find tickets: %w", err)
	}
	defer cursor.Close(ctx)

	var tickets []*models.Ticket
	for cursor.Next(ctx) {
		var ticket models.Ticket
		if err := cursor.Decode(&ticket); err != nil {
			r.logger.WithError(err).Error("Failed to decode ticket")
			continue
		}
		tickets = append(tickets, &ticket)
	}

	return tickets, cursor.Err()
}

// Count подсчитывает количество тикетов с фильтрами
func (r *ticketRepository) Count(ctx context.Context, filters repository.TicketFilters) (int64, error) {
	filter := bson.M{}

	// Применяем те же фильтры что и в List
	if filters.UserID != nil {
		filter["user_id"] = *filters.UserID
	}
	if filters.Status != nil {
		filter["status"] = *filters.Status
	}
	if filters.Priority != nil {
		filter["priority"] = *filters.Priority
	}
	if filters.AssignedTo != nil {
		filter["assigned_to"] = *filters.AssignedTo
	}
	if len(filters.Tags) > 0 {
		filter["tags"] = bson.M{"$in": filters.Tags}
	}
	if filters.FromDate != nil || filters.ToDate != nil {
		dateFilter := bson.M{}
		if filters.FromDate != nil {
			dateFilter["$gte"] = *filters.FromDate
		}
		if filters.ToDate != nil {
			dateFilter["$lte"] = *filters.ToDate
		}
		filter["created_at"] = dateFilter
	}
	if filters.Search != "" {
		filter["$text"] = bson.M{"$search": filters.Search}
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count tickets: %w", err)
	}

	return count, nil
}

// GetOpenTickets получает все открытые тикеты
func (r *ticketRepository) GetOpenTickets(ctx context.Context) ([]*models.Ticket, error) {
	filter := bson.M{
		"status": bson.M{
			"$in": []models.TicketStatus{
				models.TicketStatusOpen,
				models.TicketStatusInProgress,
				models.TicketStatusWaiting,
			},
		},
	}

	opts := options.Find().SetSort(bson.M{"created_at": 1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find open tickets: %w", err)
	}
	defer cursor.Close(ctx)

	var tickets []*models.Ticket
	for cursor.Next(ctx) {
		var ticket models.Ticket
		if err := cursor.Decode(&ticket); err != nil {
			r.logger.WithError(err).Error("Failed to decode ticket")
			continue
		}
		tickets = append(tickets, &ticket)
	}

	return tickets, cursor.Err()
}

// GetTicketsForSurvey получает тикеты для отправки опросов
func (r *ticketRepository) GetTicketsForSurvey(ctx context.Context, since time.Time) ([]*models.Ticket, error) {
	filter := bson.M{
		"status":    models.TicketStatusClosed,
		"closed_at": bson.M{"$gte": since},
		"$or": []bson.M{
			{"survey": bson.M{"$exists": false}},
			{"survey.submitted": false},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find tickets for survey: %w", err)
	}
	defer cursor.Close(ctx)

	var tickets []*models.Ticket
	for cursor.Next(ctx) {
		var ticket models.Ticket
		if err := cursor.Decode(&ticket); err != nil {
			r.logger.WithError(err).Error("Failed to decode ticket")
			continue
		}
		tickets = append(tickets, &ticket)
	}

	return tickets, cursor.Err()
}

// GetAnalytics получает аналитику по тикетам
func (r *ticketRepository) GetAnalytics(ctx context.Context, from, to time.Time) (*repository.TicketAnalytics, error) {
	// Агрегация для получения общей статистики
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{
				"created_at": bson.M{"$gte": from, "$lte": to},
			}},
		},
		{
			{"$group", bson.M{
				"_id":           nil,
				"total_tickets": bson.M{"$sum": 1},
				"open_tickets": bson.M{"$sum": bson.M{"$cond": []interface{}{
					bson.M{"$in": []interface{}{"$status", []string{"open", "in_progress", "waiting_user"}}},
					1, 0,
				}}},
				"closed_tickets": bson.M{"$sum": bson.M{"$cond": []interface{}{
					bson.M{"$in": []interface{}{"$status", []string{"closed", "rated"}}},
					1, 0,
				}}},
				"in_progress_tickets": bson.M{"$sum": bson.M{"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", "in_progress"}},
					1, 0,
				}}},
				"avg_rating": bson.M{"$avg": "$survey.rating"},
				"total_ratings": bson.M{"$sum": bson.M{"$cond": []interface{}{
					bson.M{"$ne": []interface{}{"$survey.rating", nil}},
					1, 0,
				}}},
			}},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}
	defer cursor.Close(ctx)

	analytics := &repository.TicketAnalytics{
		TicketsByPriority:  make(map[string]int64),
		TicketsByStatus:    make(map[string]int64),
		RatingDistribution: make(map[int]int64),
		TopAgents:          []repository.AgentStats{},
		DailyStats:         []repository.DailyStats{},
	}

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode analytics: %w", err)
		}

		if val, ok := result["total_tickets"]; ok {
			analytics.TotalTickets = val.(int64)
		}
		if val, ok := result["open_tickets"]; ok {
			analytics.OpenTickets = val.(int64)
		}
		if val, ok := result["closed_tickets"]; ok {
			analytics.ClosedTickets = val.(int64)
		}
		if val, ok := result["in_progress_tickets"]; ok {
			analytics.InProgressTickets = val.(int64)
		}
		if val, ok := result["avg_rating"]; ok && val != nil {
			analytics.AverageRating = val.(float64)
		}
		if val, ok := result["total_ratings"]; ok {
			analytics.TotalRatings = val.(int64)
		}
	}

	return analytics, nil
}
