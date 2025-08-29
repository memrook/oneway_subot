package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/memrook/oneway_subot/internal/config"
	"github.com/memrook/oneway_subot/internal/models"
	"github.com/memrook/oneway_subot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

// notificationService реализация NotificationService
type notificationService struct {
	bot    *telego.Bot
	config *config.Config
	logger logger.Logger
}

// NewNotificationService создает новый сервис уведомлений
func NewNotificationService(bot *telego.Bot, config *config.Config, logger logger.Logger) NotificationService {
	return &notificationService{
		bot:    bot,
		config: config,
		logger: logger,
	}
}

// NewNotificationServiceStub создает заглушку для NotificationService (для обратной совместимости)
func NewNotificationServiceStub(logger logger.Logger) NotificationService {
	return &notificationServiceStub{
		logger: logger,
	}
}

// notificationServiceStub временная заглушка для NotificationService
type notificationServiceStub struct {
	logger logger.Logger
}

// Реализация для полного NotificationService
func (s *notificationService) SendTicketCreated(ctx context.Context, ticket *models.Ticket, user *models.User) error {
	if !s.config.Features.Notifications.Enabled {
		return nil
	}

	// Отправляем тикет в канал
	channelMessage := s.formatTicketForChannel(ticket, user)
	msg, err := s.bot.SendMessage(ctx, telegoutil.Message(
		telegoutil.ID(s.config.Bot.ChannelID),
		channelMessage,
	).WithParseMode("HTML"))

	if err != nil {
		return fmt.Errorf("failed to send ticket to channel: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":      ticket.ID.Hex(),
		"ticket_number":  ticket.TicketNumber,
		"channel_id":     s.config.Bot.ChannelID,
		"channel_msg_id": msg.MessageID,
	}).Info("Ticket published to channel")

	// Отправляем уведомление админам если настроено
	if s.config.Features.Notifications.AdminChatID != 0 {
		adminMsg := fmt.Sprintf(
			"🎫 <b>Новый тикет #%s</b>\n\n"+
				"👤 Пользователь: %s\n"+
				"📝 Тема: %s\n"+
				"⏰ Время: %s\n\n"+
				"<a href=\"https://t.me/c/%s/%d\">Перейти к тикету</a>",
			ticket.TicketNumber,
			user.GetDisplayName(),
			ticket.Subject,
			ticket.CreatedAt.Format("02.01.2006 15:04"),
			strconv.FormatInt(s.config.Bot.ChannelID, 10)[4:], // Убираем -100 префикс
			msg.MessageID,
		)

		_, err = s.bot.SendMessage(ctx, telegoutil.Message(
			telegoutil.ID(s.config.Features.Notifications.AdminChatID),
			adminMsg,
		).WithParseMode("HTML"))

		if err != nil {
			s.logger.WithError(err).Error("Failed to send admin notification")
		}
	}

	return nil
}

func (s *notificationService) SendTicketClosed(ctx context.Context, ticket *models.Ticket, user *models.User) error {
	if !s.config.Features.Notifications.Enabled {
		return nil
	}

	// Уведомляем пользователя о закрытии тикета
	message := fmt.Sprintf(
		"✅ <b>Ваш тикет #%s закрыт</b>\n\n"+
			"📝 Тема: %s\n"+
			"⏰ Время закрытия: %s\n\n"+
			"💭 <i>Если у вас есть дополнительные вопросы, создайте новый тикет.</i>",
		ticket.TicketNumber,
		ticket.Subject,
		ticket.ClosedAt.Format("02.01.2006 15:04"),
	)

	if s.config.Features.Survey.Enabled && s.config.Features.Survey.AutoSendAfterClose {
		message += "\n\n⭐ Пожалуйста, оцените качество нашей поддержки:"

		// Добавляем inline клавиатуру с оценками
		keyboard := telegoutil.InlineKeyboard(
			telegoutil.InlineKeyboardRow(
				telegoutil.InlineKeyboardButton("⭐").WithCallbackData(fmt.Sprintf("rate_%s_1", ticket.ID.Hex())),
				telegoutil.InlineKeyboardButton("⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_2", ticket.ID.Hex())),
				telegoutil.InlineKeyboardButton("⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_3", ticket.ID.Hex())),
				telegoutil.InlineKeyboardButton("⭐⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_4", ticket.ID.Hex())),
				telegoutil.InlineKeyboardButton("⭐⭐⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_5", ticket.ID.Hex())),
			),
		)

		_, err := s.bot.SendMessage(ctx, telegoutil.Message(
			telegoutil.ID(user.UserID),
			message,
		).WithParseMode("HTML").WithReplyMarkup(keyboard))

		if err != nil {
			return fmt.Errorf("failed to send closure notification with survey: %w", err)
		}
	} else {
		_, err := s.bot.SendMessage(ctx, telegoutil.Message(
			telegoutil.ID(user.UserID),
			message,
		).WithParseMode("HTML"))

		if err != nil {
			return fmt.Errorf("failed to send closure notification: %w", err)
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"ticket_id":     ticket.ID.Hex(),
		"ticket_number": ticket.TicketNumber,
		"user_id":       user.UserID,
	}).Info("Ticket closure notification sent")

	return nil
}

