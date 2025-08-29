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
	"github.com/memrook/oneway_subot/pkg/metrics"
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
	bh.Handle(func(ctx *telegohandler.Context, update telego.Update) error {
		h.handlePrivateCommands(ctx, ctx.Bot(), update)
		return nil
	}, telegohandler.AnyCommand(), func(ctx context.Context, update telego.Update) bool {
		return h.isPrivateChat(update)
	})

	// Сообщения в приватном чате (не команды)
	bh.Handle(func(ctx *telegohandler.Context, update telego.Update) error {
		h.handlePrivateMessage(ctx, ctx.Bot(), update)
		return nil
	}, telegohandler.Not(telegohandler.AnyCommand()), func(ctx context.Context, update telego.Update) bool {
		return h.isPrivateChat(update)
	})

	// Команды в группе
	bh.Handle(func(ctx *telegohandler.Context, update telego.Update) error {
		h.handleGroupCommands(ctx, ctx.Bot(), update)
		return nil
	}, telegohandler.AnyCommand(), func(ctx context.Context, update telego.Update) bool {
		return h.isGroupChat(update)
	})

	// Сообщения в комментариях канала
	bh.Handle(func(ctx *telegohandler.Context, update telego.Update) error {
		h.handleChannelComments(ctx, ctx.Bot(), update)
		return nil
	}, telegohandler.AnyMessage(), func(ctx context.Context, update telego.Update) bool {
		return h.isChannelComment(update)
	})

	// Обработка callback запросов
	bh.HandleCallbackQuery(func(ctx *telegohandler.Context, query telego.CallbackQuery) error {
		h.handleCallbackQuery(ctx, ctx.Bot(), query)
		return nil
	}, telegohandler.AnyCallbackQueryWithMessage())
}

