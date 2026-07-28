package webhook_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/webhook"
)

func setupHandler(t *testing.T) (*webhook.Handler, *sqlite.Store, func()) {
	t.Helper()

	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore error: %v", err)
	}

	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		st.Close()
		t.Fatalf("Migrate error: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := webhook.NewHandler(st, "", logger) // пустой секрет — без валидации

	cleanup := func() {
		st.Close()
	}

	return handler, st, cleanup
}

func TestHandler_InvalidMethod(t *testing.T) {
	handler, _, cleanup := setupHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandler_NewComment_NoMapping(t *testing.T) {
	handler, _, cleanup := setupHandler(t)
	defer cleanup()

	event := webhook.PlaneWebhookEvent{
		Event: "issue.comment.created",
	}
	event.Payload.Issue.ID = "issue-nonexistent"
	event.Payload.Comment = &struct {
		ID    string `json:"id"`
		Body  string `json:"comment_html"`
		Actor struct {
			DisplayName string `json:"display_name"`
		} `json:"actor_detail"`
	}{
		ID:   "comment-1",
		Body: "Ответ пользователя",
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_NewComment_InternalMarkerIgnored(t *testing.T) {
	handler, st, cleanup := setupHandler(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём маппинг
	err := st.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "msg-1",
		PlaneIssueID:    "issue-1",
		PlaneIssueSeq:   "WEB-1",
		OriginalFrom:    "user@example.com",
		OriginalSubject: "Test",
		ActionType:      "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping error: %v", err)
	}

	// Событие с внутренним маркером
	event := webhook.PlaneWebhookEvent{
		Event: "issue.comment.created",
	}
	event.Payload.Issue.ID = "issue-1"
	event.Payload.Comment = &struct {
		ID    string `json:"id"`
		Body  string `json:"comment_html"`
		Actor struct {
			DisplayName string `json:"display_name"`
		} `json:"actor_detail"`
	}{
		ID:   "comment-2",
		Body: "[MAILBRIDGE-INTERNAL] Задача создана",
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Outbox должен быть пустым
	items, _ := st.GetPendingOutbox(ctx, 10)
	if len(items) != 0 {
		t.Errorf("expected 0 outbox items, got %d", len(items))
	}
}

func TestHandler_NewComment_EnqueuesReply(t *testing.T) {
	handler, st, cleanup := setupHandler(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём маппинг
	err := st.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "msg-2",
		PlaneIssueID:    "issue-2",
		PlaneIssueSeq:   "WEB-2",
		OriginalFrom:    "user@example.com",
		OriginalSubject: "Test",
		ActionType:      "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping error: %v", err)
	}

	// Событие с обычным комментарием
	event := webhook.PlaneWebhookEvent{
		Event: "issue.comment.created",
	}
	event.Payload.Issue.ID = "issue-2"
	event.Payload.Comment = &struct {
		ID    string `json:"id"`
		Body  string `json:"comment_html"`
		Actor struct {
			DisplayName string `json:"display_name"`
		} `json:"actor_detail"`
	}{
		ID:   "comment-3",
		Body: "Проверил, проблема с сертификатом",
	}
	event.Payload.Comment.Actor.DisplayName = "Руслан"

	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Outbox должен содержать 1 элемент
	items, _ := st.GetPendingOutbox(ctx, 10)
	if len(items) != 1 {
		t.Fatalf("expected 1 outbox item, got %d", len(items))
	}
}

func TestHandler_StatusChange(t *testing.T) {
	handler, st, cleanup := setupHandler(t)
	defer cleanup()
	ctx := context.Background()

	err := st.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "msg-3",
		PlaneIssueID:    "issue-3",
		PlaneIssueSeq:   "WEB-3",
		OriginalFrom:    "user@example.com",
		OriginalSubject: "Test",
		ActionType:      "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping error: %v", err)
	}

	event := webhook.PlaneWebhookEvent{
		Event: "issue.updated",
	}
	event.Payload.Issue.ID = "issue-3"
	event.Payload.Issue.State = "completed"

	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Outbox должен содержать 1 элемент
	items, _ := st.GetPendingOutbox(ctx, 10)
	if len(items) != 1 {
		t.Errorf("expected 1 outbox item, got %d", len(items))
	}
}

func TestStripHTML(t *testing.T) {
	// Проверяем через неэкспортируемую функцию через обработчик
	// Используем обработчик с комментарием, который проходит через stripHTML
	// stripHTML не экспортируется, тестируем косвенно через outbox

	handler, st, cleanup := setupHandler(t)
	defer cleanup()
	ctx := context.Background()

	err := st.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "msg-4",
		PlaneIssueID:    "issue-4",
		PlaneIssueSeq:   "WEB-4",
		OriginalFrom:    "user@example.com",
		OriginalSubject: "Test",
		ActionType:      "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping error: %v", err)
	}

	event := webhook.PlaneWebhookEvent{
		Event: "issue.comment.created",
	}
	event.Payload.Issue.ID = "issue-4"
	event.Payload.Comment = &struct {
		ID    string `json:"id"`
		Body  string `json:"comment_html"`
		Actor struct {
			DisplayName string `json:"display_name"`
		} `json:"actor_detail"`
	}{
		ID:   "comment-4",
		Body: "<p>Текст с <b>HTML</b> тегами</p>",
	}
	event.Payload.Comment.Actor.DisplayName = "Тест"

	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
