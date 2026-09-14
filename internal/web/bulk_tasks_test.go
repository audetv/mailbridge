package web_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

// bulkPayload — тело PATCH /api/tasks (контракт шага 4): {ids, changes}.
type bulkPayload struct {
	IDs     []int64 `json:"ids"`
	Changes struct {
		Status  string `json:"status,omitempty"`
		Project string `json:"project,omitempty"`
	} `json:"changes"`
}

func patchBulk(t *testing.T, handler interface {
	BulkUpdateTasks(w http.ResponseWriter, r *http.Request)
}, payload bulkPayload) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal bulk payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.BulkUpdateTasks(w, req)
	return w
}

func createBulkTask(t *testing.T, st interface {
	CreateTask(ctx context.Context, task *store.Task) error
}, id string) *store.Task {
	t.Helper()
	task := &store.Task{MessageID: fmt.Sprintf("m-bulk-%s", id), Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	if err := st.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	return task
}

// TestBulkUpdateTasks_OK — 200 {count, statuses, errors:[]}, статусы в store.
func TestBulkUpdateTasks_OK(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	ids := []int64{}
	for i := 0; i < 3; i++ {
		ids = append(ids, createBulkTask(t, st, fmt.Sprintf("ok-%d", i)).ID)
	}

	var p bulkPayload
	p.IDs = ids
	p.Changes.Status = "in_progress"
	w := patchBulk(t, handler, p)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Count    int               `json:"count"`
		Statuses map[string]int64  `json:"statuses"`
		Errors   []json.RawMessage `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Count != 3 {
		t.Errorf("count = %d, want 3", resp.Count)
	}
	if len(resp.Errors) != 0 {
		t.Errorf("errors = %v, want empty", resp.Errors)
	}
	for _, id := range ids {
		task, err := st.GetTask(ctx, id)
		if err != nil || task == nil {
			t.Fatalf("GetTask %d: %v", id, err)
		}
		if task.Status != "in_progress" {
			t.Errorf("task %d status = %q, want in_progress", id, task.Status)
		}
	}
}

// TestBulkUpdateTasks_Project — changes.project: пакетная переноска (крит. приёмки).
func TestBulkUpdateTasks_Project(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	var ids []int64
	for i := 0; i < 2; i++ {
		ids = append(ids, createBulkTask(t, st, fmt.Sprintf("pr-%d", i)).ID)
	}

	var p bulkPayload
	p.IDs = ids
	p.Changes.Project = "Офис"
	w := patchBulk(t, handler, p)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	for _, id := range ids {
		task, err := st.GetTask(ctx, id)
		if err != nil || task == nil {
			t.Fatalf("GetTask %d: %v", id, err)
		}
		if task.Project != "Офис" {
			t.Errorf("task %d project = %q, want Офис", id, task.Project)
		}
	}
}

// TestBulkUpdateTasks_Dedup — повторные ID: одна строка истории на задачу (крит. приёмки).
func TestBulkUpdateTasks_Dedup(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	task := createBulkTask(t, st, "dup")

	var p bulkPayload
	p.IDs = []int64{task.ID, task.ID, task.ID}
	p.Changes.Status = "completed"
	w := patchBulk(t, handler, p)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	history, err := st.GetTaskStatusHistory(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskStatusHistory: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("history len = %d, want 1 (дупы схлопнуты)", len(history))
	}
}

// TestBulkUpdateTasks_InvalidStatus — недопустимый status → 400, задача не пострадала.
func TestBulkUpdateTasks_InvalidStatus(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()

	task := createBulkTask(t, st, "bad")

	var p bulkPayload
	p.IDs = []int64{task.ID}
	p.Changes.Status = "bogus"
	w := patchBulk(t, handler, p)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	got, _ := st.GetTask(context.Background(), task.ID)
	if got.Status != "new" {
		t.Errorf("task status = %q, want new (unchanged)", got.Status)
	}
}

// TestBulkUpdateTasks_EmptyIDs — пустой список → 400.
func TestBulkUpdateTasks_EmptyIDs(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	var p bulkPayload
	p.IDs = nil
	p.Changes.Status = "completed"
	w := patchBulk(t, handler, p)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBulkUpdateTasks_EmptyChanges — ни status, ни project → 400.
func TestBulkUpdateTasks_EmptyChanges(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()

	task := createBulkTask(t, st, "emptych")

	var p bulkPayload
	p.IDs = []int64{task.ID}
	w := patchBulk(t, handler, p)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBulkUpdateTasks_NotFound — есть несуществующий id → 404, атомарность.
func TestBulkUpdateTasks_NotFound(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	task := createBulkTask(t, st, "nf")

	var p bulkPayload
	p.IDs = []int64{task.ID, 999999999}
	p.Changes.Status = "completed"
	w := patchBulk(t, handler, p)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// Атомарность: реально существующая задача не менялась (откат транзакции).
	got, _ := st.GetTask(ctx, task.ID)
	if got.Status != "new" {
		t.Errorf("task status = %q, want new (atomic rollback)", got.Status)
	}
}

// TestBulkUpdateTasks_WrongMethod — не PATCH → 405.
func TestBulkUpdateTasks_WrongMethod(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	w := httptest.NewRecorder()
	handler.BulkUpdateTasks(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// TestBulkUpdateTasks_InvalidJSON — невалидное тело → 400.
func TestBulkUpdateTasks_InvalidJSON(t *testing.T) {
	handler, _, cleanup := setupAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPatch, "/api/tasks", strings.NewReader("{oops"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.BulkUpdateTasks(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
