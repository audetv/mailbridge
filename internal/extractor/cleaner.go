package extractor

import (
	"regexp"
	"strings"
)

// Cleaner очищает тело письма от истории переписки и подписей.
type Cleaner struct {
	quotePatterns []*regexp.Regexp
	sigSeparator  string
}

// NewCleaner создаёт новый Cleaner.
func NewCleaner() *Cleaner {
	return &Cleaner{
		quotePatterns: defaultQuotePatterns(),
		sigSeparator:  "-- ",
	}
}

// CleanBody очищает текст письма от цитируемой истории и подписей.
func (c *Cleaner) CleanBody(body string) string {
	body = c.removeQuotedHistory(body)
	body = c.removeSignatures(body)
	body = strings.TrimSpace(body)
	return body
}

// removeQuotedHistory удаляет пересланную/цитируемую историю.
func (c *Cleaner) removeQuotedHistory(text string) string {
	for _, pattern := range c.quotePatterns {
		loc := pattern.FindStringIndex(text)
		if loc != nil {
			text = strings.TrimSpace(text[:loc[0]])
			return text
		}
	}

	// Если есть строки с цитированием (начинаются с >),
	// но нет явного маркера — удаляем всё, начиная с последнего блока цитат
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">") && len(trimmed) > 1 {
			break
		}
		cleanLines = append(cleanLines, line)
	}

	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

// removeSignatures удаляет подпись (после "-- ").
func (c *Cleaner) removeSignatures(text string) string {
	// Стандартный разделитель "-- " (с пробелом)
	idx := strings.Index(text, "\n-- \n")
	if idx != -1 {
		return strings.TrimSpace(text[:idx])
	}

	// Вариант без пробела в конце текста
	if strings.HasSuffix(text, "\n-- ") {
		return strings.TrimSpace(text[:len(text)-4])
	}

	// Вариант "--" без пробела (нестандартный, но встречается)
	idx = strings.Index(text, "\n--\n")
	if idx != -1 {
		after := strings.TrimSpace(text[idx+4:])
		// Если после "--" идёт текст похожий на подпись (имя, должность)
		if strings.Contains(after, "\n") || len(after) < 100 {
			return strings.TrimSpace(text[:idx])
		}
	}

	return text
}

// defaultQuotePatterns возвращает паттерны для определения начала цитаты.
func defaultQuotePatterns() []*regexp.Regexp {
	patterns := []string{
		// Русские маркеры
		`\n?\s*---+.*Original Message.*---+`,
		`\n?\s*---+.*Пересылаемое сообщение.*---+`,
		`\n?\s*---+.*Исходное сообщение.*---+`,
		`\n\d{2}[./]\d{2}[./]\d{2,4}\s+\d{2}:\d{2}.*написал`,
		`\n\d{2}[./]\d{2}[./]\d{2,4}\s+\d{2}:\d{2}.*пользователь.*написал`,
		`\nOn\s+\w+,\s+\w+\s+\d+,\s+\d{4}.*wrote:`,
		`\nFrom:.*\nDate:.*\nSubject:.*\nTo:.*`,
		`\nFrom:.*\nSent:.*\nTo:.*\nSubject:.*`,
		`\n-+\s*Forwarded message\s*-+`,
		`\n_{10,}`,
		`\n\*{10,}`,
		`\n={10,}`,
	}

	var compiled []*regexp.Regexp
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}

	return compiled
}

// SanitizeHTML очищает HTML от пустых тегов, лишних br и схлопывает отступы.
func (c *Cleaner) SanitizeHTML(html string) string {
	if html == "" {
		return ""
	}

	// Удаляем пустые <p> теги (с пробелами, &nbsp;, <br>)
	emptyP := regexp.MustCompile(`(?i)<p[^>]*>(?:&nbsp;|\s|<br\s*/?>)*</p>`)
	html = emptyP.ReplaceAllString(html, "")

	// Схлопываем множественные <br> (больше 2 подряд)
	multiBr := regexp.MustCompile(`(?i)(<br\s*/?>\s*){3,}`)
	html = multiBr.ReplaceAllString(html, "<br><br>")

	// Удаляем пустые <div>
	emptyDiv := regexp.MustCompile(`(?i)<div[^>]*>(?:\s|&nbsp;)*</div>`)
	html = emptyDiv.ReplaceAllString(html, "")

	// Удаляем <base> теги — они меняют базовый URL страницы и ломают API-запросы
	baseTag := regexp.MustCompile(`(?i)<base[^>]*>`)
	html = baseTag.ReplaceAllString(html, "")

	// Удаляем множественные пустые строки
	html = regexp.MustCompile(`\n{3,}`).ReplaceAllString(html, "\n\n")

	return strings.TrimSpace(html)
}
