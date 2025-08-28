package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config структура конфигурации приложения
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Bot        BotConfig        `mapstructure:"bot"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	Features   FeaturesConfig   `mapstructure:"features"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
}

type BotConfig struct {
	Token     string          `mapstructure:"token"`
	ChannelID int64           `mapstructure:"channel_id"`
	GroupID   int64           `mapstructure:"supergroup_id"`
	Webhook   WebhookConfig   `mapstructure:"webhook"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type WebhookConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
	Port    int    `mapstructure:"port"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
}

type DatabaseConfig struct {
	URI         string        `mapstructure:"uri"`
	Name        string        `mapstructure:"name"`
	Timeout     time.Duration `mapstructure:"timeout"`
	MaxPoolSize uint64        `mapstructure:"max_pool_size"`
	MinPoolSize uint64        `mapstructure:"min_pool_size"`
	MaxIdleTime time.Duration `mapstructure:"max_idle_time"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	URI      string `mapstructure:"uri"`
	DB       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
}

type MonitoringConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	MetricsPort     int  `mapstructure:"metrics_port"`
	HealthCheckPort int  `mapstructure:"health_check_port"`
}

type FeaturesConfig struct {
	Survey        SurveyConfig        `mapstructure:"survey"`
	Notifications NotificationsConfig `mapstructure:"notifications"`
	Analytics     AnalyticsConfig     `mapstructure:"analytics"`
}

type SurveyConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	AutoSendAfterClose bool          `mapstructure:"auto_send_after_close"`
	ReminderAfter      time.Duration `mapstructure:"reminder_after"`
}

type NotificationsConfig struct {
	Enabled     bool  `mapstructure:"enabled"`
	AdminChatID int64 `mapstructure:"admin_chat_id"`
}

type AnalyticsConfig struct {
	Enabled      bool `mapstructure:"enabled"`
	DailyReports bool `mapstructure:"daily_reports"`
}

// Load загружает конфигурацию из файла и переменных окружения
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Автозамена переменных окружения
	viper.AutomaticEnv()

	// Установка значений по умолчанию
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("app.name", "OneWay Support Bot")
	viper.SetDefault("app.version", "2.0.0")
	viper.SetDefault("app.environment", "development")

	viper.SetDefault("bot.webhook.enabled", false)
	viper.SetDefault("bot.webhook.port", 8080)
	viper.SetDefault("bot.rate_limit.requests_per_minute", 30)

	viper.SetDefault("database.name", "oneway_support")
	viper.SetDefault("database.timeout", "30s")
	viper.SetDefault("database.max_pool_size", 100)
	viper.SetDefault("database.min_pool_size", 5)
	viper.SetDefault("database.max_idle_time", "5m")

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.file", "logs/bot.log")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)

	viper.SetDefault("redis.enabled", false)
	viper.SetDefault("redis.db", 0)

	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.metrics_port", 9090)
	viper.SetDefault("monitoring.health_check_port", 8081)

	viper.SetDefault("features.survey.enabled", true)
	viper.SetDefault("features.survey.auto_send_after_close", true)
	viper.SetDefault("features.survey.reminder_after", "1h")
	viper.SetDefault("features.notifications.enabled", true)
	viper.SetDefault("features.analytics.enabled", true)
	viper.SetDefault("features.analytics.daily_reports", true)
}

func validate(config *Config) error {
	if config.Bot.Token == "" {
		return fmt.Errorf("bot token is required")
	}

	if config.Bot.ChannelID == 0 {
		return fmt.Errorf("channel ID is required")
	}

	if config.Bot.GroupID == 0 {
		return fmt.Errorf("supergroup ID is required")
	}

	if config.Database.URI == "" {
		return fmt.Errorf("database URI is required")
	}

	// Создание директории для логов если нужно
	if config.Logging.File != "" {
		if err := os.MkdirAll("logs", 0755); err != nil {
			return fmt.Errorf("failed to create logs directory: %w", err)
		}
	}

	return nil
}