// handlePrivateCommands обрабатывает команды в приватном чате
func (h *TelegramHandler) handlePrivateCommands(ctx context.Context, bot *telego.Bot, update telego.Update) {
	message := update.Message
	startTime := time.Now()

	// Записываем метрику о получении команды
	metrics.RecordTelegramMessage("command", "received")

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
		metrics.RecordTelegramMessage("command", "processed")
		metrics.RecordResponseTime("start_command", time.Since(startTime).Seconds())
	case "/help":
		h.handleHelpCommand(bot, message)
		metrics.RecordTelegramMessage("command", "processed")
		metrics.RecordResponseTime("help_command", time.Since(startTime).Seconds())
	case "/status":
		h.handleStatusCommand(ctx, bot, message, log)
		metrics.RecordTelegramMessage("command", "processed")
		metrics.RecordResponseTime("status_command", time.Since(startTime).Seconds())
	case "/close":
		h.handleCloseCommand(ctx, bot, message, log)
		metrics.RecordTelegramMessage("command", "processed")
		metrics.RecordResponseTime("close_command", time.Since(startTime).Seconds())
	case "/history":
		h.handleHistoryCommand(ctx, bot, message, log)
		metrics.RecordTelegramMessage("command", "processed")
		metrics.RecordResponseTime("history_command", time.Since(startTime).Seconds())
	default:
		h.sendMessage(bot, message.Chat.ID, "❓ Неизвестная команда. Используйте /help для просмотра доступных команд.")
		metrics.RecordTelegramMessage("command", "unknown")
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

	_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
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

	_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
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
func (h *TelegramHandler) handlePrivateMessage(ctx context.Context, bot *telego.Bot, update telego.Update) {
	message := update.Message

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
		// Проверяем, не закрыт ли тикет
		if activeTicket.Status == models.TicketStatusClosed || activeTicket.Status == models.TicketStatusRated {
			// Если тикет закрыт, создаем новый
			h.sendMessage(bot, message.Chat.ID, "💬 Ваше предыдущее обращение было закрыто. Создаем новое обращение...")
			h.createNewTicket(ctx, bot, message, user, log)
		} else {
			// Добавляем сообщение к существующему тикету
			h.addMessageToTicket(ctx, bot, message, activeTicket, log)
		}
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
		metrics.RecordError("ticket_service", "create_failed")
		return
	}

	// Записываем метрику создания тикета
	metrics.RecordTicketCreated(string(ticket.Priority), "user_message")

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

	channelMessage, err := bot.SendMessage(ctx, &telego.SendMessageParams{
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

		// Отправляем инструкцию в комментарии (группу обсуждения)
		h.sendInstructionComment(ctx, bot, channelMessage.MessageID, log)
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
		MessageType: h.getMessageType(message),
	}

	err := h.services.Ticket.AddMessage(ctx, ticket.ID, msgReq)
	if err != nil {
		log.WithError(err).Error("Failed to add message to ticket")
		h.sendErrorMessage(bot, message.Chat.ID, "Произошла ошибка при добавлении сообщения")
		return
	}

	// Пересылаем сообщение пользователя в группу обсуждения как ответ на тему тикета
	if ticket.ChannelMessageID != 0 && h.config.Bot.GroupID != 0 {
		// Копируем сообщение пользователя в группу обсуждения
		_, err = bot.CopyMessage(ctx, &telego.CopyMessageParams{
			ChatID:     telegoutil.ID(h.config.Bot.GroupID),
			FromChatID: telegoutil.ID(message.Chat.ID),
			MessageID:  message.MessageID,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: ticket.ChannelMessageID,
			},
		})
		if err != nil {
			log.WithError(err).Error("Failed to copy user message to discussion group")

			// Fallback: отправляем как текстовое сообщение с указанием автора
			user, userErr := h.services.User.GetUser(ctx, message.From.ID)
			if userErr == nil {
				fallbackText := fmt.Sprintf(
					"👤 <b>%s</b> добавил сообщение:\n\n%s",
					user.GetDisplayName(),
					message.Text,
				)

				_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID:    telegoutil.ID(h.config.Bot.GroupID),
					Text:      fallbackText,
					ParseMode: "HTML",
					ReplyParameters: &telego.ReplyParameters{
						MessageID: ticket.ChannelMessageID,
					},
				})
				if err != nil {
					log.WithError(err).Error("Failed to send fallback message to discussion group")
				}
			}
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
	ctx := context.Background()
	_, err := bot.SendMessage(ctx, &telego.SendMessageParams{
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

func (h *TelegramHandler) isChannelComment(update telego.Update) bool {
	return update.Message != nil &&
		update.Message.Chat.ID == h.config.Bot.GroupID &&
		update.Message.ReplyToMessage != nil &&
		(update.Message.ReplyToMessage.SenderChat != nil &&
			update.Message.ReplyToMessage.SenderChat.ID == h.config.Bot.ChannelID ||
			update.Message.MessageThreadID != 0)
}

func (h *TelegramHandler) isGroupReply(update telego.Update) bool {
	return update.Message != nil &&
		update.Message.Chat.ID == h.config.Bot.GroupID &&
		update.Message.ReplyToMessage != nil &&
		update.Message.ReplyToMessage.From != nil &&
		update.Message.ReplyToMessage.From.ID == 0 // TODO: Get bot ID
}

// handleChannelPost обрабатывает посты из канала
func (h *TelegramHandler) handleChannelPost(ctx context.Context, bot *telego.Bot, update telego.Update) {
	// TODO: Implement channel post handling
}

// handleChannelComments обрабатывает комментарии в канале
// Примечание: для работы комментариев канал должен иметь связанную группу обсуждения (Discussion Group)
// Комментарии приходят в эту группу как reply на переадресованные сообщения из канала
func (h *TelegramHandler) handleChannelComments(ctx context.Context, bot *telego.Bot, update telego.Update) {
	message := update.Message

	log := h.logger.WithFields(map[string]interface{}{
		"user_id":      message.From.ID,
		"username":     message.From.Username,
		"channel_id":   message.Chat.ID,
		"reply_to_msg": message.ReplyToMessage.MessageID,
	})

	// Находим тикет по ID сообщения в канале или Thread ID
	var searchMessageID int
	if message.MessageThreadID != 0 {
		// Это сообщение в теме - используем Thread ID
		searchMessageID = message.MessageThreadID
	} else {
		// Это ответ на сообщение из канала
		searchMessageID = message.ReplyToMessage.MessageID
	}

	ticket, err := h.services.Ticket.GetTicketByChannelMessage(ctx, searchMessageID)
	if err != nil {
		log.WithError(err).Error("Failed to find ticket by channel message")
		return
	}

	if ticket == nil {
		log.Warn("No ticket found for channel message")
		return
	}

	// Это ответ техподдержки - пересылаем пользователю или выполняем команду
	if message.From.ID != ticket.UserID {
		// Проверяем, является ли сообщение командой
		if h.handleTicketCommand(ctx, bot, message, ticket, log) {
			return
		}
		// Обычный ответ - пересылаем пользователю
		h.forwardSupportReplyToUser(ctx, bot, message, ticket, log)
	}
}

// handleGroupReply обрабатывает ответы в группе
func (h *TelegramHandler) handleGroupReply(ctx context.Context, bot *telego.Bot, update telego.Update) {
	// TODO: Implement group reply handling
}

// handleGroupCommands обрабатывает команды в группе
func (h *TelegramHandler) handleGroupCommands(ctx context.Context, bot *telego.Bot, update telego.Update) {
	message := update.Message

	log := h.logger.WithFields(map[string]interface{}{
		"user_id":   message.From.ID,
		"username":  message.From.Username,
		"command":   message.Text,
		"chat_type": message.Chat.Type,
		"chat_id":   message.Chat.ID,
	})

	// Проверяем, что это команда в нашей группе поддержки
	if message.Chat.ID != h.config.Bot.GroupID {
		return
	}

	// Записываем метрику
	metrics.RecordTelegramMessage("group_command", "received")

	switch {
	case strings.HasPrefix(message.Text, "/stats"):
		h.handleStatsCommand(ctx, bot, message, log)
	case strings.HasPrefix(message.Text, "/assign"):
		h.handleAssignCommand(ctx, bot, message, log)
	case strings.HasPrefix(message.Text, "/close"):
		h.handleAdminCloseCommand(ctx, bot, message, log)
	// case strings.HasPrefix(message.Text, "/reopen"):
	//	h.handleReopenCommand(ctx, bot, message, log)
	// case strings.HasPrefix(message.Text, "/priority"):
	//	h.handlePriorityCommand(ctx, bot, message, log)
	case strings.HasPrefix(message.Text, "/ban"):
		h.handleBanCommand(ctx, bot, message, log)
	case strings.HasPrefix(message.Text, "/unban"):
		h.handleUnbanCommand(ctx, bot, message, log)
	}
}

// handleCallbackQuery обрабатывает callback запросы
func (h *TelegramHandler) handleCallbackQuery(ctx context.Context, bot *telego.Bot, query telego.CallbackQuery) {

	log := h.logger.WithFields(map[string]interface{}{
		"user_id":       query.From.ID,
		"callback_data": query.Data,
	})

	switch {
	case strings.HasPrefix(query.Data, "confirm_close:"):
		h.handleConfirmClose(ctx, bot, query, log)
	// case strings.HasPrefix(query.Data, "close_ticket:"):
	//	h.handleCloseTicket(ctx, bot, query, log)
	case query.Data == "cancel_close":
		h.handleCancelClose(bot, query)
	case query.Data == "refresh_status":
		h.handleRefreshStatus(ctx, bot, query, log)
	case strings.HasPrefix(query.Data, "rate_"):
		h.handleRating(ctx, bot, query, log)
	case strings.HasPrefix(query.Data, "reopen:"):
		h.handleReopenTicket(ctx, bot, query, log)
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
		metrics.RecordError("ticket_service", "close_failed")
		return
	}

	// Записываем метрику закрытия тикета
	metrics.RecordTicketClosed("closed", "user")

	// Удаляем клавиатуру и обновляем сообщение
	_, err = bot.EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:    telegoutil.ID(query.Message.GetChat().ID),
		MessageID: query.Message.GetMessageID(),
		Text:      "✅ Обращение закрыто!\n\nСпасибо за обращение. Пожалуйста, оцените качество обслуживания:",
		ParseMode: "HTML",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "⭐", CallbackData: fmt.Sprintf("rate_%s_1", ticketIDStr)},
					{Text: "⭐⭐", CallbackData: fmt.Sprintf("rate_%s_2", ticketIDStr)},
					{Text: "⭐⭐⭐", CallbackData: fmt.Sprintf("rate_%s_3", ticketIDStr)},
					{Text: "⭐⭐⭐⭐", CallbackData: fmt.Sprintf("rate_%s_4", ticketIDStr)},
					{Text: "⭐⭐⭐⭐⭐", CallbackData: fmt.Sprintf("rate_%s_5", ticketIDStr)},
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
	// Формат: rate_<ticket_id>_<rating>
	parts := strings.Split(query.Data, "_")
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
		metrics.RecordError("survey", "submit_failed")
		return
	}

	// Записываем метрику рейтинга
	metrics.RecordUserRating(rating)

	// Обновляем сообщение
	ratingText := strings.Repeat("⭐", rating)
	_, err = bot.EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:    telegoutil.ID(query.Message.GetChat().ID),
		MessageID: query.Message.GetMessageID(),
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
	ctx := context.Background()
	err := bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    telegoutil.ID(query.Message.GetChat().ID),
		MessageID: query.Message.GetMessageID(),
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete message")
	}

	h.answerCallbackQuery(bot, query.ID, "❌ Отменено")
}

