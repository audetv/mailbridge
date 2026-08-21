package ai_test

import (
	"context"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
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

func TestParseResponse_Valid(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)

	jsonStr := `{"verdicts":[{"action":"new","task":{"title":"Test"}}]}`
	result, err := o.ParseResponse(jsonStr)
	if err != nil {
		t.Fatalf("ParseResponse error: %v", err)
	}
	if len(result.Verdicts) != 1 {
		t.Errorf("expected 1 verdict, got %d", len(result.Verdicts))
	}
}

func TestParseResponse_MarkdownWrapper(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)

	jsonStr := "```json\n{\"verdicts\":[]}\n```"
	result, err := o.ParseResponse(jsonStr)
	if err != nil {
		t.Fatalf("ParseResponse error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestParseResponse_Invalid(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)

	_, err := o.ParseResponse("не JSON")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyVerdicts_NewTask(t *testing.T) {
	st, _ := sqlite.NewStore(":memory:")
	_ = st.Migrate(context.Background())
	defer st.Close()

	o := ai.NewOrchestrator(nil, st)

	email := &extractor.ExtractedEmail{
		MessageID: "msg-1",
		From:      "user@example.com",
		Subject:   "Test",
		BodyText:  "Body",
	}

	response := &ai.LLMResponse{
		Verdicts: []ai.Verdict{
			{
				Action: "new",
				Task: &ai.NewTaskData{
					Title:       "Новая задача",
					Description: "Описание",
					Priority:    "high",
					Project:     "ТРК",
					Type:        "bug",
				},
			},
		},
	}

	if err := o.ApplyVerdicts(context.Background(), email, response); err != nil {
		t.Fatalf("ApplyVerdicts error: %v", err)
	}

	tasks, _ := st.GetActiveTasksByThread(context.Background(), "msg-1")
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Subject != "Новая задача" {
		t.Errorf("Subject = %s", tasks[0].Subject)
	}
}
