package preprocessor

import (
	"os"
	"strings"
	"testing"
)

// TestRealICalFromEmL — приёмочный тест: реальный ICS из data/emails/1.eml,
// на котором баг проявился (пустое тело в AI). Файл извлечён и лежит в /tmp.
func TestRealICalFromEmL(t *testing.T) {
	data, err := os.ReadFile("testdata/real_invite.ics")
	if err != nil {
		t.Skipf("no testdata/real_invite.ics (skip): %v", err)
	}

	text := ExtractICal(data)
	t.Logf("PARSED OUTPUT:\n%s", text)

	expectations := []string{
		"[СОБЫТИЕ]",
		"онлайн продажи",                          // SUMMARY
		"26.08.2026 14:30",                        // DTSTART
		"Russian Standard Time",                   // TZID
		"у Алексея",                               // LOCATION — полное, без обрыва
		"Бухтина Анастасия Александровна",         // ORGANIZER — полный CN (unfolding)
		"Кузьмина Полина Валерьевна",              // ATTENDEE 1 — полный (unfolding)
		"Гусев Алексей",                           // ATTENDEE 2
		"Метод: REQUEST",                          // METHOD
	}
	for _, e := range expectations {
		if !strings.Contains(text, e) {
			t.Errorf("expected %q in parsed real ICS, got:\n%s", e, text)
		}
	}

	// REGRESSION: старые дефекты, которые должны исчезнуть
	if strings.Contains(text, "Место: у Алексей\n") && !strings.Contains(text, "у Алексея") {
		t.Error("LOCATION was truncated (missing final byte)")
	}
	for _, b := range []string{"ORGANIZER;CN=", "ATTENDEE;ROLE="} {
		if strings.Contains(text, b) {
			t.Errorf("raw iCalendar property %q leaked into output (should be decoded): %q", b, text)
		}
	}
}
