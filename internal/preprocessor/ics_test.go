package preprocessor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/preprocessor"
)

func TestExtractICalText(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "invite.ics")
	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Встреча по проекту
DTSTART:20260825T140000Z
DTEND:20260825T150000Z
DESCRIPTION:Обсуждение плана
LOCATION:Зум
ORGANIZER:mailto:user@example.com
END:VEVENT
END:VCALENDAR`

	if err := os.WriteFile(file, []byte(icsContent), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	result, err := p.ProcessAttachment(file, "invite.ics")
	if err != nil {
		t.Fatalf("ProcessAttachment error: %v", err)
	}

	if result.Type != "text" {
		t.Errorf("Type = %s, want text", result.Type)
	}
	if !strings.Contains(result.Content, "Встреча по проекту") {
		t.Error("Content does not contain summary")
	}
	if !strings.Contains(result.Content, "25.08.2026 14:00") {
		t.Error("Content does not contain formatted date")
	}
	if !strings.Contains(result.Content, "Зум") {
		t.Error("Content does not contain location")
	}
}

// TestExtractICal_ExchangeFolded — регрессия RFC 5545 unfolding:
// реальные Exchange-письма сгибаруют длинные строки (CRLF + пробел).
// Без unfolding ORGANIZER обрывался на «M», ATTENDEE терялись.
func TestExtractICal_ExchangeFolded(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\n" +
		"METHOD:REQUEST\r\n" +
		"VERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\n" +
		"ORGANIZER;CN=Бухтина Ан\r\n" +
		" астасия:MAILTO:abuhtina@example.ru\r\n" +
		"ATTENDEE;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;CN=Кузьм\r\n" +
		" ина Полина:MAILTO:pkuzmina@example.ru\r\n" +
		"SUMMARY;LANGUAGE=ru-RU:онлай\r\n" +
		" н продажи\r\n" +
		"DTSTART;TZID=Russian Standard Time:20260826T143000\r\n" +
		"DTEND;TZID=Russian Standard Time:20260826T150000\r\n" +
		"LOCATION;LANGUAGE=ru-RU:у Алексе\r\n" +
		" я\r\n" +
		"BEGIN:VALARM\r\n" +
		"DESCRIPTION:REMINDER\r\n" +
		"TRIGGER;RELATED=START:-PT15M\r\n" +
		"ACTION:DISPLAY\r\n" +
		"END:VALARM\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR"

	text := preprocessor.ExtractICal([]byte(ics))

	// Сгнутые строки должны быть склеены целиком
	if !strings.Contains(text, "Бухтина Анастасия") {
		t.Errorf("unfolding: ORGANIZER name broken or lost: %q", text)
	}
	if !strings.Contains(text, "Кузьмина Полина") {
		t.Errorf("unfolding: ATTENDEE lost or broken: %q", text)
	}
	if !strings.Contains(text, "онлайн продажи") {
		t.Errorf("unfolding: SUMMARY broken: %q", text)
	}
	if !strings.Contains(text, "у Алексея") {
		t.Errorf("unfolding: LOCATION broken: %q", text)
	}
	// Метод события и его участники должны присутствовать
	if !strings.Contains(text, "Метод: REQUEST") {
		t.Errorf("METHOD not surfaced: %q", text)
	}
	if !strings.Contains(text, "Участники:") {
		t.Errorf("ATTENDEEs missing: %q", text)
	}
	// VALARM не должен затирать DESCRIPTION события (здесь DESCRIPTION пуст)
	if strings.Contains(text, "REMINDER") {
		t.Errorf("VALARM content leaked into event: %q", text)
	}
	// TZID должен сохраниться в дате
	if !strings.Contains(text, "Russian Standard Time") {
		t.Errorf("TZID lost in DTSTART: %q", text)
	}
}

// TestExtractICal_MultipleEvents — несколько VEVENT-блоков в одном календаре.
func TestExtractICal_MultipleEvents(t *testing.T) {
	ics := "BEGIN:VCALENDAR\n" +
		"BEGIN:VEVENT\nSUMMARY:Встреча 1\nDTSTART:20260826T100000Z\nEND:VEVENT\n" +
		"BEGIN:VEVENT\nSUMMARY:Встреча 2\nDTSTART:20260826T120000Z\nEND:VEVENT\n" +
		"END:VCALENDAR"

	text := preprocessor.ExtractICal([]byte(ics))

	if strings.Count(text, "[СОБЫТИЕ]") != 2 {
		t.Errorf("expected 2 events, got %d: %q", strings.Count(text, "[СОБЫТИЕ]"), text)
	}
	if !strings.Contains(text, "Встреча 1") || !strings.Contains(text, "Встреча 2") {
		t.Errorf("not all events present: %q", text)
	}
}

// TestExtractICal_EscapedChars — экранированные символы RFC 5545 §3.3.11.
func TestExtractICal_EscapedChars(t *testing.T) {
	ics := "BEGIN:VCALENDAR\n" +
		"BEGIN:VEVENT\n" +
		"SUMMARY:Обед; заказ\\, кофе\n" +
		"DESCRIPTION:Строка 1\\nСтрока 2\n" +
		"END:VEVENT\nEND:VCALENDAR"

	text := preprocessor.ExtractICal([]byte(ics))

	if !strings.Contains(text, "Обед; заказ, кофе") {
		t.Errorf("escaped comma/semicolon not unescaped: %q", text)
	}
	if !strings.Contains(text, "Строка 1\nСтрока 2") {
		t.Errorf("escaped newline not unescaped: %q", text)
	}
}
