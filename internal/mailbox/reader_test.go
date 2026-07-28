package mailbox_test

import (
	"testing"

	"github.com/audetv/mailbridge/internal/mailbox"
)

func TestReader_New(t *testing.T) {
	cfg := mailbox.Config{
		Server:  "imap.example.com",
		Port:    993,
		User:    "test@example.com",
		TLS:     true,
		Inbox:   "INBOX",
		Archive: "Archive",
		Errors:  "Errors",
	}

	reader := mailbox.NewReader(cfg, nil)
	if reader == nil {
		t.Fatal("expected reader, got nil")
	}
}

func TestReader_Connect_InvalidServer(t *testing.T) {
	cfg := mailbox.Config{
		Server: "invalid.imap.example.com",
		Port:   993,
		User:   "test@example.com",
		TLS:    true,
		Inbox:  "INBOX",
	}

	reader := mailbox.NewReader(cfg, nil)
	err := reader.Connect()
	if err == nil {
		t.Fatal("expected error for invalid server")
	}
}

func TestRawEmail(t *testing.T) {
	raw := &mailbox.RawEmail{
		UID:  42,
		Data: []byte("test email data"),
	}

	if raw.UID != 42 {
		t.Errorf("UID = %d, want 42", raw.UID)
	}
	if string(raw.Data) != "test email data" {
		t.Errorf("Data = %s", string(raw.Data))
	}
}