func (h *TelegramHandler) handleRefreshStatus(ctx context.Context, bot *telego.Bot, query telego.CallbackQuery, log logger.Logger) {
	// Перезапускаем команду /status
	message := &telego.Message{
		MessageID: query.Message.GetMessageID(),
		From:      &query.From,
		Chat:      query.Message.GetChat(),
		Text:      "/status",
	}

	h.handleStatusCommand(ctx, bot, message, log)
	h.answerCallbackQuery(bot, query.ID, "🔄 Статус обновлен")
}

func (h *TelegramHandler) answerCallbackQuery(bot *telego.Bot, queryID string, text string) {
	ctx := context.Background()
	err := bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
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

// Administrative commands

// handleStatsCommand показывает статистику тикетов
func (h *TelegramHandler) handleStatsCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	// Получаем статистику открытых тикетов
	openTickets, err := h.services.Ticket.GetOpenTickets(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get open tickets")
		h.sendMessage(bot, message.Chat.ID, "❌ Ошибка при получении статистики")
		return
	}

	// Подсчитываем по статусам
	stats := make(map[string]int)
	for _, ticket := range openTickets {
		stats[string(ticket.Status)]++
	}

	statsText := "📊 <b>Статистика тикетов</b>\n\n"
	statsText += fmt.Sprintf("🟢 Новые: %d\n", stats["open"])
	statsText += fmt.Sprintf("🔵 В работе: %d\n", stats["in_progress"])
	statsText += fmt.Sprintf("🟡 Ожидание: %d\n", stats["waiting"])
	statsText += fmt.Sprintf("\n📈 Всего активных: %d", len(openTickets))

	h.sendMessage(bot, message.Chat.ID, statsText)
}

