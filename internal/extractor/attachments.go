package extractor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Attachment описывает вложение письма.
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	Data        []byte
	StoragePath string
}

// AttachmentStore сохраняет вложения в content-addressable storage.
type AttachmentStore struct {
	baseDir string
}

// NewAttachmentStore создаёт новое хранилище вложений.
func NewAttachmentStore(baseDir string) (*AttachmentStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create attachment dir %s: %w", baseDir, err)
	}
	return &AttachmentStore{baseDir: baseDir}, nil
}

// Save сохраняет вложение и возвращает относительный путь.
// Путь: {hash[0:2]}/{hash[2:4]}/{hash}
func (s *AttachmentStore) Save(att *Attachment) (string, error) {
	// Вычисляем hash
	hash := computeHash(att.Data)

	// Строим путь
	relPath := filepath.Join(hash[0:2], hash[2:4], hash)

	fullPath := filepath.Join(s.baseDir, relPath)

	// Если файл уже существует — не перезаписываем
	if _, err := os.Stat(fullPath); err == nil {
		return relPath, nil
	}

	// Создаём директории
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create dir: %w", err)
	}

	// Сохраняем
	if err := os.WriteFile(fullPath, att.Data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return relPath, nil
}

// BaseDir возвращает базовую директорию хранилища.
func (s *AttachmentStore) BaseDir() string {
	return s.baseDir
}

// computeHash вычисляет SHA-256 данных.
func computeHash(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

// CopyData копирует данные из reader в слайс байт.
func CopyData(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
