// Package preprocessor подготавливает вложения писем для передачи в LLM.
package preprocessor

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
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

// processPDF извлекает текст из PDF через pdftotext, при неудаче конвертирует в изображения.
func (p *Preprocessor) processPDF(path string) (*ProcessedAttachment, error) {
	// Пробуем pdftotext
	text, err := exec.Command("pdftotext", path, "-").Output()
	if err == nil && len(text) > 50 {
		return &ProcessedAttachment{Type: "text", Content: string(text)}, nil
	}

	// Если текста мало — конвертируем в PNG через pdftoppm
	tmpDir, err := os.MkdirTemp("", "pdf-pages-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPrefix := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-r", "150", path, outputPrefix)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm failed: %w", err)
	}

	// Собираем все сгенерированные PNG
	pages, err := filepath.Glob(outputPrefix + "*.png")
	if err != nil || len(pages) == 0 {
		return nil, fmt.Errorf("no pages generated")
	}

	// Кодируем все страницы в Base64 (объединяем через разделитель)
	var base64Pages []string
	for _, page := range pages {
		data, err := os.ReadFile(page)
		if err != nil {
			continue
		}
		base64Pages = append(base64Pages, base64.StdEncoding.EncodeToString(data))
	}

	return &ProcessedAttachment{
		Type:    "image",
		Content: strings.Join(base64Pages, ","), // разделитель для нескольких страниц
	}, nil
}

// MarkForwarded добавляет маркер перед пересланным сообщением.
// Не вырезает текст, а помечает его для LLM.
func (p *Preprocessor) MarkForwarded(body string) string {
	forwardMarkers := []string{
		"---------- Forwarded message ---------",
		"--- Пересланное сообщение ---",
		"--- Пересылаемое сообщение ---",
	}

	for _, marker := range forwardMarkers {
		idx := strings.Index(body, marker)
		if idx != -1 {
			body = body[:idx] +
				"\n\n[ВНИМАНИЕ: Это пересланное сообщение. Контекст оригинальной переписки находится ниже. Проанализируй его на наличие задач.]\n\n" +
				body[idx:]
			break
		}
	}

	return body
}
