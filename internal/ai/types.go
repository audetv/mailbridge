// Package ai предоставляет клиент для взаимодействия с LLM.
package ai

import "context"

// Client определяет интерфейс для обращения к LLM.
type Client interface {
	// Generate отправляет промпт и возвращает ответ модели.
	Generate(ctx context.Context, prompt string, images []string) (string, error)
}

// ActiveTask — краткое описание активной задачи для контекста LLM.
type ActiveTask struct {
	TaskID   int    `json:"task_id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

// NewTaskData — данные для создания новой задачи.
type NewTaskData struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Priority      string `json:"priority"`
	Project       string `json:"project"`
	Type          string `json:"type"`
	SourceEmailID string `json:"source_email_id"`
	ImageNote     string `json:"image_note,omitempty"`
}

// TaskUpdates — поля для обновления существующей задачи.
type TaskUpdates struct {
	Priority     string `json:"priority,omitempty"`
	AddComment   string `json:"add_comment,omitempty"`
	ChangeStatus string `json:"change_status,omitempty"`
}

// Verdict — одно решение LLM по письму.
type Verdict struct {
	Action    string       `json:"action"` // "new", "update", "completed", "none"
	TaskID    *int         `json:"task_id,omitempty"`
	Task      *NewTaskData `json:"task,omitempty"`
	Updates   *TaskUpdates `json:"updates,omitempty"`
	Comment   string       `json:"comment,omitempty"`
	Summary   string       `json:"summary,omitempty"`
	ImageNote string       `json:"image_note,omitempty"`
}

// LLMResponse — ответ модели.
type LLMResponse struct {
	Verdicts []Verdict `json:"verdicts"`
}
