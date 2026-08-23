package preprocessor

import (
	"fmt"
	"os"
	"strings"
)

// extractICalText извлекает события из iCalendar файла.
func extractICalText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	content := string(data)
	var result strings.Builder

	lines := strings.Split(content, "\n")
	var currentEvent []string
	inEvent := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "BEGIN:VEVENT") {
			inEvent = true
			currentEvent = nil
			continue
		}

		if strings.HasPrefix(line, "END:VEVENT") {
			inEvent = false
			result.WriteString(formatICalEvent(currentEvent))
			result.WriteString("\n")
			continue
		}

		if inEvent {
			currentEvent = append(currentEvent, line)
		}
	}

	return result.String(), nil
}

// formatICalEvent форматирует одно событие в читаемый текст.
// formatICalEvent форматирует одно событие в читаемый текст.
func formatICalEvent(lines []string) string {
	fields := make(map[string]string)

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]
		if idx := strings.Index(key, ";"); idx != -1 {
			key = key[:idx]
		}
		fields[key] = value
	}

	var sb strings.Builder
	if summary := fields["SUMMARY"]; summary != "" {
		fmt.Fprintf(&sb, "[СОБЫТИЕ] %s\n", summary)
	}
	if start := fields["DTSTART"]; start != "" {
		fmt.Fprintf(&sb, "Начало: %s\n", formatICalDateTime(start))
	}
	if end := fields["DTEND"]; end != "" {
		fmt.Fprintf(&sb, "Конец: %s\n", formatICalDateTime(end))
	}
	if desc := fields["DESCRIPTION"]; desc != "" {
		fmt.Fprintf(&sb, "Описание: %s\n", desc)
	}
	if loc := fields["LOCATION"]; loc != "" {
		fmt.Fprintf(&sb, "Место: %s\n", loc)
	}
	if org := fields["ORGANIZER"]; org != "" {
		fmt.Fprintf(&sb, "Организатор: %s\n", org)
	}

	return sb.String()
}

// formatICalDateTime преобразует 20260825T140000Z в читаемый формат.
func formatICalDateTime(s string) string {
	if len(s) >= 15 {
		dateStr := s[0:8]
		timeStr := s[9:15]

		if len(dateStr) == 8 && len(timeStr) == 6 {
			year := dateStr[0:4]
			month := dateStr[4:6]
			day := dateStr[6:8]
			hour := timeStr[0:2]
			minute := timeStr[2:4]

			result := fmt.Sprintf("%s.%s.%s %s:%s", day, month, year, hour, minute)
			if strings.HasSuffix(s, "Z") {
				result += " UTC"
			}
			return result
		}
	}
	return s
}
