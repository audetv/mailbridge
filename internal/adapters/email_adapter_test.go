package adapters_test

import (
	"os"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/adapters"
	"github.com/audetv/mailbridge/internal/extractor"
)

func setupEmailAdapter(t *testing.T) *adapters.EmailAdapter {
	tmpDir := t.TempDir()
	attStore, err := extractor.NewAttachmentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAttachmentStore error: %v", err)
	}
	ext := extractor.NewExtractor(attStore)
	return adapters.NewEmailAdapter(ext, nil, tmpDir)
}

func TestEmailAdapter_Parse(t *testing.T) {
	adapter := setupEmailAdapter(t)

	raw := []byte(`From: "Иван Петров" <ivan@example.com>
To: support@example.com
Subject: Тестовое письмо
Message-ID: <test-msg-1@example.com>
Content-Type: text/plain; charset=utf-8

Текст письма`)

	parseResult, err := adapter.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	item := parseResult.InboxItem

	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if item.Source != "email" {
		t.Errorf("Source = %s", item.Source)
	}
	if item.SourceID != "test-msg-1@example.com" {
		t.Errorf("SourceID = %s", item.SourceID)
	}
	if item.FromName != "Иван Петров" {
		t.Errorf("FromName = %s", item.FromName)
	}
	if item.Subject != "Тестовое письмо" {
		t.Errorf("Subject = %s", item.Subject)
	}
	if !strings.Contains(item.BodyText, "Текст письма") {
		t.Error("BodyText does not contain text")
	}
	if item.Status != "unread" {
		t.Errorf("Status = %s", item.Status)
	}
	if !strings.Contains(item.Meta, "message_id") {
		t.Error("Meta does not contain message_id")
	}
}

func TestEmailAdapter_ThreadIDFromReferences(t *testing.T) {
	adapter := setupEmailAdapter(t)

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Re: Test
Message-ID: <msg-2@example.com>
References: <msg-1@example.com>
In-Reply-To: <msg-1@example.com>
Content-Type: text/plain

Ответ`)

	parseResult, err := adapter.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	item := parseResult.InboxItem
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if item.ThreadID != "msg-1@example.com" {
		t.Errorf("ThreadID = %s, want msg-1@example.com", item.ThreadID)
	}
}

// TestEmailAdapter_CalendarInvite — приёмка issue #1:
// приглашение Exchange (text/calendar в multipart/alternative) должно
// дойти до InboxItem.BodyText секцией [СОБЫТИЕ], а не потеряться.
func TestEmailAdapter_CalendarInvite(t *testing.T) {
	adapter := setupEmailAdapter(t)

	raw, err := os.ReadFile("testdata/calendar_invite.eml")
	if err != nil {
		t.Fatalf("read testdata/calendar_invite.eml: %v", err)
	}

	parseResult, err := adapter.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	item := parseResult.InboxItem

	if !strings.Contains(item.BodyText, "[СОБЫТИЕ]") {
		t.Errorf("BodyText missing [СОБЫТИЕ] section, got: %q", item.BodyText)
	}
	if !strings.Contains(item.BodyText, "онлайн продажи") {
		t.Errorf("BodyText missing event summary, got: %q", item.BodyText)
	}
	if !strings.Contains(item.BodyText, "14:30") {
		t.Errorf("BodyText missing event start time, got: %q", item.BodyText)
	}
	if !strings.Contains(item.BodyText, "у Алексея") {
		t.Errorf("BodyText missing location (folded RFC 5545 line), got: %q", item.BodyText)
	}
	if !strings.Contains(item.BodyText, "Бухтина") {
		t.Errorf("BodyText missing organizer (folded RFC 5545 line), got: %q", item.BodyText)
	}
	t.Logf("Adapted body:\n%s", item.BodyText)
}

// TestEmailAdapter_CalendarCancel — ветка CANCEL (письмо 2.eml, записыв 61):
// отмена регулярных встреч тоже должна дойти до AI как [СОБЫТИЕ] с METHOD CANCEL.
func TestEmailAdapter_CalendarCancel(t *testing.T) {
	adapter := setupEmailAdapter(t)

	raw, err := os.ReadFile("testdata/calendar_cancel.eml")
	if err != nil {
		t.Fatalf("read testdata/calendar_cancel.eml: %v", err)
	}

	parseResult, err := adapter.Parse(raw)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	item := parseResult.InboxItem

	if !strings.Contains(item.BodyText, "[СОБЫТИЕ]") {
		t.Errorf("BodyText missing [СОБЫТИЕ] section, got: %q", item.BodyText)
	}
	if !strings.Contains(item.BodyText, "CANCEL") {
		t.Errorf("BodyText missing METHOD CANCEL, got: %q", item.BodyText)
	}
	if !strings.Contains(item.BodyText, "Сайт Гостиницы") {
		t.Errorf("BodyText missing event summary «Сайт Гостиницы», got: %q", item.BodyText)
	}
	t.Logf("Adapted CANCEL body:\n%s", item.BodyText)
}
