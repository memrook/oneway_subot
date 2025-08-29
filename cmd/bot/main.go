package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/memrook/oneway_subot/internal/config"
	"github.com/memrook/oneway_subot/internal/repository/mongodb"
	"github.com/memrook/oneway_subot/internal/service"
	"github.com/memrook/oneway_subot/pkg/logger"
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

	// Инициализация сервисов
	userService := service.NewUserService(userRepo, logger.WithField("component", "user_service"))
	// TODO: Добавить остальные сервисы когда они будут реализованы
	_ = ticketRepo  // Временно избегаем ошибки компиляции
	_ = userService // Временно избегаем ошибки компиляции

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

	// TODO: Инициализация и запуск бота
	// TODO: Инициализация веб-сервера для метрик и health checks

	logger.Info("Application started successfully")

	// Ожидание сигнала завершения
	<-ctx.Done()

	logger.Info("Application shutting down")

	// Graceful shutdown с таймаутом
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// TODO: Остановка бота и других компонентов

	select {
	case <-shutdownCtx.Done():
		logger.Error("Shutdown timeout exceeded")
	default:
		logger.Info("Application stopped gracefully")
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
