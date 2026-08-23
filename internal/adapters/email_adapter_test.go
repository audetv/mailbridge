package adapters_test

import (
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
