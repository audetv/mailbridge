package classifier

import (
	"context"

	"github.com/audetv/mailbridge/internal/parser"
)

// RuleBasedClassifier реализует классификацию на основе правил.
type RuleBasedClassifier struct {
	matcher *Matcher
	booster *UrgencyBooster
	parser  *parser.FieldParser
}

// NewRuleBasedClassifier создаёт новый RuleBasedClassifier.
func NewRuleBasedClassifier(rules []Rule) *RuleBasedClassifier {
	return &RuleBasedClassifier{
		matcher: NewMatcher(rules),
		booster: NewUrgencyBooster(DefaultUrgencyPatterns()),
		parser:  parser.NewFieldParser(),
	}
}

// Classify классифицирует текст письма.
func (c *RuleBasedClassifier) Classify(_ context.Context, text string, _, _ []string) (*Classification, error) {
	// Шаг 1: пробуем распарсить явные поля
	fields := c.parser.Parse(text)
	if fields.HasFields {
		result := &Classification{
			Project:    fields.Project,
			Type:       fields.Type,
			Priority:   fields.Priority,
			Confidence: 0.95,
		}
		// Заполняем недостающие поля из правил, если есть тело после полей
		if fields.Body != "" {
			c.fillMissingFields(fields.Body, result)
		}
		// Проверяем urgency по исходному тексту
		result.Priority = c.booster.Boost(text, result.Priority)
		if result.Priority == "" {
			result.Priority = "medium"
		}
		return result, nil
	}

	// Шаг 2: классификация по правилам
	result := &Classification{}
	c.fillFromRules(text, result)

	// Шаг 3: urgency booster
	result.Priority = c.booster.Boost(text, result.Priority)

	// Шаг 4: значения по умолчанию и проверка необходимости триажа
	if result.Project == "" {
		result.NeedsTriage = true
		result.Confidence = 0.0
	} else if result.Confidence < 0.4 {
		result.NeedsTriage = true
	}

	if result.Priority == "" {
		result.Priority = "medium"
	}

	return result, nil
}

// fillMissingFields заполняет только пустые поля из правил, не трогая confidence.
func (c *RuleBasedClassifier) fillMissingFields(text string, result *Classification) {
	matches := c.matcher.Match(text)
	if len(matches) == 0 {
		return
	}

	for _, match := range matches {
		if result.Project == "" && match.Rule.Project != "" {
			result.Project = match.Rule.Project
		}
		if result.Type == "" && match.Rule.Type != "" {
			result.Type = match.Rule.Type
		}
		if result.Priority == "" && match.Rule.Priority != "" {
			result.Priority = match.Rule.Priority
		}
		if result.Project != "" && result.Type != "" && result.Priority != "" {
			break
		}
	}
}

// fillFromRules заполняет поля классификации из совпадений правил.
func (c *RuleBasedClassifier) fillFromRules(text string, result *Classification) {
	matches := c.matcher.Match(text)
	if len(matches) == 0 {
		return
	}

	bestScore := matches[0].Score
	if bestScore > 0 {
		result.Confidence = float64(bestScore) / float64(bestScore+3)
		if result.Confidence > 1.0 {
			result.Confidence = 1.0
		}
	}

	for _, match := range matches {
		if result.Project == "" && match.Rule.Project != "" {
			result.Project = match.Rule.Project
		}
		if result.Type == "" && match.Rule.Type != "" {
			result.Type = match.Rule.Type
		}
		if result.Priority == "" && match.Rule.Priority != "" {
			result.Priority = match.Rule.Priority
		}
		if result.Project != "" && result.Type != "" && result.Priority != "" {
			break
		}
	}
}
