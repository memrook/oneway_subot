package service

import (
	"context"
	"fmt"

	"github.com/memrook/oneway_subot/internal/models"
	"github.com/memrook/oneway_subot/internal/repository"
	"github.com/memrook/oneway_subot/pkg/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

// userService реализация UserService
type userService struct {
	repo   repository.UserRepository
	logger logger.Logger
}

// NewUserService создает новый сервис пользователей
func NewUserService(repo repository.UserRepository, logger logger.Logger) UserService {
	return &userService{
		repo:   repo,
		logger: logger,
	}
}

// CreateUser создает нового пользователя
func (s *userService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	// Валидация
	if req.UserID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}

	if req.FirstName == "" {
		return nil, fmt.Errorf("first name is required")
	}

	// Проверяем, не существует ли пользователь уже
	existingUser, err := s.repo.GetByUserID(ctx, req.UserID)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with ID %d already exists", req.UserID)
	}

	// Создаем пользователя
	user := req.ToUser()

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":    user.UserID,
		"username":   user.Username,
		"first_name": user.FirstName,
	}).Info("User created successfully")

	return user, nil
}

// GetUser получает пользователя по ID
func (s *userService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}

	user, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser обновляет данные пользователя
func (s *userService) UpdateUser(ctx context.Context, userID int64, req *models.UpdateUserRequest) error {
	if userID == 0 {
		return fmt.Errorf("user ID is required")
	}

	// Проверяем существование пользователя
	_, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := s.repo.Update(ctx, userID, req); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.WithField("user_id", userID).Info("User updated successfully")
	return nil
}

// BlockUser блокирует пользователя
func (s *userService) BlockUser(ctx context.Context, userID int64, reason string) error {
	if userID == 0 {
		return fmt.Errorf("user ID is required")
	}

	if reason == "" {
		reason = "Blocked by administrator"
	}

	// Проверяем существование пользователя
	user, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.IsBlocked {
		return fmt.Errorf("user is already blocked")
	}

	if err := s.repo.Block(ctx, userID, reason); err != nil {
		return fmt.Errorf("failed to block user: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id": userID,
		"reason":  reason,
	}).Warn("User blocked")

	return nil
}

// UnblockUser разблокирует пользователя
func (s *userService) UnblockUser(ctx context.Context, userID int64) error {
	if userID == 0 {
		return fmt.Errorf("user ID is required")
	}

	// Проверяем существование пользователя
	user, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if !user.IsBlocked {
		return fmt.Errorf("user is not blocked")
	}

	if err := s.repo.Unblock(ctx, userID); err != nil {
		return fmt.Errorf("failed to unblock user: %w", err)
	}

	s.logger.WithField("user_id", userID).Info("User unblocked")
	return nil
}

// GetUserStats получает статистику пользователя
func (s *userService) GetUserStats(ctx context.Context, userID int64) (*models.UserStats, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}

	// Проверяем существование пользователя
	_, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	stats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return stats, nil
}

// IsUserBlocked проверяет, заблокирован ли пользователь
func (s *userService) IsUserBlocked(ctx context.Context, userID int64) (bool, error) {
	if userID == 0 {
		return false, fmt.Errorf("user ID is required")
	}

	user, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		// Если пользователь не найден, считаем что он не заблокирован
		// Это позволит создать нового пользователя при первом обращении
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check user status: %w", err)
	}

	return user.IsBlocked, nil
}

// GetOrCreateUser получает существующего пользователя или создает нового
func (s *userService) GetOrCreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	// Пытаемся получить существующего пользователя
	user, err := s.repo.GetByUserID(ctx, req.UserID)
	if err == nil {
		// Пользователь существует, обновляем его данные
		updateReq := &models.UpdateUserRequest{
			FirstName:    &req.FirstName,
			LastName:     &req.LastName,
			Username:     &req.Username,
			LanguageCode: &req.LanguageCode,
			IsPremium:    &req.IsPremium,
		}

		if err := s.repo.Update(ctx, req.UserID, updateReq); err != nil {
			s.logger.WithError(err).WithField("user_id", req.UserID).Warn("Failed to update existing user")
		} else {
			s.logger.WithField("user_id", req.UserID).Debug("Updated existing user")
		}

		// Получаем обновленного пользователя
		user, err = s.repo.GetByUserID(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated user: %w", err)
		}

		return user, nil
	}

	// Пользователь не существует, создаем нового
	user = req.ToUser()

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create new user: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":    user.UserID,
		"username":   user.Username,
		"first_name": user.FirstName,
	}).Info("New user created")

	return user, nil
}

// ValidateUserAccess проверяет доступ пользователя к системе
func (s *userService) ValidateUserAccess(ctx context.Context, userID int64) error {
	if userID == 0 {
		return fmt.Errorf("user ID is required")
	}

	user, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		// Если пользователь не найден, это не ошибка доступа
		// Новый пользователь может быть создан при первом обращении
		return nil
	}

	if user.IsBlocked {
		return fmt.Errorf("user is blocked: %s", user.BlockReason)
	}

	if user.IsBot {
		return fmt.Errorf("bots are not allowed")
	}

	return nil
}

// GetActiveUsers получает список активных пользователей
func (s *userService) GetActiveUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Фильтруем только активных пользователей
	var activeUsers []*models.User
	for _, user := range users {
		if user.IsActive() {
			activeUsers = append(activeUsers, user)
		}
	}

	return activeUsers, nil
}
