package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

// ProjectHandler обрабатывает запросы к проектам.
type ProjectHandler struct {
	store  store.Store
	broker *EventBroker
}

// NewProjectHandler создаёт новый ProjectHandler.
func NewProjectHandler(st store.Store, broker *EventBroker) *ProjectHandler {
	return &ProjectHandler{store: st, broker: broker}
}

// publishWS шлёт событие всем подписчикам (broker может быть nil — в тестах).
func (h *ProjectHandler) publishWS(eventType string, p *store.Project, message string) {
	if h.broker == nil || p == nil {
		return
	}
	h.broker.Publish(WSEvent{
		Type:    eventType,
		Message: message,
		Data:    p,
	})
}

// validateProjectName проверяет имя (не пустое, trim, <= 128).
func validateProjectName(name string) (string, string) {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return "", "name is required"
	case len([]rune(trimmed)) > 128:
		return "", "name must be at most 128 characters"
	}
	return trimmed, ""
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Ответ уже отправлен (заголовки выданы) — остаётся только залогировать.
		log.Printf("encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// ListProjects обрабатывает GET /api/projects.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	filter := &store.ProjectFilter{}
	if a := r.URL.Query().Get("archived"); a != "" {
		v := a == "true" || a == "1"
		filter.Archived = &v
	}
	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = s
	}

	projects, err := h.store.ListProjects(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []*store.Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

// ListProjectTasks обрабатывает GET /api/projects/{id}/tasks — задачи проекта.
// Фильтр — точное совпадение t.project = название проекта; пагинация через ?page&per_page.
// Project not found → 404; нечисловой id → 400.
func (h *ProjectHandler) ListProjectTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	p, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	result, err := h.store.ListTasks(r.Context(), &store.TaskFilter{
		Project:  p.Name,
		Page:     page,
		PerPage:  perPage,
		Username: extractUserFromToken(r),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if result.Tasks == nil {
		result.Tasks = []*store.TaskWithUnread{}
	}
	writeJSON(w, http.StatusOK, result)
}

// CreateProject обрабатывает POST /api/projects.
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	name, verr := validateProjectName(body.Name)
	if verr != "" {
		writeError(w, http.StatusBadRequest, verr)
		return
	}

	p := &store.Project{Name: name, Description: strings.TrimSpace(body.Description)}
	if err := h.store.CreateProject(r.Context(), p); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "project with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Перечитаем из БД, чтобы вернуть реальные created_at/updated_at (SQLite-дефолты).
	created, err := h.store.GetProject(r.Context(), p.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if created == nil {
		writeError(w, http.StatusInternalServerError, "project not found after insert")
		return
	}

	h.publishWS("project_created", created, fmt.Sprintf("Проект создан: %s", created.Name))
	writeJSON(w, http.StatusCreated, created)
}

// GetProject обрабатывает GET /api/projects/{id}.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	p, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// UpdateProject обрабатывает PUT /api/projects/{id}.
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	name, verr := validateProjectName(body.Name)
	if verr != "" {
		writeError(w, http.StatusBadRequest, verr)
		return
	}

	if err := h.store.UpdateProject(r.Context(), id, name, strings.TrimSpace(body.Description)); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "project with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	p, _ := h.store.GetProject(r.Context(), id)
	h.publishWS("project_updated", p, fmt.Sprintf("Проект обновлён: %s", name))
	writeJSON(w, http.StatusOK, p)
}

// ArchiveProject обрабатывает DELETE /api/projects/{id} (soft-archive).
func (h *ProjectHandler) ArchiveProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	existing, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	if err := h.store.SetProjectArchived(r.Context(), id, true); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	p, _ := h.store.GetProject(r.Context(), id)
	h.publishWS("project_archived", p, fmt.Sprintf("Проект заархивирован: %s", existing.Name))
	writeJSON(w, http.StatusOK, p)
}

// UnarchiveProject обрабатывает POST /api/projects/{id}/unarchive.
func (h *ProjectHandler) UnarchiveProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.store.SetProjectArchived(r.Context(), id, false); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	p, _ := h.store.GetProject(r.Context(), id)
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	h.publishWS("project_unarchived", p, fmt.Sprintf("Проект восстановлён: %s", p.Name))
	writeJSON(w, http.StatusOK, p)
}
