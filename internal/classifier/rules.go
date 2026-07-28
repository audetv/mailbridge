package classifier

import "github.com/audetv/mailbridge/internal/config"

// Rule описывает одно правило классификации.
type Rule struct {
	Keywords []string // фразы ключевых слов
	Project  string   // проект (опционально)
	Type     string   // тип задачи (опционально)
	Priority string   // приоритет (опционально)
	Weight   int      // вес правила (1-10)
}

// ConvertRules преобразует RuleDef из config в Rule классификатора.
func ConvertRules(defs []config.RuleDef) []Rule {
	rules := make([]Rule, len(defs))
	for i, d := range defs {
		rules[i] = Rule{
			Keywords: d.Keywords,
			Project:  d.Project,
			Type:     d.Type,
			Priority: d.Priority,
			Weight:   d.Weight,
		}
	}
	return rules
}
