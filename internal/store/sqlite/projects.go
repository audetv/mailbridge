package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

const projectColumns = `id, name, description, archived, created_at, updated_at`

// CreateProject создаёт проект (имя UNIQUE; повтор с тем же именем — ошибка).
func (s *Store) CreateProject(ctx context.Context, p *store.Project) error {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (name, description, archived) VALUES (?, ?, ?)`,
		name, p.Description, boolToInt(p.Archived))
	if err != nil {
		return fmt.Errorf("failed to create project %q: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get project id: %w", err)
	}
	p.ID = id
	return nil
}

// GetProject возвращает проект по ID (nil при отсутствии).
func (s *Store) GetProject(ctx context.Context, id int64) (*store.Project, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE id = ?`, id)
	return scanProject(row)
}

// GetProjectByName возвращает проект по имени (nil при отсутствии).
func (s *Store) GetProjectByName(ctx context.Context, name string) (*store.Project, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE name = ?`, strings.TrimSpace(name))
	return scanProject(row)
}

// ListProjects возвращает список проектов с фильтрами.
func (s *Store) ListProjects(ctx context.Context, filter *store.ProjectFilter) ([]*store.Project, error) {
	if filter == nil {
		filter = &store.ProjectFilter{}
	}

	var conds []string
	var args []interface{}
	if filter.Archived != nil {
		conds = append(conds, "archived = ?")
		args = append(args, boolToInt(*filter.Archived))
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		conds = append(conds, "name LIKE ?")
		args = append(args, "%"+search+"%")
	}

	query := `SELECT ` + projectColumns + ` FROM projects`
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY archived ASC, name COLLATE NOCASE ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]*store.Project, 0)
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// UpdateProject переименовывает проект и/или меняет описание.
func (s *Store) UpdateProject(ctx context.Context, id int64, name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, description, id)
	if err != nil {
		return fmt.Errorf("failed to update project %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check project update: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("project %d not found", id)
	}
	return nil
}

// SetProjectArchived отмечает проект как архивный (soft-archive) или возвращает.
func (s *Store) SetProjectArchived(ctx context.Context, id int64, archived bool) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET archived = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		boolToInt(archived), id)
	if err != nil {
		return fmt.Errorf("failed to archive project %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check project archive: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("project %d not found", id)
	}
	return nil
}

// scanProject сканирует строку таблиц projects в Project (nil при sql.ErrNoRows).
func scanProject(row interface{ Scan(...interface{}) error }) (*store.Project, error) {
	p := &store.Project{}
	var archived int
	err := row.Scan(&p.ID, &p.Name, &p.Description, &archived, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}
	p.Archived = archived != 0
	return p, nil
}

// boolToInt приводит bool к sqlite-integer.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
