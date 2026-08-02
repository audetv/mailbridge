// Package store определяет интерфейс хранилища и модели данных Mailbridge.
package store

import (
	"context"
	"time"
)

// Task представляет задачу в helpdesk.
type Task struct {
	ID        int64     `json:"id"`
	MessageID string    `json:"message_id"`
	Subject   string    `json:"subject"`
	BodyText  string    `json:"body_text"`
	BodyHTML  string    `json:"body_html"`
	FromEmail string    `json:"from_email"`
	FromName  string    `json:"from_name"`
	Project   string    `json:"project"`
	Type      string    `json:"type"`
	Priority  string    `json:"priority"`
	Status    string    `json:"status"`
	Assignee  string    `json:"assignee"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskComment представляет комментарий к задаче.
type TaskComment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	Direction string    `json:"direction"` // "in" от клиента, "out" ответ
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
	Status   string
	Assignee string
	Type     string
	Priority string
	Search   string
	Page     int
	PerPage  int
}

// TaskListResult содержит результат запроса списка задач.
type TaskListResult struct {
	Tasks   []*Task `json:"tasks"`
	Total   int64   `json:"total"`
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
}

// EmailMapping связывает email-сообщение с задачей.
type EmailMapping struct {
	ID               int64
	MessageID        string
	PlaneIssueID     string
	PlaneProjectID   string
	PlaneIssueSeq    string
	OriginalFrom     string
	OriginalSubject  string
	ThreadReferences []string
	ActionType       string
	CreatedAt        time.Time
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

	// Email Mapping
	SaveMapping(ctx context.Context, m *EmailMapping) error
	GetMappingByMessageID(ctx context.Context, msgID string) (*EmailMapping, error)
	GetLatestMappingByIssueID(ctx context.Context, issueID string) (*EmailMapping, error)
	MessageExists(ctx context.Context, msgID string) (bool, error)
	FindMappingByReferences(ctx context.Context, refs []string) (*EmailMapping, error)

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

	// Ping
	Ping(ctx context.Context) error

	// Close
	Close() error
}
