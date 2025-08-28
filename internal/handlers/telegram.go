package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/memrook/oneway_subot/internal/config"
	"github.com/memrook/oneway_subot/internal/models"
	"github.com/memrook/oneway_subot/internal/service"
	"github.com/memrook/oneway_subot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TelegramHandler обработчик Telegram сообщений
type TelegramHandler struct {
	bot      *telego.Bot
	services *service.Services
	config   *config.Config
	logger   logger.Logger
}

// NewTelegramHandler создает новый обработчик
func NewTelegramHandler(
	bot *telego.Bot,
	services *service.Services,
	config *config.Config,
	logger logger.Logger,
) *TelegramHandler {
	return &TelegramHandler{
		bot:      bot,
		services: services,
		config:   config,
		logger:   logger,
	}
}

// RegisterHandlers регистрирует все обработчики
func (h *TelegramHandler) RegisterHandlers(bh *telegohandler.BotHandler) {
	// Команды в приватном чате
	bh.Handle(h.handlePrivateCommands, telegohandler.AnyCommand(), h.isPrivateChat)

	// Сообщения в приватном чате (не команды)
	bh.Handle(h.handlePrivateMessage, telegohandler.Not(telegohandler.AnyCommand()), h.isPrivateChat)

	// Сообщения из канала в группе (автопересылка)
	bh.Handle(h.handleChannelPost, h.isChannelForward)

	// Ответы в группе на сообщения бота
	bh.Handle(h.handleGroupReply, h.isGroupReply)

	// Команды в группе
	bh.Handle(h.handleGroupCommands, telegohandler.AnyCommand(), h.isGroupChat)

	// Обработка callback запросов
	bh.HandleCallbackQuery(h.handleCallbackQuery, telegohandler.AnyCallbackQueryWithMessage())
}

// handlePrivateCommands обрабатывает команды в приватном чате
func (h *TelegramHandler) handlePrivateCommands(bot *telego.Bot, update telego.Update) {
	message := update.Message
	ctx := context.Background()

	log := h.logger.WithFields(map[string]interface{}{
		"user_id":   message.From.ID,
		"username":  message.From.Username,
		"command":   message.Text,
		"chat_type": message.Chat.Type,
	})

	// Проверка доступа пользователя
	isBlocked, err := h.services.User.IsUserBlocked(ctx, message.From.ID)
	if err != nil {
		log.WithError(err).Error("Failed to check user block status")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при проверке доступа")
		return
	}

	if isBlocked {
		h.sendMessage(bot, message.Chat.ID, "❌ Ваш доступ к боту заблокирован")
		return
	}

	switch message.Text {
	case "/start":
		h.handleStartCommand(ctx, bot, message, log)
	case "/help":
		h.handleHelpCommand(bot, message)
	case "/status":
		h.handleStatusCommand(ctx, bot, message, log)
	case "/close":
		h.handleCloseCommand(ctx, bot, message, log)
	case "/history":
		h.handleHistoryCommand(ctx, bot, message, log)
	default:
		h.sendMessage(bot, message.Chat.ID, "❓ Неизвестная команда. Используйте /help для просмотра доступных команд.")
	}
}

// handleStartCommand обрабатывает команду /start
func (h *TelegramHandler) handleStartCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	userReq := &models.CreateUserRequest{
		UserID:       message.From.ID,
		FirstName:    message.From.FirstName,
		LastName:     message.From.LastName,
		Username:     message.From.Username,
		LanguageCode: message.From.LanguageCode,
		IsPremium:    message.From.IsPremium,
		IsBot:        message.From.IsBot,
	}

	user, err := h.services.User.GetOrCreateUser(ctx, userReq)
	if err != nil {
		log.WithError(err).Error("Failed to get or create user")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при инициализации")
		return
	}

	welcomeText := fmt.Sprintf(
		"👋 Добро пожаловать в службу техподдержки, %s!\n\n"+
			"🔹 Опишите вашу проблему <b>одним сообщением</b>\n"+
			"🔹 Наши специалисты ответят вам в кратчайшие сроки\n"+
			"🔹 Используйте /help для просмотра команд\n\n"+
			"💡 <i>Чем подробнее опишете проблему, тем быстрее сможем помочь!</i>",
		user.GetDisplayName(),
	)

	if user.CreatedAt.After(time.Now().Add(-time.Minute)) {
		log.Info("New user registered")
		welcomeText = "🎉 " + welcomeText
	} else {
		welcomeText = "🔄 С возвращением! " + welcomeText
	}

	h.sendMessage(bot, message.Chat.ID, welcomeText)
}

