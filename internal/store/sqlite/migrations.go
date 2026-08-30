// Package sqlite реализует интерфейс Store для SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// Migrate выполняет миграции схемы.
func (s *Store) Migrate(ctx context.Context) error {
	migrations := []string{
		// Таблица задач
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id TEXT NOT NULL UNIQUE,
			subject TEXT NOT NULL,
			body_text TEXT NOT NULL DEFAULT '',
			body_html TEXT NOT NULL DEFAULT '',
			from_email TEXT NOT NULL,
			from_name TEXT NOT NULL DEFAULT '',
			project TEXT NOT NULL DEFAULT 'Входящие',
			type TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL DEFAULT 'medium',
			status TEXT NOT NULL DEFAULT 'new',
			assignee TEXT NOT NULL DEFAULT '',
			thread_id TEXT NOT NULL DEFAULT '',
			source_email_id TEXT NOT NULL DEFAULT '',
			ai_verdict TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_message_id ON tasks(message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_assignee ON tasks(assignee)`,

		// Таблица комментариев
		`CREATE TABLE IF NOT EXISTS task_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			author TEXT NOT NULL,
			body TEXT NOT NULL,
			direction TEXT NOT NULL DEFAULT 'in',
			kind TEXT NOT NULL DEFAULT 'user_comment',
			inbox_item_id INTEGER REFERENCES inbox_items(id),
			verdict_json TEXT,
			approved INTEGER,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_comments_task_id ON task_comments(task_id)`,

		// Таблица вложений
		`CREATE TABLE IF NOT EXISTS task_attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			filename TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			storage_path TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_attachments_task_id ON task_attachments(task_id)`,

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

		`CREATE TABLE IF NOT EXISTS task_reads (
			task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			username TEXT NOT NULL,
			read_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (task_id, username)
		)`,

		`CREATE TABLE IF NOT EXISTS threads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			thread_id TEXT NOT NULL UNIQUE,
			source TEXT NOT NULL DEFAULT 'email',
			subject TEXT NOT NULL DEFAULT '',
			participants TEXT NOT NULL DEFAULT '[]',
			summary TEXT NOT NULL DEFAULT '',
			last_item_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_threads_thread_id ON threads(thread_id)`,

		`CREATE TABLE IF NOT EXISTS inbox_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL DEFAULT 'email',
			source_id TEXT NOT NULL,
			thread_id TEXT NOT NULL DEFAULT '',
			from_contact TEXT NOT NULL DEFAULT '',
			from_name TEXT NOT NULL DEFAULT '',
			subject TEXT NOT NULL DEFAULT '',
			body_text TEXT NOT NULL DEFAULT '',
			body_html TEXT NOT NULL DEFAULT '',
			meta TEXT NOT NULL DEFAULT '{}',
			received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			ai_processed INTEGER NOT NULL DEFAULT 0,
			ai_attempts INTEGER NOT NULL DEFAULT 0,
			ai_verdict TEXT NOT NULL DEFAULT '[]',
			ai_summary TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'unread',
			UNIQUE(source, source_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_inbox_items_thread_id ON inbox_items(thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_inbox_items_status ON inbox_items(status)`,
		`CREATE INDEX IF NOT EXISTS idx_inbox_items_ai_processed ON inbox_items(ai_processed)`,

		`CREATE TABLE IF NOT EXISTS task_inbox_items (
			task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			inbox_item_id INTEGER NOT NULL REFERENCES inbox_items(id) ON DELETE CASCADE,
			relation TEXT NOT NULL DEFAULT 'created_from',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (task_id, inbox_item_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_inbox_items_task_id ON task_inbox_items(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_task_inbox_items_inbox_item_id ON task_inbox_items(inbox_item_id)`,

		// Таблица проектов (v0.22.0): иерархия Проект → Модуль (эпик) → Задача
		`CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			archived INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name)`,

		// Модули (эпики) v0.22.0: проект → модуль → задача
		`CREATE TABLE IF NOT EXISTS epics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			number INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'open',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, number)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_epics_project_id ON epics(project_id)`,

		// Универсальная таблица вложений
		`CREATE TABLE IF NOT EXISTS attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT NOT NULL UNIQUE,
			filename TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			storage_path TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_hash ON attachments(hash)`,

		// Связь входящих с вложениями
		`CREATE TABLE IF NOT EXISTS inbox_attachments (
			inbox_item_id INTEGER NOT NULL REFERENCES inbox_items(id) ON DELETE CASCADE,
			attachment_id INTEGER NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
			PRIMARY KEY (inbox_item_id, attachment_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_inbox_attachments_inbox ON inbox_attachments(inbox_item_id)`,
		`CREATE INDEX IF NOT EXISTS idx_inbox_attachments_att ON inbox_attachments(attachment_id)`,

		// Связь задач с вложениями (обновлённая)
		`CREATE TABLE IF NOT EXISTS task_attachments (
			task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			attachment_id INTEGER NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
			PRIMARY KEY (task_id, attachment_id)
		)`,

		// Связь комментариев с вложениями (на будущее)
		`CREATE TABLE IF NOT EXISTS comment_attachments (
			comment_id INTEGER NOT NULL REFERENCES task_comments(id) ON DELETE CASCADE,
			attachment_id INTEGER NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
			PRIMARY KEY (comment_id, attachment_id)
		)`,
	}

	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if err := s.migrateSchema(ctx); err != nil {
		return fmt.Errorf("schema migration failed: %w", err)
	}

	if err := s.seedProjects(ctx); err != nil {
		return fmt.Errorf("project seed failed: %w", err)
	}

	return nil
}

// seedProjects наполняет таблицу projects исходным набором:
// «Входящие» (fallback-проект) + проекты задач (distinct tasks.project).
// Идемпотентно: INSERT OR IGNORE по UNIQUE(name), повторный запуск не меняет данные.
func (s *Store) seedProjects(ctx context.Context) error {
	names, err := s.distinctTaskProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list distinct task projects: %w", err)
	}

	upsert := `INSERT OR IGNORE INTO projects (name, description) VALUES (?, '')`
	for _, name := range append([]string{"Входящие"}, names...) {
		if _, err := s.db.ExecContext(ctx, upsert, name); err != nil {
			return fmt.Errorf("failed to seed project %q: %w", name, err)
		}
	}

	return nil
}

// distinctTaskProjects возвращает уникальные непустые значения tasks.project.
func (s *Store) distinctTaskProjects(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT project FROM tasks
		WHERE project IS NOT NULL AND project <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return names, nil
}

// migrateSchema выполняет миграции для обновления существующих таблиц.
func (s *Store) migrateSchema(ctx context.Context) error {
	// Добавляем AI-колонки в tasks если их нет
	aiColumns := map[string]string{
		"thread_id":       "TEXT NOT NULL DEFAULT ''",
		"source_email_id": "TEXT NOT NULL DEFAULT ''",
		"ai_verdict":      "TEXT NOT NULL DEFAULT ''",
	}
	for col, typ := range aiColumns {
		has, err := s.columnExists(ctx, "tasks", col)
		if err != nil {
			return fmt.Errorf("failed to check column %s: %w", col, err)
		}
		if !has {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE tasks ADD COLUMN %s %s", col, typ)); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col, err)
			}
		}
	}

	// Ссылка задачи на модуль (v0.22.0); nullable, удаление модуля не бьёт по задачам.
	epicColumns := map[string]string{
		"epic_id": "INTEGER REFERENCES epics(id) ON DELETE SET NULL",
	}
	for col, typ := range epicColumns {
		has, err := s.columnExists(ctx, "tasks", col)
		if err != nil {
			return fmt.Errorf("failed to check column %s: %w", col, err)
		}
		if !has {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE tasks ADD COLUMN %s %s", col, typ)); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col, err)
			}
		}
	}

	// Модули: в ранней версии схемы (v0.22 шаг 3) epics создавались без description/status;
	// идемпотентно дособираем колонки, если их нет.
	epicsBackfillColumns := map[string]string{
		"description": "TEXT NOT NULL DEFAULT ''",
		"status":      "TEXT NOT NULL DEFAULT 'open'",
	}
	for col, typ := range epicsBackfillColumns {
		has, err := s.columnExists(ctx, "epics", col)
		if err != nil {
			return fmt.Errorf("failed to check epics column %s: %w", col, err)
		}
		if !has {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE epics ADD COLUMN %s %s", col, typ)); err != nil {
				return fmt.Errorf("failed to add epics column %s: %w", col, err)
			}
		}
	}

	// Добавляем новые колонки в threads если их нет
	threadColumns := map[string]string{
		"source":       "TEXT NOT NULL DEFAULT 'email'",
		"subject":      "TEXT NOT NULL DEFAULT ''",
		"participants": "TEXT NOT NULL DEFAULT '[]'",
		"last_item_at": "TIMESTAMP",
	}
	for col, typ := range threadColumns {
		has, err := s.columnExists(ctx, "threads", col)
		if err != nil {
			return fmt.Errorf("failed to check column %s: %w", col, err)
		}
		if !has {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE threads ADD COLUMN %s %s", col, typ)); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col, err)
			}
		}
	}

	// Миграция старой схемы task_attachments
	if err := s.migrateTaskAttachments(ctx); err != nil {
		return fmt.Errorf("failed to migrate task_attachments: %w", err)
	}

	// Индексы для новых колонок
	indexMigrations := []string{
		`CREATE INDEX IF NOT EXISTS idx_tasks_thread_id ON tasks(thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_source_email_id ON tasks(source_email_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_epic_id ON tasks(epic_id)`,
	}
	for _, idx := range indexMigrations {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Миграция task_comments: удалить старую, создать новую
	if err := s.migrateTaskComments(ctx); err != nil {
		return fmt.Errorf("failed to migrate task_comments: %w", err)
	}

	// task_comments.approved — модерационный флаг (ФАЗА 4, 2026-08-30):
	// approved 0/1 на комментарии, NULL = не утверждён. Для существующих БД — backfill.
	commentApprovedColumns := map[string]string{
		"approved": "INTEGER",
	}
	for col, typ := range commentApprovedColumns {
		has, err := s.columnExists(ctx, "task_comments", col)
		if err != nil {
			return fmt.Errorf("failed to check task_comments column %s: %w", col, err)
		}
		if !has {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE task_comments ADD COLUMN %s %s", col, typ)); err != nil {
				return fmt.Errorf("failed to add task_comments column %s: %w", col, err)
			}
		}
	}

	return nil
}

// migrateTaskAttachments переносит данные из старой схемы task_attachments в новую.
func (s *Store) migrateTaskAttachments(ctx context.Context) error {
	// Проверяем что старая схема существует (с колонкой filename)
	hasFilename, err := s.columnExists(ctx, "task_attachments", "filename")
	if err != nil {
		return err
	}
	if !hasFilename {
		return nil // уже новая схема
	}

	// 1. Создаём временную таблицу связей
	createTemp := `CREATE TABLE IF NOT EXISTS temp_task_attachments (
		task_id INTEGER NOT NULL,
		storage_path TEXT NOT NULL
	)`
	if _, err := s.db.ExecContext(ctx, createTemp); err != nil {
		return err
	}

	// 2. Копируем связи во временную
	copyTemp := `INSERT INTO temp_task_attachments (task_id, storage_path)
		SELECT task_id, storage_path FROM task_attachments`
	if _, err := s.db.ExecContext(ctx, copyTemp); err != nil {
		return err
	}

	// 3. Удаляем старую таблицу
	if _, err := s.db.ExecContext(ctx, "DROP TABLE task_attachments"); err != nil {
		return err
	}

	// 4. Создаём новую таблицу связей
	createNew := `CREATE TABLE IF NOT EXISTS task_attachments (
		task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		attachment_id INTEGER NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
		PRIMARY KEY (task_id, attachment_id)
	)`
	if _, err := s.db.ExecContext(ctx, createNew); err != nil {
		return err
	}

	// 5. Переносим файлы в attachments (временный hash = storage_path)
	insertAtt := `INSERT OR IGNORE INTO attachments (hash, filename, content_type, size, storage_path)
		SELECT storage_path, storage_path, '', 0, storage_path
		FROM temp_task_attachments`
	if _, err := s.db.ExecContext(ctx, insertAtt); err != nil {
		return err
	}

	// 6. Восстанавливаем связи
	linkQuery := `INSERT OR IGNORE INTO task_attachments (task_id, attachment_id)
		SELECT t.task_id, a.id
		FROM temp_task_attachments t
		JOIN attachments a ON a.storage_path = t.storage_path`
	if _, err := s.db.ExecContext(ctx, linkQuery); err != nil {
		return err
	}

	// 7. Удаляем временную таблицу
	if _, err := s.db.ExecContext(ctx, "DROP TABLE temp_task_attachments"); err != nil {
		return err
	}

	// 8. Создаём индексы
	if _, err := s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_task_attachments_task ON task_attachments(task_id)"); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_task_attachments_att ON task_attachments(attachment_id)"); err != nil {
		return err
	}

	return nil
}

// columnExists проверяет существование колонки в таблице.
func (s *Store) columnExists(ctx context.Context, table, column string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// ColumnExistsForTest — экспортируемая обёртка для тестов.
func (s *Store) ColumnExistsForTest(ctx context.Context, table, column string) (bool, error) {
	return s.columnExists(ctx, table, column)
}

// migrateTaskComments удаляет старую таблицу и создаёт новую.
func (s *Store) migrateTaskComments(ctx context.Context) error {
	// Проверяем есть ли старая таблица
	hasKind, err := s.columnExists(ctx, "task_comments", "kind")
	if err == nil && hasKind {
		return nil // уже новая схема
	}

	// Удаляем старую
	if _, err := s.db.ExecContext(ctx, "DROP TABLE IF EXISTS task_comments"); err != nil {
		return err
	}

	// Создаём новую
	createNew := `CREATE TABLE IF NOT EXISTS task_comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		author TEXT NOT NULL,
		body TEXT NOT NULL,
		direction TEXT NOT NULL DEFAULT 'in',
		kind TEXT NOT NULL DEFAULT 'user_comment',
		inbox_item_id INTEGER REFERENCES inbox_items(id),
		verdict_json TEXT,
		approved INTEGER,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := s.db.ExecContext(ctx, createNew); err != nil {
		return err
	}

	// Индексы
	if _, err := s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_task_comments_task_id ON task_comments(task_id)"); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_task_comments_inbox_item ON task_comments(inbox_item_id)"); err != nil {
		return err
	}

	return nil
}
