// Package sqlite реализует интерфейс Store для SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // драйвер SQLite

	"github.com/audetv/mailbridge/internal/extractor"
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

// CreateInboxItem создаёт новый элемент ленты.
func (s *Store) CreateInboxItem(ctx context.Context, item *store.InboxItem) error {
	query := `INSERT INTO inbox_items 
		(source, source_id, thread_id, from_contact, from_name, subject, body_text, body_html, meta, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		item.Source, item.SourceID, item.ThreadID,
		item.FromContact, item.FromName, item.Subject,
		item.BodyText, item.BodyHTML, item.Meta, item.Status)
	if err != nil {
		return fmt.Errorf("failed to create inbox item: %w", err)
	}

	id, _ := result.LastInsertId()
	item.ID = id
	item.ReceivedAt = time.Now()
	return nil
}

// GetInboxItemByID возвращает элемент ленты по ID.
func (s *Store) GetInboxItemByID(ctx context.Context, id int64) (*store.InboxItem, error) {
	query := `SELECT id, source, source_id, thread_id, from_contact, from_name, subject, body_text, body_html, meta, received_at, ai_processed, ai_attempts, ai_verdict, ai_summary, status
		FROM inbox_items WHERE id = ?`

	row := s.db.QueryRowContext(ctx, query, id)
	return scanInboxItem(row)
}

// GetInboxItemBySourceID возвращает элемент ленты по source и source_id.
func (s *Store) GetInboxItemBySourceID(ctx context.Context, source, sourceID string) (*store.InboxItem, error) {
	query := `SELECT id, source, source_id, thread_id, from_contact, from_name, subject, body_text, body_html, meta, received_at, ai_processed, ai_attempts, ai_verdict, ai_summary, status
		FROM inbox_items WHERE source = ? AND source_id = ?`

	row := s.db.QueryRowContext(ctx, query, source, sourceID)
	return scanInboxItem(row)
}

// ListInboxItems возвращает список элементов ленты.
func (s *Store) ListInboxItems(ctx context.Context, filter *store.InboxFilter) (*store.InboxListResult, error) {
	if filter == nil {
		filter = &store.InboxFilter{Page: 1, PerPage: 50}
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 200 {
		filter.PerPage = 50
	}

	var conditions []string
	var args []interface{}

	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, filter.Source)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM inbox_items %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count inbox items: %w", err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	dataQuery := fmt.Sprintf(`SELECT id, source, source_id, thread_id, from_contact, from_name, subject, body_text, body_html, meta, received_at, ai_processed, ai_attempts, ai_verdict, ai_summary, status
		FROM inbox_items %s ORDER BY received_at DESC LIMIT ? OFFSET ?`, where)

	dataArgs := append(args, filter.PerPage, offset)
	rows, err := s.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list inbox items: %w", err)
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

	return &store.InboxListResult{
		Items:   items,
		Total:   total,
		Page:    filter.Page,
		PerPage: filter.PerPage,
	}, rows.Err()
}

// UpdateInboxItemStatus обновляет статус элемента ленты.
func (s *Store) UpdateInboxItemStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE inbox_items SET status = ? WHERE id = ?", status, id)
	return err
}

// UpdateInboxItemAI обновляет AI-поля элемента ленты.
func (s *Store) UpdateInboxItemAI(ctx context.Context, id int64, processed int, verdict, summary string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE inbox_items SET ai_processed = ?, ai_verdict = ?, ai_summary = ? WHERE id = ?",
		processed, verdict, summary, id)
	return err
}

// scanInboxItem сканирует строку в InboxItem.
func scanInboxItem(row interface{ Scan(...interface{}) error }) (*store.InboxItem, error) {
	item := &store.InboxItem{}
	err := row.Scan(&item.ID, &item.Source, &item.SourceID, &item.ThreadID,
		&item.FromContact, &item.FromName, &item.Subject,
		&item.BodyText, &item.BodyHTML, &item.Meta, &item.ReceivedAt,
		&item.AIProcessed, &item.AIAttempts, &item.AIVerdict, &item.AISummary, &item.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan inbox item: %w", err)
	}
	return item, nil
}

// LinkTaskToInboxItem создаёт связь между задачей и элементом ленты.
func (s *Store) LinkTaskToInboxItem(ctx context.Context, taskID, inboxItemID int64, relation string) error {
	query := `INSERT OR IGNORE INTO task_inbox_items (task_id, inbox_item_id, relation) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, taskID, inboxItemID, relation)
	return err
}

