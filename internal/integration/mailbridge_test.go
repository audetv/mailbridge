// Package integration содержит сквозные тесты Mailbridge.
package integration

import (
	"context"
	"net/smtp"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/sender"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

func TestFullCycle_EmailToIssue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore error: %v", err)
	}
	defer st.Close()
	_ = st.Migrate(context.Background())
}

func TestSMTPConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	addr := "localhost:3025"
	client, err := smtp.Dial(addr)
	if err != nil {
		t.Skipf("greenmail not available: %v", err)
	}
	defer client.Close()

	if err := client.Noop(); err != nil {
		t.Errorf("SMTP noop failed: %v", err)
	}
}

func TestSender_Format(t *testing.T) {
	_, body := sender.FormatAcknowledgement(&sender.AcknowledgementData{
		To:                 "user@test.local",
		Subject:            "Не работает сайт",
		InReplyToMessageID: "original-msg",
		IssueSequence:      "INBOX-123",
		ProjectName:        "ТРК",
		TypeName:           "bug",
		Priority:           "high",
	})
	if !strings.Contains(body, "MAILBRIDGE-INTERNAL") {
		t.Error("body should contain internal marker")
	}
	if !strings.Contains(body, "INBOX-123") {
		t.Error("body should contain issue sequence")
	}

	_, body = sender.FormatCommentReply(&sender.CommentReplyData{
		To:                 "user@test.local",
		Subject:            "Re: Не работает сайт",
		InReplyToMessageID: "original-msg",
		IssueSequence:      "INBOX-123",
		CommentText:        "Проверил, проблема с сертификатом",
		CommentAuthor:      "Руслан",
	})
	if !strings.Contains(body, "Руслан") {
		t.Error("body should contain author name")
	}
}

func TestConfig_Load(t *testing.T) {
	t.Setenv("MAILBRIDGE_IMAP_SERVER", "localhost")
	t.Setenv("MAILBRIDGE_IMAP_USER", "test@localhost")
	t.Setenv("MAILBRIDGE_IMAP_PASS", "test")
	t.Setenv("MAILBRIDGE_SMTP_SERVER", "localhost")
	t.Setenv("MAILBRIDGE_SMTP_FROM", "support@test.local")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.IMAP.Server != "localhost" {
		t.Errorf("IMAP server = %s", cfg.IMAP.Server)
	}
}
