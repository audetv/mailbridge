package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

// newManualID генерирует 16 hex-символов для уникального message_id ручной задачи.
func newManualID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b)
}

// deref возвращает значение строки по указателю или "".
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// TaskHandler обрабатывает запросы к задачам.
type TaskHandler struct {
	store  store.Store
	broker *EventBroker
	// adminUser — имя админа (approve только он, ФАЗА 4).
	adminUser string
}

// NewTaskHandler создаёт новый TaskHandler.
func NewTaskHandler(st store.Store, broker *EventBroker, adminUser string) *TaskHandler {
	if adminUser == "" {
		adminUser = getEnv("MAILBRIDGE_AUTH_USER", "admin")
	}
	return &TaskHandler{store: st, broker: broker, adminUser: adminUser}
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

	// Вложения входящего достают в задачу (идемпотентно)
	if n, err := h.store.CopyInboxAttachmentsToTask(r.Context(), task.ID, item.ID); err != nil {
		log.Printf("failed to copy inbox attachments to task: %v", err)
	} else if n > 0 {
		log.Printf("copied %d attachment(s) from inbox item %d to task %d", n, item.ID, task.ID)
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

	// Фильтр по модулю (эпику)
	var epicID *int64
	if raw := q.Get("epic_id"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			epicID = &v
		}
	}

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
		EpicID:   epicID,
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

// CreateTask обрабатывает POST /api/tasks — ручное создание задачи.
// Обязательно: title, project (имя). Опционально: description, epic_id.
// Статус всегда new. WS: task_created.
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Title       string  `json:"title"`
		Project     string  `json:"project"`
		Description *string `json:"description"`
		EpicID      *int64  `json:"epic_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 500 {
		writeError(w, http.StatusBadRequest, "title is required (1..500 chars)")
		return
	}
	req.Project = strings.TrimSpace(req.Project)
	if req.Project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}

	p, err := h.store.GetProjectByName(r.Context(), req.Project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if p.Archived {
		writeError(w, http.StatusBadRequest, "project is archived")
		return
	}

	if req.EpicID != nil {
		e, err := h.store.GetEpic(r.Context(), *req.EpicID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if e == nil {
			writeError(w, http.StatusBadRequest, "epic not found")
			return
		}
		if e.ProjectID != p.ID {
			writeError(w, http.StatusBadRequest, "epic doesn't belong to project")
			return
		}
	}

	task := &store.Task{
		MessageID: "manual-" + newManualID(),
		Subject:   req.Title,
		BodyText:  deref(req.Description),
		Project:   req.Project,
		Status:    string(store.StatusNew),
	}
	if err := h.store.CreateTask(r.Context(), task); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if req.EpicID != nil {
		if err := h.store.SetTaskEpic(r.Context(), task.ID, *req.EpicID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		task.EpicID = req.EpicID
	}

	if h.broker != nil {
		h.broker.Publish(WSEvent{
			Type:    "task_created",
			TaskID:  task.ID,
			Message: fmt.Sprintf("Новая задача #%d: %s", task.ID, task.Subject),
			Data:    task,
		})
	}

	writeJSON(w, http.StatusCreated, task)
}

// GetTaskStatusHistory обрабатывает GET /api/tasks/{id}/history
// — хронология статусов задачи (v0.23, шаг 2).
func (h *TaskHandler) GetTaskStatusHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"}); err != nil {
			return
		}
		return
	}

	history, err := h.store.GetTaskStatusHistory(r.Context(), id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); encErr != nil {
			log.Printf("encode error: %v", encErr)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(map[string]interface{}{"history": history}); encErr != nil {
		log.Printf("encode error: %v", encErr)
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

	// Разрешённые поля для обновления.
	// status — отдельно, через SetTaskStatus: единственный путь смены статуса,
	// атомарно обновляет статус и пишет строку в task_status_history
	// (v0.23, шаг 2). Обновлять его здесь повторно нельзя — история затеряется.
	allowedFields := map[string]bool{
		"project": true, "assignee": true,
		"type": true, "priority": true, "epic_id": true,
	}

	filtered := make(map[string]interface{})
	newStatus := ""
	for k, v := range updates {
		if k == "status" {
			if s, ok := v.(string); ok && s != "" {
				newStatus = s
			}
			continue
		}
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

	// Смена статуса → SetTaskStatus: обновит статус + напишет в историю
	// (by = текущий пользователь из токена: admin/hermes).
	if newStatus != "" {
		if err := h.store.SetTaskStatus(r.Context(), id, newStatus, extractUserFromToken(r)); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
				log.Printf("encode error: %v", err)
			}
			return
		}
	}

	task, _ := h.store.GetTask(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"task": task}); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// BulkUpdateTasks — PATCH /api/tasks {ids:[], changes:{status?, project?}} (v0.23, шаг 4).
// Пакетная смена status и/или project отобранных задач.
// Контракт: 200 {count, statuses:{status:count}, errors:[…]} | 400 {error} | 404 {error}.
//   - id не найден → 404 (контракт шага 4: «404 задача не найдена»);
//   - идемпотентно: статус уже = to → строки истории нет (то же, что SetTaskStatus);
//   - aтомарность не обязательна планом (на SQLite транзакция внутри store).
//
// WS: одно событие batch_update, не N × task_updated (шаг 4 п.4: «по простоте UI»).
func (h *TaskHandler) BulkUpdateTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed: use PATCH"})
		return
	}

	var req struct {
		IDs     []int64 `json:"ids"`
		Changes struct {
			Status  string `json:"status"`
			Project string `json:"project"`
		} `json:"changes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "ids: empty list")
		return
	}
	if req.Changes.Status == "" && req.Changes.Project == "" {
		writeError(w, http.StatusBadRequest, "changes: specify at least one of `status` or `project`")
		return
	}
	if req.Changes.Status != "" {
		valid := false
		for _, allowed := range store.ValidStatuses() {
			if string(allowed) == req.Changes.Status {
				valid = true
				break
			}
		}
		if !valid {
			writeError(w, http.StatusBadRequest,
				"changes.status: expected one of new|backlog|in_progress|completed|closed")
			return
		}
	}

	by := extractUserFromToken(r)
	if by == "" {
		by = "unknown"
	}

	// Дупы схлопываем — иначе дублировались бы строки истории на одну задачу.
	seen := make(map[int64]struct{}, len(req.IDs))
	uniq := make([]int64, 0, len(req.IDs))
	for _, id := range req.IDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	// status через BulkUpdateTasks (история), project через BulkUpdateProject.
	changedCount := 0
	if req.Changes.Status != "" {
		c, err := h.store.BulkUpdateTasks(r.Context(), uniq, req.Changes.Status, by)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, store.ErrTaskNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		changedCount = c
	}
	if req.Changes.Project != "" {
		c, err := h.store.BulkUpdateProject(r.Context(), uniq, req.Changes.Project)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if changedCount == 0 {
			changedCount = c
		}
	}

	// WS — одно пакетное событие (шаг 4 п.4): данные — список затронутых ID.
	if h.broker != nil {
		h.broker.Publish(WSEvent{
			Type:    "batch_update",
			Count:   len(uniq),
			Message: fmt.Sprintf("bulk update: %d задач, status=%s, project=%q", len(uniq), req.Changes.Status, req.Changes.Project),
			Data:    uniq,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"count":    len(uniq),
		"changed":  changedCount,
		"statuses": map[string]interface{}{},
		"errors":   []string{},
	}
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		log.Printf("encode error: %v", encErr)
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
		// kind: user_comment (по умолчанию) | report | reply (ФАЗА 4)
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	switch req.Kind {
	case "":
		req.Kind = "user_comment"
	case "user_comment", "report", "reply":
	default:
		http.Error(w, `{"error":"invalid kind: expected user_comment|report|reply"}`, http.StatusBadRequest)
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
		Kind:      req.Kind,
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

// ApproveComment обрабатывает PATCH /api/comments/{id}/approve (ФАЗА 4, шаг 19).
// Только admin (на этапе — пользователь MAILBRIDGE_AUTH_USER), только kind=reply.
// Idempotent: повторный approve не ошибка. WS: comment_approved.
func (h *TaskHandler) ApproveComment(w http.ResponseWriter, r *http.Request) {
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

	username := extractUserFromToken(r)
	if username == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if username != h.adminUser {
		writeError(w, http.StatusForbidden, "approve available only to admin")
		return
	}

	comment, err := h.store.GetTaskComment(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrCommentNotFound) {
			writeError(w, http.StatusNotFound, "comment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if comment.Kind != "reply" {
		writeError(w, http.StatusBadRequest, "only comments with kind=reply can be approved")
		return
	}

	one := 1
	if comment.Approved == nil || *comment.Approved != 1 {
		if err := h.store.SetTaskCommentApproved(r.Context(), id, true); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		comment.Approved = &one
	}

	if h.broker != nil {
		h.broker.Publish(WSEvent{
			Type:    "comment_approved",
			TaskID:  comment.TaskID,
			Message: fmt.Sprintf("Комментарий #%d утверждён", id),
			Data:    comment,
		})
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
