package web

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/audetv/mailbridge/internal/store"
)

// TaskHandler обрабатывает запросы к задачам.
type TaskHandler struct {
	store store.Store
}

// NewTaskHandler создаёт новый TaskHandler.
func NewTaskHandler(st store.Store) *TaskHandler {
	return &TaskHandler{store: st}
}

// ListTasks обрабатывает GET /api/tasks
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	username := extractUserFromToken(r)
	// Статусы — может быть несколько ?status=new&status=in_progress
	var statuses []string
	for _, s := range q["status"] {
		if s != "" {
			statuses = append(statuses, s)
		}
	}

	filter := &store.TaskFilter{
		Project:  q.Get("project"),
		Statuses: statuses,
		Assignee: q.Get("assignee"),
		Type:     q.Get("type"),
		Priority: q.Get("priority"),
		Search:   q.Get("search"),
		Page:     page,
		PerPage:  perPage,
		Username: username,
	}

	result, err := h.store.ListTasks(r.Context(), filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	if result.Tasks == nil {
		result.Tasks = []*store.TaskWithUnread{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// GetTask обрабатывает GET /api/tasks/{id}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	task, err := h.store.GetTask(r.Context(), id)
	if err != nil || task == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "task not found"}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	comments, _ := h.store.GetTaskComments(r.Context(), id)
	attachments, _ := h.store.GetTaskAttachments(r.Context(), id)

	if comments == nil {
		comments = []*store.TaskComment{}
	}
	if attachments == nil {
		attachments = []*store.TaskAttachment{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"task":        task,
		"comments":    comments,
		"attachments": attachments,
	}); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// UpdateTask обрабатывает PATCH /api/tasks/{id}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	// Разрешённые поля для обновления
	allowedFields := map[string]bool{
		"project": true, "status": true, "assignee": true,
		"type": true, "priority": true,
	}

	filtered := make(map[string]interface{})
	for k, v := range updates {
		if allowedFields[k] {
			filtered[k] = v
		}
	}

	if err := h.store.UpdateTask(r.Context(), id, filtered); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	task, _ := h.store.GetTask(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"task": task}); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// ReplyTask обрабатывает POST /api/tasks/{id}/reply
func (h *TaskHandler) ReplyTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	username := extractUserFromToken(r)
	if username == "" {
		username = "support"
	}

	comment := &store.TaskComment{
		TaskID:    id,
		Author:    username,
		Body:      req.Body,
		Direction: "out",
	}

	if err := h.store.AddTaskComment(r.Context(), comment); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"comment": comment}); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// GetAttachment обрабатывает GET /api/attachments/{path...}
func (h *TaskHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Путь к файлу вложений
	fullPath := filepath.Join("data", "attachments", filepath.Clean(path))

	// Проверяем что файл существует
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullPath)
}

// MarkRead обрабатывает POST /api/tasks/{id}/mark-read
func (h *TaskHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	username := extractUserFromToken(r)
	if username == "" {
		username = "anonymous"
	}

	if err := h.store.MarkTaskRead(r.Context(), id, username); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("encode error: %v", err)
	}
}
