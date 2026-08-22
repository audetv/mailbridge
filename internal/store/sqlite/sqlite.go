// Package sqlite реализует интерфейс Store для SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

	// Пытаемся включить ICU для корректной работы LOWER() с кириллицей
	// Если ICU недоступен — продолжаем без него, поиск по кириллице будет чувствителен к регистру
	_, _ = db.Exec("SELECT icu_load_collation('ru_RU', 'ru')")

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &Store{db: db}, nil
}

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
	}

	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if err := s.migrateSchema(ctx); err != nil {
		return fmt.Errorf("schema migration failed: %w", err)
	}

	// Удаляем старую таблицу после миграции данных
	if err := s.dropEmailMapping(ctx); err != nil {
		return fmt.Errorf("failed to drop email_mapping: %w", err)
	}

	// Миграция данных из email_mapping в inbox_items (однократно)
	if err := s.migrateEmailMappingToInbox(ctx); err != nil {
		return fmt.Errorf("failed to migrate email_mapping: %w", err)
	}

	return nil
}

// dropEmailMapping удаляет старую таблицу email_mapping после переноса данных.
func (s *Store) dropEmailMapping(ctx context.Context) error {
	exists, err := s.TableExists(ctx, "email_mapping")
	if err != nil || !exists {
		return nil
	}

	_, err = s.db.ExecContext(ctx, "DROP TABLE email_mapping")
	if err != nil {
		return fmt.Errorf("failed to drop email_mapping: %w", err)
	}
	return nil
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

	// Индексы для новых колонок
	indexMigrations := []string{
		`CREATE INDEX IF NOT EXISTS idx_tasks_thread_id ON tasks(thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_source_email_id ON tasks(source_email_id)`,
	}
	for _, idx := range indexMigrations {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
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

// CreateTask создаёт новую задачу.
func (s *Store) CreateTask(ctx context.Context, task *store.Task) error {
	query := `INSERT INTO tasks (message_id, subject, body_text, body_html, from_email, from_name, project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		task.MessageID, task.Subject, task.BodyText, task.BodyHTML,
		task.FromEmail, task.FromName, task.Project, task.Type,
		task.Priority, task.Status, task.Assignee,
		task.ThreadID, task.SourceEmailID, task.AIVerdict)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	id, _ := result.LastInsertId()
	task.ID = id
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	return nil
}

// GetTask возвращает задачу по ID.
func (s *Store) GetTask(ctx context.Context, id int64) (*store.Task, error) {
	query := `SELECT id, message_id, subject, body_text, body_html, from_email, from_name,
		project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict, created_at, updated_at
		FROM tasks WHERE id = ?`

	row := s.db.QueryRowContext(ctx, query, id)
	return scanTask(row)
}

// GetTaskByMessageID возвращает задачу по Message-ID.
func (s *Store) GetTaskByMessageID(ctx context.Context, messageID string) (*store.Task, error) {
	query := `SELECT id, message_id, subject, body_text, body_html, from_email, from_name,
		project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict, created_at, updated_at
		FROM tasks WHERE message_id = ?`

	row := s.db.QueryRowContext(ctx, query, messageID)
	return scanTask(row)
}

// ListTasks возвращает список задач с фильтрацией, пагинацией и счётчиком непрочитанных.
func (s *Store) ListTasks(ctx context.Context, filter *store.TaskFilter) (*store.TaskListResult, error) {
	if filter == nil {
		filter = &store.TaskFilter{Page: 1, PerPage: 50}
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 200 {
		filter.PerPage = 50
	}

	var conditions []string
	var args []interface{}

	if filter.Project != "" {
		conditions = append(conditions, "t.project = ?")
		args = append(args, filter.Project)
	}
	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			placeholders[i] = "?"
			args = append(args, s)
		}
		conditions = append(conditions, fmt.Sprintf("t.status IN (%s)", strings.Join(placeholders, ",")))
	}
	if filter.Assignee != "" {
		conditions = append(conditions, "t.assignee = ?")
		args = append(args, filter.Assignee)
	}
	if filter.Type != "" {
		conditions = append(conditions, "t.type = ?")
		args = append(args, filter.Type)
	}
	if filter.Priority != "" {
		conditions = append(conditions, "t.priority = ?")
		args = append(args, filter.Priority)
	}
	if filter.Search != "" {
		conditions = append(conditions, "(LOWER(t.subject) LIKE LOWER(?) OR LOWER(t.body_text) LIKE LOWER(?) OR LOWER(t.from_email) LIKE LOWER(?))")
		search := "%" + filter.Search + "%"
		args = append(args, search, search, search)
	}

	username := filter.Username
	if username == "" {
		username = ""
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks t %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	dataQuery := fmt.Sprintf(`SELECT t.id, t.message_id, t.subject, t.body_text, t.body_html, t.from_email, t.from_name,
		t.project, t.type, t.priority, t.status, t.assignee, t.thread_id, t.source_email_id, t.ai_verdict, t.created_at, t.updated_at,
		(SELECT COUNT(*) FROM task_comments tc 
		 WHERE tc.task_id = t.id 
		 AND tc.direction = 'in' 
		 AND tc.created_at > COALESCE(
		   (SELECT read_at FROM task_reads WHERE task_id = t.id AND username = ?1), 
		   '1970-01-01')
		) + 
		CASE WHEN (SELECT read_at FROM task_reads WHERE task_id = t.id AND username = ?1) IS NULL THEN 1 ELSE 0 END
		as unread_comments
		FROM tasks t %s ORDER BY t.created_at DESC LIMIT ? OFFSET ?`, where)

	dataArgs := append([]interface{}{username}, args...)
	dataArgs = append(dataArgs, filter.PerPage, offset)

	rows, err := s.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*store.TaskWithUnread
	for rows.Next() {
		task := &store.Task{}
		unread := 0
		err := rows.Scan(&task.ID, &task.MessageID, &task.Subject, &task.BodyText, &task.BodyHTML,
			&task.FromEmail, &task.FromName, &task.Project, &task.Type, &task.Priority, &task.Status, &task.Assignee,
			&task.ThreadID, &task.SourceEmailID, &task.AIVerdict, &task.CreatedAt, &task.UpdatedAt, &unread)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &store.TaskWithUnread{Task: task, UnreadComments: unread})
	}

	return &store.TaskListResult{
		Tasks:   tasks,
		Total:   total,
		Page:    filter.Page,
		PerPage: filter.PerPage,
	}, rows.Err()
}

// UpdateTask обновляет поля задачи.
func (s *Store) UpdateTask(ctx context.Context, id int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	var setClauses []string
	var args []interface{}

	for field, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", field))
		args = append(args, value)
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE tasks SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}

// AddTaskComment добавляет комментарий к задаче.
func (s *Store) AddTaskComment(ctx context.Context, comment *store.TaskComment) error {
	query := `INSERT INTO task_comments (task_id, author, body, direction) VALUES (?, ?, ?, ?)`
	result, err := s.db.ExecContext(ctx, query, comment.TaskID, comment.Author, comment.Body, comment.Direction)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	id, _ := result.LastInsertId()
	comment.ID = id
	comment.CreatedAt = time.Now()
	return nil
}

// GetTaskComments возвращает комментарии к задаче.
func (s *Store) GetTaskComments(ctx context.Context, taskID int64) ([]*store.TaskComment, error) {
	query := `SELECT id, task_id, author, body, direction, created_at
		FROM task_comments WHERE task_id = ? ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	var comments []*store.TaskComment
	for rows.Next() {
		c := &store.TaskComment{}
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &c.Direction, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// AddTaskAttachment добавляет вложение к задаче.
func (s *Store) AddTaskAttachment(ctx context.Context, att *store.TaskAttachment) error {
	query := `INSERT INTO task_attachments (task_id, filename, content_type, size, storage_path) VALUES (?, ?, ?, ?, ?)`
	result, err := s.db.ExecContext(ctx, query, att.TaskID, att.Filename, att.ContentType, att.Size, att.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to add attachment: %w", err)
	}

	id, _ := result.LastInsertId()
	att.ID = id
	att.CreatedAt = time.Now()
	return nil
}

// GetTaskAttachments возвращает вложения задачи.
func (s *Store) GetTaskAttachments(ctx context.Context, taskID int64) ([]*store.TaskAttachment, error) {
	query := `SELECT id, task_id, filename, content_type, size, storage_path, created_at
		FROM task_attachments WHERE task_id = ? ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*store.TaskAttachment
	for rows.Next() {
		a := &store.TaskAttachment{}
		if err := rows.Scan(&a.ID, &a.TaskID, &a.Filename, &a.ContentType, &a.Size, &a.StoragePath, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

// SaveReplyLog сохраняет запись об отправленном ответе.
func (s *Store) SaveReplyLog(ctx context.Context, log *store.ReplyLog) error {
	query := `INSERT INTO reply_log (message_id, in_reply_to, plane_issue_id) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, log.MessageID, log.InReplyTo, log.PlaneIssueID)
	return err
}

// ReplyExists проверяет существование ответа.
func (s *Store) ReplyExists(ctx context.Context, msgID string) (bool, error) {
	query := `SELECT COUNT(*) FROM reply_log WHERE message_id = ?`
	var count int
	err := s.db.QueryRowContext(ctx, query, msgID).Scan(&count)
	return count > 0, err
}

// EnqueueOutbox добавляет письмо в очередь.
func (s *Store) EnqueueOutbox(ctx context.Context, payload string) error {
	query := `INSERT INTO outbox (payload) VALUES (?)`
	_, err := s.db.ExecContext(ctx, query, payload)
	return err
}

// GetPendingOutbox возвращает pending-элементы очереди.
func (s *Store) GetPendingOutbox(ctx context.Context, limit int) ([]*store.OutboxItem, error) {
	query := `SELECT id, payload, status, attempts, last_attempt_at, created_at
		FROM outbox WHERE status = 'pending' ORDER BY created_at ASC LIMIT ?`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*store.OutboxItem
	for rows.Next() {
		item := &store.OutboxItem{}
		var lastAttempt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Payload, &item.Status, &item.Attempts, &lastAttempt, &item.CreatedAt); err != nil {
			return nil, err
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
	_, err := s.db.ExecContext(ctx, "UPDATE outbox SET status = 'sent', last_attempt_at = ? WHERE id = ?", time.Now(), id)
	return err
}

// MarkOutboxFailed помечает элемент очереди как ошибочный.
func (s *Store) MarkOutboxFailed(ctx context.Context, id int64, _ string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE outbox SET status = 'failed', attempts = attempts + 1, last_attempt_at = ? WHERE id = ?", time.Now(), id)
	return err
}

// MarkTaskRead отмечает задачу прочитанной пользователем.
func (s *Store) MarkTaskRead(ctx context.Context, taskID int64, username string) error {
	query := `INSERT OR IGNORE INTO task_reads (task_id, username) VALUES (?, ?)`
	_, err := s.db.ExecContext(ctx, query, taskID, username)
	return err
}

// ResetTaskReads сбрасывает статус прочтения для задачи (при новом входящем комментарии).
func (s *Store) ResetTaskReads(ctx context.Context, taskID int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM task_reads WHERE task_id = ?", taskID)
	return err
}

// Ping проверяет соединение с БД.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close закрывает соединение с БД.
func (s *Store) Close() error {
	return s.db.Close()
}

// TableExists проверяет существование таблицы.
func (s *Store) TableExists(ctx context.Context, table string) (bool, error) {
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?`
	var count int
	err := s.db.QueryRowContext(ctx, query, table).Scan(&count)
	return count > 0, err
}

// scanTask сканирует строку в Task.
func scanTask(row interface{ Scan(...interface{}) error }) (*store.Task, error) {
	t := &store.Task{}
	err := row.Scan(&t.ID, &t.MessageID, &t.Subject, &t.BodyText, &t.BodyHTML,
		&t.FromEmail, &t.FromName, &t.Project, &t.Type, &t.Priority, &t.Status, &t.Assignee,
		&t.ThreadID, &t.SourceEmailID, &t.AIVerdict, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}
	return t, nil
}

// CreateThread создаёт новую цепочку писем.
func (s *Store) CreateThread(ctx context.Context, thread *store.Thread) error {
	query := `INSERT INTO threads (thread_id, source, subject, participants, summary, last_item_at) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := s.db.ExecContext(ctx, query,
		thread.ThreadID, thread.Source, thread.Subject, thread.Participants, thread.Summary, thread.LastItemAt)
	if err != nil {
		return fmt.Errorf("failed to create thread: %w", err)
	}
	id, _ := result.LastInsertId()
	thread.ID = id
	thread.CreatedAt = time.Now()
	thread.UpdatedAt = time.Now()
	return nil
}

// GetThread возвращает цепочку по thread_id.
func (s *Store) GetThread(ctx context.Context, threadID string) (*store.Thread, error) {
	query := `SELECT id, thread_id, source, subject, participants, summary, last_item_at, created_at, updated_at FROM threads WHERE thread_id = ?`
	row := s.db.QueryRowContext(ctx, query, threadID)

	t := &store.Thread{}
	var lastItemAt sql.NullTime
	err := row.Scan(&t.ID, &t.ThreadID, &t.Source, &t.Subject, &t.Participants, &t.Summary, &lastItemAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan thread: %w", err)
	}
	if lastItemAt.Valid {
		t.LastItemAt = &lastItemAt.Time
	}
	return t, nil
}

// UpdateThreadSummary обновляет summary цепочки.
func (s *Store) UpdateThreadSummary(ctx context.Context, threadID, summary string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE threads SET summary = ?, updated_at = ? WHERE thread_id = ?",
		summary, time.Now(), threadID)
	return err
}

// GetActiveTasksByThread возвращает активные задачи цепочки.
func (s *Store) GetActiveTasksByThread(ctx context.Context, threadID string) ([]*store.Task, error) {
	query := `SELECT id, message_id, subject, body_text, body_html, from_email, from_name,
		project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict, created_at, updated_at
		FROM tasks WHERE thread_id = ? AND status IN ('new', 'in_progress', 'resolved', 'info_only') ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks: %w", err)
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

// migrateEmailMappingToInbox переносит данные из старой таблицы email_mapping в inbox_items.
func (s *Store) migrateEmailMappingToInbox(ctx context.Context) error {
	// Проверяем что email_mapping существует
	exists, err := s.TableExists(ctx, "email_mapping")
	if err != nil || !exists {
		return nil
	}

	// Переносим данные (INSERT OR IGNORE для идемпотентности)
	query := `INSERT OR IGNORE INTO inbox_items 
		(source, source_id, thread_id, from_contact, from_name, subject, body_text, meta, received_at, status)
		SELECT 
			'email',
			message_id,
			message_id,  -- thread_id = message_id (нет данных о цепочке в старой схеме)
			original_from,
			'',
			original_subject,
			'',
			'{}',
			created_at,
			'read'  -- старые письма считаем прочитанными
		FROM email_mapping`

	_, err = s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to migrate email_mapping: %w", err)
	}

	return nil
}

// MigrateEmailMappingToInboxForTest — экспортируемая обёртка для тестов.
func (s *Store) MigrateEmailMappingToInboxForTest(ctx context.Context) error {
	return s.migrateEmailMappingToInbox(ctx)
}

// QueryRowForTest — экспортируемый метод для тестов.
func (s *Store) QueryRowForTest(ctx context.Context, query string) *sql.Row {
	return s.db.QueryRowContext(ctx, query)
}
