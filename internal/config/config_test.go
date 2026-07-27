package config_test

import (
	"os"
	"testing"

	"github.com/audetv/mailbridge/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Устанавливаем только обязательные поля
	envs := map[string]string{
		"MAILBRIDGE_IMAP_SERVER":    "imap.example.com",
		"MAILBRIDGE_IMAP_USER":      "user@example.com",
		"MAILBRIDGE_IMAP_PASS":      "secret",
		"MAILBRIDGE_SMTP_SERVER":    "smtp.example.com",
		"MAILBRIDGE_SMTP_FROM":      "support@example.com",
		"MAILBRIDGE_PLANE_BASE_URL": "https://plane.example.com",
		"MAILBRIDGE_PLANE_API_KEY":  "plane-key-123",
		"MAILBRIDGE_WEBHOOK_SECRET": "webhook-secret-456",
	}
	setEnvs(t, envs)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.IMAP.Port != 993 {
		t.Errorf("default IMAP port = %d, want 993", cfg.IMAP.Port)
	}
	if cfg.Storage.Driver != "sqlite" {
		t.Errorf("default storage driver = %s, want sqlite", cfg.Storage.Driver)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("default log level = %s, want info", cfg.Logging.Level)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// Не задаём обязательные поля
	os.Clearenv()

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	envs := map[string]string{
		"MAILBRIDGE_IMAP_SERVER":    "imap.custom.com",
		"MAILBRIDGE_IMAP_USER":      "user@custom.com",
		"MAILBRIDGE_IMAP_PASS":      "pass123",
		"MAILBRIDGE_IMAP_PORT":      "143",
		"MAILBRIDGE_IMAP_TLS":       "false",
		"MAILBRIDGE_SMTP_SERVER":    "smtp.custom.com",
		"MAILBRIDGE_SMTP_FROM":      "noreply@custom.com",
		"MAILBRIDGE_SMTP_PORT":      "25",
		"MAILBRIDGE_PLANE_BASE_URL": "https://plane.custom.com",
		"MAILBRIDGE_PLANE_API_KEY":  "custom-key",
		"MAILBRIDGE_WEBHOOK_SECRET": "custom-secret",
		"MAILBRIDGE_STORAGE_DRIVER": "postgres",
		"MAILBRIDGE_LOG_LEVEL":      "debug",
		"MAILBRIDGE_LOG_FORMAT":     "text",
	}
	setEnvs(t, envs)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.IMAP.Port != 143 {
		t.Errorf("IMAP port = %d, want 143", cfg.IMAP.Port)
	}
	if cfg.IMAP.TLS {
		t.Error("IMAP TLS should be false")
	}
	if cfg.SMTP.Port != 25 {
		t.Errorf("SMTP port = %d, want 25", cfg.SMTP.Port)
	}
	if cfg.Storage.Driver != "postgres" {
		t.Errorf("storage driver = %s, want postgres", cfg.Storage.Driver)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("log level = %s, want debug", cfg.Logging.Level)
	}
}

// setEnvs устанавливает переменные окружения и возвращает функцию очистки.
func setEnvs(t *testing.T, envs map[string]string) {
	t.Helper()
	for k, v := range envs {
		t.Setenv(k, v)
	}
}
