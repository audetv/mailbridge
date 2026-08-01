// Package worker содержит воркеры для обработки почты.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/audetv/mailbridge/internal/mailbox"
	"github.com/audetv/mailbridge/internal/processor"
)

// InboundWorker периодически сканирует IMAP-ящик и обрабатывает письма.
type InboundWorker struct {
	reader    *mailbox.Reader
	processor *processor.MessageProcessor
	interval  time.Duration
	logger    *slog.Logger
	stopCh    chan struct{}
}

// NewInboundWorker создаёт новый InboundWorker.
func NewInboundWorker(
	reader *mailbox.Reader,
	proc *processor.MessageProcessor,
	interval time.Duration,
	logger *slog.Logger,
) *InboundWorker {
	return &InboundWorker{
		reader:    reader,
		processor: proc,
		interval:  interval,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

// Start запускает цикл обработки.
func (w *InboundWorker) Start(ctx context.Context) error {
	w.logger.Info("inbound worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("inbound worker stopped by context")
			return nil
		case <-w.stopCh:
			w.logger.Info("inbound worker stopped")
			return nil
		case <-ticker.C:
			w.processUnseenMessages(ctx)
		}
	}
}

// Stop останавливает воркер.
func (w *InboundWorker) Stop(_ context.Context) error {
	close(w.stopCh)
	return nil
}

// processUnseenMessages обрабатывает непрочитанные письма.
func (w *InboundWorker) processUnseenMessages(ctx context.Context) {
	rawEmails, err := w.reader.FetchUnseen(ctx)
	if err != nil {
		// Если соединение потеряно — пробуем переподключиться
		if err.Error() == "not connected" || !w.reader.IsConnected() {
			w.logger.Warn("IMAP disconnected, attempting reconnect")
			if reconnectErr := w.reader.Reconnect(); reconnectErr != nil {
				w.logger.Error("IMAP reconnect failed", "error", reconnectErr)
				return
			}
			w.logger.Info("IMAP reconnected, resuming normal operation")
		} else {
			w.logger.Error("failed to fetch unseen messages", "error", err)
		}
		return
	}

	if len(rawEmails) == 0 {
		return
	}

	w.logger.Info("processing unseen messages", "count", len(rawEmails))

	for _, raw := range rawEmails {
		result, err := w.processor.Process(ctx, raw.Data)
		if err != nil {
			w.logger.Error("failed to process email",
				"uid", raw.UID,
				"error", err,
			)
			if markErr := w.reader.MarkErrored(raw.UID); markErr != nil {
				w.logger.Error("failed to mark as errored", "uid", raw.UID, "error", markErr)
			}
			continue
		}

		switch result.Action {
		case processor.ActionCreateIssue:
			w.logger.Info("issue created from email",
				"uid", raw.UID,
				"task_id", result.TaskID,
			)
		case processor.ActionAddComment:
			w.logger.Info("comment added from email",
				"uid", raw.UID,
				"task_id", result.TaskID,
			)
		case processor.ActionIgnore:
			w.logger.Info("email ignored",
				"uid", raw.UID,
				"reason", "duplicate",
			)
		}

		if markErr := w.reader.MarkProcessed(raw.UID); markErr != nil {
			w.logger.Error("failed to mark as processed", "uid", raw.UID, "error", markErr)
		}
	}
}

// String возвращает имя воркера.
func (w *InboundWorker) String() string {
	return fmt.Sprintf("InboundWorker(interval=%s)", w.interval)
}