// handleAssignCommand назначает тикет на администратора
func (h *TelegramHandler) handleAssignCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		h.sendMessage(bot, message.Chat.ID, "❗ Использование: /assign <номер_тикета> [@пользователь]")
		return
	}

	ticketNumber := parts[1]
	ticket, err := h.services.Ticket.GetTicketByNumber(ctx, ticketNumber)
	if err != nil {
		log.WithError(err).Error("Failed to get ticket")
		h.sendMessage(bot, message.Chat.ID, "❌ Тикет не найден")
		return
	}

	var assignedTo int64 = message.From.ID

	err = h.services.Ticket.AssignTicket(ctx, ticket.ID, assignedTo)
	if err != nil {
		log.WithError(err).Error("Failed to assign ticket")
		h.sendMessage(bot, message.Chat.ID, "❌ Ошибка при назначении тикета")
		metrics.RecordError("ticket_service", "assign_failed")
		return
	}

	h.sendMessage(bot, message.Chat.ID, fmt.Sprintf("✅ Тикет #%s назначен на %s", ticketNumber, message.From.FirstName))
}

// handleAdminCloseCommand закрывает тикет администратором
func (h *TelegramHandler) handleAdminCloseCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		h.sendMessage(bot, message.Chat.ID, "❗ Использование: /close <номер_тикета>")
		return
	}

	ticketNumber := parts[1]
	ticket, err := h.services.Ticket.GetTicketByNumber(ctx, ticketNumber)
	if err != nil {
		log.WithError(err).Error("Failed to get ticket")
		h.sendMessage(bot, message.Chat.ID, "❌ Тикет не найден")
		return
	}

	err = h.services.Ticket.CloseTicket(ctx, ticket.ID, message.From.ID)
	if err != nil {
		log.WithError(err).Error("Failed to close ticket")
		h.sendMessage(bot, message.Chat.ID, "❌ Ошибка при закрытии тикета")
		metrics.RecordError("ticket_service", "admin_close_failed")
		return
	}

	metrics.RecordTicketClosed("closed", "admin")
	h.sendMessage(bot, message.Chat.ID, fmt.Sprintf("✅ Тикет #%s закрыт администратором", ticketNumber))
}

