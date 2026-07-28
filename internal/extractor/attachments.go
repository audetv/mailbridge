package extractor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Attachment описывает вложение письма.
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	Data        []byte
	StoragePath string // путь после сохранения
}

// AttachmentStore сохраняет вложения в файловую систему.
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

// Save сохраняет вложение в директорию с датой.
// Возвращает путь относительно baseDir.
func (s *AttachmentStore) Save(att *Attachment) (string, error) {
	dateDir := time.Now().Format("2006-01-02")
	dir := filepath.Join(s.baseDir, dateDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create date dir: %w", err)
	}

	safeFilename := sanitizeFilename(att.Filename)
	path := filepath.Join(dir, safeFilename)

	// Если файл существует, добавляем timestamp
	if _, err := os.Stat(path); err == nil {
		ext := filepath.Ext(safeFilename)
		name := safeFilename[:len(safeFilename)-len(ext)]
		safeFilename = fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext)
		path = filepath.Join(dir, safeFilename)
	}

	if err := os.WriteFile(path, att.Data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write attachment: %w", err)
	}

	relPath, _ := filepath.Rel(s.baseDir, path)
	return relPath, nil
}

// BaseDir возвращает базовую директорию хранилища.
func (s *AttachmentStore) BaseDir() string {
	return s.baseDir
}

// sanitizeFilename удаляет опасные символы из имени файла.
func sanitizeFilename(name string) string {
	// Удаляем путь
	name = filepath.Base(name)

	// Заменяем небезопасные символы на подчёркивание
	reg := regexp.MustCompile(`[^a-zA-Z0-9а-яА-ЯёЁ_.-]`)
	name = reg.ReplaceAllString(name, "_")

	// Убираем множественные подчёркивания
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// Обрезаем до 255 символов
	if len(name) > 255 {
		ext := filepath.Ext(name)
		name = name[:255-len(ext)] + ext
	}

	if name == "" || name == "." || name == ".." {
		name = fmt.Sprintf("attachment_%d", time.Now().UnixNano())
	}

	return name
}

// CopyData копирует данные из reader в слайс байт.
func CopyData(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
