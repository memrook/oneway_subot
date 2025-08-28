package service

import (
	"context"
	"testing"
	"time"

	"github.com/memrook/oneway_subot/internal/models"
	"github.com/memrook/oneway_subot/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockUserRepository мок для UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByUserID(ctx context.Context, userID int64) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, userID int64, update *models.UpdateUserRequest) error {
	args := m.Called(ctx, userID, update)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) GetStats(ctx context.Context, userID int64) (*models.UserStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserStats), args.Error(1)
}

func (m *MockUserRepository) Block(ctx context.Context, userID int64, reason string) error {
	args := m.Called(ctx, userID, reason)
	return args.Error(0)
}

func (m *MockUserRepository) Unblock(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepository) GetActiveUsers(ctx context.Context, since time.Time) ([]*models.User, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.CreateUserRequest
		setup   func(*MockUserRepository)
		wantErr bool
	}{
		{
			name: "successful user creation",
			req: &models.CreateUserRequest{
				UserID:    123456789,
				FirstName: "John",
				LastName:  "Doe",
				Username:  "johndoe",
			},
			setup: func(repo *MockUserRepository) {
				repo.On("GetByUserID", mock.Anything, int64(123456789)).Return(nil, assert.AnError)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "user already exists",
			req: &models.CreateUserRequest{
				UserID:    123456789,
				FirstName: "John",
			},
			setup: func(repo *MockUserRepository) {
				existingUser := &models.User{
					UserID:    123456789,
					FirstName: "John",
				}
				repo.On("GetByUserID", mock.Anything, int64(123456789)).Return(existingUser, nil)
			},
			wantErr: true,
		},
		{
			name: "missing user ID",
			req: &models.CreateUserRequest{
				FirstName: "John",
			},
			setup:   func(repo *MockUserRepository) {},
			wantErr: true,
		},
		{
			name: "missing first name",
			req: &models.CreateUserRequest{
				UserID: 123456789,
			},
			setup:   func(repo *MockUserRepository) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			tt.setup(mockRepo)

			// Создаем простой логгер для тестов
			testLogger := &testLogger{}

			service := NewUserService(mockRepo, testLogger)

			_, err := service.CreateUser(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetOrCreateUser(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.CreateUserRequest
		setup   func(*MockUserRepository)
		wantErr bool
	}{
		{
			name: "get existing user",
			req: &models.CreateUserRequest{
				UserID:    123456789,
				FirstName: "John",
				LastName:  "Doe",
				Username:  "johndoe",
			},
			setup: func(repo *MockUserRepository) {
				existingUser := &models.User{
					UserID:    123456789,
					FirstName: "John",
					LastName:  "Doe",
					Username:  "johndoe",
				}
				repo.On("GetByUserID", mock.Anything, int64(123456789)).Return(existingUser, nil)
				repo.On("Update", mock.Anything, int64(123456789), mock.AnythingOfType("*models.UpdateUserRequest")).Return(nil)
				repo.On("GetByUserID", mock.Anything, int64(123456789)).Return(existingUser, nil)
			},
			wantErr: false,
		},
		{
			name: "create new user",
			req: &models.CreateUserRequest{
				UserID:    123456789,
				FirstName: "John",
				LastName:  "Doe",
				Username:  "johndoe",
			},
			setup: func(repo *MockUserRepository) {
				repo.On("GetByUserID", mock.Anything, int64(123456789)).Return(nil, assert.AnError)
				repo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			tt.setup(mockRepo)

			testLogger := &testLogger{}
			service := NewUserService(mockRepo, testLogger)

			user, err := service.GetOrCreateUser(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.req.UserID, user.UserID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_BlockUser(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		reason  string
		setup   func(*MockUserRepository)
		wantErr bool
	}{
		{
			name:   "successful block",
			userID: 123456789,
			reason: "Spam",
			setup: func(repo *MockUserRepository) {
				user := &models.User{
					UserID:    123456789,
					FirstName: "John",
					IsBlocked: false,
				}
				repo.On("GetByUserID", mock.Anything, int64(123456789)).Return(user, nil)
				repo.On("Block", mock.Anything, int64(123456789), "Spam").Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "user already blocked",
			userID: 123456789,
			reason: "Spam",
			setup: func(repo *MockUserRepository) {
				user := &models.User{
					UserID:    123456789,
					FirstName: "John",
					IsBlocked: true,
				}
				repo.On("GetByUserID", mock.Anything, int64(123456789)).Return(user, nil)
			},
			wantErr: true,
		},
		{
			name:   "user not found",
			userID: 123456789,
			reason: "Spam",
			setup: func(repo *MockUserRepository) {
				repo.On("GetByUserID", mock.Anything, int64(123456789)).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			tt.setup(mockRepo)

			testLogger := &testLogger{}
			service := NewUserService(mockRepo, testLogger)

			err := service.BlockUser(context.Background(), tt.userID, tt.reason)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// testLogger простая реализация логгера для тестов
type testLogger struct{}

func (l *testLogger) Debug(args ...interface{})                              {}
func (l *testLogger) Info(args ...interface{})                               {}
func (l *testLogger) Warn(args ...interface{})                               {}
func (l *testLogger) Error(args ...interface{})                              {}
func (l *testLogger) Fatal(args ...interface{})                              {}
func (l *testLogger) Debugf(format string, args ...interface{})              {}
func (l *testLogger) Infof(format string, args ...interface{})               {}
func (l *testLogger) Warnf(format string, args ...interface{})               {}
func (l *testLogger) Errorf(format string, args ...interface{})              {}
func (l *testLogger) Fatalf(format string, args ...interface{})              {}
func (l *testLogger) WithField(key string, value interface{}) logger.Logger  { return l }
func (l *testLogger) WithFields(fields map[string]interface{}) logger.Logger { return l }
func (l *testLogger) WithError(err error) logger.Logger                      { return l }
