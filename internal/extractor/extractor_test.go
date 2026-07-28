package extractor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/extractor"
)

func TestExtractor_PlainText(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := extractor.NewAttachmentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAttachmentStore error: %v", err)
	}

	ext := extractor.NewExtractor(store)

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Test Subject
Message-ID: <abc123@example.com>
Date: Mon, 28 Jul 2026 10:00:00 +0300
Content-Type: text/plain; charset=utf-8

Это тестовое сообщение.
Вторая строка.`)

	result, err := ext.Extract(raw)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if result.From != "user@example.com" {
		t.Errorf("From = %q, want %q", result.From, "user@example.com")
	}
	if result.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", result.Subject, "Test Subject")
	}
	if result.MessageID != "abc123@example.com" {
		t.Errorf("MessageID = %q", result.MessageID)
	}
	if !strings.Contains(result.BodyText, "Это тестовое сообщение") {
		t.Errorf("BodyText does not contain expected text: %q", result.BodyText)
	}
}

func TestExtractor_WithAttachments(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := extractor.NewAttachmentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAttachmentStore error: %v", err)
	}

	ext := extractor.NewExtractor(store)

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Screenshot
Message-ID: <def456@example.com>
Content-Type: multipart/mixed; boundary="boundary123"

--boundary123
Content-Type: text/plain; charset=utf-8

Посмотрите скриншот ошибки.

--boundary123
Content-Type: image/png; name="screenshot.png"
Content-Disposition: attachment; filename="screenshot.png"
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGA
WjR9awAAAABJRU5ErkJggg==

--boundary123--`)

	result, err := ext.Extract(raw)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if !strings.Contains(result.BodyText, "Посмотрите скриншот ошибки") {
		t.Errorf("BodyText does not contain expected text: %q", result.BodyText)
	}

	if len(result.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(result.Attachments))
	}

	att := result.Attachments[0]
	if att.Filename != "screenshot.png" {
		t.Errorf("filename = %q, want %q", att.Filename, "screenshot.png")
	}
	if att.ContentType != "image/png" {
		t.Errorf("contentType = %q, want %q", att.ContentType, "image/png")
	}
	if att.Size == 0 {
		t.Error("attachment size is 0")
	}
	if att.StoragePath == "" {
		t.Error("storage path is empty")
	}

	// Проверяем, что файл реально сохранён
	fullPath := filepath.Join(tmpDir, att.StoragePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("attachment file not found: %s", fullPath)
	}
}

func TestCleaner_RemoveQuotedHistory(t *testing.T) {
	c := extractor.NewCleaner()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "gmail-style",
			input: `Новый текст ответа.

On Mon, Jul 28, 2026 at 10:00 AM, User wrote:
> Оригинальное сообщение
> Вторая строка`,
			want: "Новый текст ответа.",
		},
		{
			name: "outlook-style",
			input: `Новый текст.

From: User <user@example.com>
Sent: Monday, July 28, 2026 10:00 AM
To: Support <support@example.com>
Subject: Old subject

Старое сообщение`,
			want: "Новый текст.",
		},
		{
			name: "russian-marker",
			input: `Мой ответ.

28.07.2026 10:00, пользователь написал:
> Исходное сообщение`,
			want: "Мой ответ.",
		},
		{
			name: "dashed-separator",
			input: `Ответ.

-----Original Message-----
From: user@example.com
Old message text`,
			want: "Ответ.",
		},
		{
			name:  "no-quote",
			input: "Просто текст без цитирования.",
			want:  "Просто текст без цитирования.",
		},
	}

	for _, tt := range tests {
		got := c.CleanBody(tt.input)
		if got != tt.want {
			t.Errorf("%s: CleanBody = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestCleaner_RemoveSignatures(t *testing.T) {
	c := extractor.NewCleaner()

	input := `Текст письма.

--
С уважением,
Иван Иванов`

	want := "Текст письма."
	got := c.CleanBody(input)
	if got != want {
		t.Errorf("CleanBody = %q, want %q", got, want)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := extractor.NewAttachmentStore(tmpDir)

	att := &extractor.Attachment{
		Filename:    "../../etc/passwd",
		ContentType: "text/plain",
		Data:        []byte("test"),
		Size:        4,
	}

	path, err := store.Save(att)
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Путь не должен содержать ".."
	if strings.Contains(path, "..") {
		t.Errorf("path contains '..': %s", path)
	}

	// Имя файла должно быть безопасным — passwd без пути etc
	if !strings.HasSuffix(path, "passwd") {
		t.Errorf("unexpected sanitized name in path: %s", path)
	}
}

func TestAttachmentStore_DuplicateNames(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := extractor.NewAttachmentStore(tmpDir)

	att := &extractor.Attachment{
		Filename:    "file.txt",
		ContentType: "text/plain",
		Data:        []byte("first"),
		Size:        5,
	}

	path1, _ := store.Save(att)
	path2, _ := store.Save(att)

	if path1 == path2 {
		t.Errorf("duplicate files should have different paths: %s == %s", path1, path2)
	}
}

func TestExtractor_References(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := extractor.NewAttachmentStore(tmpDir)
	ext := extractor.NewExtractor(store)

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Re: Test
Message-ID: <new@example.com>
References: <old1@example.com> <old2@example.com>
In-Reply-To: <old2@example.com>
Content-Type: text/plain

Ответ на цепочку писем.`)

	result, err := ext.Extract(raw)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if len(result.References) != 2 {
		t.Errorf("expected 2 references, got %d: %v", len(result.References), result.References)
	}
	if result.InReplyTo != "old2@example.com" {
		t.Errorf("InReplyTo = %q", result.InReplyTo)
	}
}
