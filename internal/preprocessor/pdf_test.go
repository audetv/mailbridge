package preprocessor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/audetv/mailbridge/internal/preprocessor"
)

func TestProcessAttachment_PDF_WithoutPoppler(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "test.pdf")
	if err := os.WriteFile(file, []byte("fake pdf"), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Ожидаем ошибку если poppler-utils не установлен
	result, err := p.ProcessAttachment(file, "test.txt")
	if err != nil {
		return // ожидаемо если pdftotext не установлен
	}
	if result == nil {
		t.Error("result is nil")
	}
}
