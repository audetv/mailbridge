package preprocessor_test

import (
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/preprocessor"
)

func TestMarkForwarded(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	body := `Сделайте это.

---------- Forwarded message ---------
От: Клиент
Тема: Важная задача`

	result := p.MarkForwarded(body)

	if !strings.Contains(result, "[ВНИМАНИЕ: Это пересланное сообщение") {
		t.Error("marker not inserted")
	}
	if !strings.Contains(result, "---------- Forwarded message ---------") {
		t.Error("original forwarded text not preserved")
	}
}

func TestMarkForwarded_NoMarker(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	body := "Обычное письмо без пересылки"
	result := p.MarkForwarded(body)

	if result != body {
		t.Errorf("body changed: %q", result)
	}
}