func (s *notificationService) SendSurveyRequest(ctx context.Context, ticket *models.Ticket, user *models.User) error {
	if !s.config.Features.Survey.Enabled {
		return nil
	}

	message := fmt.Sprintf(
		"⭐ <b>Оценка качества поддержки</b>\n\n"+
			"Тикет #%s завершен.\n"+
			"Пожалуйста, оцените качество нашей работы:",
		ticket.TicketNumber,
	)

	keyboard := telegoutil.InlineKeyboard(
		telegoutil.InlineKeyboardRow(
			telegoutil.InlineKeyboardButton("⭐").WithCallbackData(fmt.Sprintf("rate_%s_1", ticket.ID.Hex())),
			telegoutil.InlineKeyboardButton("⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_2", ticket.ID.Hex())),
			telegoutil.InlineKeyboardButton("⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_3", ticket.ID.Hex())),
			telegoutil.InlineKeyboardButton("⭐⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_4", ticket.ID.Hex())),
			telegoutil.InlineKeyboardButton("⭐⭐⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_5", ticket.ID.Hex())),
		),
	)

	_, err := s.bot.SendMessage(ctx, telegoutil.Message(
		telegoutil.ID(user.UserID),
		message,
	).WithParseMode("HTML").WithReplyMarkup(keyboard))

	if err != nil {
		return fmt.Errorf("failed to send survey request: %w", err)
	}

	return nil
}

func (s *notificationService) SendSurveyReminder(ctx context.Context, ticket *models.Ticket, user *models.User) error {
	message := fmt.Sprintf(
		"⭐ <b>Напоминание об оценке</b>\n\n"+
			"Вы еще не оценили качество поддержки по тикету #%s.\n"+
			"Ваша оценка поможет нам стать лучше:",
		ticket.TicketNumber,
	)

	keyboard := telegoutil.InlineKeyboard(
		telegoutil.InlineKeyboardRow(
			telegoutil.InlineKeyboardButton("⭐").WithCallbackData(fmt.Sprintf("rate_%s_1", ticket.ID.Hex())),
			telegoutil.InlineKeyboardButton("⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_2", ticket.ID.Hex())),
			telegoutil.InlineKeyboardButton("⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_3", ticket.ID.Hex())),
			telegoutil.InlineKeyboardButton("⭐⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_4", ticket.ID.Hex())),
			telegoutil.InlineKeyboardButton("⭐⭐⭐⭐⭐").WithCallbackData(fmt.Sprintf("rate_%s_5", ticket.ID.Hex())),
		),
	)

	_, err := s.bot.SendMessage(ctx, telegoutil.Message(
		telegoutil.ID(user.UserID),
		message,
	).WithParseMode("HTML").WithReplyMarkup(keyboard))

	if err != nil {
		return fmt.Errorf("failed to send survey reminder: %w", err)
	}

	return nil
}

