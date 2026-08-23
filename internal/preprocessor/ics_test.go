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
