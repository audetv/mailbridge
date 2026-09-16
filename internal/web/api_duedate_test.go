package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func TestAPI_UpdateTaskDueDate(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	manual := "manual"
	err := st.CreateTask(ctx, &store.Task{
		MessageID: "due-api-1", Subject: "Дедлайн-1", BodyText: "B", FromEmail: "u@e.com",
		Project: "P1", Status: string(store.StatusNew), DueDate: &manual,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// PATCH: установка срока.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"title":"Дедлайн-1","project":"P1","due_date":"2026-12-01"}`))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	handler.UpdateTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT due: status=%d body=%s", w.Code, w.Body.String())
	}

	got, err := st.GetTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.DueDate == nil || *got.DueDate != "2026-12-01" {
		t.Errorf("due_date after PUT = %v, want 2026-12-01", got.DueDate)
	}
	if got.DueSource == nil || *got.DueSource != "manual" {
		t.Errorf("due_source = %v, want manual (ручное приоритетнее AI)", got.DueSource)
	}
}

func TestAPI_UpdateTaskDueDateRejectsBadFormat(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	err := st.CreateTask(ctx, &store.Task{MessageID: "due-api-2", Subject: "Bad", BodyText: "B", FromEmail: "u@e.com", Project: "P1", Status: string(store.StatusNew)})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	cases := []string{
		`{"title":"Bad","project":"P1","due_date":"01.12.2026"}`,           // DD.MM.YYYY — не канон
		`{"title":"Bad","project":"P1","due_date":"2026-13-01"}`,           // месяц 13 — не существует
		`{"title":"Bad","project":"P1","due_date":"2026-02-31"}`,           // дня нет
		`{"title":"Bad","project":"P1","due_date":"2026-12-01T00:00:00Z"}`, // это время, а не DATE
	}
	for _, body := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(body))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/json")
		handler.UpdateTask(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body=%s status=%d, want 400", body, w.Code)
		}
	}

	// Срок не должен измениться после отклонённых попыток.
	got, _ := st.GetTask(ctx, 1)
	if got.DueDate != nil {
		t.Errorf("due_date should stay nil, got %v", got.DueDate)
	}
}

func TestAPI_CreateTaskWithDueDate(t *testing.T) {
	e := newCreateTaskEnv(t)
	e.mkProject(t, "Деск Due")

	// Валидный срок.
	w := e.post(t, `{"title":"С сроком","project":"Деск Due","due_date":"2026-11-11"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("created status=%d body=%s", w.Code, w.Body.String())
	}
	tk := e.decodeTask(t, w)
	if tk.DueDate == nil || *tk.DueDate != "2026-11-11" {
		t.Errorf("due_date = %v, want 2026-11-11", tk.DueDate)
	}
	if tk.DueSource == nil || *tk.DueSource != "manual" {
		t.Errorf("due_source = %v, want manual", tk.DueSource)
	}

	// Невалидный — 400, задача не создаётся.
	w2 := e.post(t, `{"title":"С плохим","project":"Деск Due","due_date":"13.13.2026"}`)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("bad due status=%d, want 400", w2.Code)
	}
}
