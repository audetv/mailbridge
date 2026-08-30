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

// TestLoad_AIConfig — новые поля AI-секции (решение §7 #16: system-prompt + temperature из конфига).
func TestLoad_AIConfig(t *testing.T) {
	envs := map[string]string{
		"MAILBRIDGE_IMAP_SERVER":    "imap.example.com",
		"MAILBRIDGE_IMAP_USER":      "user@example.com",
		"MAILBRIDGE_IMAP_PASS":      "secret",
		"MAILBRIDGE_SMTP_SERVER":    "smtp.example.com",
		"MAILBRIDGE_SMTP_FROM":      "support@example.com",
		"MAILBRIDGE_PLANE_BASE_URL": "https://plane.example.com",
		"MAILBRIDGE_PLANE_API_KEY":  "plane-key-123",
		"MAILBRIDGE_WEBHOOK_SECRET": "webhook-secret-456",
		"MAILBRIDGE_AI_ENABLED":     "true",
		"MAILBRIDGE_AI_MODEL":       "qwen3.8-74k:latest",
		"MAILBRIDGE_AI_SYSTEM_FILE": "configs/email-assistant-v2.system.txt",
		"MAILBRIDGE_AI_TEMPERATURE": "0.2",
	}
	setEnvs(t, envs)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.AI.Enabled {
		t.Error("AI.Enabled should be true")
	}
	if cfg.AI.Model != "qwen3.8-74k:latest" {
		t.Errorf("AI.Model = %s, want qwen3.8-74k:latest", cfg.AI.Model)
	}
	if cfg.AI.SystemPromptFile != "configs/email-assistant-v2.system.txt" {
		t.Errorf("AI.SystemPromptFile = %s", cfg.AI.SystemPromptFile)
	}
	if cfg.AI.Temperature != 0.2 {
		t.Errorf("AI.Temperature = %v, want 0.2", cfg.AI.Temperature)
	}
}

// TestLoad_AIConfig_TemperatureDefault — по умолчанию 0.1 (как в Modelfile).
func TestLoad_AIConfig_TemperatureDefault(t *testing.T) {
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

	if cfg.AI.Temperature != 0.1 {
		t.Errorf("AI.Temperature default = %v, want 0.1", cfg.AI.Temperature)
	}
	if cfg.AI.SystemPromptFile != "" {
		t.Errorf("AI.SystemPromptFile default = %s, want empty", cfg.AI.SystemPromptFile)
	}
}

// setEnvs устанавливает переменные окружения и возвращает функцию очистки.
func setEnvs(t *testing.T, envs map[string]string) {
	t.Helper()
	for k, v := range envs {
		t.Setenv(k, v)
	}
}
