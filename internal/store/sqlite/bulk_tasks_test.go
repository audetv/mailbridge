package sqlite_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

// TestBulkUpdateTasks — пакетная смена статуса (v0.23, шаг 4):
// N задач одной транзакцией, dups схлопываются, idempotent строки не пишутся.
func TestBulkUpdateTasks(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Три задачи: две new, одна уже completed (bulk на completed → no-op).
	var ids []int64
	for i := 0; i < 3; i++ {
		status := "new"
		if i == 2 {
			status = "completed"
		}
		task := &store.Task{MessageID: fmt.Sprintf("m-bulk-%d", i), Subject: "B", BodyText: "x", FromEmail: "u@e.com", Status: status}
		mustCreateStatusTask(t, s, task)
		ids = append(ids, task.ID)
	}
	for _, id := range []int64{ids[0], ids[1]} {
		tt, _ := s.GetTask(ctx, id)
		if tt.Status != "new" {
			t.Fatalf("setup: task %d status=%q, want new", id, tt.Status)
		}
	}

	// Bulk: новые 2 + no-op 1 (уже completed) + dup (тот же ID дважды).
	touched, err := s.BulkUpdateTasks(ctx, []int64{ids[0], ids[1], ids[2], ids[1]}, "completed", "admin")
	if err != nil {
		t.Fatalf("BulkUpdateTasks error: %v", err)
	}
	if touched != 3 {
		t.Errorf("touched = %d, want 3 (dups схлопнуты)", touched)
	}

	// Статусы: все completed; история: одна строка на реально перешедшую задачу,
	// у no-op-задачи строк нет (отсутствовали и до bulk).
	for i, id := range ids {
		tt, err := s.GetTask(ctx, id)
		if err != nil || tt == nil {
			t.Fatalf("GetTask %d error: %v", i, err)
		}
		if tt.Status != "completed" {
			t.Errorf("task %d status = %q, want completed", id, tt.Status)
		}
		history, err := s.GetTaskStatusHistory(ctx, id)
		if err != nil {
			t.Fatalf("GetTaskStatusHistory %d: %v", id, err)
		}
		if i == 2 {
			if len(history) != 0 {
				t.Errorf("task %d (no-op) history len = %d, want 0", id, len(history))
			}
			continue
		}
		if len(history) != 1 {
			t.Fatalf("task %d history len = %d, want 1", id, len(history))
		}
		last := history[len(history)-1]
		if last.FromStatus == nil || *last.FromStatus != "new" {
			t.Errorf("task %d from_status = %v, want new", id, last.FromStatus)
		}
		if last.ToStatus != "completed" || last.By != "admin" {
			t.Errorf("task %d last = %q/%q, want completed/admin", id, last.ToStatus, last.By)
		}
	}

	// Обновлены updatedAt у всех трёх.
	for _, id := range ids {
		tt, _ := s.GetTask(ctx, id)
		if tt.UpdatedAt.IsZero() {
			t.Errorf("task %d updated_at is zero, want set", id)
		}
	}
}

// TestBulkUpdateTasks_NotFound — одна несуществующая задача → ошибка (весь пакет откатывается).
func TestBulkUpdateTasks_NotFound(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{MessageID: "m-bulk-nf", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	mustCreateStatusTask(t, s, task)

	_, err := s.BulkUpdateTasks(ctx, []int64{task.ID, 99999}, "completed", "admin")
	if err == nil {
		t.Fatal("BulkUpdateTasks with missing id: want error, got nil")
	}
	if !errors.Is(err, store.ErrTaskNotFound) {
		t.Errorf("err = %v, want wrap store.ErrTaskNotFound", err)
	}

	// Транзакция откатилась: статус первой задачи не изменился.
	got, _ := s.GetTask(ctx, task.ID)
	if got.Status != "new" {
		t.Errorf("task status after rollback = %q, want new (atomic)", got.Status)
	}
}

// TestBulkUpdateTasks_Empty — пустой список → no-op (nil, nil).
func TestBulkUpdateTasks_Empty(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	if _, err := s.BulkUpdateTasks(context.Background(), nil, "completed", "admin"); err != nil {
		t.Fatalf("BulkUpdateTasks(empty) error: %v", err)
	}
}

// TestBulkUpdateProject — «К проекту X» (крит. приёмки): пакетная переноска, dups схлопнуты.
func TestBulkUpdateProject(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	var ids []int64
	for i := 0; i < 2; i++ {
		task := &store.Task{MessageID: fmt.Sprintf("m-bulkproj-%d", i), Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
		mustCreateStatusTask(t, s, task)
		ids = append(ids, task.ID)
	}

	touched, err := s.BulkUpdateProject(ctx, []int64{ids[0], ids[0], ids[1]}, "Офис")
	if err != nil {
		t.Fatalf("BulkUpdateProject error: %v", err)
	}
	if touched != 2 {
		t.Errorf("touched = %d, want 2 (dup схлопнут)", touched)
	}
	for _, id := range ids {
		tt, err := s.GetTask(ctx, id)
		if err != nil || tt == nil {
			t.Fatalf("GetTask %d: %v", id, err)
		}
		if tt.Project != "Офис" {
			t.Errorf("task %d project = %q, want Офис", id, tt.Project)
		}
		// Статус не трогаем, истории не пишем.
		if tt.Status != "new" {
			t.Errorf("task %d status = %q, want new (unchanged)", id, tt.Status)
		}
	}
}
