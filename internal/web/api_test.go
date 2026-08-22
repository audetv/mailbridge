package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/web"
)

func setupAPI(t *testing.T) (*web.TaskHandler, *sqlite.Store, func()) {
	t.Helper()

	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore error: %v", err)
	}

	if err := st.Migrate(context.Background()); err != nil {
		st.Close()
		t.Fatalf("Migrate error: %v", err)
	}

	handler := web.NewTaskHandler(st)

	cleanup := func() {
		st.Close()
	}

	return handler, st, cleanup
}

func TestListInbox_Empty(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/inbox", nil)
	w := httptest.NewRecorder()

	handler.ListInbox(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestListTasks_Empty(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	w := httptest.NewRecorder()

	handler.ListTasks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result store.TaskListResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(result.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(result.Tasks))
	}
}

func TestListTasks_WithFilters(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateTask(ctx, &store.Task{MessageID: "m1", Subject: "S1", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := st.CreateTask(ctx, &store.Task{MessageID: "m2", Subject: "S2", BodyText: "B", FromEmail: "u@e.com", Project: "Отель", Status: "in_progress"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks?project=ТРК", nil)
	w := httptest.NewRecorder()

	handler.ListTasks(w, req)

	var result store.TaskListResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Errorf("expected 1 task for filter, got %d", len(result.Tasks))
	}
}

func TestListTasks_MultipleStatuses(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateTask(ctx, &store.Task{MessageID: "s1", Subject: "S1", BodyText: "B", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := st.CreateTask(ctx, &store.Task{MessageID: "s2", Subject: "S2", BodyText: "B", FromEmail: "u@e.com", Status: "in_progress"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := st.CreateTask(ctx, &store.Task{MessageID: "s3", Subject: "S3", BodyText: "B", FromEmail: "u@e.com", Status: "closed"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks?status=new&status=in_progress", nil)
	w := httptest.NewRecorder()

	handler.ListTasks(w, req)

	var result store.TaskListResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(result.Tasks) != 2 {
		t.Errorf("expected 2 tasks (new + in_progress), got %d", len(result.Tasks))
	}
}

func TestGetTask_NotFound(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	handler.GetTask(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetTask_Found(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateTask(ctx, &store.Task{MessageID: "m3", Subject: "Test", BodyText: "Body", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.GetTask(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdateTask(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateTask(ctx, &store.Task{MessageID: "m4", Subject: "Test", BodyText: "Body", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	body := `{"status":"in_progress","assignee":"Иванов"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateTask(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	task, _ := st.GetTask(ctx, 1)
	if task.Status != "in_progress" {
		t.Errorf("status = %s", task.Status)
	}
}

func TestReplyTask(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateTask(ctx, &store.Task{MessageID: "m5", Subject: "Test", BodyText: "Body", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	body := `{"body":"Ответ клиенту"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/reply", strings.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token-admin-20260101")
	w := httptest.NewRecorder()

	handler.ReplyTask(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	comments, _ := st.GetTaskComments(ctx, 1)
	if len(comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(comments))
	}
}

func TestGetAttachment_NotFound(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/attachments/nonexistent/file.pdf", nil)
	req.SetPathValue("path", "nonexistent/file.pdf")
	w := httptest.NewRecorder()

	handler.GetAttachment(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetAttachment_Found(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	// Создаём файл в data/attachments относительно рабочей директории
	attPath := filepath.Join("data", "attachments", "test-file.txt")
	if err := os.MkdirAll(filepath.Dir(attPath), 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := os.WriteFile(attPath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	defer os.RemoveAll("data")

	req := httptest.NewRequest(http.MethodGet, "/api/attachments/test-file.txt", nil)
	req.SetPathValue("path", "test-file.txt")
	w := httptest.NewRecorder()

	handler.GetAttachment(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "test content" {
		t.Errorf("body = %q", w.Body.String())
	}
}

func TestMarkRead(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateTask(ctx, &store.Task{MessageID: "m6", Subject: "Test", BodyText: "Body", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Добавляем входящий комментарий
	if err := st.AddTaskComment(ctx, &store.TaskComment{TaskID: 1, Author: "client@e.com", Body: "Проблема", Direction: "in"}); err != nil {
		t.Fatalf("AddTaskComment error: %v", err)
	}

	// Проверяем что есть непрочитанные
	filter := &store.TaskFilter{Username: "admin", Page: 1, PerPage: 10}
	result, _ := st.ListTasks(ctx, filter)
	if len(result.Tasks) == 0 {
		t.Fatal("no tasks returned")
	}
	if result.Tasks[0].UnreadComments != 2 {
		t.Errorf("expected 2 unread (1 new task + 1 comment), got %d", result.Tasks[0].UnreadComments)
	}

	// Отмечаем прочитанным
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/mark-read", nil)
	req.SetPathValue("id", "1")
	req.Header.Set("Authorization", "Bearer token-admin-20260101")
	w := httptest.NewRecorder()

	handler.MarkRead(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Проверяем что непрочитанных больше нет
	result, _ = st.ListTasks(ctx, filter)
	if result.Tasks[0].UnreadComments != 0 {
		t.Errorf("expected 0 unread after mark, got %d", result.Tasks[0].UnreadComments)
	}
}
