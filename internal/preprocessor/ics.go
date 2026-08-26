// Package preprocessor подготавливает вложения писем для передачи в LLM.
package preprocessor

import (
	"fmt"
	"os"
	"strings"
)

// unfoldICal выполняет unfolding строк по RFC 5545 §3.1:
// строка, начинающаяся с пробела или TAB, является продолжением предыдущей
// (ведущий символ-пробел отбрасывается). CRLF нормализуется в LF.
func unfoldICal(data []byte) []string {
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	var out []string
	for i, line := range lines {
		if i > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			if len(out) > 0 {
				out[len(out)-1] += line[1:]
				continue
			}
		}
		out = append(out, line)
	}
	return out
}

// icalProp — свойство iCalendar: имя, параметры и значение.
type icalProp struct {
	name   string
	params map[string]string
	value  string
}

// parseICalProp разбирает строку вида "NAME;p1=v1;p2:VALUE" (уже unfolded).
func parseICalProp(line string) (icalProp, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return icalProp{}, false
	}

	namepart := line[:idx]
	value := unescapeICalValue(line[idx+1:])

	name := namepart
	params := map[string]string{}
	if semi := strings.Index(namepart, ";"); semi != -1 {
		name = namepart[:semi]
		for _, p := range strings.Split(namepart[semi+1:], ";") {
			if eq := strings.Index(p, "="); eq > 0 {
				params[p[:eq]] = strings.Trim(p[eq+1:], `"`)
			}
		}
	}

	return icalProp{name: strings.ToUpper(name), params: params, value: value}, true
}

// unescapeICalValue декодирует экранирования RFC 5545 §3.3.11 (\n, \r\n, \;, \,, \\, \").
func unescapeICalValue(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n', 'N':
				b.WriteByte('\n')
			case 'r', 'R':
				// \r\n схлопывается в один перевод строки (\n обработаем на след. итерации)
			case ';', ',', '\\', '"':
				b.WriteByte(s[i+1])
			default:
				b.WriteByte(s[i])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// ExtractICal извлекает события из iCalendar-данных (RFC 5545) в читаемый текст.
func ExtractICal(data []byte) string {
	return extractICalFromData(data)
}

// extractICalText извлекает события из iCalendar-файла (legacy API по пути).
func extractICalText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return extractICalFromData(data), nil
}

// extractICalFromData — ядро парсера: unfolding + сборка VEVENT-блоков.
func extractICalFromData(data []byte) string {
	lines := unfoldICal(data)

	var result strings.Builder
	var currentEvent []string
	inEvent := false
	method := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// METHOD живёт на уровне VCALENDAR (встречается до BEGIN:VEVENT)
		if strings.HasPrefix(line, "METHOD:") {
			method = strings.TrimSpace(strings.TrimPrefix(line, "METHOD:"))
			continue
		}

		if strings.HasPrefix(line, "BEGIN:VEVENT") {
			inEvent = true
			currentEvent = nil
			continue
		}

		if strings.HasPrefix(line, "END:VEVENT") {
			inEvent = false
			result.WriteString(formatICalEvent(currentEvent, method))
			result.WriteString("\n")
			continue
		}

		if inEvent {
			currentEvent = append(currentEvent, line)
		}
	}

	return result.String()
}

// formatICalEvent форматирует одно VEVENT-событие в читаемый текст.
func formatICalEvent(lines []string, method string) string {
	// Разбираем свойства, пропуская вложенные суб-компоненты (VALARM и пр.)
	var props []icalProp
	subdepth := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "BEGIN:") {
			subdepth++
			continue
		}
		if strings.HasPrefix(line, "END:") {
			if subdepth > 0 {
				subdepth--
			}
			continue
		}
		if subdepth > 0 {
			continue // содержимое VALARM/прочих суб-компонентов не наше дело
		}
		if p, ok := parseICalProp(line); ok {
			props = append(props, p)
		}
	}

	get := func(name string) (string, bool) {
		for _, p := range props {
			if p.name == name {
				return p.value, true
			}
		}
		return "", false
	}
	tzidOf := func(name string) string {
		for _, p := range props {
			if p.name == name {
				return p.params["TZID"]
			}
		}
		return ""
	}

	var sb strings.Builder
	if summary, ok := get("SUMMARY"); ok && strings.TrimSpace(summary) != "" {
		fmt.Fprintf(&sb, "[СОБЫТИЕ] %s\n", strings.TrimSpace(summary))
	}
	if method != "" {
		fmt.Fprintf(&sb, "Метод: %s (REQUEST — приглашение, CANCEL — отмена встречи)\n", method)
	}
	if start, ok := get("DTSTART"); ok && start != "" {
		fmt.Fprintf(&sb, "Начало: %s\n", formatICalDateTime(start, tzidOf("DTSTART")))
	}
	if end, ok := get("DTEND"); ok && end != "" {
		fmt.Fprintf(&sb, "Конец: %s\n", formatICalDateTime(end, tzidOf("DTEND")))
	}
	if desc, ok := get("DESCRIPTION"); ok && strings.TrimSpace(desc) != "" {
		fmt.Fprintf(&sb, "Описание: %s\n", strings.TrimSpace(desc))
	}
	if loc, ok := get("LOCATION"); ok && loc != "" {
		fmt.Fprintf(&sb, "Место: %s\n", loc)
	}
	if org, ok := get("ORGANIZER"); ok && org != "" {
		// ORGANIZER;CN=«Имя» с EMAIL-значением — людям имя читабельнее адреса
		for _, p := range props {
			if p.name == "ORGANIZER" {
				if cn, has := p.params["CN"]; has && cn != "" {
					org = cn
				}
				break
			}
		}
		fmt.Fprintf(&sb, "Организатор: %s\n", org)
	}

	// Участники: ATTENDEE может повторяться (берём CN, если есть)
	var attendees []string
	for _, p := range props {
		if p.name != "ATTENDEE" {
			continue
		}
		name := ""
		if cn, has := p.params["CN"]; has && cn != "" {
			name = cn
		} else {
			name = strings.TrimSpace(p.value)
		}
		if name != "" {
			attendees = append(attendees, name)
		}
	}
	if len(attendees) > 0 {
		fmt.Fprintf(&sb, "Участники: %s\n", strings.Join(attendees, ", "))
	}

	return sb.String()
}

// formatICalDateTime преобразует 20260825T140000Z в читаемый формат;
// при наличии TZID-параметра добавляет название часового пояса.
func formatICalDateTime(s, tzid string) string {
	if len(s) >= 15 {
		dateStr := s[0:8]
		timeStr := s[9:15]
		if len(dateStr) == 8 && len(timeStr) == 6 {
			result := fmt.Sprintf("%s.%s.%s %s:%s", dateStr[6:8], dateStr[4:6], dateStr[0:4], timeStr[0:2], timeStr[2:4])
			if strings.HasSuffix(s, "Z") {
				result += " UTC"
			} else if tzid != "" {
				result += " (" + tzid + ")"
			}
			return result
		}
	}
	return s
}
