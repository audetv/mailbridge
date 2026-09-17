package web_test

// Шаг 7c v0.25 (Due date: подтверждение в UI) — бэкенд-контракт решения
// человека по AI-предложению (due_ai_pending, 7b) через PATCH /api/tasks/{id}:
//   - due_ai_resolve:"accept" — due_date := ai_due_date, due_source="ai",
//     pending=0; ai_due_date ОСТАЁТСЯ (база ошибок AI);
//   - due_ai_resolve:"reject" — pending=0, due_date/ai_due_date нетронуты;
//   - due_date (ручной) — приоритетнее: due_source="manual", pending=0 («изменить»);
//   - accept без ai_due_date — 422; due_ai_resolve вне allow — 400;
//   - пустой PATCH (только status) не меняет срок-поля.
// Тесты без LLM: AI-предложение эмулируется SetTaskDueAIPending (как 7b).
// due_ai_set — тестовый сид AI-предложения через PATCH (без Ollama, dev-стенд).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/web"
)

// seedDuePending — задача без срока + «предложение AI» (ai_due_date, pending=1).
func seedDuePending(t *testing.T, handler *web.TaskHandler, st *sqlite.Store, msgID string) {
	t.Helper()
	_ = handler
	ctx := t.Context()
	err := st.CreateTask(ctx, &store.Task{
		Subject:   "AI-due",
		BodyText:  "B",
		FromEmail: "u@e.com",
		Project:   "P1",
		Status:    string(store.StatusNew),
		MessageID: msgID,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	ai := "2027-01-15"
	if err := st.SetTaskDueAIPending(ctx, 1, &ai, true); err != nil {
		t.Fatalf("SetTaskDueAIPending: %v", err)
	}
}

func TestAPI_ResolveAIDue_Accept(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	seedDuePending(t, handler, st, "ai-accept-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"due_ai_resolve":"accept"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("accept: status=%d body=%s", w.Code, w.Body.String())
	}

	got, err := st.GetTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.DueDate == nil || *got.DueDate != "2027-01-15" {
		t.Errorf("due_date = %v, want 2027-01-15 (AI-срок принят)", got.DueDate)
	}
	if got.DueSource == nil || *got.DueSource != "ai" {
		t.Errorf("due_source = %v, want ai", got.DueSource)
	}
	if got.DueAIPending == nil || *got.DueAIPending {
		t.Errorf("due_ai_pending = %v, want false (pending снят)", got.DueAIPending)
	}
	// ai_due_date остаётся — база ошибок AI (приёмка 7c).
	if got.AIDueDate == nil || *got.AIDueDate != "2027-01-15" {
		t.Errorf("ai_due_date = %v, want 2027-01-15 (сохраняется после accept)", got.AIDueDate)
	}
}

func TestAPI_ResolveAIDue_Reject(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	seedDuePending(t, handler, st, "ai-reject-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"due_ai_resolve":"reject"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("reject: status=%d body=%s", w.Code, w.Body.String())
	}

	got, err := st.GetTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.DueDate != nil {
		t.Errorf("due_date = %v, want nil (отклонено — срок не ставится)", got.DueDate)
	}
	if got.DueSource != nil {
		t.Errorf("due_source = %v, want nil (не задавался)", got.DueSource)
	}
	if got.DueAIPending == nil || *got.DueAIPending {
		t.Errorf("due_ai_pending = %v, want false (pending снят)", got.DueAIPending)
	}
	// Приёмка 7c: отклонённое предложение остаётся в ai_due_date (база ошибок AI).
	if got.AIDueDate == nil || *got.AIDueDate != "2027-01-15" {
		t.Errorf("ai_due_date = %v, want 2027-01-15 (отклонённое ХРАНИТСЯ)", got.AIDueDate)
	}
}

func TestAPI_ResolveAIDue_ManualOverridesPending(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	seedDuePending(t, handler, st, "ai-manual-1")

	// «Изменить» после AI-предложения = решение человека = manual (PLAN 7c).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"due_date":"2027-02-20"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("manual: status=%d body=%s", w.Code, w.Body.String())
	}

	got, err := st.GetTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.DueDate == nil || *got.DueDate != "2027-02-20" {
		t.Errorf("due_date = %v, want 2027-02-20 (ручной срок)", got.DueDate)
	}
	if got.DueSource == nil || *got.DueSource != "manual" {
		t.Errorf("due_source = %v, want manual (ручное приоритетнее)", got.DueSource)
	}
	if got.DueAIPending == nil || *got.DueAIPending {
		t.Errorf("due_ai_pending = %v, want false (ручное решение снимает pending)", got.DueAIPending)
	}
	if got.AIDueDate == nil || *got.AIDueDate != "2027-01-15" {
		t.Errorf("ai_due_date = %v, want 2027-01-15 (история предложения хранится)", got.AIDueDate)
	}
}

func TestAPI_ResolveAIDue_Errors(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	seedDuePending(t, handler, st, "ai-err-1")

	// accept БЕЗ ai_due_date — 422 (предложить нечего принимать).
	if err := st.SetTaskDueAIPending(ctx, 1, nil, false); err != nil {
		t.Fatalf("clear proposal: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"due_ai_resolve":"accept"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("accept без ai_due_date: status=%d, want 422", w.Code)
	}

	// Мусорные значения — 400.
	for _, body := range []string{
		`{"due_ai_resolve":"approve"}`,
		`{"due_ai_resolve":123}`,
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(body))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/json")
		handler.UpdateTask(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body=%s status=%d, want 400", body, w.Code)
		}
	}
}

// due_ai_set — тестовый сид AI-предложения (7c): установка/смена/снятие через
// PATCH, валидация формата. Продуктовый UI этот ключ НЕ шлёт.
func TestAPI_AIDueSet_Seed(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	// Создание задачи напрямую (без pending) — seed создаёт предложение.
	err := st.CreateTask(ctx, &store.Task{
		Subject: "AI-set", BodyText: "B", FromEmail: "u@e.com",
		Project: "P1", Status: string(store.StatusNew), MessageID: "ai-set-1",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Установка предложения.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"due_ai_set":"2027-03-10"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("set: status=%d body=%s", w.Code, w.Body.String())
	}
	got, err := st.GetTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.AIDueDate == nil || *got.AIDueDate != "2027-03-10" {
		t.Errorf("ai_due_date = %v, want 2027-03-10", got.AIDueDate)
	}
	if got.DueAIPending == nil || !*got.DueAIPending {
		t.Errorf("due_ai_pending = %v, want true", got.DueAIPending)
	}
	if got.DueDate != nil {
		t.Errorf("due_date = %v, want nil (предложение не ставит срок)", got.DueDate)
	}

	// Смена предложения.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"due_ai_set":"2027-04-05"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("replace: status=%d body=%s", w.Code, w.Body.String())
	}
	got, _ = st.GetTask(ctx, 1)
	if got.AIDueDate == nil || *got.AIDueDate != "2027-04-05" {
		t.Errorf("ai_due_date = %v, want 2027-04-05 (перестановлена)", got.AIDueDate)
	}

	// Невалидный формат — 400.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"due_ai_set":"15.04.2027"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad format: status=%d, want 400", w.Code)
	}
}

func TestAPI_UpdateTask_WithoutDueSignalsKeepsPending(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	seedDuePending(t, handler, st, "ai-keep-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"priority":"high"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("patch: status=%d body=%s", w.Code, w.Body.String())
	}

	got, err := st.GetTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.DueAIPending == nil || !*got.DueAIPending {
		t.Errorf("due_ai_pending = %v, want true (без due-сигналов pending не снимается)", got.DueAIPending)
	}
	if got.AIDueDate == nil || *got.AIDueDate != "2027-01-15" {
		t.Errorf("ai_due_date = %v, want 2027-01-15 (не тронута)", got.AIDueDate)
	}
}
