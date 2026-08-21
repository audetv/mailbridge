// Package preprocessor подготавливает вложения писем для передачи в LLM.
package preprocessor

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Preprocessor обрабатывает вложения писем.
type Preprocessor struct{}

// NewPreprocessor создаёт новый Preprocessor.
func NewPreprocessor() *Preprocessor {
	return &Preprocessor{}
}

// ProcessAttachment обрабатывает один файл и возвращает текстовое представление
// или Base64 для изображений.
func (p *Preprocessor) ProcessAttachment(path string) (*ProcessedAttachment, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".txt", ".csv", ".md", ".log":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		return &ProcessedAttachment{
			Type:    "text",
			Content: string(data),
		}, nil

	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read image: %w", err)
		}
		return &ProcessedAttachment{
			Type:    "image",
			Content: base64.StdEncoding.EncodeToString(data),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// ProcessedAttachment — результат обработки вложения.
type ProcessedAttachment struct {
	Type    string // "text" или "image"
	Content string // текст или Base64
}
