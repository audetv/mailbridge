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

	handler := web.NewTaskHandler(st, web.NewEventBroker(), "admin")

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

func TestCreateTaskFromInbox(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateInboxItem(ctx, &store.InboxItem{Source: "email", SourceID: "msg-1", Subject: "Test", BodyText: "Body", FromContact: "u@e.com", Status: "unread"}); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/inbox/1/task", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.CreateTaskFromInbox(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	tasks, _ := st.GetTasksByInboxItem(ctx, 1)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task link, got %d", len(tasks))
	}
}

func TestUpdateInboxStatus_Read(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateInboxItem(ctx, &store.InboxItem{Source: "email", SourceID: "msg-1", Status: "unread"}); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/inbox/1/read", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.UpdateInboxStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	item, _ := st.GetInboxItemByID(ctx, 1)
	if item.Status != "read" {
		t.Errorf("Status = %s, want read", item.Status)
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

// TestReplyTaskKind — валидация поля kind (ФАЗА 4, шаг 18.3).
func TestReplyTaskKind(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode int
		wantKind string
	}{
		{name: "empty kind -> user_comment", body: `{"body":"a"}`, wantCode: http.StatusOK, wantKind: "user_comment"},
		{name: "explicit user_comment", body: `{"body":"b","kind":"user_comment"}`, wantCode: http.StatusOK, wantKind: "user_comment"},
		{name: "report", body: `{"body":"c","kind":"report"}`, wantCode: http.StatusOK, wantKind: "report"},
		{name: "reply", body: `{"body":"d","kind":"reply"}`, wantCode: http.StatusOK, wantKind: "reply"},
		{name: "invalid kind", body: `{"body":"e","kind":"junk"}`, wantCode: http.StatusBadRequest, wantKind: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, st, cleanup := setupAPI(t)
			defer cleanup()
			ctx := context.Background()
			if err := st.CreateTask(ctx, &store.Task{MessageID: "kind-" + tc.name, Subject: "t", BodyText: "b", FromEmail: "u@e.com", Status: "new"}); err != nil {
				t.Fatalf("CreateTask: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/reply", strings.NewReader(tc.body))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer token-hermes-20260830")
			w := httptest.NewRecorder()
			handler.ReplyTask(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d: %s", tc.wantCode, w.Code, w.Body.String())
			}
			if tc.wantCode != http.StatusOK {
				return
			}
			comments, err := st.GetTaskComments(ctx, 1)
			if err != nil {
				t.Fatalf("GetTaskComments: %v", err)
			}
			if len(comments) != 1 {
				t.Fatalf("expected 1 comment, got %d", len(comments))
			}
			if comments[0].Kind != tc.wantKind {
				t.Errorf("kind = %s, want %s", comments[0].Kind, tc.wantKind)
			}
			if comments[0].Direction != "out" {
				t.Errorf("direction = %s, want out", comments[0].Direction)
			}
			if comments[0].Author != "hermes" {
				t.Errorf("author = %s, want hermes", comments[0].Author)
			}
		})
	}
}

// TestApproveComment — PATCH /api/comments/{id}/approve (ФАЗА 4, шаг 19).
func TestApproveComment(t *testing.T) {
	cases := []struct {
		name     string
		token    string // Authorization: Bearer <token>
		kind     string // kind создаваемого комментария
		wantCode int
		wantAppr *int // nil = NULL
	}{
		{name: "admin approves reply", token: "token-admin-20260101", kind: "reply", wantCode: http.StatusOK, wantAppr: ip(1)},
		{name: "admin idempotent second call", token: "token-admin-20260101", kind: "reply", wantCode: http.StatusOK, wantAppr: ip(1)},
		{name: "agent hermes forbidden", token: "token-hermes-20260101", kind: "reply", wantCode: http.StatusForbidden},
		{name: "no auth 401", token: "", kind: "reply", wantCode: http.StatusUnauthorized},
		{name: "non-reply kind forbidden", token: "token-admin-20260101", kind: "report", wantCode: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, st, cleanup := setupAPI(t)
			defer cleanup()
			ctx := context.Background()
			if err := st.CreateTask(ctx, &store.Task{MessageID: "appr-" + tc.name, Subject: "t", BodyText: "b", FromEmail: "u@e.com", Status: "new"}); err != nil {
				t.Fatalf("CreateTask: %v", err)
			}
			cm := &store.TaskComment{TaskID: 1, Author: "hermes", Body: "text", Direction: "out", Kind: tc.kind}
			if err := st.AddTaskComment(ctx, cm); err != nil {
				t.Fatalf("AddTaskComment: %v", err)
			}

			authHdr := ""
			if tc.token != "" {
				authHdr = "Bearer " + tc.token
			}
			req := httptest.NewRequest(http.MethodPatch, "/api/comments/1/approve", nil)
			req.SetPathValue("id", "1")
			if authHdr != "" {
				req.Header.Set("Authorization", authHdr)
			}
			w := httptest.NewRecorder()
			handler.ApproveComment(w, req)

			if w.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d: %s", tc.wantCode, w.Code, w.Body.String())
			}
			if tc.wantAppr == nil {
				return
			}
			// идемпотентность: второй вызов — тоже 200 (шаг 19)
			req2 := httptest.NewRequest(http.MethodPatch, "/api/comments/1/approve", nil)
			req2.SetPathValue("id", "1")
			req2.Header.Set("Authorization", "Bearer "+tc.token)
			w2 := httptest.NewRecorder()
			handler.ApproveComment(w2, req2)
			if w2.Code != http.StatusOK {
				t.Fatalf("idempotent second call expected 200, got %d: %s", w2.Code, w2.Body.String())
			}
			got, err := st.GetTaskComment(ctx, cm.ID)
			if err != nil {
				t.Fatalf("GetTaskComment: %v", err)
			}
			if got.Approved == nil || *got.Approved != *tc.wantAppr {
				t.Errorf("approved = %v, want %v", got.Approved, tc.wantAppr)
			}
		})
	}
}