func (s *notificationService) SendAdminNotification(ctx context.Context, message string, data map[string]interface{}) error {
	if s.config.Features.Notifications.AdminChatID == 0 {
		return nil
	}

	var parts []string
	parts = append(parts, "🔔 <b>Системное уведомление</b>\n")
	parts = append(parts, message)

	if len(data) > 0 {
		parts = append(parts, "\n📊 <b>Данные:</b>")
		for key, value := range data {
			parts = append(parts, fmt.Sprintf("• %s: %v", key, value))
		}
	}

	_, err := s.bot.SendMessage(ctx, telegoutil.Message(
		telegoutil.ID(s.config.Features.Notifications.AdminChatID),
		strings.Join(parts, "\n"),
	).WithParseMode("HTML"))

	if err != nil {
		return fmt.Errorf("failed to send admin notification: %w", err)
	}

	return nil
}

func (s *notificationService) SendUserMessage(ctx context.Context, userID int64, message string) error {
	_, err := s.bot.SendMessage(ctx, telegoutil.Message(
		telegoutil.ID(userID),
		message,
	).WithParseMode("HTML"))

	if err != nil {
		return fmt.Errorf("failed to send user message: %w", err)
	}

	return nil
}

func (s *notificationService) SendSupportMessage(ctx context.Context, chatID int64, message string, threadID int) error {
	msg := telegoutil.Message(telegoutil.ID(chatID), message).WithParseMode("HTML")

	if threadID != 0 {
		msg = msg.WithMessageThreadID(threadID)
	}

	_, err := s.bot.SendMessage(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send support message: %w", err)
	}

	return nil
}

func (s *notificationService) formatTicketForChannel(ticket *models.Ticket, user *models.User) string {
	priorityEmoji := "🔵"
	switch ticket.Priority {
	case models.PriorityLow:
		priorityEmoji = "🟢"
	case models.PriorityNormal:
		priorityEmoji = "🔵"
	case models.PriorityHigh:
		priorityEmoji = "🟠"
	case models.PriorityCritical:
		priorityEmoji = "🔴"
	}

	var tags string
	if len(ticket.Tags) > 0 {
		tags = "\n🏷 Теги: " + strings.Join(ticket.Tags, ", ")
	}

	return fmt.Sprintf(
		"🎫 <b>Новый тикет #%s</b>\n\n"+
			"👤 <b>Пользователь:</b> %s\n"+
			"📝 <b>Тема:</b> %s\n"+
			"%s <b>Приоритет:</b> %s\n"+
			"⏰ <b>Создан:</b> %s%s\n\n"+
			"📄 <b>Описание:</b>\n%s",
		ticket.TicketNumber,
		user.GetDisplayName(),
		ticket.Subject,
		priorityEmoji,
		ticket.Priority,
		ticket.CreatedAt.Format("02.01.2006 15:04"),
		tags,
		ticket.Description,
	)
}

// Заглушки для обратной совместимости
func (s *notificationServiceStub) SendTicketCreated(ctx context.Context, ticket *models.Ticket, user *models.User) error {
	s.logger.Info("SendTicketCreated called (stub)")
	return nil
}

func (s *notificationServiceStub) SendTicketClosed(ctx context.Context, ticket *models.Ticket, user *models.User) error {
	s.logger.Info("SendTicketClosed called (stub)")
	return nil
}

func (s *notificationServiceStub) SendSurveyRequest(ctx context.Context, ticket *models.Ticket, user *models.User) error {
	s.logger.Info("SendSurveyRequest called (stub)")
	return nil
}

func (s *notificationServiceStub) SendSurveyReminder(ctx context.Context, ticket *models.Ticket, user *models.User) error {
	s.logger.Info("SendSurveyReminder called (stub)")
	return nil
}

func (s *notificationServiceStub) SendAdminNotification(ctx context.Context, message string, data map[string]interface{}) error {
	s.logger.Info("SendAdminNotification called (stub)")
	return nil
}

func (s *notificationServiceStub) SendUserMessage(ctx context.Context, userID int64, message string) error {
	s.logger.Info("SendUserMessage called (stub)")
	return nil
}

func (s *notificationServiceStub) SendSupportMessage(ctx context.Context, chatID int64, message string, threadID int) error {
	s.logger.Info("SendSupportMessage called (stub)")
	return nil
}