// handleHelpCommand обрабатывает команду /help
func (h *TelegramHandler) handleHelpCommand(bot *telego.Bot, message *telego.Message) {
	helpText := `📋 <b>Доступные команды:</b>

/start - Начать работу с ботом
/help - Показать это сообщение
/status - Проверить статус активных обращений
/close - Закрыть текущее обращение
/history - История ваших обращений

📝 <b>Как создать обращение:</b>
Просто отправьте сообщение с описанием проблемы

⏱ <b>Время работы поддержки:</b>
Понедельник - Пятница: 9:00 - 18:00 (МСК)
Суббота - Воскресенье: 10:00 - 16:00 (МСК)

📞 <b>Экстренная связь:</b>
По критичным вопросам: @admin`

	h.sendMessage(bot, message.Chat.ID, helpText)
}

// handleStatusCommand обрабатывает команду /status
func (h *TelegramHandler) handleStatusCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	ticket, err := h.services.Ticket.GetActiveTicket(ctx, message.From.ID)
	if err != nil {
		log.WithError(err).Error("Failed to get active ticket")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при получении статуса")
		return
	}

	if ticket == nil {
		h.sendMessage(bot, message.Chat.ID, "📋 У вас нет активных обращений")
		return
	}

	statusEmoji := map[models.TicketStatus]string{
		models.TicketStatusOpen:       "🔵",
		models.TicketStatusInProgress: "🟡",
		models.TicketStatusWaiting:    "🟠",
	}

	statusText := map[models.TicketStatus]string{
		models.TicketStatusOpen:       "Ожидает обработки",
		models.TicketStatusInProgress: "В работе",
		models.TicketStatusWaiting:    "Ожидает ответа",
	}

	createdTime := ticket.CreatedAt.Format("02.01.2006 15:04")
	elapsedTime := time.Since(ticket.CreatedAt).Round(time.Minute)

	text := fmt.Sprintf(
		"%s <b>Обращение #%s</b>\n\n"+
			"📅 Создано: %s\n"+
			"⏱ Прошло времени: %s\n"+
			"🏷 Статус: %s\n"+
			"📝 Тема: %s\n\n"+
			"💬 Сообщений: %d",
		statusEmoji[ticket.Status],
		ticket.TicketNumber,
		createdTime,
		elapsedTime,
		statusText[ticket.Status],
		ticket.Subject,
		len(ticket.Messages),
	)

	if ticket.FirstResponseAt != nil {
		responseTime := ticket.FirstResponseAt.Sub(ticket.CreatedAt).Round(time.Minute)
		text += fmt.Sprintf("\n⚡ Время первого ответа: %s", responseTime)
	}

	keyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "🔄 Обновить", CallbackData: "refresh_status"},
				{Text: "❌ Закрыть обращение", CallbackData: fmt.Sprintf("close_ticket:%s", ticket.ID.Hex())},
			},
		},
	}

	_, err = bot.SendMessage(&telego.SendMessageParams{
		ChatID:      telegoutil.ID(message.Chat.ID),
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: keyboard,
	})
	if err != nil {
		log.WithError(err).Error("Failed to send status message")
	}
}

// handleCloseCommand обрабатывает команду /close
func (h *TelegramHandler) handleCloseCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	ticket, err := h.services.Ticket.GetActiveTicket(ctx, message.From.ID)
	if err != nil {
		log.WithError(err).Error("Failed to get active ticket")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при поиске обращения")
		return
	}

	if ticket == nil {
		h.sendMessage(bot, message.Chat.ID, "📋 У вас нет активных обращений для закрытия")
		return
	}

	keyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "✅ Да, закрыть", CallbackData: fmt.Sprintf("confirm_close:%s", ticket.ID.Hex())},
				{Text: "❌ Отмена", CallbackData: "cancel_close"},
			},
		},
	}

	text := fmt.Sprintf(
		"❓ Вы уверены, что хотите закрыть обращение #%s?\n\n"+
			"После закрытия вы сможете оценить качество обслуживания.",
		ticket.TicketNumber,
	)

	_, err = bot.SendMessage(&telego.SendMessageParams{
		ChatID:      telegoutil.ID(message.Chat.ID),
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: keyboard,
	})
	if err != nil {
		log.WithError(err).Error("Failed to send close confirmation")
	}
}

