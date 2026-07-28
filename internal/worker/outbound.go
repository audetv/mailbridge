package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/audetv/mailbridge/internal/sender"
	"github.com/audetv/mailbridge/internal/store"
)

// OutboundWorker обрабатывает очередь исходящих писем.
type OutboundWorker struct {
	store    store.Store
	sender   *sender.Sender
	interval time.Duration
	logger   *slog.Logger
	stopCh   chan struct{}
}

// NewOutboundWorker создаёт новый OutboundWorker.
func NewOutboundWorker(
	st store.Store,
	s *sender.Sender,
	interval time.Duration,
	logger *slog.Logger,
) *OutboundWorker {
	return &OutboundWorker{
		store:    st,
		sender:   s,
		interval: interval,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

// Start запускает цикл обработки очереди.
func (w *OutboundWorker) Start(ctx context.Context) error {
	w.logger.Info("outbound worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("outbound worker stopped by context")
			return nil
		case <-w.stopCh:
			w.logger.Info("outbound worker stopped")
			return nil
		case <-ticker.C:
			w.processOutbox(ctx)
		}
	}
}

// Stop останавливает воркер.
func (w *OutboundWorker) Stop(_ context.Context) error {
	close(w.stopCh)
	return nil
}

// processOutbox обрабатывает pending-элементы очереди.
func (w *OutboundWorker) processOutbox(ctx context.Context) {
	items, err := w.store.GetPendingOutbox(ctx, 10)
	if err != nil {
		w.logger.Error("failed to get pending outbox", "error", err)
		return
	}

	if len(items) == 0 {
		return
	}

	w.logger.Info("processing outbox", "count", len(items))

	for _, item := range items {
		if err := w.sendItem(item); err != nil {
			w.logger.Error("failed to send outbox item",
				"id", item.ID,
				"attempts", item.Attempts,
				"error", err,
			)
			if markErr := w.store.MarkOutboxFailed(ctx, item.ID, err.Error()); markErr != nil {
				w.logger.Error("failed to mark outbox failed", "error", markErr)
			}
			continue
		}

		if markErr := w.store.MarkOutboxSent(ctx, item.ID); markErr != nil {
			w.logger.Error("failed to mark outbox sent", "error", markErr)
		}
	}
}

// sendItem отправляет одно письмо из очереди.
func (w *OutboundWorker) sendItem(item *store.OutboxItem) error {
	var payload struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal([]byte(item.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	switch payload.Type {
	case "acknowledgement":
		var data sender.AcknowledgementData
		if err := json.Unmarshal(payload.Data, &data); err != nil {
			return fmt.Errorf("failed to unmarshal ack data: %w", err)
		}
		return w.sender.SendAcknowledgement(&data)

	case "comment_reply":
		var data sender.CommentReplyData
		if err := json.Unmarshal(payload.Data, &data); err != nil {
			return fmt.Errorf("failed to unmarshal comment data: %w", err)
		}
		return w.sender.SendCommentReply(&data)

	case "status_change":
		var data sender.StatusChangeData
		if err := json.Unmarshal(payload.Data, &data); err != nil {
			return fmt.Errorf("failed to unmarshal status data: %w", err)
		}
		return w.sender.SendStatusChange(&data)

	default:
		return fmt.Errorf("unknown payload type: %s", payload.Type)
	}
}

// String возвращает имя воркера.
func (w *OutboundWorker) String() string {
	return fmt.Sprintf("OutboundWorker(interval=%s)", w.interval)
}

// EnqueueAcknowledgement добавляет подтверждение в очередь.
func EnqueueAcknowledgement(ctx context.Context, st store.Store, data *sender.AcknowledgementData) error {
	return enqueue(ctx, st, "acknowledgement", data)
}

// EnqueueCommentReply добавляет уведомление о комментарии в очередь.
func EnqueueCommentReply(ctx context.Context, st store.Store, data *sender.CommentReplyData) error {
	return enqueue(ctx, st, "comment_reply", data)
}

// EnqueueStatusChange добавляет уведомление о смене статуса в очередь.
func EnqueueStatusChange(ctx context.Context, st store.Store, data *sender.StatusChangeData) error {
	return enqueue(ctx, st, "status_change", data)
}

func enqueue(ctx context.Context, st store.Store, payloadType string, data interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	payload := map[string]interface{}{
		"type": payloadType,
		"data": json.RawMessage(dataJSON),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return st.EnqueueOutbox(ctx, string(payloadJSON))
}
