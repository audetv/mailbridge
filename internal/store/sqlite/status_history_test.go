package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func mustCreateStatusTask(t *testing.T, s interface {
	CreateTask(context.Context, *store.Task) error
}, task *store.Task) {
	t.Helper()
	if err := s.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
}

// TestSetTaskStatus проверяет единый путь смены статуса:
// переход пишется в task_status_history (from→to, by), повторный статус тем же
// значением строки НЕ пишет (идемпотентность).
func TestSetTaskStatus(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{MessageID: "m-status", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	mustCreateStatusTask(t, s, task)

	// 1) new → in_progress: строка пишется, from=new, to=in_progress, by=admin.
	if err := s.SetTaskStatus(ctx, task.ID, "in_progress", "admin"); err != nil {
		t.Fatalf("SetTaskStatus #1 error: %v", err)
	}
	gotTask, err := s.GetTask(ctx, task.ID)
	if err != nil || gotTask == nil {
		t.Fatalf("GetTask after SetTaskStatus error: %v", err)
	}
	if gotTask.Status != "in_progress" {
		t.Errorf("task status = %q, want in_progress", gotTask.Status)
	}

	history, err := s.GetTaskStatusHistory(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskStatusHistory error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	h := history[0]
	if h.FromStatus == nil || *h.FromStatus != "new" {
		t.Errorf("from_status = %v, want new", h.FromStatus)
	}
	if h.ToStatus != "in_progress" {
		t.Errorf("to_status = %q, want in_progress", h.ToStatus)
	}
	if h.By != "admin" {
		t.Errorf("by = %q, want admin", h.By)
	}
	if h.TaskID != task.ID {
		t.Errorf("task_id = %d, want %d", h.TaskID, task.ID)
	}
	if h.At.IsZero() {
		t.Error("at is zero, want set")
	}

	// 2) Повторный вызов с тем же статусом — строка НЕ пишется (идемпотентность).
	if err := s.SetTaskStatus(ctx, task.ID, "in_progress", "admin"); err != nil {
		t.Fatalf("SetTaskStatus #2 (no-op) error: %v", err)
	}
	history, _ = s.GetTaskStatusHistory(ctx, task.ID)
	if len(history) != 1 {
		t.Errorf("history len after no-op = %d, want 1", len(history))
	}

	// 3) in_progress → completed (by=ai).
	if err := s.SetTaskStatus(ctx, task.ID, "completed", "ai"); err != nil {
		t.Fatalf("SetTaskStatus #3 error: %v", err)
	}
	history, _ = s.GetTaskStatusHistory(ctx, task.ID)
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2", len(history))
	}
	last := history[len(history)-1]
	if last.FromStatus == nil || *last.FromStatus != "in_progress" {
		t.Errorf("last.from_status = %v, want in_progress", last.FromStatus)
	}
	if last.ToStatus != "completed" || last.By != "ai" {
		t.Errorf("last = %q/%q, want completed/ai", last.ToStatus, last.By)
	}

	// Сортировка: по at asc (первая строка — самая ранняя).
	if history[0].ToStatus != "in_progress" {
		t.Errorf("first history entry out of order: to=%q", history[0].ToStatus)
	}
}

// TestSetTaskStatus_NotFound — смена статуса несуществующей задачи → error.
func TestSetTaskStatus_NotFound(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	err := s.SetTaskStatus(context.Background(), 99999, "in_progress", "admin")
	if err == nil {
		t.Fatal("SetTaskStatus for missing task: want error, got nil")
	}
	if !errors.Is(err, store.ErrTaskNotFound) {
		t.Errorf("err = %v, want wrap store.ErrTaskNotFound", err)
	}
}

// TestGetTaskStatusHistory_Empty — у задачи-сироты история пуста, не nil.
func TestGetTaskStatusHistory_Empty(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{MessageID: "m-empty", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	mustCreateStatusTask(t, s, task)

	history, err := s.GetTaskStatusHistory(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskStatusHistory error: %v", err)
	}
	if history == nil {
		t.Error("history = nil, want empty non-nil slice")
	}
	if len(history) != 0 {
		t.Errorf("history len = %d, want 0", len(history))
	}
}