// handleBanCommand блокирует пользователя
func (h *TelegramHandler) handleBanCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		h.sendMessage(bot, message.Chat.ID, "❗ Использование: /ban <user_id> [причина]")
		return
	}

	userIDStr := parts[1]
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.sendMessage(bot, message.Chat.ID, "❌ Неверный ID пользователя")
		return
	}

	reason := "Заблокирован администратором"
	if len(parts) > 2 {
		reason = strings.Join(parts[2:], " ")
	}

	err = h.services.User.BlockUser(ctx, userID, reason)
	if err != nil {
		log.WithError(err).Error("Failed to block user")
		h.sendMessage(bot, message.Chat.ID, "❌ Ошибка при блокировке пользователя")
		return
	}

	h.sendMessage(bot, message.Chat.ID, fmt.Sprintf("✅ Пользователь %d заблокирован", userID))
}

// handleUnbanCommand разблокирует пользователя
func (h *TelegramHandler) handleUnbanCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, log logger.Logger) {
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		h.sendMessage(bot, message.Chat.ID, "❗ Использование: /unban <user_id>")
		return
	}

	userIDStr := parts[1]
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.sendMessage(bot, message.Chat.ID, "❌ Неверный ID пользователя")
		return
	}

	err = h.services.User.UnblockUser(ctx, userID)
	if err != nil {
		log.WithError(err).Error("Failed to unblock user")
		h.sendMessage(bot, message.Chat.ID, "❌ Ошибка при разблокировке пользователя")
		return
	}

	h.sendMessage(bot, message.Chat.ID, fmt.Sprintf("✅ Пользователь %d разблокирован", userID))
}

// forwardSupportReplyToUser пересылает ответ техподдержки пользователю
func (h *TelegramHandler) forwardSupportReplyToUser(ctx context.Context, bot *telego.Bot, message *telego.Message, ticket *models.Ticket, log logger.Logger) {
	// Добавляем сообщение к тикету
	msgReq := &models.AddMessageRequest{
		MessageID:   message.MessageID,
		FromUserID:  message.From.ID,
		FromSupport: true,
		Text:        message.Text,
		MessageType: h.getMessageType(message),
	}

	err := h.services.Ticket.AddMessage(ctx, ticket.ID, msgReq)
	if err != nil {
		log.WithError(err).Error("Failed to add support message to ticket")
		return
	}

	// Формируем имя сотрудника техподдержки
	var supportName string
	if message.From.FirstName != "" {
		supportName = message.From.FirstName
		if message.From.LastName != "" {
			supportName += " " + message.From.LastName
		}
	} else if message.From.Username != "" {
		supportName = "@" + message.From.Username
	} else {
		supportName = "Техподдержка"
	}

	// Отправляем заголовок с информацией о том, кто ответил
	headerText := fmt.Sprintf("💬 <b>%s</b> ответил по обращению #%s:", supportName, ticket.TicketNumber)
	_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telegoutil.ID(ticket.UserID),
		Text:      headerText,
		ParseMode: "HTML",
	})
	if err != nil {
		log.WithError(err).Error("Failed to send support reply header to user")
	}

	// Копируем оригинальное сообщение техподдержки пользователю
	_, err = bot.CopyMessage(ctx, &telego.CopyMessageParams{
		ChatID:     telegoutil.ID(ticket.UserID),
		FromChatID: telegoutil.ID(message.Chat.ID),
		MessageID:  message.MessageID,
	})
	if err != nil {
		log.WithError(err).Error("Failed to copy support message to user")

		// Fallback: отправляем только текст, если копирование не удалось
		if message.Text != "" {
			fallbackText := fmt.Sprintf("📝 %s", message.Text)
			_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
				ChatID:    telegoutil.ID(ticket.UserID),
				Text:      fallbackText,
				ParseMode: "HTML",
			})
			if err != nil {
				log.WithError(err).Error("Failed to send fallback support message to user")
			}
		}
	}

	// Обновляем статус тикета на "в работе"
	err = h.services.Ticket.ChangeTicketStatus(ctx, ticket.ID, models.TicketStatusInProgress)
	if err != nil {
		log.WithError(err).Error("Failed to update ticket status")
	}

	log.Info("Support reply forwarded to user")
}

