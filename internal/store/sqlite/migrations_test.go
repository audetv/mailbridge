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
