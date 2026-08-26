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

func TestAttachmentStore_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := extractor.NewAttachmentStore(tmpDir)

	att := &extractor.Attachment{
		Filename:    "file.txt",
		ContentType: "text/plain",
		Data:        []byte("same content"),
		Size:        12,
	}

	path1, _ := store.Save(att)
	path2, _ := store.Save(att)

	if path1 != path2 {
		t.Errorf("expected same path for same content, got %s and %s", path1, path2)
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

func TestCleaner_SanitizeHTML(t *testing.T) {
	c := extractor.NewCleaner()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty paragraphs",
			input: `<p>&nbsp;</p><p>  </p><p>Текст</p>`,
			want:  `<p>Текст</p>`,
		},
		{
			name:  "multiple br",
			input: `Текст<br><br><br><br>конец`,
			want:  `Текст<br><br>конец`,
		},
		{
			name:  "empty divs",
			input: `<div></div><div>Содержимое</div><div> </div>`,
			want:  `<div>Содержимое</div>`,
		},
		{
			name:  "base tag removed",
			input: `<base href="https://e.mail.ru/"><p>Текст</p>`,
			want:  `<p>Текст</p>`,
		},
		{
			name:  "cid images preserved for inline replacement",
			input: `<p>Текст</p><img src="cid:ii_123">`,
			want:  `<p>Текст</p><img src="cid:ii_123">`,
		},
		{
			name:  "clean text passes through",
			input: "Простой текст без HTML",
			want:  "Простой текст без HTML",
		},
	}

	for _, tt := range tests {
		got := c.SanitizeHTML(tt.input)
		if got != tt.want {
			t.Errorf("%s: SanitizeHTML = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestExtractor_InlineImages(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := extractor.NewAttachmentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAttachmentStore error: %v", err)
	}

	ext := extractor.NewExtractor(store)

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Screenshot
Message-ID: <inline-test@example.com>
Content-Type: multipart/related; boundary="boundary456"

--boundary456
Content-Type: text/html; charset=utf-8

<html><body><p>Смотрите скриншот:</p><img src="cid:img123"></body></html>

--boundary456
Content-Type: image/png; name="screenshot.png"
Content-Disposition: inline; filename="screenshot.png"
Content-ID: <img123>
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGA
WjR9awAAAABJRU5ErkJggg==

--boundary456--`)

	result, err := ext.Extract(raw)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	// HTML должен содержать путь к файлу вместо cid
	if strings.Contains(result.BodyHTML, "cid:img123") {
		t.Error("HTML should not contain cid: reference after extraction")
	}
	if !strings.Contains(result.BodyHTML, "/api/attachments/") {
		t.Error("HTML should contain path to saved inline image")
	}

	// Вложение должно быть сохранено
	if len(result.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(result.Attachments))
	}
}

func TestExtractor_InlineImageInAttachments(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := extractor.NewAttachmentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAttachmentStore error: %v", err)
	}

	ext := extractor.NewExtractor(store)

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Screenshot
Message-ID: <inline-test@example.com>
Content-Type: multipart/related; boundary="boundary456"

--boundary456
Content-Type: text/html; charset=utf-8

<html><body><p>Смотрите скриншот:</p><img src="cid:img123"></body></html>

--boundary456
Content-Type: image/png; name="image001.png"
Content-Disposition: inline; filename="image001.png"
Content-ID: <img123>
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGA
WjR9awAAAABJRU5ErkJggg==

--boundary456--`)

	result, err := ext.Extract(raw)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if len(result.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(result.Attachments))
	}

	att := result.Attachments[0]
	if att.ContentType != "image/png" {
		t.Errorf("ContentType = %s, want image/png", att.ContentType)
	}
	if att.StoragePath == "" {
		t.Error("StoragePath is empty")
	}
	t.Logf("Attachment: filename=%s, type=%s, path=%s", att.Filename, att.ContentType, att.StoragePath)
}

func TestExtractor_MultipleAttachmentTypes(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := extractor.NewAttachmentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAttachmentStore error: %v", err)
	}

	ext := extractor.NewExtractor(store)

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Документы
Message-ID: <multi-types@example.com>
Content-Type: multipart/mixed; boundary="b123"

--b123
Content-Type: text/plain; charset=utf-8

Прикладываю документы.

--b123
Content-Type: application/pdf; name="doc.pdf"
Content-Disposition: attachment; filename="doc.pdf"
Content-Transfer-Encoding: base64

JVBERi0xLjQKJcOkw7zDtsOfCjIgMCBvYmoKPDwvTGVuZ3RoIDMgMCBSL0ZpbHRlci9GbGF0ZURl
Y29kZT4+CnN0cmVhbQp4nDPQM1Qo5ypUMFAwALJMLQsABlMDCmVuZHN0cmVhbQplbmRvYmoK

