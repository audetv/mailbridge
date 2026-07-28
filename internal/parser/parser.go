// Package parser извлекает структурированные поля из тела email-сообщения.
// Поддерживаются русские и английские названия ключей.
package parser

import (
	"regexp"
	"strings"
)

// ParsedFields содержит результат парсинга полей из тела письма.
type ParsedFields struct {
	// Project — название проекта (например, "ТРК", "Отель").
	Project string
	// Type — тип задачи: bug, feature, support, access, seo, content.
	Type string
	// Priority — приоритет: urgent, high, medium, low.
	Priority string
	// Deadline — срок выполнения в свободном формате.
	Deadline string
	// Assignee — предполагаемый исполнитель.
	Assignee string
	// Body — текст письма без строк с извлечёнными полями.
	Body string
	// HasFields — true, если был найден хотя бы один ключ.
	HasFields bool
}

// FieldParser извлекает структурированные поля из текста.
type FieldParser struct {
	keyPattern *regexp.Regexp
	validTypes map[string]bool
	validPrios map[string]bool
}

// fieldTarget связывает ключ с указателем на поле в ParsedFields.
type fieldTarget struct {
	ptr     *string
	allowed map[string]bool
}

// NewFieldParser создаёт новый FieldParser с заданными допустимыми значениями.
func NewFieldParser(validTypes, validPriorities map[string]bool) *FieldParser {
	return &FieldParser{
		keyPattern: regexp.MustCompile(`^([a-zA-Zа-яА-ЯёЁ]+)\s*:\s*(.+)$`),
		validTypes: validTypes,
		validPrios: validPriorities,
	}
}

// Parse разбирает текст и возвращает извлечённые поля.
func (p *FieldParser) Parse(body string) *ParsedFields {
	result := &ParsedFields{Body: body}

	keyMap := make(map[string]*fieldTarget)

	projectTarget := &fieldTarget{ptr: &result.Project}
	for _, k := range []string{"проект", "project"} {
		keyMap[k] = projectTarget
	}

	typeTarget := &fieldTarget{ptr: &result.Type, allowed: p.validTypes}
	for _, k := range []string{"тип", "type"} {
		keyMap[k] = typeTarget
	}

	prioTarget := &fieldTarget{ptr: &result.Priority, allowed: p.validPrios}
	for _, k := range []string{"приоритет", "priority"} {
		keyMap[k] = prioTarget
	}

	deadlineTarget := &fieldTarget{ptr: &result.Deadline}
	for _, k := range []string{"дедлайн", "deadline"} {
		keyMap[k] = deadlineTarget
	}

	assigneeTarget := &fieldTarget{ptr: &result.Assignee}
	for _, k := range []string{"исполнитель", "assignee"} {
		keyMap[k] = assigneeTarget
	}

	lines := strings.Split(body, "\n")
	var remaining []string
	headerEnded := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !headerEnded {
			if trimmed == "" {
				headerEnded = true
				continue
			}

			matches := p.keyPattern.FindStringSubmatch(trimmed)
			if matches != nil {
				key := strings.ToLower(matches[1])
				value := strings.TrimSpace(matches[2])

				if target, ok := keyMap[key]; ok && *target.ptr == "" {
					if target.allowed != nil {
						normalized := strings.ToLower(value)
						if !target.allowed[normalized] {
							remaining = append(remaining, line)
							result.HasFields = true
							continue
						}
					}
					*target.ptr = value
					result.HasFields = true
					continue
				}
			}

			if trimmed != "" {
				headerEnded = true
			}
		}

		remaining = append(remaining, line)
	}

	if result.HasFields {
		bodyText := strings.TrimSpace(strings.Join(remaining, "\n"))
		result.Body = bodyText
	}

	result.Priority = normalizeCase(result.Priority)

	return result
}

// normalizeCase приводит строку к формату "First letter uppercase, rest lowercase".
func normalizeCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) == 1 {
		return strings.ToUpper(string(runes[0]))
	}
	return strings.ToUpper(string(runes[0])) + strings.ToLower(string(runes[1:]))
}
