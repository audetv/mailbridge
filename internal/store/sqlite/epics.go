package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

const epicColumns = `id, project_id, name, description, number, status, created_at, updated_at`

var epicStatuses = map[string]bool{"open": true, "in_progress": true, "done": true}

// CreateEpic создаёт модуль проекта (имя обязательно; номер — если не задан, max+1 по проекту).
func (s *Store) CreateEpic(ctx context.Context, e *store.Epic) error {
	name := strings.TrimSpace(e.Name)
	if name == "" {
		return fmt.Errorf("epic name is required")
	}
	if e.ProjectID == 0 {
		return fmt.Errorf("epic project_id is required")
	}
	e.Status = strings.TrimSpace(e.Status)
	if !epicStatuses[e.Status] {
		e.Status = "open"
	}

	if proj, err := s.GetProject(ctx, e.ProjectID); err != nil {
		return fmt.Errorf("project %d not found: %w", e.ProjectID, err)
	} else if proj == nil {
		return fmt.Errorf("project %d not found", e.ProjectID)
	}

	if e.Number <= 0 {
		var maxNum sql.NullInt64
		if err := s.db.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(number), 0) FROM epics WHERE project_id = ?`, e.ProjectID).
			Scan(&maxNum); err != nil {
			return fmt.Errorf("failed to compute epic number: %w", err)
		}
		e.Number = int(maxNum.Int64) + 1
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO epics (project_id, name, description, number, status) VALUES (?, ?, ?, ?, ?)`,
		e.ProjectID, name, e.Description, e.Number, e.Status)
	if err != nil {
		return fmt.Errorf("failed to create epic %q: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get epic id: %w", err)
	}
	e.ID = id
	return nil
}

// GetEpic возвращает модуль по ID (nil при отсутствии).
func (s *Store) GetEpic(ctx context.Context, id int64) (*store.Epic, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+epicColumns+` FROM epics WHERE id = ?`, id)
	return scanEpic(row)
}

// ListEpics возвращает модули проекта по возрастанию номера.
func (s *Store) ListEpics(ctx context.Context, projectID int64) ([]*store.Epic, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+epicColumns+` FROM epics WHERE project_id = ? ORDER BY number ASC, id ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list epics: %w", err)
	}
	defer rows.Close()

	epics := make([]*store.Epic, 0)
	for rows.Next() {
		e, err := scanEpic(rows)
		if err != nil {
			return nil, err
		}
		epics = append(epics, e)
	}
	return epics, rows.Err()
}

// UpdateEpic меняет имя, описание и/или статус модуля.
func (s *Store) UpdateEpic(ctx context.Context, id int64, name, description, status string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("epic name cannot be empty")
	}
	status = strings.TrimSpace(status)
	if !epicStatuses[status] {
		return fmt.Errorf("invalid epic status %q", status)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE epics SET name = ?, description = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, description, status, id)
	if err != nil {
		return fmt.Errorf("failed to update epic %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check epic update: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("epic %d not found", id)
	}
	return nil
}

// DeleteEpic удаляет модуль; задачи остаются (epic_id → NULL, ON DELETE SET NULL).
func (s *Store) DeleteEpic(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM epics WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete epic %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check epic delete: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("epic %d not found", id)
	}
	return nil
}

// CountTasksInEpic возвращает число задач в модуле.
func (s *Store) CountTasksInEpic(ctx context.Context, epicID int64) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE epic_id = ?`, epicID).Scan(&n); err != nil {
		return 0, fmt.Errorf("failed to count epic tasks: %w", err)
	}
	return n, nil
}

// EpicProgress считает задачи эпика: done = completed|closed, open = остальные.
func (s *Store) EpicProgress(ctx context.Context, epicID int64) (*store.EpicProgress, error) {
	var p store.EpicProgress
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status IN ('completed', 'closed') THEN 1 ELSE 0 END), 0)
		FROM tasks WHERE epic_id = ?`, epicID).Scan(&p.Total, &p.Done); err != nil {
		return nil, fmt.Errorf("failed to compute epic progress: %w", err)
	}
	p.Open = p.Total - p.Done
	return &p, nil
}

// SetTaskEpic привязывает задачу к модулю; epicID = 0 — отвязать (epic_id = NULL).
func (s *Store) SetTaskEpic(ctx context.Context, taskID, epicID int64) error {
	var val interface{}
	if epicID != 0 {
		val = epicID
	}
	res, err := s.db.ExecContext(ctx, `UPDATE tasks SET epic_id = ? WHERE id = ?`, val, taskID)
	if err != nil {
		return fmt.Errorf("failed to set task epic: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check task epic update: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("task %d not found", taskID)
	}
	return nil
}

// scanEpic сканирует строку таблиц epics в Epic (nil при sql.ErrNoRows).
func scanEpic(row interface{ Scan(...interface{}) error }) (*store.Epic, error) {
	e := &store.Epic{}
	if err := row.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Description, &e.Number, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan epic: %w", err)
	}
	return e, nil
}
