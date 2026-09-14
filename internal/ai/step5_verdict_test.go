package ai_test

// Шаг 5 v0.23 (AI-вердикты): автор-отправитель, quote → verdict_json,
// промпт-контракт с полем quote. Фиксирует решение владельца 2026-09-14.
//
// Правила кода: тесты без LLM-модели (реальные данные dev-БД + sqlite :memory:).

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

// senderLabel вынесен как test-only хелпер — проверяем контракты,
// а не реализацию (реальный вызов в verdicts.go через ApplyVerdicts).

func TestPromptContract_HasQuoteField(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)
	email := &extractor.ExtractedEmail{
		From:     "vika@example.com",
		Subject:  "Тест",
		BodyText: "Текст письма",
	}
	prompt := o.BuildPrompt("резюме", []*store.Task{
		{ID: 42, Subject: "Задача", Status: "new", Priority: "high"},
	}, email)

	if !strings.Contains(prompt, `"quote"`) {
		t.Error("prompt JSON contract missing 'quote' field")
	}
	if !strings.Contains(prompt, `"add_comment"`) {
		t.Error("prompt JSON contract missing 'add_comment' field")
	}
}

func TestParseResponse_QuoteInUpdates(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)
	res, err := o.ParseResponse(`{"verdicts":[{"action":"update","task_id":7,"updates":{"add_comment":"Саммари","quote":"дословный фрагмент"}}]}`)
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if len(res.Verdicts) != 1 {
		t.Fatalf("expected 1 verdict, got %d", len(res.Verdicts))
	}
	if got := res.Verdicts[0].Updates.Quote; got != "дословный фрагмент" {
		t.Errorf("Updates.Quote = %q, want %q", got, "дословный фрагмент")
	}
}

// setupUpdateStep5: цепочка + inbox-письмо + активная задача — каркас
// для проверки update/completed путей.
func setupUpdateStep5(t *testing.T, st *sqlite.Store, email *extractor.ExtractedEmail) (inboxItemID int64, taskID int) {
	t.Helper()
	ctx := context.Background()

	inboxItem := &store.InboxItem{
		Source:   "email",
		SourceID: email.MessageID,
		Subject:  email.Subject,
		BodyText: email.BodyText,
		Status:   "unread",
	}
	if err := st.CreateInboxItem(ctx, inboxItem); err != nil {
		t.Fatalf("CreateInboxItem: %v", err)
	}

	if err := st.CreateThread(ctx, &store.Thread{
		ThreadID:     email.MessageID,
		Source:       "email",
		Subject:      email.Subject,
		Participants: `["vika@example.com"]`,
		Summary:      "Резюме цепочки",
	}); err != nil {
		t.Fatalf("CreateThread: %v", err)
	}

	task := &store.Task{
		MessageID: email.MessageID,
		Subject:   "Согласовать сроки",
		BodyText:  "Согласовать сроки с клиентом",
		FromEmail: email.From,
		FromName:  extractNameFromEmail(t, email.From),
		Project:   "Отель",
		Type:      "task",
		Priority:  "medium",
		Status:    "in_progress",
		ThreadID:  email.MessageID,
	}
	if err := st.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	return inboxItem.ID, int(task.ID)
}

func extractNameFromEmail(t *testing.T, from string) string {
	t.Helper()
	idx := strings.Index(from, "<")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(from[:idx])
}

