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

// userRepository реализация UserRepository для MongoDB
type userRepository struct {
	collection *mongo.Collection
	logger     logger.Logger
}

// NewUserRepository создает новый репозиторий пользователей
func NewUserRepository(db *Database, log logger.Logger) repository.UserRepository {
	return &userRepository{
		collection: db.GetCollection("users"),
		logger:     log,
	}
}

// Create создает нового пользователя
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}

	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("user with ID %d already exists", user.UserID)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"user_id":    user.UserID,
		"username":   user.Username,
		"first_name": user.FirstName,
	}).Info("User created successfully")

	return nil
}

// GetByUserID получает пользователя по Telegram User ID
func (r *userRepository) GetByUserID(ctx context.Context, userID int64) (*models.User, error) {
	filter := bson.M{"user_id": userID}

	var user models.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user with ID %d not found", userID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByID получает пользователя по ObjectID
func (r *userRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	filter := bson.M{"_id": id}

	var user models.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user with ID %s not found", id.Hex())
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Update обновляет данные пользователя
func (r *userRepository) Update(ctx context.Context, userID int64, update *models.UpdateUserRequest) error {
	filter := bson.M{"user_id": userID}
	updateDoc := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// Добавляем только не-nil поля
	set := updateDoc["$set"].(bson.M)
	if update.FirstName != nil {
		set["first_name"] = *update.FirstName
	}
	if update.LastName != nil {
		set["last_name"] = *update.LastName
	}
	if update.Username != nil {
		set["username"] = *update.Username
	}
	if update.LanguageCode != nil {
		set["language_code"] = *update.LanguageCode
	}
	if update.IsPremium != nil {
		set["is_premium"] = *update.IsPremium
	}
	if update.IsBlocked != nil {
		set["is_blocked"] = *update.IsBlocked
	}
	if update.BlockReason != nil {
		set["block_reason"] = *update.BlockReason
	}

	result, err := r.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user with ID %d not found", userID)
	}

	r.logger.WithField("user_id", userID).Info("User updated successfully")
	return nil
}

// Delete удаляет пользователя
func (r *userRepository) Delete(ctx context.Context, userID int64) error {
	filter := bson.M{"user_id": userID}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user with ID %d not found", userID)
	}

	r.logger.WithField("user_id", userID).Info("User deleted successfully")
	return nil
}

// List получает список пользователей с пагинацией
func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	opts.SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			r.logger.WithError(err).Error("Failed to decode user")
			continue
		}
		users = append(users, &user)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return users, nil
}

// GetStats получает статистику пользователя
func (r *userRepository) GetStats(ctx context.Context, userID int64) (*models.UserStats, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{"user_id": userID}},
		},
		{
			{"$lookup", bson.M{
				"from":         "tickets",
				"localField":   "user_id",
				"foreignField": "user_id",
				"as":           "tickets",
			}},
		},
		{
			{"$addFields", bson.M{
				"total_tickets": bson.M{"$size": "$tickets"},
				"open_tickets": bson.M{"$size": bson.M{"$filter": bson.M{
					"input": "$tickets",
					"cond":  bson.M{"$in": []interface{}{"$$this.status", []string{"open", "in_progress", "waiting_user"}}},
				}}},
				"closed_tickets": bson.M{"$size": bson.M{"$filter": bson.M{
					"input": "$tickets",
					"cond":  bson.M{"$in": []interface{}{"$$this.status", []string{"closed", "rated"}}},
				}}},
				"average_rating": bson.M{"$avg": bson.M{"$map": bson.M{
					"input": bson.M{"$filter": bson.M{
						"input": "$tickets",
						"cond":  bson.M{"$ne": []interface{}{"$$this.survey.rating", nil}},
					}},
					"in": "$$this.survey.rating",
				}}},
				"total_messages": bson.M{"$sum": bson.M{"$map": bson.M{
					"input": "$tickets",
					"in":    bson.M{"$size": "$$this.messages"},
				}}},
				"last_activity":     bson.M{"$max": "$tickets.updated_at"},
				"first_ticket_date": bson.M{"$min": "$tickets.created_at"},
			}},
		},
		{
			{"$project", bson.M{
				"user_id":           "$user_id",
				"total_tickets":     1,
				"open_tickets":      1,
				"closed_tickets":    1,
				"average_rating":    1,
				"total_messages":    1,
				"last_activity":     1,
				"first_ticket_date": 1,
			}},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return &models.UserStats{
			UserID: userID,
		}, nil
	}

	var stats models.UserStats
	if err := cursor.Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode user stats: %w", err)
	}

	return &stats, nil
}

// Block блокирует пользователя
func (r *userRepository) Block(ctx context.Context, userID int64, reason string) error {
	update := &models.UpdateUserRequest{
		IsBlocked:   &[]bool{true}[0],
		BlockReason: &reason,
	}

	err := r.Update(ctx, userID, update)
	if err != nil {
		return fmt.Errorf("failed to block user: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"user_id": userID,
		"reason":  reason,
	}).Info("User blocked")

	return nil
}

// Unblock разблокирует пользователя
func (r *userRepository) Unblock(ctx context.Context, userID int64) error {
	update := &models.UpdateUserRequest{
		IsBlocked:   &[]bool{false}[0],
		BlockReason: &[]string{""}[0],
	}

	err := r.Update(ctx, userID, update)
	if err != nil {
		return fmt.Errorf("failed to unblock user: %w", err)
	}

	r.logger.WithField("user_id", userID).Info("User unblocked")
	return nil
}

// GetActiveUsers получает активных пользователей с определенного времени
func (r *userRepository) GetActiveUsers(ctx context.Context, since time.Time) ([]*models.User, error) {
	pipeline := mongo.Pipeline{
		{
			{"$lookup", bson.M{
				"from":         "tickets",
				"localField":   "user_id",
				"foreignField": "user_id",
				"as":           "tickets",
			}},
		},
		{
			{"$match", bson.M{
				"$or": []bson.M{
					{"updated_at": bson.M{"$gte": since}},
					{"tickets.updated_at": bson.M{"$gte": since}},
				},
				"is_blocked": bson.M{"$ne": true},
			}},
		},
		{
			{"$project", bson.M{
				"user_id":       1,
				"first_name":    1,
				"last_name":     1,
				"username":      1,
				"language_code": 1,
				"is_premium":    1,
				"is_bot":        1,
				"is_blocked":    1,
				"created_at":    1,
				"updated_at":    1,
			}},
		},
		{
			{"$sort", bson.M{"updated_at": -1}},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			r.logger.WithError(err).Error("Failed to decode active user")
			continue
		}
		users = append(users, &user)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return users, nil
}
