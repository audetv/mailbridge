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
	for _, v := range []string{"today", "tomorrow", "week", "none"} {
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

// v0.29.0 (issue #81): ?sort=created|due|updated — валидация (400 на мусор);
// порядок сортировки проверяется на уровне store (TestListTasks_ListSort*).
func TestAPI_ListTasksSort(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := t.Context()

	_ = st.CreateTask(ctx, &store.Task{
		MessageID: "sort-api-1", Subject: "Сортировка API", BodyText: "B", FromEmail: "u@e.com",
		Project: "P1", Status: string(store.StatusNew),
	})

	// Невалидное значение — 400 (тихий fallback скрывал бы typo; как ?due= / ?plan=).
	// (без пробела — httptest.NewRequest парсит query-строку)
	for _, v := range []string{"bogus", "CREATED", "asc"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/tasks?sort="+v, nil)
		handler.ListTasks(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("sort=%s: status=%d, want 400 body=%s", v, w.Code, w.Body.String())
		}
	}

	// Валидные — 200 + список задач приходит.
	for _, v := range []string{"created", "due", "updated"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/tasks?sort="+v, nil)
		handler.ListTasks(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("sort=%s: status=%d, want 200 body=%s", v, w.Code, w.Body.String())
		}
	}

	// Без param — дефолт "created" (200, v0.29.0).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	handler.ListTasks(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("no sort: status=%d, want 200", w.Code)
	}
}
