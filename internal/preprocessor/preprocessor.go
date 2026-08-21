// Package preprocessor подготавливает вложения писем для передачи в LLM.
package preprocessor

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nguyenthenguyen/docx"
	"github.com/xuri/excelize/v2"
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

	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// ProcessedAttachment — результат обработки вложения.
type ProcessedAttachment struct {
	Type    string // "text" или "image"
	Content string // текст или Base64
}

// extractDocxText извлекает текст из DOCX-файла.
// extractDocxText извлекает текст из DOCX-файла.
func extractDocxText(path string) (string, error) {
	doc, err := docx.ReadDocxFile(path)
	if err != nil {
		return "", err
	}
	defer doc.Close()

	return doc.Editable().GetContent(), nil
}

// extractXlsxText извлекает текст из XLSX-файла.
func extractXlsxText(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var text strings.Builder
	for _, sheet := range f.GetSheetList() {
		text.WriteString("=== Лист: " + sheet + " ===\n")
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for _, row := range rows {
			text.WriteString(strings.Join(row, " | "))
			text.WriteString("\n")
		}
	}
	return text.String(), nil
}
