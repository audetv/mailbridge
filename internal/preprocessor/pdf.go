package preprocessor

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// processPDF извлекает текст из PDF через pdftotext, при неудаче конвертирует в изображения.
func (p *Preprocessor) processPDF(path string) (*ProcessedAttachment, error) {
	// Пробуем pdftotext
	text, err := exec.Command("pdftotext", path, "-").Output()
	if err == nil && len(text) > 50 {
		return &ProcessedAttachment{
			Type:    "text",
			Content: truncateText(string(text)),
		}, nil
	}

	// Если текста мало (скан) — конвертируем в PNG через pdftoppm
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

	// Ограничиваем количество страниц (первые 5)
	const maxPages = 5
	if len(pages) > maxPages {
		pages = pages[:maxPages]
	}

	// Кодируем страницы в Base64
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
		Content: strings.Join(base64Pages, ","),
	}, nil
}