// handleHistoryCommand обрабатывает команду /history
func (h *TelegramHandler) handleHistoryCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	tickets, err := h.services.Ticket.GetUserTickets(ctx, message.From.ID, "")
	if err != nil {
		log.WithError(err).Error("Failed to get user tickets")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при получении истории")
		return
	}

	if len(tickets) == 0 {
		h.sendMessage(bot, message.Chat.ID, "📋 У вас пока нет обращений")
		return
	}

	text := "📚 <b>История ваших обращений:</b>\n\n"

	for i, ticket := range tickets {
		if i >= 10 { // Показываем только последние 10
			break
		}

		statusEmoji := map[models.TicketStatus]string{
			models.TicketStatusOpen:       "🔵",
			models.TicketStatusInProgress: "🟡",
			models.TicketStatusWaiting:    "🟠",
			models.TicketStatusClosed:     "✅",
			models.TicketStatusRated:      "⭐",
		}

		emoji, exists := statusEmoji[ticket.Status]
		if !exists {
			emoji = "❓"
		}

		createdDate := ticket.CreatedAt.Format("02.01.2006")
		subject := ticket.Subject
		if len(subject) > 30 {
			subject = subject[:30] + "..."
		}

		text += fmt.Sprintf(
			"%s <code>#%s</code> - %s\n📅 %s",
			emoji,
			ticket.TicketNumber,
			subject,
			createdDate,
		)

		if ticket.Survey != nil && ticket.Survey.Rating > 0 {
			text += fmt.Sprintf(" | ⭐ %d/5", ticket.Survey.Rating)
		}

		text += "\n\n"
	}

	if len(tickets) > 10 {
		text += fmt.Sprintf("... и еще %d обращений", len(tickets)-10)
	}

	h.sendMessage(bot, message.Chat.ID, text)
}

// handlePrivateMessage обрабатывает обычные сообщения в приватном чате
func (h *TelegramHandler) handlePrivateMessage(bot *telego.Bot, update telego.Update) {
	message := update.Message
	ctx := context.Background()

	log := h.logger.WithFields(map[string]interface{}{
		"user_id":   message.From.ID,
		"username":  message.From.Username,
		"chat_type": message.Chat.Type,
	})

	// Проверка доступа пользователя
	isBlocked, err := h.services.User.IsUserBlocked(ctx, message.From.ID)
	if err != nil {
		log.WithError(err).Error("Failed to check user block status")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при проверке доступа")
		return
	}

	if isBlocked {
		h.sendMessage(bot, message.Chat.ID, "❌ Ваш доступ к боту заблокирован")
		return
	}

	// Получаем или создаем пользователя
	userReq := &models.CreateUserRequest{
		UserID:       message.From.ID,
		FirstName:    message.From.FirstName,
		LastName:     message.From.LastName,
		Username:     message.From.Username,
		LanguageCode: message.From.LanguageCode,
		IsPremium:    message.From.IsPremium,
		IsBot:        message.From.IsBot,
	}

	user, err := h.services.User.GetOrCreateUser(ctx, userReq)
	if err != nil {
		log.WithError(err).Error("Failed to get or create user")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при обработке запроса")
		return
	}

	// Проверяем активный тикет
	activeTicket, err := h.services.Ticket.GetActiveTicket(ctx, message.From.ID)
	if err != nil {
		log.WithError(err).Error("Failed to get active ticket")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при поиске активного обращения")
		return
	}

	if activeTicket == nil {
		// Создаем новый тикет
		h.createNewTicket(ctx, bot, message, user, log)
	} else {
		// Добавляем сообщение к существующему тикету
		h.addMessageToTicket(ctx, bot, message, activeTicket, log)
	}
}

