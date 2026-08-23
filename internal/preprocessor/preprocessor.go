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
// или Base64 для изображений. originalFilename используется для определения типа.
func (p *Preprocessor) ProcessAttachment(path string, originalFilename string) (*ProcessedAttachment, error) {
	// Используем оригинальное имя для определения типа
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(path))
	}

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

	case ".docx":
		text, err := extractDocxText(path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract docx: %w", err)
		}
		return &ProcessedAttachment{Type: "text", Content: text}, nil

	case ".xlsx":
		text, err := extractXlsxText(path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract xlsx: %w", err)
		}
		return &ProcessedAttachment{Type: "text", Content: text}, nil

	case ".pdf":
		return p.processPDF(path)

	case ".ics":
		text, err := extractICalText(path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract ics: %w", err)
		}
		return &ProcessedAttachment{Type: "text", Content: text}, nil

	case ".rtf":
		text, err := extractRtfText(path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract rtf: %w", err)
		}
		return &ProcessedAttachment{Type: "text", Content: text}, nil

	case ".pptx":
		text, err := extractPptxText(path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract pptx: %w", err)
		}
		return &ProcessedAttachment{Type: "text", Content: text}, nil

	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// ProcessedAttachment — результат обработки вложения.
type ProcessedAttachment struct {
	Type    string // "text" или "image"
	Content string // текст или Base64
}
