// Package store определяет интерфейс хранилища и модели данных Mailbridge.
package store

import (
	"context"
	"time"
)

// EmailMapping связывает email-сообщение с задачей в Plane.
type EmailMapping struct {
	ID               int64
	MessageID        string
	PlaneIssueID     string
	PlaneIssueSeq    string
	OriginalFrom     string
	OriginalSubject  string
	ThreadReferences []string
	ActionType       string // "CREATE" или "REPLY"
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
	Status      string // "pending", "sent", "failed"
	Attempts    int
	LastAttempt *time.Time
	CreatedAt   time.Time
}

// Store определяет интерфейс хранилища данных.
type Store interface {
	// Migrate выполняет миграции схемы.
	Migrate(ctx context.Context) error

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

	// Close закрывает соединение с хранилищем.
	Close() error
}
