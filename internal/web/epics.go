package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

// EpicHandler обрабатывает запросы к модулям (эпикам).
type EpicHandler struct {
	store  store.Store
	broker *EventBroker
}

// NewEpicHandler создаёт новый EpicHandler.
func NewEpicHandler(st store.Store, broker *EventBroker) *EpicHandler {
	return &EpicHandler{store: st, broker: broker}
}

// parseID извлекает числовой путь `{pathKey}` из запроса.
func parseID(r *http.Request, pathKey string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(pathKey), 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// validateEpicName проверяет имя модуля (не пустое, trim, <= 128).
func validateEpicName(name string) (string, string) {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return "", "name is required"
	case len([]rune(trimmed)) > 128:
		return "", "name must be at most 128 characters"
	}
	return trimmed, ""
}

// validateEpicStatus разрешает open | in_progress | done; пустая строка → "open".
func validateEpicStatus(status string) string {
	switch status {
	case "", "open", "in_progress", "done":
		if status == "" {
			return "open"
		}
		return status
	default:
		return ""
	}
}

// publishWS шлёт событие всем подписчикам.
func (h *EpicHandler) publishWS(eventType string, e *store.Epic, message string) {
	if h.broker == nil || e == nil {
		return
	}
	h.broker.Publish(WSEvent{
		Type:    eventType,
		Message: message,
		Data:    e,
	})
}

// GetEpicDetail — модуль вместе с прогрессом его задач (для GET /api/epics/{id}).
type GetEpicDetail struct {
	*store.Epic
	Progress *store.EpicProgress `json:"progress"`
}

// ListEpicsList обрабатывает GET /api/projects/{id}/epics.
func (h *EpicHandler) ListEpicsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	if p, err := h.store.GetProject(r.Context(), projectID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	epics, err := h.store.ListEpics(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if epics == nil {
		epics = []*store.Epic{}
	}
	writeJSON(w, http.StatusOK, epics)
}

// CreateEpicList обрабатывает POST /api/projects/{id}/epics.
func (h *EpicHandler) CreateEpicList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	if p, err := h.store.GetProject(r.Context(), projectID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name, verr := validateEpicName(body.Name)
	if verr != "" {
		writeError(w, http.StatusBadRequest, verr)
		return
	}
	status := validateEpicStatus(body.Status)
	e := &store.Epic{
		ProjectID:   projectID,
		Name:        name,
		Description: strings.TrimSpace(body.Description),
		Status:      status,
	}
	if err := h.store.CreateEpic(r.Context(), e); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "epic with this number already exists in the project")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	created, err := h.store.GetEpic(r.Context(), e.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.publishWS("epic_created", created, fmt.Sprintf("Модуль создан: %s", created.Name))
	writeJSON(w, http.StatusCreated, created)
}

// GetEpicDetailHandler обрабатывает GET /api/epics/{id}.
func (h *EpicHandler) GetEpicDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parseID(r, "epic_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid epic id")
		return
	}
	e, err := h.store.GetEpic(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if e == nil {
		writeError(w, http.StatusNotFound, "epic not found")
		return
	}
	progress, err := h.store.EpicProgress(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, GetEpicDetail{Epic: e, Progress: progress})
}

// UpdateEpicDetail обрабатывает PUT /api/epics/{id}.
func (h *EpicHandler) UpdateEpicDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parseID(r, "epic_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid epic id")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name, verr := validateEpicName(body.Name)
	if verr != "" {
		writeError(w, http.StatusBadRequest, verr)
		return
	}
	status := validateEpicStatus(body.Status)
	if status == "" {
		writeError(w, http.StatusBadRequest, "status must be one of: open, in_progress, done")
		return
	}
	if err := h.store.UpdateEpic(r.Context(), id, name, strings.TrimSpace(body.Description), status); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	e, _ := h.store.GetEpic(r.Context(), id)
	h.publishWS("epic_updated", e, fmt.Sprintf("Модуль обновлён: %s", name))
	writeJSON(w, http.StatusOK, e)
}

// DeleteEpicDetail обрабатывает DELETE /api/epics/{id}.
func (h *EpicHandler) DeleteEpicDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parseID(r, "epic_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid epic id")
		return
	}
	e, err := h.store.GetEpic(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if e == nil {
		writeError(w, http.StatusNotFound, "epic not found")
		return
	}
	if err := h.store.DeleteEpic(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.publishWS("epic_deleted", e, fmt.Sprintf("Модуль удалён: %s", e.Name))
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// LinkTaskEpic обрабатывает POST /api/epics/{id}/tasks/{taskId}.
func (h *EpicHandler) LinkTaskEpic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	epicID, ok := parseID(r, "epic_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid epic id")
		return
	}
	taskID, ok := parseID(r, "taskId")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	if e, err := h.store.GetEpic(r.Context(), epicID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if e == nil {
		writeError(w, http.StatusNotFound, "epic not found")
		return
	}
	if t, err := h.store.GetTask(r.Context(), taskID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if t == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err := h.store.SetTaskEpic(r.Context(), taskID, epicID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.publishWS("epic_task_linked", nil, fmt.Sprintf("Задача #%d привязана к модулю #%d", taskID, epicID))
	writeJSON(w, http.StatusOK, map[string]string{"status": "linked"})
}

// UnlinkTaskEpic обрабатывает DELETE /api/epics/{id}/tasks/{taskId}.
func (h *EpicHandler) UnlinkTaskEpic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	epicID, ok := parseID(r, "epic_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid epic id")
		return
	}
	taskID, ok := parseID(r, "taskId")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	if e, err := h.store.GetEpic(r.Context(), epicID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if e == nil {
		writeError(w, http.StatusNotFound, "epic not found")
		return
	}
	if t, err := h.store.GetTask(r.Context(), taskID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if t == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err := h.store.SetTaskEpic(r.Context(), taskID, 0); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.publishWS("epic_task_unlinked", nil, fmt.Sprintf("Задача #%d отвязана от модуля #%d", taskID, epicID))
	writeJSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
}
