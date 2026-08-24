package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

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

// ListInbox обрабатывает GET /api/inbox
func (h *TaskHandler) ListInbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))

	perPageStr := q.Get("perPage")
	if perPageStr == "" {
		perPageStr = q.Get("per_page")
	}
	perPage, _ := strconv.Atoi(perPageStr)

	filter := &store.InboxFilter{
		Status:  q.Get("status"),
		Source:  q.Get("source"),
		Page:    page,
		PerPage: perPage,
	}

	result, err := h.store.ListInboxItems(r.Context(), filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	if result.Items == nil {
		result.Items = []*store.InboxItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// GetInboxItem обрабатывает GET /api/inbox/{id}
func (h *TaskHandler) GetInboxItem(w http.ResponseWriter, r *http.Request) {
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

	item, err := h.store.GetInboxItemByID(r.Context(), id)
	if err != nil || item == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "not found"}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(item); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// GetTaskInboxItems обрабатывает GET /api/tasks/{id}/inbox
func (h *TaskHandler) GetTaskInboxItems(w http.ResponseWriter, r *http.Request) {
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

	links, err := h.store.GetInboxItemsByTask(r.Context(), id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	// Загружаем полные InboxItem для каждой связи
	var items []*store.InboxItem
	for _, link := range links {
		item, err := h.store.GetInboxItemByID(r.Context(), link.InboxItemID)
		if err == nil && item != nil {
			items = append(items, item)
		}
	}

	if items == nil {
		items = []*store.InboxItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// GetInboxItemTasks обрабатывает GET /api/inbox/{id}/tasks
func (h *TaskHandler) GetInboxItemTasks(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	tasks, err := h.store.GetTasksByInboxItem(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Гарантируем что не null
	if tasks == nil {
		tasks = []*store.TaskInboxItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// UpdateInboxStatus обрабатывает POST /api/inbox/{id}/read, /unread, /archive
func (h *TaskHandler) UpdateInboxStatus(w http.ResponseWriter, r *http.Request) {
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

	// Определяем новый статус по URL
	var newStatus string
	switch {
	case strings.HasSuffix(r.URL.Path, "/read"):
		newStatus = "read"
	case strings.HasSuffix(r.URL.Path, "/unread"):
		newStatus = "unread"
	case strings.HasSuffix(r.URL.Path, "/archive"):
		newStatus = "archived"
	default:
		http.Error(w, `{"error":"unknown action"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateInboxItemStatus(r.Context(), id, newStatus); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": newStatus}); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// CreateTaskFromInbox обрабатывает POST /api/inbox/{id}/task
func (h *TaskHandler) CreateTaskFromInbox(w http.ResponseWriter, r *http.Request) {
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

	item, err := h.store.GetInboxItemByID(r.Context(), id)
	if err != nil || item == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "inbox item not found"}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	// Создаём задачу из ленты
	task := &store.Task{
		MessageID:     item.SourceID,
		Subject:       item.Subject,
		BodyText:      item.BodyText,
		BodyHTML:      item.BodyHTML,
		FromEmail:     item.FromContact,
		FromName:      item.FromName,
		Project:       "Входящие",
		Status:        string(store.StatusNew),
		ThreadID:      item.ThreadID,
		SourceEmailID: item.SourceID,
	}

	if err := h.store.CreateTask(r.Context(), task); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	// Связываем с лентой
	if err := h.store.LinkTaskToInboxItem(r.Context(), task.ID, item.ID, "created_manually"); err != nil {
		log.Printf("failed to link task to inbox: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"task": task}); err != nil {
		log.Printf("encode error: %v", err)
	}
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
