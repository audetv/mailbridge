package ai

import (
	"context"
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

	// Загружаем контекст
	thread, err := o.store.GetThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	summary := ""
	if thread != nil {
		summary = thread.Summary
	} else {
		// Создаём новый тред
		if err := o.store.CreateThread(ctx, &store.Thread{ThreadID: threadID}); err != nil {
			return nil, fmt.Errorf("failed to create thread: %w", err)
		}
	}

	activeTasks, err := o.store.GetActiveTasksByThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks: %w", err)
	}

	// Формируем промпт (следующий шаг)
	prompt := o.buildPrompt(summary, activeTasks, email)

	// Вызываем LLM
	response, err := o.client.Generate(ctx, prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM generate failed: %w", err)
	}

	// Парсим JSON (следующий шаг)
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

// parseResponse — будет реализовано в шаге AI-4.3
func (o *Orchestrator) parseResponse(_ string) (*LLMResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
