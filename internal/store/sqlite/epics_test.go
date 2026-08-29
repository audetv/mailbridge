package sqlite_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func TestMigrate_EpicsTable(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	if exists, err := s.TableExists(ctx, "epics"); err != nil {
		t.Fatalf("TableExists(epics) error: %v", err)
	} else if !exists {
		t.Fatal("epics table not created")
	}

	if exists, err := s.ColumnExistsForTest(ctx, "tasks", "epic_id"); err != nil {
		t.Fatalf("ColumnExistsForTest(tasks, epic_id) error: %v", err)
	} else if !exists {
		t.Fatal("tasks.epic_id column missing")
	}

	// Ссылка задачи на модуль работает, удаление модуля не бьёт по задачам
	proj := &store.Project{Name: "Проект эпиков"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	res, err := s.ExecForTest(ctx, "INSERT INTO epics (project_id, number, name) VALUES (?, 1, 'М-1')", proj.ID)
	if err != nil {
		t.Fatalf("insert epics: %v", err)
	}
	epicID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	if _, err := s.ExecForTest(ctx, "INSERT INTO epics (project_id, number, name) VALUES (?, 2, 'М-2')", proj.ID); err != nil {
		t.Fatalf("insert epics 2: %v", err)
	}

	task := &store.Task{MessageID: "epic-test-1", Subject: "Задача", BodyText: "текст", Status: "active", Project: "Проект эпиков"}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := s.ExecForTest(ctx, "UPDATE tasks SET epic_id = ? WHERE id = ?", epicID, task.ID); err != nil {
		t.Fatalf("set epic_id: %v", err)
	}
	got, err := s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.EpicID == nil || *got.EpicID != epicID {
		t.Fatalf("expected epic_id=%d, got %v", epicID, got.EpicID)
	}

	// Удаление модуля — задача остаётся с epic_id = NULL
	if _, err := s.ExecForTest(ctx, "DELETE FROM epics WHERE id = ?", epicID); err != nil {
		t.Fatalf("delete epic: %v", err)
	}
	got, err = s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask after epic delete: %v", err)
	}
	if got.EpicID != nil {
		t.Fatalf("expected epic_id NULL after epic delete, got %v", got.EpicID)
	}
}

func TestMigrate_EpicsIdempotent(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Повторный запуск миграции не должен ломать схему
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate error: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("third Migrate error: %v", err)
	}

	if exists, err := s.TableExists(ctx, "epics"); err != nil || !exists {
		t.Fatalf("epics table after re-migrate: %v %v", exists, err)
	}
	if exists, err := s.ColumnExistsForTest(ctx, "tasks", "epic_id"); err != nil || !exists {
		t.Fatalf("tasks.epic_id after re-migrate: %v %v", exists, err)
	}

	// UNIQUE(project_id, number): дубль номера в одном проекте отклоняется
	proj := &store.Project{Name: "Проект UNIQUE"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if _, err := s.ExecForTest(ctx, "INSERT INTO epics (project_id, number, name) VALUES (?, 7, 'A')", proj.ID); err != nil {
		t.Fatalf("insert epic: %v", err)
	}
	if _, err := s.ExecForTest(ctx, "INSERT INTO epics (project_id, number, name) VALUES (?, 7, 'B')", proj.ID); err == nil {
		t.Fatal("duplicate epic number accepted — expected UNIQUE violation")
	}
}