// handleTicketCommand обрабатывает команды управления тикетом в комментариях
func (h *TelegramHandler) handleTicketCommand(ctx context.Context, bot *telego.Bot, message *telego.Message, ticket *models.Ticket, log logger.Logger) bool {
	text := strings.TrimSpace(message.Text)

	switch {
	case text == "/resolve" || text == "/solved":
		return h.handleResolveTicket(ctx, bot, message, ticket, log)
	case text == "/close":
		return h.handleCloseTicketFromChannel(ctx, bot, message, ticket, log)
	case strings.HasPrefix(text, "/priority"):
		return h.handleChangePriority(ctx, bot, message, ticket, log)
	case strings.HasPrefix(text, "/assign"):
		return h.handleAssignFromChannel(ctx, bot, message, ticket, log)
	default:
		return false // Не команда
	}
}

// handleResolveTicket помечает тикет как решенный
func (h *TelegramHandler) handleResolveTicket(ctx context.Context, bot *telego.Bot, message *telego.Message, ticket *models.Ticket, log logger.Logger) bool {
	// Обновляем статус на "решен"
	err := h.services.Ticket.ChangeTicketStatus(ctx, ticket.ID, models.TicketStatusClosed)
	if err != nil {
		log.WithError(err).Error("Failed to resolve ticket")
		return true
	}

	// Уведомляем пользователя
	userMessage := fmt.Sprintf(
		"✅ <b>Ваше обращение #%s помечено как решенное</b>\n\n"+
			"Если проблема решена, нажмите «Закрыть обращение».\n"+
			"Если нужна дополнительная помощь, просто отправьте сообщение.",
		ticket.TicketNumber,
	)

	keyboard := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "✅ Закрыть обращение", CallbackData: fmt.Sprintf("confirm_close:%s", ticket.ID.Hex())},
				{Text: "🔄 Возобновить", CallbackData: fmt.Sprintf("reopen:%s", ticket.ID.Hex())},
			},
		},
	}

	_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:      telegoutil.ID(ticket.UserID),
		Text:        userMessage,
		ParseMode:   "HTML",
		ReplyMarkup: keyboard,
	})
	if err != nil {
		log.WithError(err).Error("Failed to send resolution notification to user")
	}

	// Отвечаем в группе обсуждения
	_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telegoutil.ID(message.Chat.ID),
		Text:   "✅ Тикет помечен как решенный. Пользователь уведомлен.",
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	if err != nil {
		log.WithError(err).Error("Failed to send resolution confirmation")
	}

	return true
}

// handleCloseTicketFromChannel закрывает тикет из канала
func (h *TelegramHandler) handleCloseTicketFromChannel(ctx context.Context, bot *telego.Bot, message *telego.Message, ticket *models.Ticket, log logger.Logger) bool {
	err := h.services.Ticket.CloseTicket(ctx, ticket.ID, message.From.ID)
	if err != nil {
		log.WithError(err).Error("Failed to close ticket from channel")
		return true
	}

	// Записываем метрику
	metrics.RecordTicketClosed("closed", "admin")

	// Уведомляем пользователя
	userMessage := fmt.Sprintf(
		"✅ <b>Ваше обращение #%s закрыто</b>\n\n"+
			"Спасибо за обращение! Если у вас возникнут новые вопросы, "+
			"просто отправьте сообщение боту.",
		ticket.TicketNumber,
	)

	_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telegoutil.ID(ticket.UserID),
		Text:      userMessage,
		ParseMode: "HTML",
	})
	if err != nil {
		log.WithError(err).Error("Failed to send closure notification to user")
	}

	return true
}

// handleChangePriority изменяет приоритет тикета
func (h *TelegramHandler) handleChangePriority(ctx context.Context, bot *telego.Bot, message *telego.Message, ticket *models.Ticket, log logger.Logger) bool {
	parts := strings.Fields(message.Text)
	if len(parts) < 2 {
		return true
	}

	var priority models.TicketPriority
	switch strings.ToLower(parts[1]) {
	case "low", "низкий":
		priority = models.PriorityLow
	case "normal", "обычный":
		priority = models.PriorityNormal
	case "high", "высокий":
		priority = models.PriorityHigh
	case "critical", "критический":
		priority = models.PriorityCritical
	default:
		return true
	}

	updateReq := &models.UpdateTicketRequest{
		Priority: &priority,
	}

	err := h.services.Ticket.UpdateTicket(ctx, ticket.ID, updateReq)
	if err != nil {
		log.WithError(err).Error("Failed to update ticket priority")
		return true
	}

	_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telegoutil.ID(message.Chat.ID),
		Text:   fmt.Sprintf("✅ Приоритет тикета изменен на: %s", priority),
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	if err != nil {
		log.WithError(err).Error("Failed to send priority change confirmation")
	}

	return true
}

