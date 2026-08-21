package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
)

// Orchestrator координирует обработку писем через LLM.
type Orchestrator struct {
	client Client
	store  store.Store
}

// NewOrchestrator создаёт новый Orchestrator.
func NewOrchestrator(client Client, st store.Store) *Orchestrator {
	return &Orchestrator{
		client: client,
		store:  st,
	}
}

// ProcessEmail обрабатывает новое письмо через LLM и возвращает вердикты.
func (o *Orchestrator) ProcessEmail(ctx context.Context, email *extractor.ExtractedEmail) (*LLMResponse, error) {
	threadID := determineThreadID(email)

	thread, err := o.store.GetThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	summary := ""
	if thread != nil {
		summary = thread.Summary
	} else {
		if err := o.store.CreateThread(ctx, &store.Thread{ThreadID: threadID}); err != nil {
			return nil, fmt.Errorf("failed to create thread: %w", err)
		}
	}

	activeTasks, err := o.store.GetActiveTasksByThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks: %w", err)
	}

	prompt := o.buildPrompt(summary, activeTasks, email)

	response, err := o.client.Generate(ctx, prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM generate failed: %w", err)
	}

	return o.parseResponse(response)
}

// ParseResponse — экспортируемая обёртка для тестирования.
func (o *Orchestrator) ParseResponse(response string) (*LLMResponse, error) {
	return o.parseResponse(response)
}

// BuildPrompt — экспортируемая обёртка для тестирования.
func (o *Orchestrator) BuildPrompt(summary string, activeTasks []*store.Task, email *extractor.ExtractedEmail) string {
	return o.buildPrompt(summary, activeTasks, email)
}

// determineThreadID определяет ID цепочки по References или Message-ID.
func determineThreadID(email *extractor.ExtractedEmail) string {
	if len(email.References) > 0 {
		return email.References[0]
	}
	return email.MessageID
}