func TestApproveComment_NotFound(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPatch, "/api/comments/999/approve", nil)
	req.SetPathValue("id", "999")
	req.Header.Set("Authorization", "Bearer token-admin-20260101")
	w := httptest.NewRecorder()
	handler.ApproveComment(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApproveComment_WrongMethod(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/comments/1/approve", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	handler.ApproveComment(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func ip(v int) *int { return &v }

// TestGetTaskCommentStore — чтение одного комментария + approved (шаг 17.1/19).
func TestGetTaskCommentStore(t *testing.T) {
	_, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()
	if err := st.CreateTask(ctx, &store.Task{MessageID: "gc-store", Subject: "t", BodyText: "b", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	cm := &store.TaskComment{TaskID: 1, Author: "admin", Body: "hello", Direction: "out", Kind: "reply"}
	if err := st.AddTaskComment(ctx, cm); err != nil {
		t.Fatalf("AddTaskComment: %v", err)
	}
	got, err := st.GetTaskComment(ctx, cm.ID)
	if err != nil {
		t.Fatalf("GetTaskComment: %v", err)
	}
	if got.Body != "hello" || got.Kind != "reply" || got.Author != "admin" {
		t.Errorf("comment mismatch: %+v", got)
	}
	if _, err := st.GetTaskComment(ctx, 424242); err != store.ErrCommentNotFound {
		t.Errorf("want ErrCommentNotFound, got %v", err)
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
	if err := st.AddTaskComment(ctx, &store.TaskComment{TaskID: 1, Author: "client@e.com", Body: "Проблема", Direction: "in", Kind: "user_comment"}); err != nil {
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

func TestGetTaskInboxItems(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateTask(ctx, &store.Task{MessageID: "m-t", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := st.CreateInboxItem(ctx, &store.InboxItem{Source: "email", SourceID: "m-i", Subject: "Inbox", Status: "unread"}); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}
	if err := st.LinkTaskToInboxItem(ctx, 1, 1, "created_from"); err != nil {
		t.Fatalf("LinkTaskToInboxItem error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1/inbox", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.GetTaskInboxItems(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