// createNewTicket создает новый тикет
func (h *TelegramHandler) createNewTicket(ctx context.Context, bot *telego.Bot, message *telego.Message, user *models.User, log logger.Logger) {
	// Создаем тикет
	ticketReq := &models.CreateTicketRequest{
		UserID:      message.From.ID,
		Subject:     h.extractSubject(message.Text),
		Description: message.Text,
		Priority:    models.PriorityNormal,
	}

	ticket, err := h.services.Ticket.CreateTicket(ctx, ticketReq)
	if err != nil {
		log.WithError(err).Error("Failed to create ticket")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при создании обращения")
		return
	}

	// Отправляем сообщение в канал
	channelText := fmt.Sprintf(
		"🎫 <b>Новое обращение #%s</b>\n"+
			"👤 От: %s\n"+
			"📅 Дата: %s\n\n"+
			"📝 <b>Описание:</b>\n%s",
		ticket.TicketNumber,
		user.GetDisplayName(),
		ticket.CreatedAt.Format("02.01.2006 15:04"),
		ticket.Description,
	)

	channelMessage, err := bot.SendMessage(&telego.SendMessageParams{
		ChatID:    telegoutil.ID(h.config.Bot.ChannelID),
		Text:      channelText,
		ParseMode: "HTML",
	})
	if err != nil {
		log.WithError(err).Error("Failed to send message to channel")
	} else {
		// Сохраняем ID сообщения в канале
		err = h.services.Ticket.SetChannelMessage(ctx, ticket.ID, channelMessage.MessageID)
		if err != nil {
			log.WithError(err).Error("Failed to set channel message ID")
		}
	}

	// Подтверждение пользователю
	confirmText := fmt.Sprintf(
		"✅ <b>Ваше обращение принято!</b>\n\n"+
			"🎫 Номер: #%s\n"+
			"📅 Создано: %s\n\n"+
			"💬 Наши специалисты свяжутся с вами в ближайшее время.\n"+
			"📱 Вы получите уведомление о каждом ответе.",
		ticket.TicketNumber,
		ticket.CreatedAt.Format("02.01.2006 15:04"),
	)

	h.sendMessage(bot, message.Chat.ID, confirmText)

	log.WithFields(map[string]interface{}{
		"ticket_id":     ticket.ID.Hex(),
		"ticket_number": ticket.TicketNumber,
	}).Info("New ticket created")
}

// addMessageToTicket добавляет сообщение к существующему тикету
func (h *TelegramHandler) addMessageToTicket(ctx context.Context, bot *telego.Bot, message *telego.Message, ticket *models.Ticket, log logger.Logger) {
	// Добавляем сообщение
	msgReq := &models.AddMessageRequest{
		MessageID:   message.MessageID,
		FromUserID:  message.From.ID,
		FromSupport: false,
		Text:        message.Text,
		MessageType: "text",
	}

	err := h.services.Ticket.AddMessage(ctx, ticket.ID, msgReq)
	if err != nil {
		log.WithError(err).Error("Failed to add message to ticket")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при добавлении сообщения")
		return
	}

	// Пересылаем сообщение в группу поддержки
	if ticket.ThreadID != 0 {
		_, err = bot.CopyMessage(&telego.CopyMessageParams{
			ChatID:           telegoutil.ID(h.config.Bot.GroupID),
			FromChatID:       telegoutil.ID(message.Chat.ID),
			MessageID:        message.MessageID,
			ReplyToMessageID: &ticket.ThreadID,
		})
		if err != nil {
			log.WithError(err).Error("Failed to copy message to support group")
		}
	}

	// Обновляем статус тикета на "ожидает ответа"
	err = h.services.Ticket.ChangeTicketStatus(ctx, ticket.ID, models.TicketStatusWaiting)
	if err != nil {
		log.WithError(err).Error("Failed to update ticket status")
	}

	h.sendMessage(bot, message.Chat.ID, "📩 Сообщение добавлено к вашему обращению #"+ticket.TicketNumber)

	log.WithFields(map[string]interface{}{
		"ticket_id":     ticket.ID.Hex(),
		"ticket_number": ticket.TicketNumber,
		"message_id":    message.MessageID,
	}).Debug("Message added to ticket")
}

// Вспомогательные методы

func (h *TelegramHandler) extractSubject(text string) string {
	lines := strings.Split(text, "\n")
	subject := lines[0]
	if len(subject) > 100 {
		subject = subject[:97] + "..."
	}
	return subject
}

