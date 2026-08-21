package preprocessor_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/preprocessor"
)

func TestProcessAttachment_Text(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(file, []byte("hello text"), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	result, err := p.ProcessAttachment(file)
	if err != nil {
		t.Fatalf("ProcessAttachment error: %v", err)
	}

	if result.Type != "text" {
		t.Errorf("Type = %s, want text", result.Type)
	}
	if result.Content != "hello text" {
		t.Errorf("Content = %s", result.Content)
	}
}

func TestProcessAttachment_Image(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "test.png")
	rawData := []byte{0x89, 0x50, 0x4E, 0x47, 0x01, 0x02}
	if err := os.WriteFile(file, rawData, 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	result, err := p.ProcessAttachment(file)
	if err != nil {
		t.Fatalf("ProcessAttachment error: %v", err)
	}

	if result.Type != "image" {
		t.Errorf("Type = %s, want image", result.Type)
	}

	// Проверяем что это валидный Base64
	decoded, err := base64.StdEncoding.DecodeString(result.Content)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}
	if string(decoded) != string(rawData) {
		t.Error("decoded data mismatch")
	}
}

func TestProcessAttachment_Unsupported(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "test.exe")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	_, err := p.ProcessAttachment(file)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestProcessAttachment_PDF_WithoutPoppler(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "test.pdf")
	if err := os.WriteFile(file, []byte("fake pdf"), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Ожидаем ошибку если poppler-utils не установлен
	result, err := p.ProcessAttachment(file)
	if err != nil {
		return // ожидаемо если pdftotext не установлен
	}
	if result == nil {
		t.Error("result is nil")
	}
}

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
