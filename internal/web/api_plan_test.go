package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

// v0.28, шаг 28b: ?plan=today|tomorrow|week — валидация (400 на мусор);
// срез сам по себе проверяет TestPlan_FilterByPlanBuckets (store-уровень).
func TestAPI_ListTasksPlan(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	_ = st.CreateTask(ctx, &store.Task{
		MessageID: "plan-api-1", Subject: "План API", BodyText: "B", FromEmail: "u@e.com",
		Project: "P1", Status: string(store.StatusNew),
	})

	// Невалидное значение — 400 (тихий fallback скрывал бы typo; как ?due= 7d).
	for _, v := range []string{"bogus", "next", "TODAY"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/tasks?plan="+v, nil)
		handler.ListTasks(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("plan=%s: status=%d, want 400 body=%s", v, w.Code, w.Body.String())
		}
	}

	// Валидные — 200 + список задач приходит.
	for _, v := range []string{"today", "tomorrow", "week"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/tasks?plan="+v, nil)
		handler.ListTasks(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("plan=%s: status=%d, want 200 body=%s", v, w.Code, w.Body.String())
		}
	}

	// Без param — как раньше (200): ?plan= опционален.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	handler.ListTasks(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("no plan: status=%d, want 200", w.Code)
	}
}
