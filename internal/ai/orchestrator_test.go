package ai_test

import (
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
)

// BuildPrompt — экспортируемая обёртка для тестирования.
func TestBuildPrompt(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)

	email := &extractor.ExtractedEmail{
		From:     "user@example.com",
		Subject:  "Тестовое письмо",
		BodyText: "Текст письма",
	}

	prompt := o.BuildPrompt("Резюме цепочки", []*store.Task{
		{ID: 42, Subject: "Задача 1", Status: "new", Priority: "high"},
	}, email)

	if !strings.Contains(prompt, "=== РЕЗЮМЕ ЦЕПОЧКИ ===") {
		t.Error("prompt does not contain summary section")
	}
	if !strings.Contains(prompt, "=== АКТИВНЫЕ ЗАДАЧИ ===") {
		t.Error("prompt does not contain active tasks section")
	}
	if !strings.Contains(prompt, "=== НОВОЕ ПИСЬМО ===") {
		t.Error("prompt does not contain new email section")
	}
	if !strings.Contains(prompt, "Task #42") {
		t.Error("prompt does not contain task ID")
	}
	if !strings.Contains(prompt, "ОТВЕТЬ СТРОГО В JSON") {
		t.Error("prompt does not contain JSON instruction")
	}
}
