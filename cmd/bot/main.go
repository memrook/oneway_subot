package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/memrook/oneway_subot/internal/config"
	"github.com/memrook/oneway_subot/internal/handlers"
	"github.com/memrook/oneway_subot/internal/repository/mongodb"
	"github.com/memrook/oneway_subot/internal/service"
	"github.com/memrook/oneway_subot/pkg/logger"
	"github.com/memrook/oneway_subot/pkg/metrics"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализация логгера
	loggerConfig := logger.Config{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		File:       cfg.Logging.File,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
	}

	if err := logger.InitDefault(loggerConfig); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Infof("Starting %s v%s", cfg.App.Name, cfg.App.Version)

	// Подключение к базе данных
	db, err := mongodb.New(cfg.Database, logger.WithField("component", "database"))
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.Close(ctx); err != nil {
			logger.Errorf("Failed to close database connection: %v", err)
		}
	}()

	// Инициализация репозиториев
	userRepo := mongodb.NewUserRepository(db, logger.WithField("component", "user_repo"))
	ticketRepo := mongodb.NewTicketRepository(db, logger.WithField("component", "ticket_repo"))

	// Инициализация Telegram бота
	bot, err := telego.NewBot(cfg.Bot.Token, telego.WithDefaultDebugLogger())
	if err != nil {
		logger.Fatalf("Failed to create bot: %v", err)
	}

	// Инициализация сервисов
	userService := service.NewUserService(userRepo, logger.WithField("component", "user_service"))
	ticketService := service.NewTicketService(ticketRepo, logger.WithField("component", "ticket_service"))
	notificationService := service.NewNotificationService(bot, cfg, logger.WithField("component", "notification_service"))
	analyticsService := service.NewAnalyticsService(ticketRepo, userRepo, logger.WithField("component", "analytics_service"))

	// Создаем контейнер сервисов
	services := &service.Services{
		User:         userService,
		Ticket:       ticketService,
		Notification: notificationService,
		Analytics:    analyticsService,
	}

	// Создание обработчика бота
	updates, err := bot.UpdatesViaLongPolling(nil)
	if err != nil {
		logger.Fatalf("Failed to start polling: %v", err)
	}

	// Создание handler для обработки обновлений
	bh, err := telegohandler.NewBotHandler(bot, updates)
	if err != nil {
		logger.Fatalf("Failed to create bot handler: %v", err)
	}

	// Регистрация обработчиков Telegram
	telegramHandler := handlers.NewTelegramHandler(bot, services, cfg, logger.WithField("component", "telegram_handler"))
	telegramHandler.RegisterHandlers(bh)

	// Создание контекста для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Infof("Received signal %s, shutting down gracefully", sig)
		cancel()
	}()

	// Запуск health check сервера
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", healthCheckHandler)
		mux.HandleFunc("/ready", readinessCheckHandler)

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Monitoring.HealthCheckPort),
			Handler: mux,
		}

		logger.Infof("Starting health check server on port %d", cfg.Monitoring.HealthCheckPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Health check server error: %v", err)
		}
	}()

	// Запуск metrics сервера
	if cfg.Monitoring.Enabled {
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.Handler())

			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", cfg.Monitoring.MetricsPort),
				Handler: mux,
			}

			logger.Infof("Starting metrics server on port %d", cfg.Monitoring.MetricsPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Errorf("Metrics server error: %v", err)
			}
		}()
	}

	// Запуск обработки обновлений
	go bh.Start()
	defer bh.Stop()

	// Устанавливаем статус компонентов в метриках
	metrics.SetBotStatus("telegram_bot", 1)
	metrics.SetBotStatus("database", 1)
	if cfg.Monitoring.Enabled {
		metrics.SetBotStatus("metrics", 1)
	}

	logger.Info("Bot started successfully and is listening for updates")

	// Ожидание сигнала завершения
	<-ctx.Done()

	logger.Info("Bot shutting down")

	// Graceful shutdown с таймаутом
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Остановка бота
	bh.Stop()
	bot.StopLongPolling()

	// Обновляем статус компонентов
	metrics.SetBotStatus("telegram_bot", 0)

	select {
	case <-shutdownCtx.Done():
		logger.Error("Shutdown timeout exceeded")
	default:
		logger.Info("Bot stopped gracefully")
	}
}

// printBanner выводит баннер приложения
func printBanner(cfg *config.Config) {
	banner := fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════╗
║                    OneWay Support Bot                        ║
║                       Version %s                        ║
║                                                              ║
║  Modern Telegram Support Bot with Advanced Features         ║
║                                                              ║
║  Environment: %-10s                                    ║
║  Database:    MongoDB                                        ║
║  Monitoring:  %s                                         ║
╚══════════════════════════════════════════════════════════════╝
`, cfg.App.Version, cfg.App.Environment, getMonitoringStatus(cfg.Monitoring.Enabled))

	fmt.Print(banner)
}

func getMonitoringStatus(enabled bool) string {
	if enabled {
		return "Enabled "
	}
	return "Disabled"
}

// validateEnvironment проверяет переменные окружения
func validateEnvironment() error {
	required := []string{
		"BOT_TOKEN",
		"DATABASE_URI",
		"CHANNEL_ID",
		"SUPERGROUP_ID",
	}

	for _, env := range required {
		if os.Getenv(env) == "" {
			return fmt.Errorf("required environment variable %s is not set", env)
		}
	}

	return nil
}

// healthCheckHandler обрабатывает запросы health check
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// readinessCheckHandler обрабатывает запросы readiness check
func readinessCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}
