package sqlite_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

func TestMigrate_ProjectsTable(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	exists, err := s.TableExists(context.Background(), "projects")
	if err != nil {
		t.Fatalf("TableExists error: %v", err)
	}
	if !exists {
		t.Fatal("projects table not created")
	}
}

func TestStore_CreateAndGetProject(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	p := &store.Project{Name: "Лидер Спорт", Description: "магазин спорттоваров"}
	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatalf("CreateProject error: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero id")
	}

	got, err := s.GetProject(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetProject error: %v", err)
	}
	if got == nil {
		t.Fatal("project not found")
	}
	if got.Name != "Лидер Спорт" || got.Description != "магазин спорттоваров" || got.Archived {
		t.Errorf("unexpected project: %+v", got)
	}
}

func TestStore_CreateProject_NameRequired(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	if err := s.CreateProject(context.Background(), &store.Project{Name: "   "}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestStore_CreateProject_DuplicateName(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := s.CreateProject(ctx, &store.Project{Name: "Дубль"}); err != nil {
		t.Fatalf("first CreateProject error: %v", err)
	}
	if err := s.CreateProject(ctx, &store.Project{Name: "Дубль"}); err == nil {
		t.Fatal("expected unique-name violation on second create")
	}
}

func TestStore_GetProjectByName(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	p := &store.Project{Name: "ТРК "} // пробел в конце должен обрезаться
	_ = s.CreateProject(ctx, p)

	got, err := s.GetProjectByName(ctx, "ТРК")
	if err != nil {
		t.Fatalf("GetProjectByName error: %v", err)
	}
	if got == nil {
		t.Fatal("project not found by name")
	}
	if got.Name != "ТРК" {
		t.Errorf("name = %q, want %q", got.Name, "ТРК")
	}

	missing, err := s.GetProjectByName(ctx, "Нет такого")
	if err != nil {
		t.Fatalf("GetProjectByName(missing) error: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for missing project, got %+v", missing)
	}
}

func TestStore_ListProjects_Filters(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	for _, name := range []string{"Альфа", "Бета", "Кабак"} {
		_ = s.CreateProject(ctx, &store.Project{Name: name})
	}
	if err := s.SetProjectArchived(ctx, findByName(t, s, "Бета").ID, true); err != nil {
		t.Fatalf("SetProjectArchived error: %v", err)
	}

	// Только активные (сид добавляет «Входящие» — он активный)
	active := false
	projects, err := s.ListProjects(ctx, &store.ProjectFilter{Archived: &active})
	if err != nil {
		t.Fatalf("ListProjects error: %v", err)
	}
	if len(projects) != 3 { // Входящие (сид), Альфа, Кабак
		t.Fatalf("active projects = %d, want 3: %+v", len(projects), projects)
	}
	for _, p := range projects {
		if p.Archived {
			t.Errorf("archived project leaked into active list: %+v", p)
		}
	}

	// Поиск по названию: «ф» встречается только в «Альфа»
	projects, err = s.ListProjects(ctx, &store.ProjectFilter{Search: "ф"})
	if err != nil {
		t.Fatalf("ListProjects(search) error: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "Альфа" {
		t.Fatalf("search projects = %+v, want [Альфа]", projects)
	}

	// Все без фильтра (Входящие + 3 созданных)
	all, _ := s.ListProjects(ctx, nil)
	if len(all) != 4 {
		t.Fatalf("all projects = %d, want 4", len(all))
	}
}

func TestStore_UpdateProject(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	p := &store.Project{Name: "Старое"}
	_ = s.CreateProject(ctx, p)

	if err := s.UpdateProject(ctx, p.ID, "Новое имя", "новое описание"); err != nil {
		t.Fatalf("UpdateProject error: %v", err)
	}
	got, err := s.GetProject(ctx, p.ID)
	if err != nil || got == nil {
		t.Fatalf("GetProject after update: %v %v", got, err)
	}
	if got.Name != "Новое имя" || got.Description != "новое описание" {
		t.Errorf("unexpected after update: %+v", got)
	}

	// Обновление несуществующего
	if err := s.UpdateProject(ctx, 99999, "X", ""); err == nil {
		t.Fatal("expected error for updating missing project")
	}
	// Пустое имя
	if err := s.UpdateProject(ctx, p.ID, "  ", ""); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestStore_SetProjectArchived_RoundTrip(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	p := &store.Project{Name: "Арх"}
	_ = s.CreateProject(ctx, p)

	if err := s.SetProjectArchived(ctx, p.ID, true); err != nil {
		t.Fatalf("archive error: %v", err)
	}
	got, _ := s.GetProject(ctx, p.ID)
	if !got.Archived {
		t.Error("project not archived")
	}

	if err := s.SetProjectArchived(ctx, p.ID, false); err != nil {
		t.Fatalf("unarchive error: %v", err)
	}
	got, _ = s.GetProject(ctx, p.ID)
	if got.Archived {
		t.Error("project still archived after unarchive")
	}

	if err := s.SetProjectArchived(ctx, 99999, true); err == nil {
		t.Fatal("expected error for archiving missing project")
	}
}

func TestMigrate_SeedsInkhodyashi(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	got, err := s.GetProjectByName(ctx, "Входящие")
	if err != nil {
		t.Fatalf("GetProjectByName error: %v", err)
	}
	if got == nil {
		t.Fatal("seed project «Входящие» missing after migrate")
	}
	_ = ctx
}

func TestMigrate_SeedsDistinctTaskProjects(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Симулируем БД со старыми задачами: вставим проекты задач СНАЧАЛА,
	// затем запустим Migrate повторно — сид должен поднять их в projects.
	task := &store.Task{
		MessageID: "seed-probe-1",
		Subject:   "legacy task",
		Project:   "Ледовая арена",
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask (probe) error: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate error: %v", err)
	}

	got, err := s.GetProjectByName(ctx, "Ледовая арена")
	if err != nil {
		t.Fatalf("GetProjectByName error: %v", err)
	}
	if got == nil {
		t.Fatal("legacy task project not seeded into projects")
	}
}

func TestMigrate_SeedIdempotent(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	p := &store.Project{Name: "Постоянный", Description: "первое описание"}
	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatalf("CreateProject error: %v", err)
	}
	// Смена описания вручную, затем повторный Migrate — сид не должен его перезаписать.
	if err := s.UpdateProject(ctx, p.ID, "Постоянный", "изменённое описание"); err != nil {
		t.Fatalf("UpdateProject error: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate idempotency error: %v", err)
	}

	got, _ := s.GetProject(ctx, p.ID)
	if got.Description != "изменённое описание" {
		t.Errorf("seed clobbered description: %q", got.Description)
	}

	all, _ := s.ListProjects(ctx, nil)
	count := 0
	for _, pr := range all {
		if pr.Name == "Постоянный" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("duplicated project rows: %d", count)
	}
}

func findByName(t *testing.T, s *sqlite.Store, name string) *store.Project {
	t.Helper()
	got, err := s.GetProjectByName(context.Background(), name)
	if err != nil {
		t.Fatalf("GetProjectByName(%q): %v", name, err)
	}
	if got == nil {
		t.Fatalf("project %q not found", name)
	}
	return got
}