// GetInboxItemsByTask возвращает элементы ленты, связанные с задачей.
func (s *Store) GetInboxItemsByTask(ctx context.Context, taskID int64) ([]*store.TaskInboxItem, error) {
	query := `SELECT task_id, inbox_item_id, relation, created_at FROM task_inbox_items WHERE task_id = ? ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*store.TaskInboxItem
	for rows.Next() {
		link := &store.TaskInboxItem{}
		if err := rows.Scan(&link.TaskID, &link.InboxItemID, &link.Relation, &link.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// GetTasksByInboxItem возвращает задачи, связанные с элементом ленты.
func (s *Store) GetTasksByInboxItem(ctx context.Context, inboxItemID int64) ([]*store.TaskInboxItem, error) {
	query := `SELECT task_id, inbox_item_id, relation, created_at FROM task_inbox_items WHERE inbox_item_id = ? ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, query, inboxItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*store.TaskInboxItem
	for rows.Next() {
		link := &store.TaskInboxItem{}
		if err := rows.Scan(&link.TaskID, &link.InboxItemID, &link.Relation, &link.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// CreateTask создаёт новую задачу.
func (s *Store) CreateTask(ctx context.Context, task *store.Task) error {
	query := `INSERT INTO tasks (message_id, subject, body_text, body_html, from_email, from_name, project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict, due_date, due_source, scheduled_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		task.MessageID, task.Subject, task.BodyText, task.BodyHTML,
		task.FromEmail, task.FromName, task.Project, task.Type,
		task.Priority, task.Status, task.Assignee,
		task.ThreadID, task.SourceEmailID, task.AIVerdict,
		task.DueDate, task.DueSource, task.ScheduledDate)
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
		project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict, epic_id, requestor_id, assignee_id, due_date, ai_due_date, due_source, due_ai_pending, scheduled_date, created_at, updated_at
		FROM tasks WHERE id = ?`

	row := s.db.QueryRowContext(ctx, query, id)
	return scanTask(row)
}

// GetTaskByMessageID возвращает задачу по Message-ID.
func (s *Store) GetTaskByMessageID(ctx context.Context, messageID string) (*store.Task, error) {
	query := `SELECT id, message_id, subject, body_text, body_html, from_email, from_name,
		project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict, epic_id, requestor_id, assignee_id, due_date, ai_due_date, due_source, due_ai_pending, scheduled_date, created_at, updated_at
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
	if filter.EpicID != nil {
		conditions = append(conditions, "t.epic_id = ?")
		args = append(args, *filter.EpicID)
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
	// Персоны (шаг 6): фильтр по роли на задаче.
	if filter.RequestorID != nil {
		conditions = append(conditions, "t.requestor_id = ?")
		args = append(args, string(*filter.RequestorID))
	}
	if filter.AssigneeID != nil {
		conditions = append(conditions, "t.assignee_id = ?")
		args = append(args, string(*filter.AssigneeID))
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

	// v0.26, шаг 7d: фильтры по срокам (одно значение за запрос).
	// «Сегодня» — часовой пояс сервера: те же часы, что и time.Now() при
	// записи due_date/updated_at, значит сравнение канонических YYYY-MM-DD
	// согласовано. Невалидные значения игнорируются (API-слой валидирует).
	var today time.Time
	switch filter.Due {
	case "overdue":
		conditions = append(conditions, "t.due_date < ?")
		args = append(args, time.Now().Format("2006-01-02"))
	case "today":
		conditions = append(conditions, "t.due_date = ?")
		args = append(args, time.Now().Format("2006-01-02"))
	case "tomorrow":
		conditions = append(conditions, "t.due_date = ?")
		args = append(args, time.Now().AddDate(0, 0, 1).Format("2006-01-02"))
	case "7d", "30d":
		days, _ := strconv.Atoi(filter.Due[:len(filter.Due)-1])
		if days > 0 {
			today = time.Now()
			conditions = append(conditions, "t.due_date BETWEEN ? AND ?")
			args = append(args, today.Format("2006-01-02"), today.AddDate(0, 0, days).Format("2006-01-02"))
		}
	case "none":
		conditions = append(conditions, "t.due_date IS NULL")
	case "due_pending":
		conditions = append(conditions, "t.due_ai_pending = 1")
	}

	// v0.28, шаг 28b: срез «План» — scheduled в окне + ВСЕГДА просроченные due
	// (due_date < сегодня, только активные статусы new/in_progress/backlog;
	// completed/closed вне плана — решение владельца). Одно OR-сложение
	// в одном условии — не разбивать на несколько AND.
	switch filter.Plan {
	case "today":
		conditions = append(conditions, "(t.scheduled_date = ? OR (t.due_date < ? AND t.status IN ('new','in_progress','backlog')))")
		args = append(args, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	case "tomorrow":
		conditions = append(conditions, "(t.scheduled_date = ? OR (t.due_date < ? AND t.status IN ('new','in_progress','backlog')))")
		args = append(args, time.Now().AddDate(0, 0, 1).Format("2006-01-02"), time.Now().Format("2006-01-02"))
	case "week":
		conditions = append(conditions, "(t.scheduled_date BETWEEN ? AND ? OR (t.due_date < ? AND t.status IN ('new','in_progress','backlog')))")
		args = append(args, time.Now().Format("2006-01-02"), time.Now().AddDate(0, 0, 6).Format("2006-01-02"), time.Now().Format("2006-01-02"))
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

	// v0.25, шаг 7a: серверная сортировка (стабильна через пагинацию).
	// "due" (дефолт): просроченные сверху (due_date ASC — мин. срок = самый просроченный
	// = приоритет), без срока — внизу (NULLS LAST); "updated": по свежести обновлений.
	// Тикер по id — полная детерминированность (без дублей/склеек между страницами).
	orderClause := "ORDER BY t.due_date IS NULL ASC, t.due_date ASC, t.created_at DESC, t.id DESC" // v0.27.1 хотфикс: в группе «без срока» — новые выше (created_at DESC); id DESC — детерминированный ключ для пагинации
	if sortKey := strings.TrimSpace(filter.Sort); sortKey == "updated" {
		orderClause = "ORDER BY t.updated_at DESC, t.id DESC"
	}

	dataQuery := fmt.Sprintf(`SELECT t.id, t.message_id, t.subject, t.body_text, t.body_html, t.from_email, t.from_name,
		t.project, t.type, t.priority, t.status, t.assignee, t.thread_id, t.source_email_id, t.ai_verdict, t.epic_id, t.requestor_id, t.assignee_id, t.due_date, t.ai_due_date, t.due_source, t.due_ai_pending, t.scheduled_date, t.created_at, t.updated_at,
		(SELECT COUNT(*) FROM task_comments tc 
		 WHERE tc.task_id = t.id 
		 AND tc.direction = 'in' 
		 AND tc.kind = 'user_comment'
		 AND tc.created_at > COALESCE(
		   (SELECT read_at FROM task_reads WHERE task_id = t.id AND username = ?1), 
		   '1970-01-01')
		) + 
		CASE WHEN (SELECT read_at FROM task_reads WHERE task_id = t.id AND username = ?1) IS NULL THEN 1 ELSE 0 END
		as unread_comments
		FROM tasks t %s %s LIMIT ? OFFSET ?`, where, orderClause)

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
		var epicID sql.NullInt64
		var reqID sql.NullString
		var asnID sql.NullString
		var dueDate, aiDueDate, dueSource, scheduledDate sql.NullString
		var dueAIPending sql.NullInt64
		err := rows.Scan(&task.ID, &task.MessageID, &task.Subject, &task.BodyText, &task.BodyHTML,
			&task.FromEmail, &task.FromName, &task.Project, &task.Type, &task.Priority, &task.Status, &task.Assignee,
			&task.ThreadID, &task.SourceEmailID, &task.AIVerdict, &epicID, &reqID, &asnID,
			&dueDate, &aiDueDate, &dueSource, &dueAIPending, &scheduledDate,
			&task.CreatedAt, &task.UpdatedAt, &unread)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		if epicID.Valid {
			task.EpicID = &epicID.Int64
		}
		if reqID.Valid {
			pid := store.PersonID(reqID.String)
			task.RequestorID = &pid
		}
		if asnID.Valid {
			pid := store.PersonID(asnID.String)
			task.AssigneeID = &pid
		}
		if dueDate.Valid {
			task.DueDate = &dueDate.String
		}
		if aiDueDate.Valid {
			task.AIDueDate = &aiDueDate.String
		}
		if dueSource.Valid {
			task.DueSource = &dueSource.String
		}
		if dueAIPending.Valid {
			v := dueAIPending.Int64 != 0
			task.DueAIPending = &v
		}
		if scheduledDate.Valid {
			task.ScheduledDate = &scheduledDate.String
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

// SetTaskStatus — единственный путь смены статуса задачи:
// в транзакции читает текущий статус (для истории), обновляет статус
// и пишет строку в task_status_history. Идемпотентен для «статус не изменился»
// (строка не пишется, ошибка не возвращается).
func (s *Store) SetTaskStatus(ctx context.Context, taskID int64, toStatus, by string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Текущий статус: пустая строка — задача не найдена (внешний FK сработал бы
	// при UPDATE, но проверяем явно ради читаемой ошибки/пустой истории).
	var fromStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&fromStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("task %d: %w", taskID, store.ErrTaskNotFound)
		}
		return fmt.Errorf("failed to read task status: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?", toStatus, time.Now(), taskID); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// v0.24, шаг 6 (Персоны): авто-назначение assignee_id = тот, кто подтвердил
	// (режим A). Срабатывает на переходе в закрытый статус; не затирает ручной assign.
	// Best-effort: ошибка auto-assign не фейлит сам переход статуса (история уже важна).
	if isClosedStatus(toStatus) {
		// Best-effort: ошибка auto-assign не фейлит переход статуса (режим A:
		// персоны не критичны — система работает и без них). TODO(step7/8): log hook.
		_ = autoAssignLastConfirmer(ctx, tx, taskID)
	}

	// История: строка пишется только при реальном переходе (смена статуса,
	// включая первый — from_status NULL). Повторный write с тем же статусом
	// строки не даёт (идемпотентность: batch/bulk не заваливают историю).
	if fromStatus != toStatus {
		var fromInterface interface{}
		if fromStatus == "" {
			fromInterface = nil
		} else {
			fromInterface = fromStatus
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_status_history (task_id, from_status, to_status, by) VALUES (?, ?, ?, ?)`,
			taskID, fromInterface, toStatus, by); err != nil {
			return fmt.Errorf("failed to write status history: %w", err)
		}
	}

	return tx.Commit()
}

// SetTaskDueAIPending — v0.25 шаг 7b: AI-предложение срока задачи.
// ai_due_date хранится ВСЕГДА (в т.ч. NULL — «AI срока не извлек»: база ошибок,
// шаг 8 метрики), due_ai_pending=1 — предложение ждёт решения человека (7c).
// due_date/due_source НЕ изменяются: решение человека только принимает срок (7c).
func (s *Store) SetTaskDueAIPending(ctx context.Context, taskID int64, aiDueDate *string, pending bool) error {
	p := 0
	if pending {
		p = 1
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET ai_due_date = ?, due_ai_pending = ?, updated_at = ? WHERE id = ?`,
		aiDueDate, p, time.Now(), taskID); err != nil {
		return fmt.Errorf("failed to set AI due date: %w", err)
	}
	// Задача отсутствовала — 0 строк; ошибка, чтобы вердикт-путь это увидел.
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT changes()`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("task %d: %w", taskID, store.ErrTaskNotFound)
	}
	return nil
}

// isClosedStatus — статус, означающий «задача закрыта подтверждениями» (решение владельца:
// «статут задачи = done или closed»). В v0.24-набор статусов (api.go): new|backlog|
// in_progress|completed|closed — закрытые это completed и closed (legacy «done» больше нет).
func isClosedStatus(status string) bool {
	return status == "completed" || status == "closed"
}

// lastIncomingConfirmationEmail — автор последнего входящего подтверждения по задаче.
// Канон (онтология §7.7, решение 3, шаг 6): auto-assignee = тот, кто подтвердил.
// Источники (по приоритету):
//  1. task_comments(direction='in') — входящие комментарии по задаче;
//  2. from_email задачи — отправитель первого письма.
//
// Возвращает пустую строку, если подтверждения не найдено (режим A «без персон»).
// db — интерфейс с QueryRowContext: работает и с *sql.DB, и с *sql.Tx.
func lastIncomingConfirmationEmail(ctx context.Context, db queryer, taskID int64) (string, error) {
	// 1) входящие комментарии — кто подтвердил по задаче (direction: 'in' входящее, 'out' исходящее).
	var email string
	err := db.QueryRowContext(ctx,
		"SELECT c.author FROM task_comments c"+
			" WHERE c.task_id = ? AND c.direction = 'in'"+
			" ORDER BY c.created_at DESC LIMIT 1", taskID).Scan(&email)
	if err == nil && email != "" {
		return email, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("last confirmation via comments: %w", err)
	}

	// 2) fallback: the task's own from_email (first-message sender in the thread).
	if err := db.QueryRowContext(ctx, "SELECT from_email FROM tasks WHERE id = ?", taskID).Scan(&email); err == nil {
		return email, nil
	}
	return "", nil
}

// queryer — общий минимальный интерфейс для *sql.DB и *sql.Tx (контекстный доступ
// без привязки к конкретному типу — авто-ассигнмент работает в tx смены статуса).
type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ensurePersonInTx = upsertPersonByEmailInTx, но с сигнатурой (string, error):
// found=false трактует как no-op (пустую персона).
// 6-H: источник (task_comments.author) — сырой строка хедера («Имя
// <email>»); identity обязаны хранить только чистый email (канон 6-F),
// поэтому парсим RFC822 ДО upsert — так не создаётся сирот на сырых
// хедерах (prod 2026-09-17: именно так в prod попали грязные identity).
func ensurePersonInTx(ctx context.Context, tx *sql.Tx, email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", nil
	}
	if parsed, _ := extractor.ParseFromHeader(email); parsed != "" {
		email = strings.ToLower(strings.TrimSpace(parsed))
	}
	pid, found, err := upsertPersonByEmailInTx(ctx, tx, email)
	if err != nil {
		return "", err
	}
	if !found {
		return "", nil
	}
	return pid, nil
}

