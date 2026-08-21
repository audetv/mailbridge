package ai

import (
	"context"
	"fmt"

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

// determineThreadID определяет ID цепочки по References или Message-ID.
func determineThreadID(email *extractor.ExtractedEmail) string {
	if len(email.References) > 0 {
		return email.References[0]
	}
	return email.MessageID
}

func (o *Orchestrator) buildPrompt(summary string, activeTasks []*store.Task, email *extractor.ExtractedEmail) string {
	// TODO: Шаг AI-4.2
	return ""
}

func (o *Orchestrator) parseResponse(response string) (*LLMResponse, error) {
	// TODO: Шаг AI-4.3
	return nil, fmt.Errorf("not implemented")
}
