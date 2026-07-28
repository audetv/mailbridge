package classifier

import "strings"

// UrgencyBooster повышает приоритет при обнаружении срочных слов.
type UrgencyBooster struct {
	patterns []string
}

// NewUrgencyBooster создаёт новый UrgencyBooster с заданными паттернами.
func NewUrgencyBooster(patterns []string) *UrgencyBooster {
	return &UrgencyBooster{patterns: patterns}
}

// DefaultUrgencyPatterns возвращает стандартный набор срочных слов.
func DefaultUrgencyPatterns() []string {
	return []string{
		"срочно", "срочн", "горит", "упал", "не работает совсем",
		"клиент жалуется", "потеря денег", "прямо сейчас",
		"asap", "urgent", "critical", "всё сломалось",
		"ничего не работает", "пожар", "аврал",
	}
}

// Boost проверяет текст на наличие срочных паттернов.
// Если найдено — возвращает "urgent", иначе текущий приоритет.
func (b *UrgencyBooster) Boost(text string, currentPriority string) string {
	lower := strings.ToLower(text)

	for _, pattern := range b.patterns {
		if strings.Contains(lower, pattern) {
			return "urgent"
		}
	}

	return currentPriority
}
