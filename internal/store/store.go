// Package store определяет интерфейс хранилища и модели данных Mailbridge.
package store

import (
	"context"
	"time"
)

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

// Store определяет интерфейс хранилища данных.
type Store interface {
	// Migrate выполняет миграции схемы.
	Migrate(ctx context.Context) error

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
