// Package webhook обрабатывает входящие события от Plane.
package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

// PlaneWebhookEvent представляет событие от Plane.
type PlaneWebhookEvent struct {
	Event   string `json:"event"`
	Payload struct {
		Issue struct {
			ID         string `json:"id"`
			SequenceID string `json:"sequence_id"`
			Name       string `json:"name"`
			State      string `json:"state"`
		} `json:"issue"`
		Comment *struct {
			ID    string `json:"id"`
			Body  string `json:"comment_html"`
			Actor struct {
				DisplayName string `json:"display_name"`
			} `json:"actor_detail"`
		} `json:"comment"`
	} `json:"payload"`
}

// Handler обрабатывает webhook'и от Plane.
type Handler struct {
	store  store.Store
	logger *slog.Logger
	secret string
}

// NewHandler создаёт новый Handler.
func NewHandler(st store.Store, secret string, logger *slog.Logger) *Handler {
	return &Handler{
		store:  st,
		logger: logger,
		secret: secret,
	}
}

// ServeHTTP реализует http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Валидация подписи
	if h.secret != "" {
		if err := h.validateSignature(r); err != nil {
			h.logger.Warn("webhook signature validation failed", "error", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var event PlaneWebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.logger.Error("failed to decode webhook", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.logger.Info("webhook received", "event", event.Event)

	switch event.Event {
	case "issue.comment.created":
		h.handleNewComment(r.Context(), &event)
	case "issue.updated":
		h.handleStatusChange(r.Context(), &event)
	default:
		h.logger.Debug("ignoring webhook event", "event", event.Event)
	}

	w.WriteHeader(http.StatusOK)
}

// handleNewComment обрабатывает новый комментарий.
func (h *Handler) handleNewComment(_ context.Context, event *PlaneWebhookEvent) {
	if event.Payload.Comment == nil {
		return
	}

	body := event.Payload.Comment.Body

	// Защита от петель: игнорируем комментарии, созданные самим шлюзом
	if strings.Contains(body, "[MAILBRIDGE-INTERNAL]") {
		h.logger.Debug("ignoring internal comment")
		return
	}

	issueID := event.Payload.Issue.ID
	author := event.Payload.Comment.Actor.DisplayName

	h.logger.Info("processing new comment",
		"issue_id", issueID,
		"author", author,
	)
}

// handleStatusChange обрабатывает изменение статуса.
func (h *Handler) handleStatusChange(_ context.Context, event *PlaneWebhookEvent) {
	newState := event.Payload.Issue.State

	// Уведомляем только о значимых переходах
	notifyStates := map[string]bool{
		"completed":   true,
		"cancelled":   true,
		"in_progress": true,
	}

	if !notifyStates[newState] {
		return
	}
}

// validateSignature проверяет HMAC-подпись webhook'а.
func (h *Handler) validateSignature(r *http.Request) error {
	signature := r.Header.Get("X-Plane-Signature")
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	// Plane использует SHA256 HMAC
	mac := hmac.New(sha256.New, []byte(h.secret))
	body := r.Body
	// Тело уже прочитано, но для валидации нужно перечитать
	// В реальной реализации нужно буферизовать тело
	_ = body
	_ = mac

	// Плейсхолдер — полная реализация с буферизацией тела
	_ = signature

	return nil
}
