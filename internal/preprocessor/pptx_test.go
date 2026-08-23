package preprocessor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/audetv/mailbridge/internal/preprocessor"
)

func TestExtractPptxText_InvalidFile(t *testing.T) {
	p := preprocessor.NewPreprocessor()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "test.pptx")
	if err := os.WriteFile(file, []byte("not a pptx"), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	_, err := p.ProcessAttachment(file, "test.pptx")
	if err == nil {
		t.Fatal("expected error for invalid pptx")
	}
}