func (h *TelegramHandler) sendMessage(bot *telego.Bot, chatID int64, text string) {
	_, err := bot.SendMessage(&telego.SendMessageParams{
		ChatID:    telegoutil.ID(chatID),
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		h.logger.WithError(err).WithField("chat_id", chatID).Error("Failed to send message")
	}
}

func (h *TelegramHandler) sendErrorMessage(bot *telego.Bot, chatID int64, text string) {
	errorText := "⚠️ " + text + "\n\nПопробуйте еще раз или обратитесь к администратору."
	h.sendMessage(bot, chatID, errorText)
}

// Фильтры для обработчиков

func (h *TelegramHandler) isPrivateChat(update telego.Update) bool {
	return update.Message != nil && update.Message.Chat.Type == "private"
}

func (h *TelegramHandler) isGroupChat(update telego.Update) bool {
	return update.Message != nil &&
		update.Message.Chat.ID == h.config.Bot.GroupID
}

func (h *TelegramHandler) isChannelForward(update telego.Update) bool {
	return update.Message != nil &&
		update.Message.From != nil &&
		update.Message.From.ID == 777000 && // Telegram service account
		update.Message.IsAutomaticForward &&
		update.Message.Chat.ID == h.config.Bot.GroupID &&
		update.Message.SenderChat != nil &&
		update.Message.SenderChat.ID == h.config.Bot.ChannelID
}

func (h *TelegramHandler) isGroupReply(update telego.Update) bool {
	return update.Message != nil &&
		update.Message.Chat.ID == h.config.Bot.GroupID &&
		update.Message.ReplyToMessage != nil &&
		update.Message.ReplyToMessage.From != nil &&
		update.Message.ReplyToMessage.From.ID == h.config.Bot.GetBotUser().ID
}

// handleChannelPost обрабатывает посты из канала
func (h *TelegramHandler) handleChannelPost(bot *telego.Bot, update telego.Update) {
	// TODO: Implement channel post handling
}

// handleGroupReply обрабатывает ответы в группе
func (h *TelegramHandler) handleGroupReply(bot *telego.Bot, update telego.Update) {
	// TODO: Implement group reply handling
}

// handleGroupCommands обрабатывает команды в группе
func (h *TelegramHandler) handleGroupCommands(bot *telego.Bot, update telego.Update) {
	// TODO: Implement group commands handling
}

// handleCallbackQuery обрабатывает callback запросы
func (h *TelegramHandler) handleCallbackQuery(bot *telego.Bot, query telego.CallbackQuery) {
	ctx := context.Background()

	log := h.logger.WithFields(map[string]interface{}{
		"user_id":       query.From.ID,
		"callback_data": query.Data,
	})

	switch {
	case strings.HasPrefix(query.Data, "confirm_close:"):
		h.handleConfirmClose(ctx, bot, query, log)
	case strings.HasPrefix(query.Data, "close_ticket:"):
		h.handleCloseTicket(ctx, bot, query, log)
	case query.Data == "cancel_close":
		h.handleCancelClose(bot, query)
	case query.Data == "refresh_status":
		h.handleRefreshStatus(ctx, bot, query, log)
	case strings.HasPrefix(query.Data, "rate:"):
		h.handleRating(ctx, bot, query, log)
	default:
		h.answerCallbackQuery(bot, query.ID, "❓ Неизвестное действие")
	}
}

func (h *TelegramHandler) handleConfirmClose(ctx context.Context, bot *telego.Bot, query telego.CallbackQuery, log logger.Logger) {
	ticketIDStr := strings.TrimPrefix(query.Data, "confirm_close:")
	ticketID, err := h.parseObjectID(ticketIDStr)
	if err != nil {
		log.WithError(err).Error("Invalid ticket ID")
		h.answerCallbackQuery(bot, query.ID, "❌ Неверный ID обращения")
		return
	}

	err = h.services.Ticket.CloseTicket(ctx, ticketID, query.From.ID)
	if err != nil {
		log.WithError(err).Error("Failed to close ticket")
		h.answerCallbackQuery(bot, query.ID, "❌ Ошибка при закрытии обращения")
		return
	}

	// Удаляем клавиатуру и обновляем сообщение
	_, err = bot.EditMessageText(&telego.EditMessageTextParams{
		ChatID:    telegoutil.ID(query.Message.Chat.ID),
		MessageID: query.Message.MessageID,
		Text:      "✅ Обращение закрыто!\n\nСпасибо за обращение. Пожалуйста, оцените качество обслуживания:",
		ParseMode: "HTML",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "⭐", CallbackData: fmt.Sprintf("rate:%s:1", ticketIDStr)},
					{Text: "⭐⭐", CallbackData: fmt.Sprintf("rate:%s:2", ticketIDStr)},
					{Text: "⭐⭐⭐", CallbackData: fmt.Sprintf("rate:%s:3", ticketIDStr)},
					{Text: "⭐⭐⭐⭐", CallbackData: fmt.Sprintf("rate:%s:4", ticketIDStr)},
					{Text: "⭐⭐⭐⭐⭐", CallbackData: fmt.Sprintf("rate:%s:5", ticketIDStr)},
				},
			},
		},
	})
	if err != nil {
		log.WithError(err).Error("Failed to edit message")
	}

	h.answerCallbackQuery(bot, query.ID, "✅ Обращение закрыто")
}

