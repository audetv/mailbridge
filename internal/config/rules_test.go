package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/audetv/mailbridge/internal/config"
)

func TestLoadRules_Valid(t *testing.T) {
	path := writeTempRules(t, validRulesYAML)

	cfg, err := config.LoadRules(path)
	if err != nil {
		t.Fatalf("LoadRules error: %v", err)
	}

	if len(cfg.Rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(cfg.Rules))
	}
	if len(cfg.UrgencyPatterns) != 2 {
		t.Errorf("expected 2 urgency patterns, got %d", len(cfg.UrgencyPatterns))
	}
	if len(cfg.ValidTypes) != 2 {
		t.Errorf("expected 2 valid types, got %d", len(cfg.ValidTypes))
	}
}

func TestLoadRules_FileNotFound(t *testing.T) {
	_, err := config.LoadRules("/nonexistent/path/rules.yml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadRules_NoRules(t *testing.T) {
	path := writeTempRules(t, `
urgency_patterns: ["срочно"]
valid_types: ["bug"]
valid_priorities: ["high"]
`)

	_, err := config.LoadRules(path)
	if err == nil {
		t.Fatal("expected error for empty rules")
	}
}

func TestLoadRules_NoKeywords(t *testing.T) {
	path := writeTempRules(t, `
rules:
  - weight: 2
urgency_patterns: ["срочно"]
valid_types: ["bug"]
valid_priorities: ["high"]
`)

	_, err := config.LoadRules(path)
	if err == nil {
		t.Fatal("expected error for missing keywords")
	}
}

func TestRulesConfig_Maps(t *testing.T) {
	cfg := &config.RulesConfig{
		ValidTypes:      []string{"bug", "feature"},
		ValidPriorities: []string{"urgent", "high"},
	}

	typesMap := cfg.ValidTypesMap()
	if !typesMap["bug"] || !typesMap["feature"] {
		t.Error("valid types map missing entries")
	}

	priosMap := cfg.ValidPrioritiesMap()
	if !priosMap["urgent"] || !priosMap["high"] {
		t.Error("valid priorities map missing entries")
	}
}

// writeTempRules создаёт временный файл и возвращает путь.
func writeTempRules(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

const validRulesYAML = `
rules:
  - keywords: ["ошибка", "упал"]
    type: bug
    priority: high
    weight: 5
  - keywords: ["трк", "арендатор"]
    project: "ТРК"
    weight: 2
  - keywords: ["отель"]
    project: "Отель"
    weight: 2
urgency_patterns: ["срочно", "горит"]
valid_types: ["bug", "feature"]
valid_priorities: ["urgent", "high"]
`
