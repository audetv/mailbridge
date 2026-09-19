// Package store определяет интерфейс хранилища и модели данных Mailbridge.
package store

import (
	"context"
	"errors"
	"time"
)

// Attachment представляет файл в системе.
type Attachment struct {
	ID          int64     `json:"id"`
	Hash        string    `json:"hash"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	StoragePath string    `json:"storage_path"`
	CreatedAt   time.Time `json:"created_at"`
}

// Project представляет проект (внутренний «проект», контейнер модулей/эпиков и задач).
type Project struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Archived    bool      `json:"archived"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectFilter фильтрует список проектов.
type ProjectFilter struct {
	Archived *bool // nil = все, true = только архивные, false = только активные
	Search   string
}

// Epic представляет модуль (эпик) внутри проекта.
type Epic struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Number      int       `json:"number"`
	Status      string    `json:"status"` // open | in_progress | done
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EpicProgress — сводка задач эпика для карточки.
type EpicProgress struct {
	Total int `json:"total"`
	Open  int `json:"open"`
	Done  int `json:"done"` // задачи со статусом completed|closed
}

// InboxItem представляет элемент ленты входящих.
type InboxItem struct {
	ID          int64     `json:"id"`
	Source      string    `json:"source"`
	SourceID    string    `json:"source_id"`
	ThreadID    string    `json:"thread_id"`
	FromContact string    `json:"from_contact"`
	FromName    string    `json:"from_name"`
	FromEmail   string    `json:"from_email"`
	Subject     string    `json:"subject"`
	BodyText    string    `json:"body_text"`
	BodyHTML    string    `json:"body_html"`
	Meta        string    `json:"meta"` // JSON
	ReceivedAt  time.Time `json:"received_at"`
	AIProcessed int       `json:"ai_processed"`
	AIAttempts  int       `json:"ai_attempts"`
	AIVerdict   string    `json:"ai_verdict"`
	AISummary   string    `json:"ai_summary"`
	Status      string    `json:"status"`
}

// TaskInboxItem связывает задачу с элементом ленты.
type TaskInboxItem struct {
	TaskID      int64     `json:"task_id"`
	InboxItemID int64     `json:"inbox_item_id"`
	Relation    string    `json:"relation"`
	CreatedAt   time.Time `json:"created_at"`
}

// Task представляет задачу в helpdesk.
type Task struct {
	ID            int64  `json:"id"`
	MessageID     string `json:"message_id"`
	Subject       string `json:"subject"`
	BodyText      string `json:"body_text"`
	BodyHTML      string `json:"body_html"`
	FromEmail     string `json:"from_email"`
	FromName      string `json:"from_name"`
	Project       string `json:"project"`
	Type          string `json:"type"`
	Priority      string `json:"priority"`
	Status        string `json:"status"`
	Assignee      string `json:"assignee"`
	ThreadID      string `json:"thread_id"`
	SourceEmailID string `json:"source_email_id"`
	AIVerdict     string `json:"ai_verdict"`
	EpicID        *int64 `json:"epic_id"`
	// v0.24, шаг 6 (Персоны): роли — контекст связи, не свойство персоны.
	// RequestorID — кто обратился (task.requestor → person); nil = не привязана.
	// AssigneeID — кто выполняет (task.assignee → person); nil = не назначен.
	// Legacy-текст Assignee/FromEmail остаётся до конца шага (совместимость).
	RequestorID *PersonID `json:"requestor_id,omitempty"`
	AssigneeID  *PersonID `json:"assignee_id,omitempty"`
	// v0.25, шаг 7a: срок задачи — DATE в формате YYYY-MM-DD (без времени).
	// Канон: онтология v0.5.2 §7.5. Ручной срок приоритетнее AI;
	// AI-предложение (ai_due_date, due_source) появится в шаге 7b.
	DueDate *string `json:"due_date,omitempty"`
	// v0.28, шаг 28a: план задачи — «когда делаю» (DATE, те же правила, что due_date).
	// Решение владельца 2026-09-19: две независимые даты — due (обещание) + scheduled (план).
	ScheduledDate *string `json:"scheduled_date,omitempty"`
	// AI-предложение срока (пополняется в шаге 7b): хранится ВСЕГДА (и отклонённые — база ошибок AI).
	AIDueDate    *string   `json:"ai_due_date,omitempty"`
	DueSource    *string   `json:"due_source,omitempty"`     // 'ai' | 'manual'
	DueAIPending *bool     `json:"due_ai_pending,omitempty"` // pending-предложение AI (7b/7c)
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Thread представляет цепочку входящих.
type Thread struct {
	ID           int64      `json:"id"`
	ThreadID     string     `json:"thread_id"`
	Source       string     `json:"source"`
	Subject      string     `json:"subject"`
	Participants string     `json:"participants"` // JSON-массив
	Summary      string     `json:"summary"`
	LastItemAt   *time.Time `json:"last_item_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TaskWithUnread расширяет Task полем UnreadComments для ответа API.
type TaskWithUnread struct {
	*Task
	UnreadComments int `json:"unread_comments"`
}

// TaskComment представляет комментарий к задаче.
type TaskComment struct {
	ID          int64     `json:"id"`
	TaskID      int64     `json:"task_id"`
	Author      string    `json:"author"`
	Body        string    `json:"body"`
	Direction   string    `json:"direction"`
	Kind        string    `json:"kind"`
	InboxItemID *int64    `json:"inbox_item_id,omitempty"`
	VerdictJSON string    `json:"verdict_json,omitempty"`
	Approved    *int      `json:"approved,omitempty"` // NULL = не утверждён; 0/1 — модерация ответа (ФАЗА 4)
	CreatedAt   time.Time `json:"created_at"`

	// v0.24, шаг 6 (Персоны): кто подтверждал/решал (comment.author → person); nil = не привязан.
	// Legacy-текст Author остаётся до конца шага (совместимость).
	AuthorPersonID *PersonID `json:"author_person_id,omitempty"`
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
	EpicID   *int64   // фильтр по модулю (epic)
	Statuses []string // множественный фильтр по статусам
	Assignee string
	Type     string
	Priority string
	Search   string
	Username string
	Page     int
	PerPage  int

	// v0.24, шаг 6 (Персоны): фильтры по ролям (контекст связи).
	// RequestorID/AssigneeID задаются по person UUID;
	// legacy-текст Assignee по-прежнему работает (по email — совмещение).
	RequestorID *PersonID `json:"requestor_id,omitempty"`
	AssigneeID  *PersonID `json:"assignee_id,omitempty"`

	// v0.25, шаг 7a: серверная сортировка списка. "due" | "updated".
	// Дефолт (пусто) = "due" — просроченные сверху, без срока внизу;
	// "updated" — по свежести активности.
	// v0.27.1 (хотфикс): тайбрейкер "due" для задач БЕЗ срока — created_at
	// DESC, т.е. новые созданные — наверху (раньше: id ASC — старейшие сверху).
	// Задачи СО сроком — приоритет по min(due_date) сохранён; id DESC —
	// детерминированный ключ для пагинации.
	Sort string

	// v0.26, шаг 7d: фильтр по срокам (одно значение за запрос):
	// "overdue"        — срок в прошлом (due_date < сегодня),
	// "today"          — срок = сегодня,
	// "tomorrow"       — срок = завтра,
	// "7d" / "30d"     — срок в ближайшие N дней (включая сегодня),
	// "none"           — срок не установлен (due_date IS NULL),
	// "due_pending"    — срок на подтверждении (due_ai_pending=1).
	// Даты сравнения — канонические YYYY-MM-DD, «сегодня» — часовой пояс сервера.
	Due string

	// v0.28, шаг 28b: срез «План» (одно значение за запрос):
	// "today"    — план на сегодня (scheduled_date = H),
	// "tomorrow" — план на завтра (scheduled_date = H+1),
	// "week"     — план на 7 дней (H..H+6).
	// ПЛЮС в любом случае — просроченные due (due_date < H) у АКТИВНЫХ
	// статусов new/in_progress/backlog (решение владельца: «совсем не видеть
	// просроченные» отклонено). completed/closed — вне плана.
	// H — локальный день сервера (как ?due= в 7a).
	Plan string
}

// TaskListResult содержит результат запроса списка задач.
type TaskListResult struct {
	Tasks   []*TaskWithUnread `json:"tasks"`
	Total   int64             `json:"total"`
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
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

// InboxFilter содержит параметры фильтрации ленты.
type InboxFilter struct {
	Status  string // unread, read, archived, "" = все
	Source  string
	Page    int
	PerPage int
}

// InboxListResult содержит результат запроса ленты.
type InboxListResult struct {
	Items   []*InboxItem `json:"items"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
}

// ErrCommentNotFound — комментарий не найден.
var ErrCommentNotFound = errors.New("comment not found")

// ErrTaskNotFound — задача не найдена.
var ErrTaskNotFound = errors.New("task not found")

// TaskStatusHistory — строка истории статусов задачи (v0.23, шаг 2).
// FromStatus — NULL при первом записи (задача только создана).
type TaskStatusHistory struct {
	ID         int64     `json:"id"`
	TaskID     int64     `json:"task_id"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	By         string    `json:"by"`
	At         time.Time `json:"at"`
}

// Store определяет интерфейс хранилища данных.
type Store interface {
	// Migrate выполняет миграции схемы.
	Migrate(ctx context.Context) error

	// Projects
	CreateProject(ctx context.Context, p *Project) error
	GetProject(ctx context.Context, id int64) (*Project, error)
	GetProjectByName(ctx context.Context, name string) (*Project, error)
	ListProjects(ctx context.Context, filter *ProjectFilter) ([]*Project, error)
	UpdateProject(ctx context.Context, id int64, name, description string) error
	SetProjectArchived(ctx context.Context, id int64, archived bool) error

	// Epics (модули)
	CreateEpic(ctx context.Context, e *Epic) error
	GetEpic(ctx context.Context, id int64) (*Epic, error)
	ListEpics(ctx context.Context, projectID int64) ([]*Epic, error)
	UpdateEpic(ctx context.Context, id int64, name, description, status string) error
	DeleteEpic(ctx context.Context, id int64) error
	EpicProgress(ctx context.Context, epicID int64) (*EpicProgress, error)
	// SetTaskEpic привязывает задачу к модулю (epicID = 0 — отвязать).
	SetTaskEpic(ctx context.Context, taskID, epicID int64) error

	// Attachments
	CreateAttachment(ctx context.Context, att *Attachment) error
	GetAttachmentByHash(ctx context.Context, hash string) (*Attachment, error)
	GetAttachmentByID(ctx context.Context, id int64) (*Attachment, error)
	LinkAttachmentToInbox(ctx context.Context, inboxItemID, attachmentID int64) error
	LinkAttachmentToTask(ctx context.Context, taskID, attachmentID int64) error
	UnlinkAttachmentFromTask(ctx context.Context, taskID, attachmentID int64) error
	GetAttachmentsByInbox(ctx context.Context, inboxItemID int64) ([]*Attachment, error)
	GetAttachmentsByTask(ctx context.Context, taskID int64) ([]*Attachment, error)
	// CopyInboxAttachmentsToTask переносит вложения входящего в задачу
	// (идемпотентно). Возвращает число новых связей.
	CopyInboxAttachmentsToTask(ctx context.Context, taskID, inboxItemID int64) (int, error)

	// GetAttachmentsByComment возвращает вложения комментария.
	GetAttachmentsByComment(ctx context.Context, commentID int64) ([]*Attachment, error)

	// Inbox Items
	CreateInboxItem(ctx context.Context, item *InboxItem) error
	GetInboxItemByID(ctx context.Context, id int64) (*InboxItem, error)
	GetInboxItemBySourceID(ctx context.Context, source, sourceID string) (*InboxItem, error)

	ListInboxItems(ctx context.Context, filter *InboxFilter) (*InboxListResult, error)
	UpdateInboxItemStatus(ctx context.Context, id int64, status string) error
	UpdateInboxItemAI(ctx context.Context, id int64, processed int, verdict, summary string) error

	// GetInboxItemsByThread возвращает все входящие цепочки.
	GetInboxItemsByThread(ctx context.Context, threadID string) ([]*InboxItem, error)
	// GetTasksByThread возвращает все задачи цепочки.
	GetTasksByThread(ctx context.Context, threadID string) ([]*Task, error)

	// Task-Inbox связь
	LinkTaskToInboxItem(ctx context.Context, taskID, inboxItemID int64, relation string) error
	GetInboxItemsByTask(ctx context.Context, taskID int64) ([]*TaskInboxItem, error)
	GetTasksByInboxItem(ctx context.Context, inboxItemID int64) ([]*TaskInboxItem, error)

	// GetPendingAIItems возвращает входящие, ожидающие AI-обработки.
	GetPendingAIItems(ctx context.Context) ([]*InboxItem, error)

	// Tasks
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id int64) (*Task, error)
	GetTaskByMessageID(ctx context.Context, messageID string) (*Task, error)
	ListTasks(ctx context.Context, filter *TaskFilter) (*TaskListResult, error)
	UpdateTask(ctx context.Context, id int64, updates map[string]interface{}) error
	// SetTaskStatus — единственный путь смены статуса задачи:
	// транзакционно обновляет tasks.status и при реальном переходе пишет
	// строку в task_status_history (from_status читается из текущего
	// состояния; при совпадении с текущим строки нет).
	SetTaskStatus(ctx context.Context, taskID int64, toStatus, by string) error
	// BulkUpdateTasks — пакетная смена статуса N задач (v0.23, шаг 4).
	// Поведение по строкам: идентично циклу SetTaskStatus (история from/to/by,
	// «нет строки при совпадении с текущим»). Атомарно: либо все, либо откат.
	// Возвращает число затронутых задач (дупы в taskIDs схлопнуты, порядок неважен).
	BulkUpdateTasks(ctx context.Context, taskIDs []int64, toStatus, by string) (touched int, err error)
	// BulkUpdateProject — пакетная смена проекта N задач (v0.23, шаг 4, «К проекту X»).
	// Возвращает число затронутых задач (дупы схлопнуты).
	BulkUpdateProject(ctx context.Context, taskIDs []int64, project string) (touched int, err error)
	// GetTaskStatusHistory возвращает хронологию статусов задачи (по at asc).
	GetTaskStatusHistory(ctx context.Context, taskID int64) ([]*TaskStatusHistory, error)
	// SetTaskPersonRoles привязывает/отвязывает роли задачи (v0.24, шаг 6):
	// requestor/assignee по person UUID; nil — сохранить текущее, &"" (пустой ID) — отвязать.
	// Ручное назначение имеет приоритет над авто (правило решения 6).
	SetTaskPersonRoles(ctx context.Context, taskID int64, requestorID, assigneeID *PersonID) error

	// SetTaskDueAIPending — v0.25 шаг 7b: AI-предложение срока задачи.
	// Пишет ai_due_date (NULL — срок не извлечён, база ошибок AI:
	// попал/упустил/заврался) и due_ai_pending (1 — предложение ждёт решения
	// человека; приём/отмена — 7c, там же пишется due_date/due_source).
	// due_date/due_source НЕ изменяются: решение человека только в 7c.
	// Флаг MAILBRIDGE_AI_AUTO_APPLY_DUE (авто-принятие) — шаг 7d.
	SetTaskDueAIPending(ctx context.Context, taskID int64, aiDueDate *string, pending bool) error

	// Persons (v0.24, шаг 6) — справочник действующих лиц.
	// CreatePerson создаёт персону; ID должен быть задан приложением (UUID).
	CreatePerson(ctx context.Context, p *Person) error
	GetPerson(ctx context.Context, id PersonID) (*Person, error)
	ListPersons(ctx context.Context, filter *PersonFilter) (*PersonListResult, error)
	UpdatePerson(ctx context.Context, p *Person) error // name/org/is_internal/confirmed/archived
	// FindPersonByEmail — fast-path: (kind=email, value, case-fold) → персона; nil, nil если нет.
	FindPersonByEmail(ctx context.Context, email string) (*Person, error)
	// EnsurePersonByEmail создаёт (confirmed=false) или возвращает персону по email:
	// авто-создание при первом контакте (решение 7); идемпотентно (UNIQUE(kind,value)).
	// ВХОД — строка email («a@b.ru»): жёсткое требование вида «@ + точка после @» (6-F),
	// иначе error — мусор (RFC822-хедер) в identity не записывается.
	EnsurePersonByEmail(ctx context.Context, email string) (*Person, error)
	// EnsurePersonFromIncoming (6-F) — разбор из extractor (name, email):
	// если имя пустое и пришло — авто-заполняем persons.name;
	// если email похож на «машину» (no-reply/postfix/mailer/bot/...) — org='машина'.
	// Идемпотентна.
	EnsurePersonFromIncoming(ctx context.Context, fromName, fromEmail string) (*Person, error)

	// Identities
	AddPersonIdentity(ctx context.Context, ident *PersonIdentity) (*PersonIdentity, error)
	ListIdentities(ctx context.Context, personID PersonID) ([]*PersonIdentity, error)
	RemovePersonIdentity(ctx context.Context, id PersonID) error // unlink (без удаления записи)
	// FindIdentity — natural key (kind, value); email с case-fold; nil, nil если нет.
	FindIdentity(ctx context.Context, kind, value string) (*PersonIdentity, error)

	// Suggestions / отрицательное знание (решения 4–5).
	// SuggestMatch — предложение: идентичность вида identityKind/identityValue
	// может принадлежать персонам, похожим по имени (fuzzy); учитывает match_rejections.
	SuggestMatch(ctx context.Context, identityKind, identityValue, name string) ([]*Person, error)
	// AcceptMatch — привязать идентичность к персоне (provenance='suggested').
	AcceptMatch(ctx context.Context, identityID PersonID, personID PersonID) error
	// RejectMatch — записать match_rejections (не повторять предложение).
	RejectMatch(ctx context.Context, identityID, personID PersonID) error

	// MergePersons — слияние персон (duplicates): идентичности sourceIdentities
	// переносятся к targetID, ссылки tasks/comments переключаются; source — archived.
	MergePersons(ctx context.Context, sourceID, targetID PersonID) error

	// Task Comments
	// AddTaskComment добавляет комментарий и поднимает активность задачи:
	// tasks.updated_at обновляется до времени комментария (вклад-активность,
	// v0.26, шаг 7d — сортировка «по активности»: ответ/комментарий
	// поднимают задачу наверх, а не только метаданные).
	AddTaskComment(ctx context.Context, comment *TaskComment) error
	GetTaskComments(ctx context.Context, taskID int64) ([]*TaskComment, error)
	GetTaskComment(ctx context.Context, id int64) (*TaskComment, error)
	SetTaskCommentApproved(ctx context.Context, id int64, approved bool) error

	// Task Attachments
	AddTaskAttachment(ctx context.Context, att *TaskAttachment) error
	GetTaskAttachments(ctx context.Context, taskID int64) ([]*TaskAttachment, error)

	// LinkAttachmentToComment связывает вложение с комментарием.
	LinkAttachmentToComment(ctx context.Context, commentID, attachmentID int64) error

	// Outbox
	EnqueueOutbox(ctx context.Context, payload string) error
	GetPendingOutbox(ctx context.Context, limit int) ([]*OutboxItem, error)
	MarkOutboxSent(ctx context.Context, id int64) error
	MarkOutboxFailed(ctx context.Context, id int64, errMsg string) error

	// MarkTaskRead отмечает задачу прочитанной пользователем.
	MarkTaskRead(ctx context.Context, taskID int64, username string) error
	// ResetTaskReads сбрасывает статус прочтения для всех пользователей задачи.
	// Вызывается при добавлении нового входящего комментария.
	ResetTaskReads(ctx context.Context, taskID int64) error

	// Threads
	CreateThread(ctx context.Context, thread *Thread) error
	GetThread(ctx context.Context, threadID string) (*Thread, error)
	UpdateThreadSummary(ctx context.Context, threadID, summary string) error
	GetActiveTasksByThread(ctx context.Context, threadID string) ([]*Task, error)

	// Ping
	Ping(ctx context.Context) error

	// Close
	Close() error
}