// buildPrompt формирует промпт для LLM.
func (o *Orchestrator) buildPrompt(summary string, activeTasks []*store.Task, email *extractor.ExtractedEmail) string {
	var sb strings.Builder

	sb.WriteString("Ты — интеллектуальный ассистент для управления задачами.\n\n")
	sb.WriteString("Проанализируй НОВОЕ письмо в контексте цепочки и определи действия.\n\n")

	if summary != "" {
		sb.WriteString("=== РЕЗЮМЕ ЦЕПОЧКИ ===\n")
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	}

	if len(activeTasks) > 0 {
		sb.WriteString("=== АКТИВНЫЕ ЗАДАЧИ ===\n")
		for _, task := range activeTasks {
			fmt.Fprintf(&sb, "- Task #%d: %s (статус: %s, приоритет: %s)\n",
				task.ID, task.Subject, task.Status, task.Priority)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("=== НОВОЕ ПИСЬМО ===\n")
	fmt.Fprintf(&sb, "От: %s\n", email.From)
	fmt.Fprintf(&sb, "Тема: %s\n", email.Subject)
	sb.WriteString("\n")
	sb.WriteString(email.BodyText)
	sb.WriteString("\n\n")

	sb.WriteString(`ОТВЕТЬ СТРОГО В JSON-формате:
{
  "verdicts": [
    {
      "action": "new",
      "task": {
        "title": "...",
        "description": "...",
        "priority": "high|medium|low",
        "project": "Название проекта",
        "type": "bug|feature|support|access|seo|content",
        "source_email_id": "message-id"
      }
    },
    {
      "action": "update",
      "task_id": 42,
      "updates": {
        "priority": "urgent",
        "add_comment": "Комментарий",
        "change_status": "in_progress"
      }
    },
    {
      "action": "completed",
      "task_id": 42,
      "comment": "Автор подтвердил"
    },
    {
      "action": "info_only",
      "summary": "Краткое резюме"
    }
  ]
}`)

	return sb.String()
}

// parseResponse парсит JSON-ответ LLM.
func (o *Orchestrator) parseResponse(response string) (*LLMResponse, error) {
	// Очищаем возможные markdown-обёртки
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result LLMResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}

// ApplyVerdicts применяет решения LLM к БД.
func (o *Orchestrator) ApplyVerdicts(ctx context.Context, email *extractor.ExtractedEmail, response *LLMResponse) error {
	if response == nil || len(response.Verdicts) == 0 {
		return nil
	}

	for _, verdict := range response.Verdicts {
		switch verdict.Action {
		case "new":
			if verdict.Task != nil {
				if err := o.createTaskFromVerdict(ctx, email, verdict); err != nil {
					return fmt.Errorf("failed to create task: %w", err)
				}
			}

		case "update":
			if verdict.TaskID != nil && verdict.Updates != nil {
				if err := o.updateTaskFromVerdict(ctx, *verdict.TaskID, verdict); err != nil {
					return fmt.Errorf("failed to update task: %w", err)
				}
			}

		case "completed":
			if verdict.TaskID != nil {
				if err := o.completeTaskFromVerdict(ctx, *verdict.TaskID, verdict); err != nil {
					return fmt.Errorf("failed to complete task: %w", err)
				}
			}

		case "info_only":
			// Ничего не делаем, только сохраняем ai_verdict
			// (будет реализовано позже при необходимости)
		}
	}

	return nil
}

// createTaskFromVerdict создаёт новую задачу.
func (o *Orchestrator) createTaskFromVerdict(ctx context.Context, email *extractor.ExtractedEmail, verdict Verdict) error {
	task := &store.Task{
		MessageID:     email.MessageID,
		Subject:       verdict.Task.Title,
		BodyText:      verdict.Task.Description,
		FromEmail:     email.From,
		FromName:      extractName(email.From),
		Project:       verdict.Task.Project,
		Type:          verdict.Task.Type,
		Priority:      verdict.Task.Priority,
		Status:        "new",
		ThreadID:      determineThreadID(email),
		SourceEmailID: email.MessageID,
		AIVerdict:     verdictToJSON(verdict),
	}

	return o.store.CreateTask(ctx, task)
}

// updateTaskFromVerdict обновляет существующую задачу.
func (o *Orchestrator) updateTaskFromVerdict(ctx context.Context, taskID int, verdict Verdict) error {
	updates := make(map[string]interface{})

	if verdict.Updates.Priority != "" {
		updates["priority"] = verdict.Updates.Priority
	}
	if verdict.Updates.ChangeStatus != "" {
		updates["status"] = verdict.Updates.ChangeStatus
	}

	if len(updates) > 0 {
		if err := o.store.UpdateTask(ctx, int64(taskID), updates); err != nil {
			return err
		}
	}

	if verdict.Updates.AddComment != "" {
		comment := &store.TaskComment{
			TaskID:    int64(taskID),
			Author:    "ai",
			Body:      verdict.Updates.AddComment,
			Direction: "in",
		}
		if err := o.store.AddTaskComment(ctx, comment); err != nil {
			return err
		}
	}

	return nil
}

// completeTaskFromVerdict завершает задачу.
func (o *Orchestrator) completeTaskFromVerdict(ctx context.Context, taskID int, verdict Verdict) error {
	updates := map[string]interface{}{
		"status": "completed",
	}
	if err := o.store.UpdateTask(ctx, int64(taskID), updates); err != nil {
		return err
	}

	if verdict.Comment != "" {
		comment := &store.TaskComment{
			TaskID:    int64(taskID),
			Author:    "ai",
			Body:      verdict.Comment,
			Direction: "in",
		}
		if err := o.store.AddTaskComment(ctx, comment); err != nil {
			return err
		}
	}

	return nil
}

// verdictToJSON сериализует вердикт для аудита.
func verdictToJSON(v Verdict) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// extractName извлекает имя из адреса вида "Имя Фамилия <email>".
func extractName(from string) string {
	idx := strings.Index(from, "<")
	if idx == -1 {
		return ""
	}
	name := strings.TrimSpace(from[:idx])
	name = strings.Trim(name, `"`)
	return name
}

// UpdateSummary обновляет резюме цепочки после обработки письма.
func (o *Orchestrator) UpdateSummary(ctx context.Context, email *extractor.ExtractedEmail, response *LLMResponse) error {
	threadID := determineThreadID(email)

	// Убеждаемся что тред существует
	thread, err := o.store.GetThread(ctx, threadID)
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}
	if thread == nil {
		if err := o.store.CreateThread(ctx, &store.Thread{ThreadID: threadID}); err != nil {
			return fmt.Errorf("failed to create thread: %w", err)
		}
	}

	// Формируем запрос на обновление summary
	prompt := fmt.Sprintf(`Обнови краткое резюме цепочки писем на основе нового события.

ТЕКУЩЕЕ РЕЗЮМЕ:
%s

НОВОЕ ПИСЬМО:
От: %s
Тема: %s
%s

ВЕРДИКТЫ:
%s

Новое резюме должно быть кратким (2-3 предложения) и отражать текущее состояние цепочки. Верни ТОЛЬКО текст резюме без кавычек и markdown.`,
		o.getThreadSummary(ctx, threadID),
		email.From,
		email.Subject,
		email.BodyText,
		verdictsToJSON(response),
	)

	summary, err := o.client.Generate(ctx, prompt, nil)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}

	return o.store.UpdateThreadSummary(ctx, threadID, summary)
}

func (o *Orchestrator) getThreadSummary(ctx context.Context, threadID string) string {
	thread, err := o.store.GetThread(ctx, threadID)
	if err != nil || thread == nil {
		return "Нет резюме"
	}
	return thread.Summary
}

func verdictsToJSON(response *LLMResponse) string {
	if response == nil {
		return "[]"
	}
	data, _ := json.Marshal(response.Verdicts)
	return string(data)
}
