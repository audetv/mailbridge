package ai

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/web"
)

// Worker обрабатывает элементы очереди через LLM.
type Worker struct {
	queue        *Queue
	orchestrator *Orchestrator
	store        store.Store
	logger       *slog.Logger
	broker       *web.EventBroker
}

// NewWorker создаёт новый Worker.
func NewWorker(queue *Queue, orchestrator *Orchestrator, st store.Store, logger *slog.Logger, broker *web.EventBroker) *Worker {
	return &Worker{
		queue:        queue,
		orchestrator: orchestrator,
		store:        st,
		logger:       logger,
		broker:       broker,
	}
}

// Start запускает обработку очереди.
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("[AIWorker] started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[AIWorker] stopped")
			return
		case inboxItemID := <-w.queue.Channel():
			w.process(ctx, inboxItemID)
		}
	}
}

// process обрабатывает один элемент очереди.
func (w *Worker) process(ctx context.Context, inboxItemID int64) {
	item, err := w.store.GetInboxItemByID(ctx, inboxItemID)
	if err != nil || item == nil {
		w.logger.Error("[AIWorker] inbox item not found", "inbox_item_id", inboxItemID, "error", err)
		return
	}

	attempts := item.AIAttempts + 1
	if err := w.store.UpdateInboxItemAI(ctx, inboxItemID, 0, "[]", ""); err != nil {
		w.logger.Error("[AIWorker] failed to update attempts", "error", err)
	}

	email := w.convertToEmail(ctx, item)

	response, err := w.orchestrator.ProcessEmail(ctx, email)
	if err != nil {
		w.logger.Error("[AIWorker] failed to process", "inbox_item_id", inboxItemID, "error", err)
		if attempts >= 5 {
			if err := w.store.UpdateInboxItemAI(ctx, inboxItemID, -1, "[]", ""); err != nil {
				w.logger.Error("[AIWorker] failed to mark failed", "error", err)
			}
			w.logger.Info("[AIWorker] inbox item marked as failed", "inbox_item_id", inboxItemID)
		} else {
			go func(id int64) {
				time.Sleep(backoff(attempts))
				w.queue.Enqueue(id)
			}(inboxItemID)
		}
		return
	}

	if err := w.orchestrator.ApplyVerdicts(ctx, email, response, inboxItemID); err != nil {
		w.logger.Error("[AIWorker] failed to apply verdicts", "inbox_item_id", inboxItemID, "error", err)
		return
	}

	if err := w.store.UpdateInboxItemAI(ctx, inboxItemID, 1, verdictsToJSON(response), ""); err != nil {
		w.logger.Error("[AIWorker] failed to update processed", "error", err)
	}

	// Публикуем события о созданных задачах
	if w.broker != nil {
		links, _ := w.store.GetTasksByInboxItem(ctx, inboxItemID)
		for _, link := range links {
			task, _ := w.store.GetTask(ctx, link.TaskID)
			if task != nil {
				w.broker.Publish(web.WSEvent{
					Type:    "task_created",
					TaskID:  task.ID,
					Message: fmt.Sprintf("Новая задача #%d: %s", task.ID, task.Subject),
					Data:    task,
				})
			}
		}
	}

	w.logger.Info("[AIWorker] inbox item processed successfully", "inbox_item_id", inboxItemID)
}

// convertToEmail преобразует InboxItem в ExtractedEmail с вложениями.
func (w *Worker) convertToEmail(ctx context.Context, item *store.InboxItem) *extractor.ExtractedEmail {
	email := &extractor.ExtractedEmail{
		MessageID: item.SourceID,
		From:      item.FromContact,
		Subject:   item.Subject,
		BodyText:  item.BodyText,
		BodyHTML:  item.BodyHTML,
	}

	// Загружаем вложения
	atts, err := w.store.GetAttachmentsByInbox(ctx, item.ID)
	if err == nil {
		for _, att := range atts {
			email.Attachments = append(email.Attachments, extractor.Attachment{
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Size:        att.Size,
				StoragePath: att.StoragePath,
			})
		}
	}

	return email
}

// backoff возвращает задержку для retry.
func backoff(attempt int) time.Duration {
	delays := []time.Duration{1 * time.Minute, 5 * time.Minute, 15 * time.Minute, 1 * time.Hour}
	if attempt-1 < len(delays) {
		return delays[attempt-1]
	}
	return delays[len(delays)-1]
}

// BackoffForTest — экспортируемая обёртка.
func BackoffForTest(attempt int) time.Duration {
	return backoff(attempt)
}
