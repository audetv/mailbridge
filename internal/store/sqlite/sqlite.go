// Package sqlite реализует интерфейс Store для SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3" // драйвер SQLite

	"github.com/audetv/mailbridge/internal/store"
)

// Store реализует интерфейс store.Store для SQLite.
type Store struct {
	db *sql.DB
}

// NewStore создаёт новый Store.
func NewStore(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Настройка для конкурентного доступа
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Включаем WAL-режим
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Включаем поддержку внешних ключей
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &Store{db: db}, nil
}

// Migrate выполняет миграции схемы.
func (s *Store) Migrate(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS email_mapping (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id TEXT NOT NULL UNIQUE,
			plane_issue_id TEXT NOT NULL DEFAULT '',
			plane_project_id TEXT NOT NULL DEFAULT '',
			plane_issue_seq TEXT NOT NULL DEFAULT '',
			original_from TEXT NOT NULL,
			original_subject TEXT NOT NULL,
			thread_references TEXT NOT NULL DEFAULT '[]',
			action_type TEXT NOT NULL DEFAULT 'CREATE',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_email_mapping_message_id ON email_mapping(message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_email_mapping_plane_issue ON email_mapping(plane_issue_id)`,

		`CREATE TABLE IF NOT EXISTS reply_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id TEXT NOT NULL UNIQUE,
			in_reply_to TEXT NOT NULL,
			plane_issue_id TEXT NOT NULL,
			sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reply_log_message_id ON reply_log(message_id)`,

		`CREATE TABLE IF NOT EXISTS outbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			payload TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			last_attempt_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_outbox_status ON outbox(status)`,
	}

	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// SaveMapping сохраняет маппинг email-сообщения.
func (s *Store) SaveMapping(ctx context.Context, m *store.EmailMapping) error {
	refsJSON, err := json.Marshal(m.ThreadReferences)
	if err != nil {
		return fmt.Errorf("failed to marshal thread references: %w", err)
	}

	query := `INSERT INTO email_mapping 
		(message_id, plane_issue_id, plane_project_id, plane_issue_seq, original_from, original_subject, thread_references, action_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.ExecContext(ctx, query,
		m.MessageID, m.PlaneIssueID, m.PlaneProjectID, m.PlaneIssueSeq,
		m.OriginalFrom, m.OriginalSubject, string(refsJSON), m.ActionType)
	if err != nil {
		return fmt.Errorf("failed to save mapping: %w", err)
	}

	return nil
}

// GetMappingByMessageID возвращает маппинг по Message-ID.
func (s *Store) GetMappingByMessageID(ctx context.Context, msgID string) (*store.EmailMapping, error) {
	query := `SELECT id, message_id, plane_issue_id, plane_project_id, plane_issue_seq, 
		original_from, original_subject, thread_references, action_type, created_at
		FROM email_mapping WHERE message_id = ?`

	row := s.db.QueryRowContext(ctx, query, msgID)
	return scanMapping(row)
}

// GetLatestMappingByIssueID возвращает последний маппинг по ID задачи в Plane.
func (s *Store) GetLatestMappingByIssueID(ctx context.Context, issueID string) (*store.EmailMapping, error) {
	query := `SELECT id, message_id, plane_issue_id, plane_project_id, plane_issue_seq, 
		original_from, original_subject, thread_references, action_type, created_at
		FROM email_mapping WHERE plane_issue_id = ? ORDER BY id DESC LIMIT 1`

	row := s.db.QueryRowContext(ctx, query, issueID)
	return scanMapping(row)
}

// MessageExists проверяет существование маппинга по Message-ID.
func (s *Store) MessageExists(ctx context.Context, msgID string) (bool, error) {
	query := `SELECT COUNT(*) FROM email_mapping WHERE message_id = ?`
	var count int
	err := s.db.QueryRowContext(ctx, query, msgID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check message existence: %w", err)
	}
	return count > 0, nil
}

// FindMappingByReferences ищет маппинг по ссылкам (References/In-Reply-To).
func (s *Store) FindMappingByReferences(ctx context.Context, refs []string) (*store.EmailMapping, error) {
	for _, ref := range refs {
		m, err := s.GetMappingByMessageID(ctx, ref)
		if err == nil && m != nil {
			return m, nil
		}
	}

	for _, ref := range refs {
		query := `SELECT id, message_id, plane_issue_id, plane_project_id, plane_issue_seq, 
			original_from, original_subject, thread_references, action_type, created_at
			FROM email_mapping WHERE thread_references LIKE ?`
		row := s.db.QueryRowContext(ctx, query, "%\""+ref+"\"%")
		m, err := scanMapping(row)
		if err == nil && m != nil {
			return m, nil
		}
	}

	return nil, fmt.Errorf("mapping not found by references")
}

// SaveReplyLog сохраняет запись об отправленном ответе.
func (s *Store) SaveReplyLog(ctx context.Context, log *store.ReplyLog) error {
	query := `INSERT INTO reply_log (message_id, in_reply_to, plane_issue_id) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, log.MessageID, log.InReplyTo, log.PlaneIssueID)
	if err != nil {
		return fmt.Errorf("failed to save reply log: %w", err)
	}
	return nil
}

// ReplyExists проверяет, был ли уже отправлен ответ с данным Message-ID.
func (s *Store) ReplyExists(ctx context.Context, msgID string) (bool, error) {
	query := `SELECT COUNT(*) FROM reply_log WHERE message_id = ?`
	var count int
	err := s.db.QueryRowContext(ctx, query, msgID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check reply existence: %w", err)
	}
	return count > 0, nil
}

// EnqueueOutbox добавляет письмо в очередь отправки.
func (s *Store) EnqueueOutbox(ctx context.Context, payload string) error {
	query := `INSERT INTO outbox (payload) VALUES (?)`
	_, err := s.db.ExecContext(ctx, query, payload)
	if err != nil {
		return fmt.Errorf("failed to enqueue outbox: %w", err)
	}
	return nil
}

// GetPendingOutbox возвращает элементы очереди со статусом "pending".
func (s *Store) GetPendingOutbox(ctx context.Context, limit int) ([]*store.OutboxItem, error) {
	query := `SELECT id, payload, status, attempts, last_attempt_at, created_at
		FROM outbox WHERE status = 'pending' ORDER BY created_at ASC LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending outbox: %w", err)
	}
	defer rows.Close()

	var items []*store.OutboxItem
	for rows.Next() {
		item := &store.OutboxItem{}
		var lastAttempt sql.NullTime
		err := rows.Scan(&item.ID, &item.Payload, &item.Status, &item.Attempts, &lastAttempt, &item.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox item: %w", err)
		}
		if lastAttempt.Valid {
			item.LastAttempt = &lastAttempt.Time
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// MarkOutboxSent помечает элемент очереди как отправленный.
func (s *Store) MarkOutboxSent(ctx context.Context, id int64) error {
	query := `UPDATE outbox SET status = 'sent', last_attempt_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to mark outbox sent: %w", err)
	}
	return nil
}

// MarkOutboxFailed помечает элемент очереди как ошибочный.
func (s *Store) MarkOutboxFailed(ctx context.Context, id int64, _ string) error {
	query := `UPDATE outbox SET status = 'failed', attempts = attempts + 1, last_attempt_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to mark outbox failed: %w", err)
	}
	return nil
}

// Ping проверяет соединение с базой данных.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close закрывает соединение с БД.
func (s *Store) Close() error {
	return s.db.Close()
}

// scanMapping сканирует строку в EmailMapping.
func scanMapping(row interface{ Scan(...interface{}) error }) (*store.EmailMapping, error) {
	m := &store.EmailMapping{}
	var refsJSON string
	var createdAt time.Time

	err := row.Scan(&m.ID, &m.MessageID, &m.PlaneIssueID, &m.PlaneProjectID, &m.PlaneIssueSeq,
		&m.OriginalFrom, &m.OriginalSubject, &refsJSON, &m.ActionType, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan mapping: %w", err)
	}

	if refsJSON != "" {
		if err := json.Unmarshal([]byte(refsJSON), &m.ThreadReferences); err != nil {
			m.ThreadReferences = []string{}
		}
	}

	m.CreatedAt = createdAt
	return m, nil
}
