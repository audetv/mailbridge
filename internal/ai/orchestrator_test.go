package ai_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

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

func TestBuildPrompt_WithProjects(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)
	o.SetProjects([]string{"ТРК", "Отель", "Входящие"})

	email := &extractor.ExtractedEmail{
		From:     "user@example.com",
		Subject:  "Тестовое письмо",
		BodyText: "Текст письма",
	}

	prompt := o.BuildPrompt("Резюме", []*store.Task{}, email)

	if !strings.Contains(prompt, "=== ДОСТУПНЫЕ ПРОЕКТЫ ===") {
		t.Error("prompt does not contain projects section")
	}
	if !strings.Contains(prompt, "ТРК, Отель, Входящие") {
		t.Error("prompt does not contain project names")
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
					ImageNote:   "На скриншоте ошибка 500",
				},
			},
		},
	}

	if err := o.ApplyVerdicts(context.Background(), email, response, 0); err != nil {
		t.Fatalf("ApplyVerdicts error: %v", err)
	}

	tasks, _ := st.GetActiveTasksByThread(context.Background(), "msg-1")
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Subject != "Новая задача" {
		t.Errorf("Subject = %s", tasks[0].Subject)
	}
	if !strings.Contains(tasks[0].BodyText, "ошибка 500") {
		t.Error("ImageNote not included in BodyText")
	}
}

func TestUpdateSummary(t *testing.T) {
	st, _ := sqlite.NewStore(":memory:")
	_ = st.Migrate(context.Background())
	defer st.Close()

	mock := &mockSummaryClient{response: "Новое резюме цепочки"}
	o := ai.NewOrchestrator(mock, st)

	email := &extractor.ExtractedEmail{
		MessageID: "msg-1",
		From:      "user@example.com",
		Subject:   "Test",
		BodyText:  "Body",
	}

	response := &ai.LLMResponse{
		Verdicts: []ai.Verdict{{Action: "info_only"}},
	}

	if err := o.UpdateSummary(context.Background(), email, response); err != nil {
		t.Fatalf("UpdateSummary error: %v", err)
	}

	thread, _ := st.GetThread(context.Background(), "msg-1")
	if thread == nil {
		t.Fatal("thread not found")
	}
	if thread.Summary != "Новое резюме цепочки" {
		t.Errorf("Summary = %s", thread.Summary)
	}
}

type mockSummaryClient struct {
	response string
}

func (m *mockSummaryClient) Generate(_ context.Context, _ string, _ []string) (string, error) {
	return m.response, nil
}

// func TestApplyVerdicts_SavesAttachments(t *testing.T) {
// 	st, _ := sqlite.NewStore(":memory:")
// 	_ = st.Migrate(context.Background())
// 	defer st.Close()

// 	o := ai.NewOrchestrator(nil, st)

// 	email := &extractor.ExtractedEmail{
// 		MessageID: "msg-att-1",
// 		From:      "user@example.com",
// 		Subject:   "Скриншот ошибки",
// 		BodyText:  "Прикладываю скриншот",
// 		Attachments: []extractor.Attachment{
// 			{
// 				Filename:    "image001.png",
// 				ContentType: "image/png",
// 				Size:        54133,
// 				StoragePath: "/tmp/test/image001.png",
// 			},
// 		},
// 	}

// 	response := &ai.LLMResponse{
// 		Verdicts: []ai.Verdict{
// 			{
// 				Action: "new",
// 				Task: &ai.NewTaskData{
// 					Title:       "Ошибка",
// 					Description: "Ошибка на скриншоте",
// 					Priority:    "high",
// 					Project:     "Входящие",
// 					Type:        "bug",
// 				},
// 			},
// 		},
// 	}

// 	if err := o.ApplyVerdicts(context.Background(), email, response, 0); err != nil {
// 		t.Fatalf("ApplyVerdicts error: %v", err)
// 	}

// 	// Проверяем что задача создана
// 	tasks, _ := st.GetActiveTasksByThread(context.Background(), "msg-att-1")
// 	if len(tasks) != 1 {
// 		t.Fatalf("expected 1 task, got %d", len(tasks))
// 	}

// 	// Проверяем что вложение сохранено в БД
// 	atts, _ := st.GetTaskAttachments(context.Background(), tasks[0].ID)
// 	if len(atts) != 1 {
// 		t.Fatalf("expected 1 attachment in DB, got %d", len(atts))
// 	}
// 	if atts[0].Filename != "image001.png" {
// 		t.Errorf("Filename = %s", atts[0].Filename)
// 	}
// }

func TestBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 1 * time.Minute},
		{2, 5 * time.Minute},
		{3, 15 * time.Minute},
		{4, 1 * time.Hour},
		{5, 1 * time.Hour},
	}

	for _, tt := range tests {
		got := ai.BackoffForTest(tt.attempt)
		if got != tt.want {
			t.Errorf("backoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestBuildPromptWithAttachments(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)

	email := &extractor.ExtractedEmail{
		From:     "user@example.com",
		Subject:  "Документы",
		BodyText: "Прикладываю документы",
	}

	textAttachments := []string{
		"[ВЛОЖЕНИЕ: правки.docx]\nСодержимое документа",
		"[ВЛОЖЕНИЕ: таблица.xlsx]\n=== Лист: Лист1 ===\nколонка1 | колонка2",
	}

	prompt := o.BuildPromptWithAttachmentsForTest("", []*store.Task{}, email, textAttachments)

	if !strings.Contains(prompt, "=== СОДЕРЖИМОЕ ВЛОЖЕНИЙ ===") {
		t.Error("prompt does not contain attachments section")
	}
	if !strings.Contains(prompt, "правки.docx") {
		t.Error("prompt does not contain docx filename")
	}
	if !strings.Contains(prompt, "таблица.xlsx") {
		t.Error("prompt does not contain xlsx filename")
	}
}