// handleAssignFromChannel назначает тикет из канала
func (h *TelegramHandler) handleAssignFromChannel(ctx context.Context, bot *telego.Bot, message *telego.Message, ticket *models.Ticket, log logger.Logger) bool {
	assignedTo := message.From.ID

	err := h.services.Ticket.AssignTicket(ctx, ticket.ID, assignedTo)
	if err != nil {
		log.WithError(err).Error("Failed to assign ticket from channel")
		return true
	}

	_, err = bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telegoutil.ID(message.Chat.ID),
		Text:   fmt.Sprintf("✅ Тикет назначен на %s", message.From.FirstName),
		ReplyParameters: &telego.ReplyParameters{
			MessageID: message.MessageID,
		},
	})
	if err != nil {
		log.WithError(err).Error("Failed to send assignment confirmation")
	}

	return true
}

// handleReopenTicket возобновляет закрытый тикет
func (h *TelegramHandler) handleReopenTicket(ctx context.Context, bot *telego.Bot, query telego.CallbackQuery, log logger.Logger) {
	ticketIDStr := strings.TrimPrefix(query.Data, "reopen:")
	ticketID, err := h.parseObjectID(ticketIDStr)
	if err != nil {
		log.WithError(err).Error("Invalid ticket ID for reopen")
		h.answerCallbackQuery(bot, query.ID, "❌ Неверный ID обращения")
		return
	}

	// Меняем статус на "открыт"
	err = h.services.Ticket.ChangeTicketStatus(ctx, ticketID, models.TicketStatusOpen)
	if err != nil {
		log.WithError(err).Error("Failed to reopen ticket")
		h.answerCallbackQuery(bot, query.ID, "❌ Ошибка при возобновлении обращения")
		return
	}

	// Обновляем сообщение
	_, err = bot.EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:    telegoutil.ID(query.Message.GetChat().ID),
		MessageID: query.Message.GetMessageID(),
		Text:      "🔄 <b>Обращение возобновлено</b>\n\nВы можете продолжить отправлять сообщения по этому обращению.",
		ParseMode: "HTML",
	})
	if err != nil {
		log.WithError(err).Error("Failed to edit reopen message")
	}

	h.answerCallbackQuery(bot, query.ID, "🔄 Обращение возобновлено")

	log.Info("Ticket reopened by user")
}

// sendInstructionComment отправляет инструкцию в комментарии к новому тикету
func (h *TelegramHandler) sendInstructionComment(ctx context.Context, bot *telego.Bot, channelMessageID int, log logger.Logger) {
	instructionText := "💬 <b>Инструкция для техподдержки:</b>\n\n" +
		"🔹 Для ответа пользователю используйте функцию <b>'Ответить ⤺'</b> на любое сообщение в этой теме\n" +
		"🔹 Команды управления тикетом:\n" +
		"   • <code>/resolve</code> - пометить как решенный\n" +
		"   • <code>/close</code> - закрыть принудительно\n" +
		"   • <code>/priority [low|normal|high|critical]</code> - изменить приоритет\n" +
		"   • <code>/assign</code> - назначить на себя"

	_, err := bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:              telegoutil.ID(h.config.Bot.GroupID),
		Text:                instructionText,
		ParseMode:           "HTML",
		DisableNotification: true,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: channelMessageID,
		},
	})
	if err != nil {
		log.WithError(err).Error("Failed to send instruction comment")
	}
}

// getMessageType определяет тип сообщения
func (h *TelegramHandler) getMessageType(message *telego.Message) string {
	if message.Photo != nil {
		return "photo"
	}
	if message.Document != nil {
		return "document"
	}
	if message.Video != nil {
		return "video"
	}
	if message.Audio != nil {
		return "audio"
	}
	if message.Voice != nil {
		return "voice"
	}
	if message.VideoNote != nil {
		return "video_note"
	}
	if message.Sticker != nil {
		return "sticker"
	}
	if message.Animation != nil {
		return "animation"
	}
	if message.Location != nil {
		return "location"
	}
	if message.Contact != nil {
		return "contact"
	}
	return "text"
}
