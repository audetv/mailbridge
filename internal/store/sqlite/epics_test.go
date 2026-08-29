package sqlite_test

import (
	"context"
	"fmt"
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

// TestMigrate_EpicsDescription_Backfill — регрессия: БД, созданная ранним билдсом
// (epics без description/status), после Migrate получает недостающие колонки.
func TestMigrate_EpicsDescription_Backfill(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Имитация старой схемы: epics без description/status
	if _, err := s.ExecForTest(ctx, "DROP TABLE epics"); err != nil {
		t.Fatalf("drop epics: %v", err)
	}
	legacy := `CREATE TABLE epics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		number INTEGER NOT NULL DEFAULT 0,
		UNIQUE(project_id, number)
	)`
	if _, err := s.ExecForTest(ctx, legacy); err != nil {
		t.Fatalf("create legacy epics: %v", err)
	}
	if exists, _ := s.ColumnExistsForTest(ctx, "epics", "description"); exists {
		t.Fatal("precondition: legacy epics must NOT have description")
	}

	// Migrate должен дособирать недостающие колонки
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate on legacy schema: %v", err)
	}
	if exists, err := s.ColumnExistsForTest(ctx, "epics", "description"); err != nil || !exists {
		t.Fatalf("epics.description after Migrate: %v %v", exists, err)
	}
	if exists, err := s.ColumnExistsForTest(ctx, "epics", "status"); err != nil || !exists {
		t.Fatalf("epics.status after Migrate: %v %v", exists, err)
	}

	// CRUD теперь работает через полную схему
	proj := &store.Project{Name: "Backfill"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := s.CreateEpic(ctx, &store.Epic{ProjectID: proj.ID, Name: "После backfill"}); err != nil {
		t.Fatalf("CreateEpic after backfill: %v", err)
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

func TestEpicCRUD(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	proj := &store.Project{Name: "Проект CRUD"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Создание: номер auto (max+1), статус по умолчанию open
	e := &store.Epic{ProjectID: proj.ID, Name: "Модуль Alpha"}
	if err := s.CreateEpic(ctx, e); err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	if e.ID == 0 || e.Number != 1 || e.Status != "open" {
		t.Fatalf("unexpected new epic: id=%d number=%d status=%q", e.ID, e.Number, e.Status)
	}

	e2 := &store.Epic{ProjectID: proj.ID, Name: "Модуль Beta", Status: "in_progress"}
	if err := s.CreateEpic(ctx, e2); err != nil {
		t.Fatalf("CreateEpic 2: %v", err)
	}
	if e2.Number != 2 {
		t.Fatalf("expected auto number 2, got %d", e2.Number)
	}

	// Get / List
	got, err := s.GetEpic(ctx, e.ID)
	if err != nil || got == nil {
		t.Fatalf("GetEpic: %v %v", got, err)
	}
	if got.Name != "Модуль Alpha" {
		t.Fatalf("epic name: %q", got.Name)
	}
	list, err := s.ListEpics(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListEpics: %v", err)
	}
	if len(list) != 2 || list[0].ID != e.ID || list[1].ID != e2.ID {
		t.Fatalf("ListEpics order/len: %d", len(list))
	}

	// Update
	if err := s.UpdateEpic(ctx, e.ID, "Модуль Alpha v2", "описание", "done"); err != nil {
		t.Fatalf("UpdateEpic: %v", err)
	}
	got, _ = s.GetEpic(ctx, e.ID)
	if got.Name != "Модуль Alpha v2" || got.Description != "описание" || got.Status != "done" {
		t.Fatalf("updated epic: %+v", got)
	}
	if err := s.UpdateEpic(ctx, e.ID, "x", "", "bogus"); err == nil {
		t.Fatal("UpdateEpic accepted invalid status")
	}

	// Нет записей → empty
	if absent, err := s.GetEpic(ctx, 999999); err != nil || absent != nil {
		t.Fatalf("GetEpic(absent) = %v, %v", absent, err)
	}

	// Delete
	if err := s.DeleteEpic(ctx, e2.ID); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}
	if deleted, _ := s.GetEpic(ctx, e2.ID); deleted != nil {
		t.Fatal("deleted epic still present")
	}
	if err := s.DeleteEpic(ctx, e2.ID); err == nil {
		t.Fatal("second delete should fail")
	}
}

func TestSetTaskEpicAndProgress(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	proj := &store.Project{Name: "Проект прогресса"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	epic := &store.Epic{ProjectID: proj.ID, Name: "Модуль"}
	if err := s.CreateEpic(ctx, epic); err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}

	// 3 задачи в эпику: 2 open-ish, 1 completed
	var ids []int64
	for i, status := range []string{"active", "backlog", "completed"} {
		task := &store.Task{
			MessageID: fmt.Sprintf("epic-prog-%d", i),
			Subject:   "Задача", BodyText: "t", Status: status, Project: proj.Name,
		}
		if err := s.CreateTask(ctx, task); err != nil {
			t.Fatalf("CreateTask %d: %v", i, err)
		}
		if err := s.SetTaskEpic(ctx, task.ID, epic.ID); err != nil {
			t.Fatalf("SetTaskEpic %d: %v", i, err)
		}
		ids = append(ids, task.ID)
	}

	p, err := s.EpicProgress(ctx, epic.ID)
	if err != nil {
		t.Fatalf("EpicProgress: %v", err)
	}
	if p.Total != 3 || p.Done != 1 || p.Open != 2 {
		t.Fatalf("progress: %+v", p)
	}

	// Перенос задачи в другой модуль
	epic2 := &store.Epic{ProjectID: proj.ID, Name: "Второй"}
	if err := s.CreateEpic(ctx, epic2); err != nil {
		t.Fatalf("CreateEpic 2: %v", err)
	}
	if err := s.SetTaskEpic(ctx, ids[0], epic2.ID); err != nil {
		t.Fatalf("move task: %v", err)
	}
	if p, _ = s.EpicProgress(ctx, epic.ID); p.Total != 2 || p.Done != 1 {
		t.Fatalf("progress after move: %+v", p)
	}

	// Отвязка (epic_id = NULL)
	if err := s.SetTaskEpic(ctx, ids[1], 0); err != nil {
		t.Fatalf("unlink task: %v", err)
	}
	if got, _ := s.GetTask(ctx, ids[1]); got == nil || got.EpicID != nil {
		t.Fatalf("task after unlink: %+v", got)
	}
	if p, _ = s.EpicProgress(ctx, epic.ID); p.Total != 1 || p.Done != 1 {
		t.Fatalf("progress after unlink: %+v", p)
	}

	// Нет задачи → ошибка
	if err := s.SetTaskEpic(ctx, 999999, epic.ID); err == nil {
		t.Fatal("SetTaskEpic for missing task should fail")
	}
}

func TestDeleteEpicKeepsTasks(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	proj := &store.Project{Name: "Проект удаления"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	epic := &store.Epic{ProjectID: proj.ID, Name: "Куда уйдёт"}
	if err := s.CreateEpic(ctx, epic); err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	task := &store.Task{MessageID: "epic-del-1", Subject: "Задача", BodyText: "t", Status: "active", Project: proj.Name}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := s.SetTaskEpic(ctx, task.ID, epic.ID); err != nil {
		t.Fatalf("SetTaskEpic: %v", err)
	}

	if n, err := s.CountTasksInEpic(ctx, epic.ID); err != nil || n != 1 {
		t.Fatalf("CountTasksInEpic = %d, %v", n, err)
	}

	if err := s.DeleteEpic(ctx, epic.ID); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}
	got, err := s.GetTask(ctx, task.ID)
	if err != nil || got == nil {
		t.Fatalf("task after epic delete: %v, %v", got, err)
	}
	if got.EpicID != nil {
		t.Fatalf("task epic_id after delete = %v, want NULL", got.EpicID)
	}
}

// TestListTasks_EpicFilter проверяет фильтр списка задач по модулю (v0.22 step 10).
func TestListTasks_EpicFilter(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	proj := &store.Project{Name: "Проект фильтров"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	epic := &store.Epic{ProjectID: proj.ID, Name: "Фильтр"}
	if err := s.CreateEpic(ctx, epic); err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}

	const total = 5
	const epicCount = 3
	for i := 1; i <= total; i++ {
		task := &store.Task{MessageID: fmt.Sprintf("epic-filt-%d", i), Subject: "s", BodyText: "b", Status: "active", Project: proj.Name}
		if err := s.CreateTask(ctx, task); err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
		if i <= epicCount {
			if err := s.SetTaskEpic(ctx, task.ID, epic.ID); err != nil {
				t.Fatalf("SetTaskEpic: %v", err)
			}
		}
	}

	// Без фильтра — все задачи
	all, err := s.ListTasks(ctx, &store.TaskFilter{Project: proj.Name, Page: 1, PerPage: 50})
	if err != nil || all == nil {
		t.Fatalf("ListTasks: %v, %v", all, err)
	}
	if int(all.Total) != total {
		t.Fatalf("total = %d, want %d", all.Total, total)
	}

	// Фильтр по модулю — только его задачи
	filt, err := s.ListTasks(ctx, &store.TaskFilter{EpicID: &epic.ID, Page: 1, PerPage: 50})
	if err != nil || filt == nil {
		t.Fatalf("ListTasks epic: %v, %v", filt, err)
	}
	if int(filt.Total) != epicCount {
		t.Fatalf("filtered total = %d, want %d", filt.Total, epicCount)
	}
	for _, task := range filt.Tasks {
		if task.EpicID == nil || *task.EpicID != epic.ID {
			t.Fatalf("task in epic filter: %+v", task)
		}
	}

	// Чужой модуль — пусто
	other := int64(999999)
	none, err := s.ListTasks(ctx, &store.TaskFilter{EpicID: &other, Page: 1, PerPage: 50})
	if err != nil || none == nil || int(none.Total) != 0 {
		t.Fatalf("ListTasks other epic: %v, %v", none, err)
	}
}
