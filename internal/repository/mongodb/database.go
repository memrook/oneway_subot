package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/memrook/oneway_subot/internal/config"
	"github.com/memrook/oneway_subot/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Database структура для работы с MongoDB
type Database struct {
	client *mongo.Client
	db     *mongo.Database
	logger logger.Logger
}

// New создает новое подключение к MongoDB
func New(cfg config.DatabaseConfig, log logger.Logger) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Настройка клиента
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxConnIdleTime(cfg.MaxIdleTime).
		SetServerSelectionTimeout(cfg.Timeout)

	// Подключение к MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Проверка подключения
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Name)

	database := &Database{
		client: client,
		db:     db,
		logger: log,
	}

	// Создание индексов
	if err := database.createIndexes(ctx); err != nil {
		log.WithError(err).Warn("Failed to create indexes")
	}

	log.Info("Successfully connected to MongoDB")
	return database, nil
}

// Close закрывает подключение к базе данных
func (d *Database) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}

// GetCollection возвращает коллекцию
func (d *Database) GetCollection(name string) *mongo.Collection {
	return d.db.Collection(name)
}

// Ping проверяет подключение к базе данных
func (d *Database) Ping(ctx context.Context) error {
	return d.client.Ping(ctx, nil)
}

// createIndexes создает необходимые индексы
func (d *Database) createIndexes(ctx context.Context) error {
	// Индексы для пользователей
	userCollection := d.GetCollection("users")
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "username", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}

	if _, err := userCollection.Indexes().CreateMany(ctx, userIndexes, options.CreateIndexes()); err != nil {
		return fmt.Errorf("failed to create user indexes: %w", err)
	}

	// Индексы для тикетов
	ticketCollection := d.GetCollection("tickets")
	ticketIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "ticket_number", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "channel_message_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "group_message_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "thread_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "assigned_to", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "priority", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "closed_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "tags", Value: 1}},
		},
		{
			// Составной индекс для активных тикетов пользователя
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		{
			// Текстовый индекс для поиска
			Keys: bson.D{
				{Key: "subject", Value: "text"},
				{Key: "description", Value: "text"},
				{Key: "messages.text", Value: "text"},
			},
			Options: options.Index().SetDefaultLanguage("russian"),
		},
	}

	if _, err := ticketCollection.Indexes().CreateMany(ctx, ticketIndexes, options.CreateIndexes()); err != nil {
		return fmt.Errorf("failed to create ticket indexes: %w", err)
	}

	d.logger.Info("Successfully created database indexes")
	return nil
}

// HealthCheck проверяет состояние базы данных
func (d *Database) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return d.client.Ping(ctx, nil)
}

// GetStats возвращает статистику базы данных
func (d *Database) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Общая статистика БД
	dbStats := d.db.RunCommand(ctx, map[string]interface{}{"dbStats": 1})
	if dbStats.Err() != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", dbStats.Err())
	}

	var dbStatsResult map[string]interface{}
	if err := dbStats.Decode(&dbStatsResult); err != nil {
		return nil, fmt.Errorf("failed to decode database stats: %w", err)
	}

	stats["database"] = dbStatsResult

	// Статистика коллекций
	collections := []string{"users", "tickets"}
	collectionStats := make(map[string]interface{})

	for _, collName := range collections {
		coll := d.GetCollection(collName)
		count, err := coll.EstimatedDocumentCount(ctx)
		if err != nil {
			d.logger.WithError(err).Warnf("Failed to get count for collection %s", collName)
			continue
		}
		collectionStats[collName] = map[string]interface{}{
			"count": count,
		}
	}

	stats["collections"] = collectionStats

	return stats, nil
}
