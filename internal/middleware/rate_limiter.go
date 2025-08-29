package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/memrook/oneway_subot/internal/config"
	"github.com/memrook/oneway_subot/pkg/logger"
	"github.com/memrook/oneway_subot/pkg/metrics"
	"github.com/mymmrac/telego"
)

// RateLimiter простой rate limiter для пользователей
type RateLimiter struct {
	config      *config.Config
	logger      logger.Logger
	userLimits  map[int64]*UserLimit
	mu          sync.RWMutex
	cleanupDone chan struct{}
}

// UserLimit лимит для пользователя
type UserLimit struct {
	Count     int
	Window    time.Time
	LastReset time.Time
}

// NewRateLimiter создает новый rate limiter
func NewRateLimiter(config *config.Config, logger logger.Logger) *RateLimiter {
	rl := &RateLimiter{
		config:      config,
		logger:      logger,
		userLimits:  make(map[int64]*UserLimit),
		cleanupDone: make(chan struct{}),
	}

	// Запускаем goroutine для очистки старых записей
	go rl.cleanup()

	return rl
}

// IsAllowed проверяет, разрешено ли пользователю отправить сообщение
func (rl *RateLimiter) IsAllowed(userID int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowDuration := time.Minute
	maxRequests := rl.config.Bot.RateLimit.RequestsPerMinute

	userLimit, exists := rl.userLimits[userID]
	if !exists {
		rl.userLimits[userID] = &UserLimit{
			Count:     1,
			Window:    now,
			LastReset: now,
		}
		return true
	}

	// Проверяем, нужно ли сбросить окно
	if now.Sub(userLimit.Window) >= windowDuration {
		userLimit.Count = 1
		userLimit.Window = now
		userLimit.LastReset = now
		return true
	}

	// Проверяем лимит
	if userLimit.Count >= maxRequests {
		metrics.RecordError("rate_limiter", "limit_exceeded")
		rl.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"count":   userLimit.Count,
			"limit":   maxRequests,
		}).Warn("Rate limit exceeded")
		return false
	}

	userLimit.Count++
	return true
}

// GetUserStats возвращает статистику пользователя
func (rl *RateLimiter) GetUserStats(userID int64) (int, int, time.Time) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	userLimit, exists := rl.userLimits[userID]
	if !exists {
		return 0, rl.config.Bot.RateLimit.RequestsPerMinute, time.Time{}
	}

	return userLimit.Count, rl.config.Bot.RateLimit.RequestsPerMinute, userLimit.LastReset
}

// cleanup периодически очищает старые записи
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanupOldEntries()
		case <-rl.cleanupDone:
			return
		}
	}
}

// cleanupOldEntries удаляет старые записи
func (rl *RateLimiter) cleanupOldEntries() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for userID, userLimit := range rl.userLimits {
		// Удаляем записи старше 1 часа
		if now.Sub(userLimit.LastReset) > time.Hour {
			delete(rl.userLimits, userID)
		}
	}
}

// Stop останавливает rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.cleanupDone)
}

// RateLimitMiddleware middleware функция для rate limiting
func (rl *RateLimiter) RateLimitMiddleware(next func(*telego.Bot, telego.Update)) func(*telego.Bot, telego.Update) {
	return func(bot *telego.Bot, update telego.Update) {
		var userID int64

		// Извлекаем user ID из разных типов update
		if update.Message != nil {
			userID = update.Message.From.ID
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
		} else if update.InlineQuery != nil {
			userID = update.InlineQuery.From.ID
		} else {
			// Если не можем определить пользователя, пропускаем
			next(bot, update)
			return
		}

		// Проверяем rate limit
		if !rl.IsAllowed(userID) {
			// Отправляем сообщение о превышении лимита
			if update.Message != nil {
				rl.sendRateLimitMessage(bot, update.Message.Chat.ID)
			} else if update.CallbackQuery != nil {
				rl.answerCallbackWithError(bot, update.CallbackQuery.ID)
			}
			return
		}

		// Передаем управление следующему обработчику
		next(bot, update)
	}
}

// sendRateLimitMessage отправляет сообщение о превышении лимита
func (rl *RateLimiter) sendRateLimitMessage(bot *telego.Bot, chatID int64) {
	message := "⚠️ <b>Превышен лимит сообщений</b>\n\n" +
		"Вы отправляете сообщения слишком быстро. " +
		"Пожалуйста, подождите немного перед отправкой следующего сообщения.\n\n" +
		"🕐 Лимит: " + fmt.Sprintf("%d сообщений в минуту", rl.config.Bot.RateLimit.RequestsPerMinute)

	_, err := bot.SendMessage(&telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      message,
		ParseMode: "HTML",
	})

	if err != nil {
		rl.logger.WithError(err).Error("Failed to send rate limit message")
	}
}

// answerCallbackWithError отвечает на callback с ошибкой
func (rl *RateLimiter) answerCallbackWithError(bot *telego.Bot, callbackQueryID string) {
	_, err := bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: callbackQueryID,
		Text:            "⚠️ Слишком много запросов. Подождите немного.",
		ShowAlert:       true,
	})

	if err != nil {
		rl.logger.WithError(err).Error("Failed to answer callback query with rate limit error")
	}
}