func (h *TelegramHandler) handleRating(ctx context.Context, bot *telego.Bot, query telego.CallbackQuery, log logger.Logger) {
	parts := strings.Split(query.Data, ":")
	if len(parts) != 3 {
		log.Error("Invalid rating callback data format")
		h.answerCallbackQuery(bot, query.ID, "❌ Неверный формат данных")
		return
	}

	ticketIDStr := parts[1]
	ratingStr := parts[2]

	ticketID, err := h.parseObjectID(ticketIDStr)
	if err != nil {
		log.WithError(err).Error("Invalid ticket ID")
		h.answerCallbackQuery(bot, query.ID, "❌ Неверный ID обращения")
		return
	}

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		log.WithError(err).Error("Invalid rating value")
		h.answerCallbackQuery(bot, query.ID, "❌ Неверная оценка")
		return
	}

	surveyReq := &models.SurveyRequest{
		Rating: rating,
	}

	err = h.services.Ticket.SubmitSurvey(ctx, ticketID, surveyReq)
	if err != nil {
		log.WithError(err).Error("Failed to submit survey")
		h.answerCallbackQuery(bot, query.ID, "❌ Ошибка при сохранении оценки")
		return
	}

	// Обновляем сообщение
	ratingText := strings.Repeat("⭐", rating)
	_, err = bot.EditMessageText(&telego.EditMessageTextParams{
		ChatID:    telegoutil.ID(query.Message.Chat.ID),
		MessageID: query.Message.MessageID,
		Text:      fmt.Sprintf("✅ Спасибо за оценку!\n\n%s (%d/5)\n\nВаше мнение поможет нам стать лучше!", ratingText, rating),
		ParseMode: "HTML",
	})
	if err != nil {
		log.WithError(err).Error("Failed to edit message")
	}

	h.answerCallbackQuery(bot, query.ID, fmt.Sprintf("✅ Оценка %d/5 сохранена", rating))
}

func (h *TelegramHandler) handleCancelClose(bot *telego.Bot, query telego.CallbackQuery) {
	// Удаляем сообщение с подтверждением
	err := bot.DeleteMessage(&telego.DeleteMessageParams{
		ChatID:    telegoutil.ID(query.Message.Chat.ID),
		MessageID: query.Message.MessageID,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete message")
	}

	h.answerCallbackQuery(bot, query.ID, "❌ Отменено")
}

func (h *TelegramHandler) handleRefreshStatus(ctx context.Context, bot *telego.Bot, query telego.CallbackQuery, log logger.Logger) {
	// Перезапускаем команду /status
	message := &telego.Message{
		MessageID: query.Message.MessageID,
		From:      query.From,
		Chat:      query.Message.Chat,
		Text:      "/status",
	}

	h.handleStatusCommand(ctx, bot, message, log)
	h.answerCallbackQuery(bot, query.ID, "🔄 Статус обновлен")
}

func (h *TelegramHandler) answerCallbackQuery(bot *telego.Bot, queryID string, text string) {
	err := bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: queryID,
		Text:            text,
		ShowAlert:       false,
	})
	if err != nil {
		h.logger.WithError(err).WithField("query_id", queryID).Error("Failed to answer callback query")
	}
}

func (h *TelegramHandler) parseObjectID(idStr string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(idStr)
}

// Методы конфигурации бота
func (c *config.BotConfig) GetBotUser() *telego.User {
	// TODO: Implement bot user retrieval
	return &telego.User{ID: 0}
}
