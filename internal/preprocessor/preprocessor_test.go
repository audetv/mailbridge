package preprocessor_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
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
