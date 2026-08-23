package preprocessor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// extractRtfText извлекает текст из RTF-файла.
// Использует внешнюю утилиту unrtf, при отсутствии — простой парсер.
func extractRtfText(path string) (string, error) {
	// Пробуем unrtf (более качественное извлечение)
	if text, err := extractRtfWithUnrtf(path); err == nil && len(text) > 0 {
		return text, nil
	}

	// Fallback: простой парсер RTF
	return extractRtfSimple(path)
}

// extractRtfWithUnrtf использует внешнюю утилиту unrtf.
func extractRtfWithUnrtf(path string) (string, error) {
	output, err := exec.Command("unrtf", "--text", path).Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// extractRtfSimple — простой парсер RTF без внешних зависимостей.
func extractRtfSimple(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	content := string(data)
	var result strings.Builder

	// Убираем RTF-теги и извлекаем текст
	inGroup := 0
	i := 0

	for i < len(content) {
		c := content[i]

		switch c {
		case '{':
			inGroup++
			i++
		case '}':
			inGroup--
			i++
		case '\\':
			// Обрабатываем RTF-команду
			i++
			// Пропускаем буквы команды
			for i < len(content) && isLetter(content[i]) {
				i++
			}
			// Если команда с параметром — пропускаем число и пробел
			if i < len(content) && content[i] == '-' {
				i++
			}
			for i < len(content) && isDigit(content[i]) {
				i++
			}
			// Пропускаем пробел после команды
			if i < len(content) && content[i] == ' ' {
				i++
			}

			// Специальные команды
			if i+1 < len(content) && content[i] == '\\' {
				switch content[i+1] {
				case 'n', 'N': // \n — новая строка
					result.WriteString("\n")
					i += 2
				case 't', 'T': // \t — таб
					result.WriteString("\t")
					i += 2
				case 'p', 'P': // \p — абзац
					result.WriteString("\n")
					i += 2
				default:
					i++
				}
			}
		default:
			// Обычный символ
			if inGroup > 0 {
				result.WriteByte(c)
			}
			i++
		}
	}

	return result.String(), nil
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// ensure non-empty return
var _ = fmt.Sprintf
