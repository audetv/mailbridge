// Package store определяет интерфейс хранилища и модели данных Mailbridge.
package store

import (
	"context"
	"time"
)

// InboxItem представляет элемент ленты входящих.
type InboxItem struct {
	ID          int64     `json:"id"`
	Source      string    `json:"source"`
	SourceID    string    `json:"source_id"`
	ThreadID    string    `json:"thread_id"`
	FromContact string    `json:"from_contact"`
	FromName    string    `json:"from_name"`
	Subject     string    `json:"subject"`
	BodyText    string    `json:"body_text"`
	BodyHTML    string    `json:"body_html"`
	Meta        string    `json:"meta"` // JSON
	ReceivedAt  time.Time `json:"received_at"`
	AIProcessed int       `json:"ai_processed"`
	AIAttempts  int       `json:"ai_attempts"`
	AIVerdict   string    `json:"ai_verdict"`
	AISummary   string    `json:"ai_summary"`
	Status      string    `json:"status"`
}

// TaskInboxItem связывает задачу с элементом ленты.
type TaskInboxItem struct {
	TaskID      int64     `json:"task_id"`
	InboxItemID int64     `json:"inbox_item_id"`
	Relation    string    `json:"relation"`
	CreatedAt   time.Time `json:"created_at"`
}

// Task представляет задачу в helpdesk.
type Task struct {
	ID            int64     `json:"id"`
	MessageID     string    `json:"message_id"`
	Subject       string    `json:"subject"`
	BodyText      string    `json:"body_text"`
	BodyHTML      string    `json:"body_html"`
	FromEmail     string    `json:"from_email"`
	FromName      string    `json:"from_name"`
	Project       string    `json:"project"`
	Type          string    `json:"type"`
	Priority      string    `json:"priority"`
	Status        string    `json:"status"`
	Assignee      string    `json:"assignee"`
	ThreadID      string    `json:"thread_id"`
	SourceEmailID string    `json:"source_email_id"`
	AIVerdict     string    `json:"ai_verdict"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Thread представляет цепочку входящих.
type Thread struct {
	ID           int64      `json:"id"`
	ThreadID     string     `json:"thread_id"`
	Source       string     `json:"source"`
	Subject      string     `json:"subject"`
	Participants string     `json:"participants"` // JSON-массив
	Summary      string     `json:"summary"`
	LastItemAt   *time.Time `json:"last_item_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TaskWithUnread расширяет Task полем UnreadComments для ответа API.
type TaskWithUnread struct {
	*Task
	UnreadComments int `json:"unread_comments"`
}

// TaskComment представляет комментарий к задаче.
type TaskComment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	Direction string    `json:"direction"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskAttachment представляет вложение задачи.
type TaskAttachment struct {
	ID          int64     `json:"id"`
	TaskID      int64     `json:"task_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	StoragePath string    `json:"storage_path"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskFilter содержит параметры фильтрации списка задач.
type TaskFilter struct {
	Project  string
	Statuses []string // множественный фильтр по статусам
	Assignee string
	Type     string
	Priority string
	Search   string
	Username string
	Page     int
	PerPage  int
}

// TaskListResult содержит результат запроса списка задач.
type TaskListResult struct {
	Tasks   []*TaskWithUnread `json:"tasks"`
	Total   int64             `json:"total"`
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
}

// ReplyLog записывает отправленный ответ для предотвращения дубликатов.
type ReplyLog struct {
	ID           int64
	MessageID    string
	InReplyTo    string
	PlaneIssueID string
	SentAt       time.Time
}

// OutboxItem представляет элемент очереди исходящих писем.
type OutboxItem struct {
	ID          int64
	Payload     string
	Status      string
	Attempts    int
	LastAttempt *time.Time
	CreatedAt   time.Time
}

// InboxFilter содержит параметры фильтрации ленты.
type InboxFilter struct {
	Status  string // unread, read, archived, "" = все
	Source  string
	Page    int
	PerPage int
}

// InboxListResult содержит результат запроса ленты.
type InboxListResult struct {
	Items   []*InboxItem `json:"items"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
}

// Store определяет интерфейс хранилища данных.
type Store interface {
	// Migrate выполняет миграции схемы.
	Migrate(ctx context.Context) error

	// Inbox Items
	CreateInboxItem(ctx context.Context, item *InboxItem) error
	GetInboxItemByID(ctx context.Context, id int64) (*InboxItem, error)
	GetInboxItemBySourceID(ctx context.Context, source, sourceID string) (*InboxItem, error)
	ListInboxItems(ctx context.Context, filter *InboxFilter) (*InboxListResult, error)
	UpdateInboxItemStatus(ctx context.Context, id int64, status string) error
	UpdateInboxItemAI(ctx context.Context, id int64, processed int, verdict, summary string) error

	// Task-Inbox связь
	LinkTaskToInboxItem(ctx context.Context, taskID, inboxItemID int64, relation string) error
	GetInboxItemsByTask(ctx context.Context, taskID int64) ([]*TaskInboxItem, error)
	GetTasksByInboxItem(ctx context.Context, inboxItemID int64) ([]*TaskInboxItem, error)

	// GetPendingAIItems возвращает входящие, ожидающие AI-обработки.
	GetPendingAIItems(ctx context.Context) ([]*InboxItem, error)

	// Tasks
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id int64) (*Task, error)
	GetTaskByMessageID(ctx context.Context, messageID string) (*Task, error)
	ListTasks(ctx context.Context, filter *TaskFilter) (*TaskListResult, error)
	UpdateTask(ctx context.Context, id int64, updates map[string]interface{}) error

	// Task Comments
	AddTaskComment(ctx context.Context, comment *TaskComment) error
	GetTaskComments(ctx context.Context, taskID int64) ([]*TaskComment, error)

	// Task Attachments
	AddTaskAttachment(ctx context.Context, att *TaskAttachment) error
	GetTaskAttachments(ctx context.Context, taskID int64) ([]*TaskAttachment, error)

	// Reply Log
	SaveReplyLog(ctx context.Context, log *ReplyLog) error
	ReplyExists(ctx context.Context, msgID string) (bool, error)

	// Outbox
	EnqueueOutbox(ctx context.Context, payload string) error
	GetPendingOutbox(ctx context.Context, limit int) ([]*OutboxItem, error)
	MarkOutboxSent(ctx context.Context, id int64) error
	MarkOutboxFailed(ctx context.Context, id int64, errMsg string) error

	// MarkTaskRead отмечает задачу прочитанной пользователем.
	MarkTaskRead(ctx context.Context, taskID int64, username string) error
	// ResetTaskReads сбрасывает статус прочтения для всех пользователей задачи.
	// Вызывается при добавлении нового входящего комментария.
	ResetTaskReads(ctx context.Context, taskID int64) error

	// Threads
	CreateThread(ctx context.Context, thread *Thread) error
	GetThread(ctx context.Context, threadID string) (*Thread, error)
	UpdateThreadSummary(ctx context.Context, threadID, summary string) error
	GetActiveTasksByThread(ctx context.Context, threadID string) ([]*Task, error)

	// Ping
	Ping(ctx context.Context) error

	// Close
	Close() error
}
