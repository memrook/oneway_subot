package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Метрики для Telegram бота
var (
	// Telegram сообщения
	TelegramMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_messages_total",
			Help: "Total number of Telegram messages processed",
		},
		[]string{"type", "status"},
	)

	// Тикеты
	TicketsCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tickets_created_total",
			Help: "Total number of tickets created",
		},
		[]string{"priority", "source"},
	)

	TicketsClosedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tickets_closed_total",
			Help: "Total number of tickets closed",
		},
		[]string{"status", "closed_by"},
	)

	TicketsActiveGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tickets_active_total",
			Help: "Current number of active tickets by status",
		},
		[]string{"status"},
	)

	// Время ответа
	ResponseTimeSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_time_seconds",
			Help:    "Response time for ticket replies",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"type"},
	)

	// Рейтинг пользователей
	UserSatisfactionRating = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "user_satisfaction_rating",
			Help:    "User satisfaction rating (1-5 stars)",
			Buckets: []float64{1, 2, 3, 4, 5},
		},
		[]string{"rating"},
	)

	// Пользователи
	UsersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "users_total",
			Help: "Total number of registered users",
		},
		[]string{"status"},
	)

	// Database операции
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "collection", "status"},
	)

	DatabaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_operation_duration_seconds",
			Help:    "Duration of database operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "collection"},
	)

	// Уведомления
	NotificationsSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notifications_sent_total",
			Help: "Total number of notifications sent",
		},
		[]string{"type", "status"},
	)

	// Боты статус
	BotStatusGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bot_status",
			Help: "Bot status (1 = running, 0 = stopped)",
		},
		[]string{"component"},
	)

	// Ошибки
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors",
		},
		[]string{"component", "type"},
	)
)

// Вспомогательные функции для обновления метрик

// RecordTicketCreated записывает создание тикета
func RecordTicketCreated(priority, source string) {
	TicketsCreatedTotal.WithLabelValues(priority, source).Inc()
}

// RecordTicketClosed записывает закрытие тикета
func RecordTicketClosed(status, closedBy string) {
	TicketsClosedTotal.WithLabelValues(status, closedBy).Inc()
}

// RecordTelegramMessage записывает обработку Telegram сообщения
func RecordTelegramMessage(msgType, status string) {
	TelegramMessagesTotal.WithLabelValues(msgType, status).Inc()
}

// RecordResponseTime записывает время ответа
func RecordResponseTime(responseType string, seconds float64) {
	ResponseTimeSeconds.WithLabelValues(responseType).Observe(seconds)
}

// RecordUserRating записывает рейтинг пользователя
func RecordUserRating(rating int) {
	UserSatisfactionRating.WithLabelValues(string(rune(rating))).Observe(float64(rating))
}

// RecordDatabaseOperation записывает операцию с базой данных
func RecordDatabaseOperation(operation, collection, status string, duration float64) {
	DatabaseOperationsTotal.WithLabelValues(operation, collection, status).Inc()
	if status == "success" {
		DatabaseOperationDuration.WithLabelValues(operation, collection).Observe(duration)
	}
}

// RecordNotification записывает отправку уведомления
func RecordNotification(notificationType, status string) {
	NotificationsSentTotal.WithLabelValues(notificationType, status).Inc()
}

// SetBotStatus устанавливает статус компонента бота
func SetBotStatus(component string, status float64) {
	BotStatusGauge.WithLabelValues(component).Set(status)
}

// RecordError записывает ошибку
func RecordError(component, errorType string) {
	ErrorsTotal.WithLabelValues(component, errorType).Inc()
}

// UpdateActiveTickets обновляет количество активных тикетов
func UpdateActiveTickets(status string, count float64) {
	TicketsActiveGauge.WithLabelValues(status).Set(count)
}

// UpdateUsersTotal обновляет общее количество пользователей
func UpdateUsersTotal(status string, count float64) {
	UsersTotal.WithLabelValues(status).Set(count)
}