--b123
Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document; name="doc.docx"
Content-Disposition: attachment; filename="doc.docx"
Content-Transfer-Encoding: base64

UEsDBBQABgAIAAAAIQCq5n2yMQAAADAAAAATAAAAW0NvbnRlbnRfVHlwZXNdLnhtbI2QwWrDMAyG
7/sKRu9t42YdxYdR2Glh7DSU7TQXJdtJTWwZS2Nr376aYqU9FEsvQv8n8esv0lkdw8QYCkQF

--b123--`)

	result, err := ext.Extract(raw)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if len(result.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(result.Attachments))
	}

	for _, att := range result.Attachments {
		if att.StoragePath == "" {
			t.Error("StoragePath is empty")
		}
		if att.Size == 0 {
			t.Error("Size is 0")
		}
		t.Logf("Attachment: %s (type: %s, path: %s)", att.Filename, att.ContentType, att.StoragePath)
	}
}

func TestExtractor_CalendarFromOtherParts(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := extractor.NewAttachmentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAttachmentStore error: %v", err)
	}

	ext := extractor.NewExtractor(store)

	raw, err := os.ReadFile("testdata/calendar_invite.eml")
	if err != nil {
		t.Fatalf("read testdata/calendar_invite.eml: %v", err)
	}

	result, err := ext.Extract(raw)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	// Календарные части (enmime → OtherParts) должны попасть в Calendar
	if result.Calendar == "" {
		t.Fatal("Calendar is empty — text/calendar part lost (issue #1 root cause)")
	}
	if !strings.Contains(result.Calendar, "[СОБЫТИЕ]") {
		t.Errorf("Calendar missing [СОБЫТИЕ] header, got: %q", result.Calendar)
	}
	if !strings.Contains(result.Calendar, "онлайн продажи") {
		t.Errorf("Calendar missing SUMMARY «онлайн продажи», got: %q", result.Calendar)
	}
	if !strings.Contains(result.Calendar, "14:30") {
		t.Errorf("Calendar missing start time 14:30, got: %q", result.Calendar)
	}
	if !strings.Contains(result.Calendar, "у Алексея") {
		t.Errorf("Calendar missing LOCATION «у Алексея» (folded line!), got: %q", result.Calendar)
	}
	if !strings.Contains(result.Calendar, "Бухтина") {
		t.Errorf("Calendar missing ORGANIZER CN «Бухтина» (folded line!), got: %q", result.Calendar)
	}
	if !strings.Contains(result.Calendar, "Кузьмина") {
		t.Errorf("Calendar missing ATTENDEE «Кузьмина», got: %q", result.Calendar)
	}
	if !strings.Contains(result.Calendar, "REQUEST") {
		t.Errorf("Calendar missing METHOD REQUEST, got: %q", result.Calendar)
	}

	// BodyText при этом остаётся как есть (пустым для этого письма) —
	// календарь не смешивается с телом на уровне extractor.
	t.Logf("Calendar output:\n%s", result.Calendar)
}

func TestExtractor_NoCalendarPassthrough(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := extractor.NewAttachmentStore(tmpDir)
	ext := extractor.NewExtractor(store)

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Без календаря
Message-ID: <no-ical@example.com>
Content-Type: text/plain; charset=utf-8

Просто обычное письмо без событий.`)

	result, err := ext.Extract(raw)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if result.Calendar != "" {
		t.Errorf("Calendar should be empty for plain email, got: %q", result.Calendar)
	}
}

func TestCleaner_HTMLToText(t *testing.T) {
	c := extractor.NewCleaner()

	html := `<html><head><style>.x{color:red}</style></head>
<body>
<p>Первая строка</p>
<p>Вторая строка с <b>жирным</b> и <a href="http://example.com">ссылкой</a></p>
<ul><li>Пункт 1</li><li>Пункт 2</li></ul>
<script>console.log('test')</script>
</body></html>`

	text := c.HTMLToText(html)

	if !strings.Contains(text, "Первая строка") {
		t.Error("missing first line")
	}
	if !strings.Contains(text, "жирным") {
		t.Error("missing bold text")
	}
	if !strings.Contains(text, "ссылкой") {
		t.Error("missing link text")
	}
	if !strings.Contains(text, "Пункт 1") {
		t.Error("missing list item")
	}
	if strings.Contains(text, "console.log") {
		t.Error("script content should be removed")
	}
	if strings.Contains(text, ".x{color:red}") {
		t.Error("style content should be removed")
	}
}