func TestApplyVerdicts_Step5_Update_AuthorAndQuote(t *testing.T) {
	st, _ := sqlite.NewStore(":memory:")
	_ = st.Migrate(context.Background())
	defer st.Close()

	o := ai.NewOrchestrator(nil, st)
	ctx := context.Background()

	email := &extractor.ExtractedEmail{
		MessageID: "step5-1",
		From:      "Вика Целищева <vcel@example.com>",
		Subject:   "Сроки",
		BodyText:  "Алексей, сообщите сроки, когда ждать",
	}
	inboxItemID, taskID := setupUpdateStep5(t, st, email)

	response := &ai.LLMResponse{
		Verdicts: []ai.Verdict{
			{
				Action: "update",
				TaskID: &taskID,
				Updates: &ai.TaskUpdates{
					AddComment: "Ждёт сроков от клиента — приоритет high",
					Quote:      "назвать сроки до 15.00",
				},
			},
		},
	}
	if err := o.ApplyVerdicts(ctx, email, response, inboxItemID); err != nil {
		t.Fatalf("ApplyVerdicts: %v", err)
	}

	comments, err := st.GetTaskComments(ctx, int64(taskID))
	if err != nil {
		t.Fatalf("GetTaskComments: %v", err)
	}
	var user *store.TaskComment
	for _, c := range comments {
		if c.Kind == "user_comment" {
			user = c
		}
	}
	if user == nil {
		t.Fatal("no user_comment created")
	}

	// Автор: реальный отправитель (имя + адрес), а не «user».
	if user.Author != "Вика Целищева <vcel@example.com>" {
		t.Errorf("Author = %q, want %q", user.Author, "Вика Целищева <vcel@example.com>")
	}
	// Body: только саммари — цитата НЕ вставлена в тело (копирование MD/TXT чистое).
	if strings.Contains(user.Body, "назвать сроки") {
		t.Errorf("Body contains quote (%q) — цитата не должна дублироваться в body", user.Body)
	}
	// Quote: в verdict_json комментария.
	var parsed struct {
		Quote string `json:"quote"`
	}
	if err := json.Unmarshal([]byte(user.VerdictJSON), &parsed); err != nil {
		t.Fatalf("verdict_json не JSON: %v (%q)", err, user.VerdictJSON)
	}
	if parsed.Quote != "назвать сроки до 15.00" {
		t.Errorf("verdict_json.quote = %q, want %q", parsed.Quote, "назвать сроки до 15.00")
	}
	// inbox_item_id: UI найдёт «оригинал письма».
	if user.InboxItemID == nil || *user.InboxItemID != inboxItemID {
		t.Errorf("InboxItemID = %v, want %d", user.InboxItemID, inboxItemID)
	}
}

func TestApplyVerdicts_Step5_Complete_AuthorAndQuote(t *testing.T) {
	st, _ := sqlite.NewStore(":memory:")
	_ = st.Migrate(context.Background())
	defer st.Close()

	o := ai.NewOrchestrator(nil, st)
	ctx := context.Background()

	email := &extractor.ExtractedEmail{
		MessageID: "step5-2",
		From:      "Аня Бухтина <abuh@example.com>",
		Subject:   "Готово",
		BodyText:  "Добрый день, всё согласовано",
	}
	inboxItemID, taskID := setupUpdateStep5(t, st, email)

	response := &ai.LLMResponse{
		Verdicts: []ai.Verdict{
			{
				Action:  "completed",
				TaskID:  &taskID,
				Comment: "Клиент подтвердил — можно закрывать",
				Quote:   "всё согласовано",
			},
		},
	}
	if err := o.ApplyVerdicts(ctx, email, response, inboxItemID); err != nil {
		t.Fatalf("ApplyVerdicts: %v", err)
	}

	comments, err := st.GetTaskComments(ctx, int64(taskID))
	if err != nil {
		t.Fatalf("GetTaskComments: %v", err)
	}
	var user *store.TaskComment
	for _, c := range comments {
		if c.Kind == "user_comment" {
			user = c
		}
	}
	if user == nil {
		t.Fatal("no user_comment created")
	}
	if user.Author != "Аня Бухтина <abuh@example.com>" {
		t.Errorf("Author = %q, want %q", user.Author, "Аня Бухтина <abuh@example.com>")
	}
	var parsed struct {
		Quote string `json:"quote"`
	}
	if err := json.Unmarshal([]byte(user.VerdictJSON), &parsed); err != nil {
		t.Fatalf("verdict_json не JSON: %v (%q)", err, user.VerdictJSON)
	}
	if parsed.Quote != "всё согласовано" {
		t.Errorf("verdict_json.quote = %q", parsed.Quote)
	}

	// Задача завершена.
	task, err := st.GetTask(ctx, int64(taskID))
	if err != nil || task == nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.Status != "completed" {
		t.Errorf("Status = %q, want completed", task.Status)
	}
}
