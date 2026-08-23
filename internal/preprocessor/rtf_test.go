package preprocessor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/preprocessor"
)

func TestExtractRtfText(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "doc.rtf")
	rtfContent := `{\rtf1\ansi\deff0
{\fonttbl{\f0 Times New Roman;}}
\f0\fs24 Привет, это тестовый документ.\par
Вторая строка текста.\par
}`

	if err := os.WriteFile(file, []byte(rtfContent), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	result, err := p.ProcessAttachment(file, "doc.rtf")
	if err != nil {
		t.Fatalf("ProcessAttachment error: %v", err)
	}

	if result.Type != "text" {
		t.Errorf("Type = %s, want text", result.Type)
	}
	if !strings.Contains(result.Content, "Привет") {
		t.Error("Content does not contain extracted text")
	}
}