// autoAssignLastConfirmer — hook, вызывается из SetTaskStatus / BulkUpdateTasks
// при переходе задачи в закрытый статус (completed/closed). В tx:
//   - если assignee_id уже установлен — не трогаем (manual wins, решение 6);
//   - ищем автора последнего входящего; создаём Person (email), если нет;
//   - ставим assignee_id на задачу.
//
// Идемпотентно: повторный вызов на already-assigned task — no-op.
func autoAssignLastConfirmer(ctx context.Context, tx *sql.Tx, taskID int64) error {
	// Manual assign already set — do not overwrite (decision 3: manual is primary).
	var existingPersonID sql.NullString
	if err := tx.QueryRowContext(ctx,
		`SELECT assignee_id FROM tasks WHERE id = ?`, taskID).Scan(&existingPersonID); err == nil && existingPersonID.Valid {
		return nil
	}

	email, err := lastIncomingConfirmationEmail(ctx, tx, taskID)
	if err != nil || email == "" {
		return nil // no confirmation → no auto-assign; regime A "no persons" path
	}

	// Ensure person by email (create if missing), inside the same tx to keep
	// the auto-assign atomic with the status change.
	personID, err := ensurePersonInTx(ctx, tx, email)
	if err != nil || personID == "" {
		return nil // non-fatal: auto-assign best-effort.
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE tasks SET assignee_id = ? WHERE id = ?`, personID, taskID); err != nil {
		return fmt.Errorf("auto-assign: update tasks.assignee_id: %w", err)
	}
	return nil
}

// BulkUpdateTasks — пакетная смена статуса N задач (v0.23, шаг 4).
// Поведение по строкам идентично циклу SetTaskStatus:
//   - одна транзакция на все строки (атомарно: либо все, либо ничего);
//   - при совпадении from == to строки в истории нет;
//   - dups в taskIDs схлопываются (одна строка истории на задачу, а не N).
func (s *Store) BulkUpdateTasks(ctx context.Context, taskIDs []int64, toStatus, by string) (int, error) {
	// Дупы схлопываем: одна строка истории на задачу, а не N.
	seen := make(map[int64]struct{}, len(taskIDs))
	uniq := make([]int64, 0, len(taskIDs))
	for _, id := range taskIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	// from_status каждого элемента — до UPDATE (как в SetTaskStatus).
	type rowChange struct {
		id         int64
		fromStatus string
	}
	changes := make([]rowChange, 0, len(uniq))
	for _, id := range uniq {
		var fromStatus string
		err := s.db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, id).Scan(&fromStatus)
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("task %d: %w", id, store.ErrTaskNotFound)
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read task %d status: %w", id, err)
		}
		changes = append(changes, rowChange{id: id, fromStatus: fromStatus})
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, c := range changes {
		if _, err := tx.ExecContext(ctx,
			`UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			toStatus, c.id); err != nil {
			return 0, fmt.Errorf("failed to update task %d status: %w", c.id, err)
		}
		// v0.24, шаг 6 (Персоны): auto-assign на closed (best-effort, не ломает bulk).
		if isClosedStatus(toStatus) {
			_ = autoAssignLastConfirmer(ctx, tx, c.id)
		}
		if c.fromStatus == toStatus {
			continue // совпадение → строки истории нет (как SetTaskStatus)
		}
		var fromInterface any
		if c.fromStatus == "" {
			fromInterface = nil
		} else {
			fromInterface = c.fromStatus
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_status_history (task_id, from_status, to_status, by) VALUES (?, ?, ?, ?)`,
			c.id, fromInterface, toStatus, by); err != nil {
			return 0, fmt.Errorf("failed to write status history for task %d: %w", c.id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit bulk status update: %w", err)
	}
	return len(uniq), nil
}

// BulkUpdateProject — пакетная смена проекта N задач (v0.23, шаг 4, «К проекту X»).
// Возвращает число затронутых задач (дупы схлопнуты). История статусов не меняется.
func (s *Store) BulkUpdateProject(ctx context.Context, taskIDs []int64, project string) (int, error) {
	seen := make(map[int64]struct{}, len(taskIDs))
	uniq := make([]int64, 0, len(taskIDs))
	for _, id := range taskIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	if len(uniq) == 0 {
		return 0, nil
	}

	// Плейсхолдеры: UPDATE tasks SET project = ? WHERE id IN (?,?,...)
	placeholders := make([]string, len(uniq))
	args := make([]any, 0, len(uniq)+1)
	args = append(args, project, time.Now())
	for i, id := range uniq {
		placeholders[i] = "?"
		args = append(args, id)
	}
	res, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET project = ?, updated_at = ? WHERE id IN ("+strings.Join(placeholders, ",")+")",
		args...)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk update project: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read affected rows: %w", err)
	}
	return int(affected), nil
}

// GetTaskStatusHistory возвращает хронологию статусов задачи, отсортированную по at asc.
func (s *Store) GetTaskStatusHistory(ctx context.Context, taskID int64) ([]*store.TaskStatusHistory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, from_status, to_status, by, at
		FROM task_status_history
		WHERE task_id = ?
		ORDER BY at ASC, id ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query status history: %w", err)
	}
	defer rows.Close()

	history := make([]*store.TaskStatusHistory, 0, 4)
	for rows.Next() {
		var h store.TaskStatusHistory
		if err := rows.Scan(&h.ID, &h.TaskID, &h.FromStatus, &h.ToStatus, &h.By, &h.At); err != nil {
			return nil, fmt.Errorf("failed to scan status history: %w", err)
		}
		history = append(history, &h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return history, nil
}

// AddTaskComment добавляет комментарий к задаче.
func (s *Store) AddTaskComment(ctx context.Context, comment *store.TaskComment) error {
	query := `INSERT INTO task_comments (task_id, author, body, direction, kind, inbox_item_id, verdict_json, author_person_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := s.db.ExecContext(ctx, query,
		comment.TaskID, comment.Author, comment.Body, comment.Direction,
		comment.Kind, comment.InboxItemID, comment.VerdictJSON, personIDToSQL(comment.AuthorPersonID))
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	id, _ := result.LastInsertId()
	comment.ID = id
	comment.CreatedAt = time.Now()

	// v0.26, шаг 7d: вклад-активность — новый комментарий (ответ/входящий
	// комментарий, любой direction) поднимает tasks.updated_at, чтобы
	// сортировка «по активности» подняла задачу наверх.
	if _, err := s.db.ExecContext(ctx, "UPDATE tasks SET updated_at = ? WHERE id = ?", time.Now(), comment.TaskID); err != nil {
		return fmt.Errorf("failed to bump task activity: %w", err)
	}
	return nil
}

// SetTaskCommentApproved ставит/снимает флаг утверждения на комментарии (ФАЗА 4).
func (s *Store) SetTaskCommentApproved(ctx context.Context, id int64, approved bool) error {
	v := 0
	if approved {
		v = 1
	}
	res, err := s.db.ExecContext(ctx, "UPDATE task_comments SET approved = ? WHERE id = ?", v, id)
	if err != nil {
		return fmt.Errorf("failed to set approved on comment %d: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.ErrCommentNotFound
	}
	return nil
}

// GetTaskComment возвращает один комментарий по id (для approve, ФАЗА 4).
func (s *Store) GetTaskComment(ctx context.Context, id int64) (*store.TaskComment, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, task_id, author, body, direction, kind, inbox_item_id, verdict_json, approved, author_person_id, created_at
		 FROM task_comments WHERE id = ?`, id)
	c, err := scanComment(row)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, store.ErrCommentNotFound
	}
	return c, nil
}

// GetTaskComments возвращает список комментариев задачи.
func (s *Store) GetTaskComments(ctx context.Context, taskID int64) ([]*store.TaskComment, error) {
	query := `SELECT id, task_id, author, body, direction, kind, inbox_item_id, verdict_json, approved, author_person_id, created_at
		FROM task_comments WHERE task_id = ? ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	var comments []*store.TaskComment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		if c != nil {
			comments = append(comments, c)
		}
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
// Порядок колонок: ... assignee_id, due_date, ai_due_date, due_source, due_ai_pending, scheduled_date, created_at, updated_at
// (v0.25 шаг 7a — срок; v0.28 шаг 28a — план; все SELECT задач перечисляют их в этом порядке).
func scanTask(row interface{ Scan(...interface{}) error }) (*store.Task, error) {
	t := &store.Task{}
	var epicID sql.NullInt64
	var requestorID, assigneeID sql.NullString
	var dueDate, aiDueDate, dueSource, scheduledDate sql.NullString
	var dueAIPending sql.NullInt64
	err := row.Scan(&t.ID, &t.MessageID, &t.Subject, &t.BodyText, &t.BodyHTML,
		&t.FromEmail, &t.FromName, &t.Project, &t.Type, &t.Priority, &t.Status, &t.Assignee,
		&t.ThreadID, &t.SourceEmailID, &t.AIVerdict, &epicID, &requestorID, &assigneeID,
		&dueDate, &aiDueDate, &dueSource, &dueAIPending, &scheduledDate,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}
	if epicID.Valid {
		t.EpicID = &epicID.Int64
	}
	if requestorID.Valid {
		pid := store.PersonID(requestorID.String)
		t.RequestorID = &pid
	}
	if assigneeID.Valid {
		pid := store.PersonID(assigneeID.String)
		t.AssigneeID = &pid
	}
	if dueDate.Valid {
		t.DueDate = &dueDate.String
	}
	if aiDueDate.Valid {
		t.AIDueDate = &aiDueDate.String
	}
	if dueSource.Valid {
		t.DueSource = &dueSource.String
	}
	if dueAIPending.Valid {
		v := dueAIPending.Int64 != 0
		t.DueAIPending = &v
	}
	if scheduledDate.Valid {
		t.ScheduledDate = &scheduledDate.String
	}
	return t, nil
}

// scanComment читает строку task_comments во всех местах, где комментарии сканируются.
// Порядок колонок (Postgres-совместимый, 2026-09-15):
// id, task_id, author, body, direction, kind, inbox_item_id, verdict_json, approved, author_person_id, created_at
// NULL = автор не распознан как Персона (режим A «без персон»).
func scanComment(row interface {
	Scan(...interface{}) error
}) (*store.TaskComment, error) {
	c := &store.TaskComment{}
	var inboxItemID sql.NullInt64
	var verdictJSON sql.NullString
	var approved sql.NullInt32
	var authorPersonID sql.NullString
	err := row.Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &c.Direction,
		&c.Kind, &inboxItemID, &verdictJSON, &approved, &authorPersonID, &c.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan comment: %w", err)
	}
	if inboxItemID.Valid {
		c.InboxItemID = &inboxItemID.Int64
	}
	if verdictJSON.Valid {
		c.VerdictJSON = verdictJSON.String
	}
	if approved.Valid {
		v := int(approved.Int32)
		c.Approved = &v
	}
	if authorPersonID.Valid {
		pid := store.PersonID(authorPersonID.String)
		c.AuthorPersonID = &pid
	}
	return c, nil
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
		project, type, priority, status, assignee, thread_id, source_email_id, ai_verdict, epic_id, requestor_id, assignee_id, due_date, ai_due_date, due_source, due_ai_pending, scheduled_date, created_at, updated_at
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

// QueryRowForTest — экспортируемый метод для тестов.
func (s *Store) QueryRowForTest(ctx context.Context, query string) *sql.Row {
	return s.db.QueryRowContext(ctx, query)
}

// ExecForTest — экспортируемый метод для тестов.
func (s *Store) ExecForTest(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

// GetPendingAIItems возвращает входящие с ai_processed = 0.
func (s *Store) GetPendingAIItems(ctx context.Context) ([]*store.InboxItem, error) {
	query := `SELECT id, source, source_id, thread_id, from_contact, from_name, subject, body_text, body_html, meta, received_at, ai_processed, ai_attempts, ai_verdict, ai_summary, status
		FROM inbox_items WHERE ai_processed = 0 ORDER BY received_at ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending AI items: %w", err)
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
