package attachments_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/audetv/mailbridge/internal/attachments"
)

func TestComputeHash(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(file, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	hash, err := attachments.ComputeHash(file)
	if err != nil {
		t.Fatalf("ComputeHash error: %v", err)
	}

	// SHA-256 "hello world" = b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("hash = %s, want %s", hash, expected)
	}
}

func TestHashToPath(t *testing.T) {
	hash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	path := attachments.HashToPath(hash)

	expected := filepath.Join("b9", "4d", hash)
	if path != expected {
		t.Errorf("path = %s, want %s", path, expected)
	}
}
