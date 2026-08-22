package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
)

// Orchestrator координирует обработку писем через LLM.
type Orchestrator struct {
	client   Client
	store    store.Store
	projects []string
}

// NewOrchestrator создаёт новый Orchestrator.
func NewOrchestrator(client Client, st store.Store) *Orchestrator {
	return &Orchestrator{
		client: client,
		store:  st,
	}
}

// SetProjects устанавливает список доступных проектов.
func (o *Orchestrator) SetProjects(projects []string) {
	o.projects = projects
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

	var images []string
	for _, att := range email.Attachments {
		if isImageAttachment(att.ContentType) && att.StoragePath != "" {
			fullPath := filepath.Join("data/attachments", att.StoragePath)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				log.Printf("[AI] failed to read image %s: %v", fullPath, err)
				continue
			}
			images = append(images, base64.StdEncoding.EncodeToString(data))
		}
	}

	log.Printf("[AI] Промпт отправлен в LLM:\n%s", prompt)
	if len(images) > 0 {
		log.Printf("[AI] Изображений: %d", len(images))
	} else {
		log.Printf("[AI] Изображений: 0")
	}

	response, err := o.client.Generate(ctx, prompt, images)
	if err != nil {
		return nil, fmt.Errorf("LLM generate failed: %w", err)
	}

	log.Printf("[AI] Ответ LLM:\n%s", response)
	o.saveDebugLog(threadID, prompt, response, images)

	return o.parseResponse(response)
}

// UpdateSummary обновляет резюме цепочки после обработки письма.
func (o *Orchestrator) UpdateSummary(ctx context.Context, email *extractor.ExtractedEmail, response *LLMResponse) error {
	threadID := determineThreadID(email)

	thread, err := o.store.GetThread(ctx, threadID)
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}
	if thread == nil {
		if err := o.store.CreateThread(ctx, &store.Thread{ThreadID: threadID}); err != nil {
			return fmt.Errorf("failed to create thread: %w", err)
		}
	}

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

// BuildPrompt — экспортируемая обёртка для тестирования.
func (o *Orchestrator) BuildPrompt(summary string, activeTasks []*store.Task, email *extractor.ExtractedEmail) string {
	return o.buildPrompt(summary, activeTasks, email)
}

// ParseResponse — экспортируемая обёртка для тестирования.
func (o *Orchestrator) ParseResponse(response string) (*LLMResponse, error) {
	return o.parseResponse(response)
}

// saveDebugLog сохраняет промпт и ответ LLM в файл для отладки.
func (o *Orchestrator) saveDebugLog(threadID, prompt, response string, images []string) {
	dir := "data/ai-debug"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	filename := filepath.Join(dir, fmt.Sprintf("%s-%d.md", threadID, time.Now().Unix()))
	var sb strings.Builder
	sb.WriteString("# AI Debug Log\n\n")
	sb.WriteString("## Промпт\n\n```\n")
	sb.WriteString(prompt)
	sb.WriteString("\n```\n\n")
	fmt.Fprintf(&sb, "## Изображения: %d\n\n", len(images))
	sb.WriteString("## Ответ\n\n```json\n")
	sb.WriteString(response)
	sb.WriteString("\n```\n")

	_ = os.WriteFile(filename, []byte(sb.String()), 0o644)
}

// buildPrompt формирует промпт для LLM.
func (o *Orchestrator) buildPrompt(summary string, activeTasks []*store.Task, email *extractor.ExtractedEmail) string {
	var sb strings.Builder

	sb.WriteString("Ты — интеллектуальный ассистент для управления задачами.\n\n")
	sb.WriteString("Проанализируй НОВОЕ письмо (включая пересланные цепочки) в контексте истории.\n\n")

	if len(o.projects) > 0 {
		sb.WriteString("=== ДОСТУПНЫЕ ПРОЕКТЫ ===\n")
		sb.WriteString(strings.Join(o.projects, ", "))
		sb.WriteString("\nВАЖНО: Выбирай проект ТОЛЬКО из этого списка. Если не уверен — используй \"Входящие\".\n\n")
	}

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
        "title": "Краткий заголовок (до 60 символов)",
        "description": "Суть задачи",
        "priority": "high|medium|low",
        "project": "Название проекта",
        "type": "bug|feature|support|access|seo|content",
        "source_email_id": "message-id из заголовка",
        "image_note": "Описание скриншота или null"
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
      "task_id": null,
      "task": {
        "title": "Краткий заголовок выполненного действия",
        "description": "Что именно было сделано (извлеки из текста)",
        "priority": "medium",
        "project": "Входящие",
        "type": "support",
        "source_email_id": "message-id из заголовка"
      },
      "comment": "Дополнительный комментарий (опционально)"
    },
    {
      "action": "none",
      "summary": "Краткая информация, которая не требует действий"
    }
  ]
}

ПРИМЕР для completed с task_id = null (Пересланное письмо о выполненной работе):
{
  "verdicts": [
    {
      "action": "completed",
      "task_id": null,
      "task": {
        "title": "Размещение документа на сайте",
        "description": "Документ успешно размещён по ссылке. Заказчик подтвердил получение.",
        "priority": "medium",
        "project": "Входящие",
        "type": "content",
        "source_email_id": "<message-id-письма>"
      },
      "comment": "Работа закрыта, подтверждение получено."
    }
  ]
}`)

	return sb.String()
}

// parseResponse парсит JSON-ответ LLM.
func (o *Orchestrator) parseResponse(response string) (*LLMResponse, error) {
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
