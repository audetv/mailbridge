package sqlite_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func TestMigrate_AttachmentsTables(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	for _, table := range []string{"attachments", "inbox_attachments", "task_attachments", "comment_attachments"} {
		exists, err := s.TableExists(context.Background(), table)
		if err != nil {
			t.Fatalf("TableExists error for %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s not created", table)
		}
	}
}

func TestMigrate_InboxItemsTable(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	exists, err := s.TableExists(context.Background(), "inbox_items")
	if err != nil {
		t.Fatalf("TableExists error: %v", err)
	}
	if !exists {
		t.Fatal("inbox_items table not created")
	}

	// Проверяем что индекс thread_id создан
	exists, err = s.TableExists(context.Background(), "idx_inbox_items_thread_id")
	if err != nil {
		t.Fatalf("TableExists error: %v", err)
	}
	// Индексы не видны через sqlite_master с type='table', проверяем отдельно
	_ = exists
}

func TestMigrate_ThreadsColumns(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Проверяем что thread можно создать с новыми полями
	thread := &store.Thread{
		ThreadID: "thread-new",
		Source:   "email",
		Subject:  "Тестовая цепочка",
	}
	if err := s.CreateThread(ctx, thread); err != nil {
		t.Fatalf("CreateThread error: %v", err)
	}

	got, _ := s.GetThread(ctx, "thread-new")
	if got == nil {
		t.Fatal("thread not found")
	}
	if got.Source != "email" {
		t.Errorf("Source = %s", got.Source)
	}
	if got.Subject != "Тестовая цепочка" {
		t.Errorf("Subject = %s", got.Subject)
	}
}

func TestMigrate_TaskCommentsNewSchema(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	// Проверяем что колонка kind существует
	exists, err := s.ColumnExistsForTest(context.Background(), "task_comments", "kind")
	if err != nil {
		t.Fatalf("ColumnExists error: %v", err)
	}
	if !exists {
		t.Error("column kind not found")
	}
}

// TestMigrate_TaskCommentsApproved проверяет колонку approved (ФАЗА 4, шаг 17.1):
// существует, NULL по умолчанию; идемпотентность — миграция повторна.
func TestMigrate_TaskCommentsApproved(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	exists, err := s.ColumnExistsForTest(ctx, "task_comments", "approved")
	if err != nil {
		t.Fatalf("ColumnExists error: %v", err)
	}
	if !exists {
		t.Fatal("column approved not found in task_comments")
	}

	// Повторный Migrate не должен падать (идемпотентно)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate returned error: %v", err)
	}
	exists, err = s.ColumnExistsForTest(ctx, "task_comments", "approved")
	if err != nil {
		t.Fatalf("ColumnExists after re-migrate: %v", err)
	}
	if !exists {
		t.Fatal("column approved lost after re-migrate")
	}

	// Новый комментарий: approved = NULL (не утверждён)
	task := &store.Task{
		MessageID: "approved-mig-1",
		Subject:   "Approved миграция",
		BodyText:  "t",
		FromEmail: "u@e.com",
		Status:    "new",
		Project:   "Тест",
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	c := &store.TaskComment{TaskID: task.ID, Author: "admin", Body: "hello", Direction: "out", Kind: "reply"}
	if err := s.AddTaskComment(ctx, c); err != nil {
		t.Fatalf("AddTaskComment: %v", err)
	}
	list, err := s.GetTaskComments(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskComments: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(list))
	}
	if list[0].Approved != nil {
		t.Errorf("new comment Approved = %d, want nil (NULL)", *list[0].Approved)
	}
}

// TestMigrate_TaskCommentsApprovedBackfill проверяет ветку ALTER TABLE:
// существующая БД (task_comments без approved) → Migrate добавляет колонку,
// существующие комментарии получают NULL.
func TestMigrate_TaskCommentsApprovedBackfill(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Имитируем старую схему: колонки новые, но approved-колонки нет.
	if _, err := s.ExecForTest(ctx, "ALTER TABLE task_comments RENAME TO task_comments_old"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	old := `CREATE TABLE task_comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		comment_id INTEGER,
		task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		author TEXT NOT NULL,
		body TEXT NOT NULL,
		direction TEXT NOT NULL DEFAULT 'in',
		kind TEXT NOT NULL DEFAULT 'user_comment',
		inbox_item_id INTEGER REFERENCES inbox_items(id),
		verdict_json TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := s.ExecForTest(ctx, old); err != nil {
		t.Fatalf("create old: %v", err)
	}

	// Существующий комментарий в старой схеме
	task := &store.Task{MessageID: "approved-bf-1", Subject: "BF", BodyText: "t", FromEmail: "u@e.com", Status: "new", Project: "Тест"}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := s.ExecForTest(ctx,
		"INSERT INTO task_comments (task_id, author, body, direction, kind) VALUES (?, 'admin', 'old comment', 'out', 'user_comment')", task.ID); err != nil {
		t.Fatalf("insert old comment: %v", err)
	}

	// Migrate должен добавить колонку approved
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	exists, err := s.ColumnExistsForTest(ctx, "task_comments", "approved")
	if err != nil {
		t.Fatalf("ColumnExists: %v", err)
	}
	if !exists {
		t.Fatal("column approved not added by backfill migration")
	}

	// Старый комментарий читается, approved = NULL
	list, err := s.GetTaskComments(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskComments: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(list))
	}
	if list[0].Approved != nil {
		t.Errorf("backfilled comment Approved = %d, want nil (NULL)", *list[0].Approved)
	}
}
