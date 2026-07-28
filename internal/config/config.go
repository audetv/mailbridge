// Package config предоставляет загрузку и валидацию конфигурации Mailbridge.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config содержит все настройки приложения.
type Config struct {
	IMAP    IMAPConfig
	SMTP    SMTPConfig
	Plane   PlaneConfig
	Webhook WebhookConfig
	Storage StorageConfig
	NLP     NLPConfig
	Logging LoggingConfig
}

// IMAPConfig настройки подключения к почтовому ящику.
type IMAPConfig struct {
	Server       string
	Port         int
	User         string
	Password     string
	TLS          bool
	Inbox        string
	Archive      string
	Errors       string
	ScanInterval time.Duration
}

// SMTPConfig настройки отправки почты.
type SMTPConfig struct {
	Server   string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
}

// PlaneConfig настройки подключения к Plane API.
type PlaneConfig struct {
	BaseURL        string
	APIKey         string
	DefaultProject string
}

// WebhookConfig настройки HTTP-сервера для приёма событий Plane.
type WebhookConfig struct {
	Listen string
	Secret string
}

// StorageConfig настройки хранилища.
type StorageConfig struct {
	Driver string // "sqlite" или "postgres"
	DSN    string
}

// NLPConfig настройки NLP-классификатора.
type NLPConfig struct {
	RulesFile string
}

// LoggingConfig настройки логирования.
type LoggingConfig struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json" или "text"
}

// Load загружает конфигурацию из переменных окружения.
// Приоритет: переменная окружения > значение по умолчанию.
func Load() (*Config, error) {
	cfg := &Config{
		IMAP: IMAPConfig{
			Server:       getEnv("MAILBRIDGE_IMAP_SERVER", ""),
			Port:         getEnvAsInt("MAILBRIDGE_IMAP_PORT", 993),
			User:         getEnv("MAILBRIDGE_IMAP_USER", ""),
			Password:     getEnv("MAILBRIDGE_IMAP_PASS", ""),
			TLS:          getEnvAsBool("MAILBRIDGE_IMAP_TLS", true),
			Inbox:        getEnv("MAILBRIDGE_IMAP_INBOX", "INBOX"),
			Archive:      getEnv("MAILBRIDGE_IMAP_ARCHIVE", "Archive"),
			Errors:       getEnv("MAILBRIDGE_IMAP_ERRORS", "Errors"),
			ScanInterval: getEnvAsDuration("MAILBRIDGE_IMAP_SCAN_INTERVAL", 30*time.Second),
		},
		SMTP: SMTPConfig{
			Server:   getEnv("MAILBRIDGE_SMTP_SERVER", ""),
			Port:     getEnvAsInt("MAILBRIDGE_SMTP_PORT", 587),
			User:     getEnv("MAILBRIDGE_SMTP_USER", ""),
			Password: getEnv("MAILBRIDGE_SMTP_PASS", ""),
			From:     getEnv("MAILBRIDGE_SMTP_FROM", ""),
			TLS:      getEnvAsBool("MAILBRIDGE_SMTP_TLS", true),
		},
		Plane: PlaneConfig{
			BaseURL:        getEnv("MAILBRIDGE_PLANE_BASE_URL", ""),
			APIKey:         getEnv("MAILBRIDGE_PLANE_API_KEY", ""),
			DefaultProject: getEnv("MAILBRIDGE_PLANE_DEFAULT_PROJECT", "Входящие"),
		},
		Webhook: WebhookConfig{
			Listen: getEnv("MAILBRIDGE_WEBHOOK_LISTEN", ":8080"),
			Secret: getEnv("MAILBRIDGE_WEBHOOK_SECRET", ""),
		},
		Storage: StorageConfig{
			Driver: getEnv("MAILBRIDGE_STORAGE_DRIVER", "sqlite"),
			DSN:    getEnv("MAILBRIDGE_STORAGE_DSN", "data/mailbridge.db"),
		},
		NLP: NLPConfig{
			RulesFile: getEnv("MAILBRIDGE_NLP_RULES_FILE", "configs/rules.yml"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("MAILBRIDGE_LOG_LEVEL", "info"),
			Format: getEnv("MAILBRIDGE_LOG_FORMAT", "json"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate проверяет обязательные поля конфигурации.
func (c *Config) Validate() error {
	if c.IMAP.Server == "" {
		return fmt.Errorf("MAILBRIDGE_IMAP_SERVER is required")
	}
	if c.IMAP.User == "" {
		return fmt.Errorf("MAILBRIDGE_IMAP_USER is required")
	}
	if c.IMAP.Password == "" {
		return fmt.Errorf("MAILBRIDGE_IMAP_PASS is required")
	}
	if c.SMTP.Server == "" {
		return fmt.Errorf("MAILBRIDGE_SMTP_SERVER is required")
	}
	if c.SMTP.From == "" {
		return fmt.Errorf("MAILBRIDGE_SMTP_FROM is required")
	}
	if c.Plane.BaseURL == "" {
		return fmt.Errorf("MAILBRIDGE_PLANE_BASE_URL is required")
	}
	if c.Plane.APIKey == "" {
		return fmt.Errorf("MAILBRIDGE_PLANE_API_KEY is required")
	}
	if c.Webhook.Secret == "" {
		return fmt.Errorf("MAILBRIDGE_WEBHOOK_SECRET is required")
	}
	return nil
}

// getEnv возвращает значение переменной окружения или значение по умолчанию.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// getEnvAsInt возвращает значение переменной окружения как int.
func getEnvAsInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// getEnvAsBool возвращает значение переменной окружения как bool.
func getEnvAsBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

// getEnvAsDuration возвращает значение переменной окружения как time.Duration.
func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if dur, err := time.ParseDuration(val); err == nil {
			return dur
		}
	}
	return defaultVal
}

// ShutdownTimeout возвращает таймаут для graceful shutdown.
func (c *Config) ShutdownTimeout() time.Duration {
	return getEnvAsDuration("MAILBRIDGE_SHUTDOWN_TIMEOUT", 30*time.Second)
}
