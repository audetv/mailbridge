package sqlite

import (
	"context"
	"fmt"

	"github.com/audetv/mailbridge/internal/store"
)

// GetInboxItemsByThread возвращает все входящие цепочки.
func (s *Store) GetInboxItemsByThread(ctx context.Context, threadID string) ([]*store.InboxItem, error) {
	query := `SELECT id, source, source_id, thread_id, from_contact, from_name, subject, body_text, body_html, meta, received_at, ai_processed, ai_attempts, ai_verdict, ai_summary, status
		FROM inbox_items WHERE thread_id = ? ORDER BY received_at ASC`

	rows, err := s.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inbox items by thread: %w", err)
	}
	defer rows.Close()

	var items []*store.InboxItem
	for rows.Next() {
		item, err := scanInboxItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetTasksByThread возвращает все задачи цепочки.
func (s *Store) GetTasksByThread(ctx context.Context, threadID string) ([]*store.Task, error) {
	query := `SELECT id, message_id, subject, body_text, body_html, from_email, from_name,
		project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict, created_at, updated_at
		FROM tasks WHERE thread_id = ? ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by thread: %w", err)
	}
	defer rows.Close()

	var tasks []*store.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}
