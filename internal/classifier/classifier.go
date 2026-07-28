// Package classifier предоставляет классификацию текста обращений:
// определение проекта, типа задачи и приоритета.
package classifier

import "context"

// Classification содержит результат классификации.
type Classification struct {
	Project     string  `json:"project"`
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Confidence  float64 `json:"confidence"`
	NeedsTriage bool    `json:"needs_triage"`
}

// Classifier определяет интерфейс классификатора.
type Classifier interface {
	Classify(ctx context.Context, text string, projects, types []string) (*Classification, error)
}
