package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/audetv/mailbridge/internal/store"
)

// CreateAttachment создаёт запись о файле.
func (s *Store) CreateAttachment(ctx context.Context, att *store.Attachment) error {
	query := `INSERT INTO attachments (hash, filename, content_type, size, storage_path) VALUES (?, ?, ?, ?, ?)`
	result, err := s.db.ExecContext(ctx, query, att.Hash, att.Filename, att.ContentType, att.Size, att.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}
	id, _ := result.LastInsertId()
	att.ID = id
	att.CreatedAt = time.Now()
	return nil
}

// GetAttachmentByHash возвращает вложение по hash.
func (s *Store) GetAttachmentByHash(ctx context.Context, hash string) (*store.Attachment, error) {
	query := `SELECT id, hash, filename, content_type, size, storage_path, created_at FROM attachments WHERE hash = ?`
	row := s.db.QueryRowContext(ctx, query, hash)
	return scanAttachment(row)
}

// GetAttachmentByID возвращает вложение по ID.
func (s *Store) GetAttachmentByID(ctx context.Context, id int64) (*store.Attachment, error) {
	query := `SELECT id, hash, filename, content_type, size, storage_path, created_at FROM attachments WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)
	return scanAttachment(row)
}

// LinkAttachmentToInbox связывает вложение с входящим.
func (s *Store) LinkAttachmentToInbox(ctx context.Context, inboxItemID, attachmentID int64) error {
	query := `INSERT OR IGNORE INTO inbox_attachments (inbox_item_id, attachment_id) VALUES (?, ?)`
	_, err := s.db.ExecContext(ctx, query, inboxItemID, attachmentID)
	return err
}

// LinkAttachmentToTask связывает вложение с задачей.
func (s *Store) LinkAttachmentToTask(ctx context.Context, taskID, attachmentID int64) error {
	query := `INSERT OR IGNORE INTO task_attachments (task_id, attachment_id) VALUES (?, ?)`
	_, err := s.db.ExecContext(ctx, query, taskID, attachmentID)
	return err
}

// UnlinkAttachmentFromTask отвязывает вложение от задачи.
func (s *Store) UnlinkAttachmentFromTask(ctx context.Context, taskID, attachmentID int64) error {
	query := `DELETE FROM task_attachments WHERE task_id = ? AND attachment_id = ?`
	_, err := s.db.ExecContext(ctx, query, taskID, attachmentID)
	return err
}

// GetAttachmentsByInbox возвращает вложения входящего.
func (s *Store) GetAttachmentsByInbox(ctx context.Context, inboxItemID int64) ([]*store.Attachment, error) {
	query := `SELECT a.id, a.hash, a.filename, a.content_type, a.size, a.storage_path, a.created_at
		FROM attachments a
		JOIN inbox_attachments ia ON ia.attachment_id = a.id
		WHERE ia.inbox_item_id = ?
		ORDER BY a.created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, inboxItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAttachments(rows)
}

// GetAttachmentsByTask возвращает вложения задачи.
func (s *Store) GetAttachmentsByTask(ctx context.Context, taskID int64) ([]*store.Attachment, error) {
	query := `SELECT a.id, a.hash, a.filename, a.content_type, a.size, a.storage_path, a.created_at
		FROM attachments a
		JOIN task_attachments ta ON ta.attachment_id = a.id
		WHERE ta.task_id = ?
		ORDER BY a.created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAttachments(rows)
}

func scanAttachment(row interface{ Scan(...interface{}) error }) (*store.Attachment, error) {
	att := &store.Attachment{}
	err := row.Scan(&att.ID, &att.Hash, &att.Filename, &att.ContentType, &att.Size, &att.StoragePath, &att.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan attachment: %w", err)
	}
	return att, nil
}

func scanAttachments(rows *sql.Rows) ([]*store.Attachment, error) {
	var atts []*store.Attachment
	for rows.Next() {
		att := &store.Attachment{}
		if err := rows.Scan(&att.ID, &att.Hash, &att.Filename, &att.ContentType, &att.Size, &att.StoragePath, &att.CreatedAt); err != nil {
			return nil, err
		}
		atts = append(atts, att)
	}
	return atts, rows.Err()
}

// LinkAttachmentToComment связывает вложение с комментарием.
func (s *Store) LinkAttachmentToComment(ctx context.Context, commentID, attachmentID int64) error {
	query := `INSERT OR IGNORE INTO comment_attachments (comment_id, attachment_id) VALUES (?, ?)`
	_, err := s.db.ExecContext(ctx, query, commentID, attachmentID)
	return err
}

// GetAttachmentsByComment возвращает вложения комментария.
func (s *Store) GetAttachmentsByComment(ctx context.Context, commentID int64) ([]*store.Attachment, error) {
	query := `SELECT a.id, a.hash, a.filename, a.content_type, a.size, a.storage_path, a.created_at
		FROM attachments a
		JOIN comment_attachments ca ON ca.attachment_id = a.id
		WHERE ca.comment_id = ?
		ORDER BY a.created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, commentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment attachments: %w", err)
	}
	defer rows.Close()

	var atts []*store.Attachment
	for rows.Next() {
		att := &store.Attachment{}
		if err := rows.Scan(&att.ID, &att.Hash, &att.Filename, &att.ContentType, &att.Size, &att.StoragePath, &att.CreatedAt); err != nil {
			return nil, err
		}
		atts = append(atts, att)
	}
	return atts, rows.Err()
}
