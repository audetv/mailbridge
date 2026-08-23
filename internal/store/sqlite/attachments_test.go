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
