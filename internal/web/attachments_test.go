package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func TestGetInboxAttachments(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём входящий
	if err := st.CreateInboxItem(ctx, &store.InboxItem{Source: "email", SourceID: "s1", Status: "unread"}); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}

	// Создаём вложение
	att := &store.Attachment{Hash: "h1", Filename: "f.png", ContentType: "image/png", Size: 10, StoragePath: "f.png"}
	if err := st.CreateAttachment(ctx, att); err != nil {
		t.Fatalf("CreateAttachment error: %v", err)
	}

	// Связываем
	if err := st.LinkAttachmentToInbox(ctx, 1, att.ID); err != nil {
		t.Fatalf("LinkAttachmentToInbox error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/inbox/1/attachments", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.GetInboxAttachments(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var atts []store.Attachment
	if err := json.Unmarshal(w.Body.Bytes(), &atts); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(atts) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(atts))
	}
}

func TestGetTaskAttachments(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	// Задача
	if err := st.CreateTask(ctx, &store.Task{MessageID: "m1", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Вложение
	att := &store.Attachment{Hash: "h2", Filename: "t.png", ContentType: "image/png", Size: 10, StoragePath: "t.png"}
	if err := st.CreateAttachment(ctx, att); err != nil {
		t.Fatalf("CreateAttachment error: %v", err)
	}

	// Связь
	if err := st.LinkAttachmentToTask(ctx, 1, att.ID); err != nil {
		t.Fatalf("LinkAttachmentToTask error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1/attachments", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.GetTaskAttachments(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var atts []store.Attachment
	if err := json.Unmarshal(w.Body.Bytes(), &atts); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(atts) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(atts))
	}
}

func TestUnlinkTaskAttachment(t *testing.T) {
	handler, st, cleanup := setupAPI(t)
	defer cleanup()
	ctx := context.Background()

	if err := st.CreateTask(ctx, &store.Task{MessageID: "m2", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	att := &store.Attachment{Hash: "h3", Filename: "x.png", ContentType: "image/png", Size: 10, StoragePath: "x.png"}
	if err := st.CreateAttachment(ctx, att); err != nil {
		t.Fatalf("CreateAttachment error: %v", err)
	}

	if err := st.LinkAttachmentToTask(ctx, 1, att.ID); err != nil {
		t.Fatalf("LinkAttachmentToTask error: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/1/attachments/1", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("attId", "1")
	w := httptest.NewRecorder()

	handler.UnlinkTaskAttachment(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	atts, _ := st.GetAttachmentsByTask(ctx, 1)
	if len(atts) != 0 {
		t.Errorf("expected 0 attachments after unlink, got %d", len(atts))
	}
}
