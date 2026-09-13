package sqlite_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func TestAttachmentsCRUD(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	att := &store.Attachment{
		Hash:        "abc123",
		Filename:    "test.png",
		ContentType: "image/png",
		Size:        1024,
		StoragePath: "2026-08-23/test.png",
	}
	if err := s.CreateAttachment(ctx, att); err != nil {
		t.Fatalf("CreateAttachment error: %v", err)
	}

	// Поиск по hash
	got, err := s.GetAttachmentByHash(ctx, "abc123")
	if err != nil || got == nil {
		t.Fatalf("GetAttachmentByHash error: %v", err)
	}
	if got.Filename != "test.png" {
		t.Errorf("Filename = %s", got.Filename)
	}

	// Поиск по ID
	got, err = s.GetAttachmentByID(ctx, att.ID)
	if err != nil || got == nil {
		t.Fatalf("GetAttachmentByID error: %v", err)
	}
}

func TestAttachmentLinks(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Задача
	task := &store.Task{MessageID: "m-att", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Входящий
	item := &store.InboxItem{Source: "email", SourceID: "s-att", Status: "unread"}
	if err := s.CreateInboxItem(ctx, item); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}

	// Вложение
	att := &store.Attachment{Hash: "h1", Filename: "f.png", ContentType: "image/png", Size: 10, StoragePath: "f.png"}
	if err := s.CreateAttachment(ctx, att); err != nil {
		t.Fatalf("CreateAttachment error: %v", err)
	}

	// Связь с задачей
	if err := s.LinkAttachmentToTask(ctx, task.ID, att.ID); err != nil {
		t.Fatalf("LinkAttachmentToTask error: %v", err)
	}

	// Связь с входящим
	if err := s.LinkAttachmentToInbox(ctx, item.ID, att.ID); err != nil {
		t.Fatalf("LinkAttachmentToInbox error: %v", err)
	}

	// Проверка
	atts, _ := s.GetAttachmentsByTask(ctx, task.ID)
	if len(atts) != 1 {
		t.Errorf("expected 1 task attachment, got %d", len(atts))
	}

	atts, _ = s.GetAttachmentsByInbox(ctx, item.ID)
	if len(atts) != 1 {
		t.Errorf("expected 1 inbox attachment, got %d", len(atts))
	}

	// Отвязка
	if err := s.UnlinkAttachmentFromTask(ctx, task.ID, att.ID); err != nil {
		t.Fatalf("UnlinkAttachmentFromTask error: %v", err)
	}

	atts, _ = s.GetAttachmentsByTask(ctx, task.ID)
	if len(atts) != 0 {
		t.Errorf("expected 0 attachments after unlink, got %d", len(atts))
	}
}

func TestGetAttachmentsByComment(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Задача
	task := &store.Task{MessageID: "m-c", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Комментарий
	comment := &store.TaskComment{TaskID: task.ID, Author: "user", Body: "Comment", Direction: "in", Kind: "user_comment"}
	if err := s.AddTaskComment(ctx, comment); err != nil {
		t.Fatalf("AddTaskComment error: %v", err)
	}

	// Вложение
	att := &store.Attachment{Hash: "h-c", Filename: "f.png", ContentType: "image/png", Size: 10, StoragePath: "f.png"}
	if err := s.CreateAttachment(ctx, att); err != nil {
		t.Fatalf("CreateAttachment error: %v", err)
	}

	// Связь
	if err := s.LinkAttachmentToComment(ctx, comment.ID, att.ID); err != nil {
		t.Fatalf("LinkAttachmentToComment error: %v", err)
	}

	// Проверка
	atts, err := s.GetAttachmentsByComment(ctx, comment.ID)
	if err != nil {
		t.Fatalf("GetAttachmentsByComment error: %v", err)
	}
	if len(atts) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(atts))
	}
	if atts[0].Filename != "f.png" {
		t.Errorf("Filename = %s", atts[0].Filename)
	}
}

func TestCopyInboxAttachmentsToTask(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Задача
	task := &store.Task{MessageID: "m-copy", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Входящий
	item := &store.InboxItem{Source: "email", SourceID: "s-copy", Status: "unread"}
	if err := s.CreateInboxItem(ctx, item); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}

	// Два вложения на входящем
	for _, h := range []string{"h-a", "h-b"} {
		a := &store.Attachment{Hash: h, Filename: h + ".png", ContentType: "image/png", Size: 10, StoragePath: h + ".png"}
		if err := s.CreateAttachment(ctx, a); err != nil {
			t.Fatalf("CreateAttachment error: %v", err)
		}
		if err := s.LinkAttachmentToInbox(ctx, item.ID, a.ID); err != nil {
			t.Fatalf("LinkAttachmentToInbox error: %v", err)
		}
	}

	// Первый прогон: 2 новые связи
	n, err := s.CopyInboxAttachmentsToTask(ctx, task.ID, item.ID)
	if err != nil {
		t.Fatalf("CopyInboxAttachmentsToTask error: %v", err)
	}
	if n != 2 {
		t.Errorf("first copy = %d new links, want 2", n)
	}

	got, err := s.GetAttachmentsByTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetAttachmentsByTask error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("task has %d attachments, want 2", len(got))
	}

	// Второй прогон: идемпотентно, 0 новых
	n, err = s.CopyInboxAttachmentsToTask(ctx, task.ID, item.ID)
	if err != nil {
		t.Fatalf("second copy error: %v", err)
	}
	if n != 0 {
		t.Errorf("second copy = %d new links, want 0 (idempotency)", n)
	}

	got, _ = s.GetAttachmentsByTask(ctx, task.ID)
	if len(got) != 2 {
		t.Errorf("after second copy task has %d attachments, want 2 (no dupes)", len(got))
	}
}
