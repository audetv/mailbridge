package sqlite_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

func setupStore(t *testing.T) (*sqlite.Store, func()) {
	t.Helper()

	s, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		s.Close()
		t.Fatalf("failed to migrate: %v", err)
	}

	cleanup := func() {
		s.Close()
	}

	return s, cleanup
}

// helper для быстрого создания задачи в тестах
func mustCreateTask(t *testing.T, s *sqlite.Store, task *store.Task) {
	t.Helper()
	if err := s.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tasks
// ---------------------------------------------------------------------------

func TestCreateTask(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	task := &store.Task{
		MessageID: "msg-001",
		Subject:   "Test Subject",
		BodyText:  "Test body",
		FromEmail: "user@example.com",
		Project:   "Входящие",
		Type:      "bug",
		Priority:  "high",
		Status:    "new",
	}

	mustCreateTask(t, s, task)

	if task.ID == 0 {
		t.Error("task ID is 0")
	}
	if task.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestGetTask(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{
		MessageID: "msg-002",
		Subject:   "Get Test",
		BodyText:  "Body",
		FromEmail: "user@example.com",
		Project:   "ТРК",
		Status:    "new",
	}
	mustCreateTask(t, s, task)

	got, err := s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got == nil {
		t.Fatal("task not found")
	}
	if got.Subject != "Get Test" {
		t.Errorf("Subject = %s", got.Subject)
	}
	if got.Project != "ТРК" {
		t.Errorf("Project = %s", got.Project)
	}
}

func TestGetTaskByMessageID(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{
		MessageID: "msg-003",
		Subject:   "MessageID Test",
		BodyText:  "Body",
		FromEmail: "user@example.com",
		Status:    "new",
	}
	mustCreateTask(t, s, task)

	got, err := s.GetTaskByMessageID(ctx, "msg-003")
	if err != nil {
		t.Fatalf("GetTaskByMessageID error: %v", err)
	}
	if got == nil {
		t.Fatal("task not found by message_id")
	}
}

func TestListTasks(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		mustCreateTask(t, s, &store.Task{
			MessageID: fmt.Sprintf("list-msg-%d", i),
			Subject:   fmt.Sprintf("Task %d", i),
			BodyText:  "Body",
			FromEmail: "user@example.com",
			Project:   "ТРК",
			Status:    "new",
			Priority:  "medium",
		})
	}

	result, err := s.ListTasks(ctx, &store.TaskFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(result.Tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(result.Tasks))
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5", result.Total)
	}
}

func TestListTasks_FilterByProject(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	mustCreateTask(t, s, &store.Task{MessageID: "f1", Subject: "T1", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"})
	mustCreateTask(t, s, &store.Task{MessageID: "f2", Subject: "T2", BodyText: "B", FromEmail: "u@e.com", Project: "Отель", Status: "new"})

	result, err := s.ListTasks(ctx, &store.TaskFilter{Project: "ТРК", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Errorf("expected 1 task for project ТРК, got %d", len(result.Tasks))
	}
}

func TestListTasks_FilterByStatus(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	mustCreateTask(t, s, &store.Task{MessageID: "s1", Subject: "T1", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"})
	mustCreateTask(t, s, &store.Task{MessageID: "s2", Subject: "T2", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "in_progress"})

	result, err := s.ListTasks(ctx, &store.TaskFilter{Statuses: []string{"in_progress"}, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Errorf("expected 1 task for status in_progress, got %d", len(result.Tasks))
	}
}

// ---------------------------------------------------------------------------
// v0.25, шаг 7a — срок задачи (due_date)
// ---------------------------------------------------------------------------

func strp(v string) *string { return &v }

func TestDueDate_CreateGet(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	manual := "manual"
	t1 := &store.Task{MessageID: "due-1", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new",
		DueDate: strp("2026-10-01"), DueSource: &manual}
	mustCreateTask(t, s, t1)

	got, err := s.GetTask(ctx, t1.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.DueDate == nil || *got.DueDate != "2026-10-01" {
		t.Errorf("due_date = %v, want 2026-10-01", got.DueDate)
	}
	if got.DueSource == nil || *got.DueSource != "manual" {
		t.Errorf("due_source = %v, want manual", got.DueSource)
	}

	// Без срока — null (не ошибка, поле отсутствует).
	t2 := &store.Task{MessageID: "due-2", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
	mustCreateTask(t, s, t2)
	g2, err := s.GetTask(ctx, t2.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if g2.DueDate != nil {
		t.Errorf("due_date = %v, want nil", g2.DueDate)
	}
}

func TestDueDate_UpdateAndClear(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	tk := &store.Task{MessageID: "due-u", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
	mustCreateTask(t, s, tk)

	// Установка срока (как делает API: value + due_source=manual).
	if err := s.UpdateTask(ctx, tk.ID, map[string]interface{}{"due_date": "2026-12-31", "due_source": "manual"}); err != nil {
		t.Fatalf("UpdateTask set: %v", err)
	}
	g, _ := s.GetTask(ctx, tk.ID)
	if g.DueDate == nil || *g.DueDate != "2026-12-31" {
		t.Errorf("after set, due_date = %v, want 2026-12-31", g.DueDate)
	}

	// Снятие срока (NULL).
	if err := s.UpdateTask(ctx, tk.ID, map[string]interface{}{"due_date": nil, "due_source": "manual"}); err != nil {
		t.Fatalf("UpdateTask clear: %v", err)
	}
	g2, _ := s.GetTask(ctx, tk.ID)
	if g2.DueDate != nil {
		t.Errorf("after clear, due_date = %v, want nil", g2.DueDate)
	}
}

func TestDueDate_ListSortDefault(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// 3 с разными сроками + 1 без срока.
	d := func(id, date string) *store.Task {
		tk := &store.Task{MessageID: id, Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
		if date != "" {
			tk.DueDate = strp(date)
			tk.DueSource = strp("manual")
		}
		return tk
	}
	dFar := d("due-far", "2026-12-01")
	dMid := d("due-mid", "2026-10-15")
	dOverdue := d("due-over", "2026-01-05") // просроченный — мин. дата
	dNone := d("due-none", "")
	mustCreateTask(t, s, dOverdue)
	mustCreateTask(t, s, dMid)
	mustCreateTask(t, s, dFar)
	mustCreateTask(t, s, dNone)

	// Дефолтная сортировка ("due"): просроченные раньше, без срока — внизу.
	res, err := s.ListTasks(ctx, &store.TaskFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	var ids []string
	for _, tk := range res.Tasks {
		ids = append(ids, tk.MessageID)
	}
	if len(ids) != 4 {
		t.Fatalf("expected 4 tasks, got %d (%v)", len(ids), ids)
	}
	if ids[0] != "due-over" {
		t.Errorf("first (most overdue) = %v, want due-over", ids[0])
	}
	if ids[3] != "due-none" {
		t.Errorf("last (no due date) = %v, want due-none", ids[3])
	}
}

func TestDueDate_ListSortUpdated(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	mk := func(id string) *store.Task {
		tk := &store.Task{MessageID: id, Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
		mustCreateTask(t, s, tk)
		return tk
	}
	a := mk("upd-a")
	b := mk("upd-b")
	// Подкручиваем updated_at у b свежее.
	_ = a
	if err := s.UpdateTask(ctx, b.ID, map[string]interface{}{"priority": "high"}); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	res, err := s.ListTasks(ctx, &store.TaskFilter{Sort: "updated", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(res.Tasks) != 2 {
		t.Fatalf("expected 2, got %d", len(res.Tasks))
	}
	// b обновлён позже → сверху при sort=updated.
	if res.Tasks[0].ID != b.ID {
		t.Errorf("first with sort=updated should be b (%d), got %d", b.ID, res.Tasks[0].ID)
	}
}

func TestListTasks_Search(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{
		MessageID: "sr1",
		Subject:   "Ошибка на сайте",
		BodyText:  "Подробности проблемы с ошибкой",
		FromEmail: "u@e.com",
		Status:    "new",
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	task2 := &store.Task{
		MessageID: "sr2",
		Subject:   "Баннер",
		BodyText:  "Обновить",
		FromEmail: "u@e.com",
		Status:    "new",
	}
	if err := s.CreateTask(ctx, task2); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Проверяем что задача точно создалась
	got, _ := s.GetTaskByMessageID(ctx, "sr1")
	t.Logf("Created task: subject=%q, body=%q", got.Subject, got.BodyText)

	// Ищем разными запросами
	result, err := s.ListTasks(ctx, &store.TaskFilter{Search: "ошибка", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	t.Logf("Search 'ошибка': found %d tasks", len(result.Tasks))

	result, err = s.ListTasks(ctx, &store.TaskFilter{Search: "Ошибка", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	t.Logf("Search 'Ошибка': found %d tasks", len(result.Tasks))

	if len(result.Tasks) != 1 {
		t.Errorf("expected 1 task for search, got %d", len(result.Tasks))
	}
}

func TestUpdateTask(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{MessageID: "upd-1", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	mustCreateTask(t, s, task)

	err := s.UpdateTask(ctx, task.ID, map[string]interface{}{
		"status":   "in_progress",
		"assignee": "Иванов",
		"project":  "Отель",
	})
	if err != nil {
		t.Fatalf("UpdateTask error: %v", err)
	}

	updated, err := s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if updated.Status != "in_progress" {
		t.Errorf("Status = %s, want in_progress", updated.Status)
	}
	if updated.Assignee != "Иванов" {
		t.Errorf("Assignee = %s, want Иванов", updated.Assignee)
	}
	if updated.Project != "Отель" {
		t.Errorf("Project = %s, want Отель", updated.Project)
	}
}

// ---------------------------------------------------------------------------
// Task Comments
// ---------------------------------------------------------------------------

func TestAddAndGetTaskComments(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{MessageID: "cmt-1", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	mustCreateTask(t, s, task)

	if err := s.AddTaskComment(ctx, &store.TaskComment{TaskID: task.ID, Author: "user@example.com", Body: "Comment 1", Direction: "in", Kind: "user_comment"}); err != nil {
		t.Fatalf("AddTaskComment error: %v", err)
	}
	if err := s.AddTaskComment(ctx, &store.TaskComment{TaskID: task.ID, Author: "support", Body: "Reply", Direction: "out", Kind: "user_comment"}); err != nil {
		t.Fatalf("AddTaskComment error: %v", err)
	}

	comments, err := s.GetTaskComments(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskComments error: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].Kind != "user_comment" {
		t.Errorf("Kind = %s, want user_comment", comments[0].Kind)
	}
}

// ---------------------------------------------------------------------------
// Task Attachments
// ---------------------------------------------------------------------------

// func TestAddAndGetTaskAttachments(t *testing.T) {
// 	s, cleanup := setupStore(t)
// 	defer cleanup()
// 	ctx := context.Background()

// 	task := &store.Task{MessageID: "att-1", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
// 	mustCreateTask(t, s, task)

// 	if err := s.AddTaskAttachment(ctx, &store.TaskAttachment{
// 		TaskID:      task.ID,
// 		Filename:    "screenshot.png",
// 		ContentType: "image/png",
// 		Size:        1024,
// 		StoragePath: "2024-01-01/screenshot.png",
// 	}); err != nil {
// 		t.Fatalf("AddTaskAttachment error: %v", err)
// 	}

// 	atts, err := s.GetTaskAttachments(ctx, task.ID)
// 	if err != nil {
// 		t.Fatalf("GetTaskAttachments error: %v", err)
// 	}
// 	if len(atts) != 1 {
// 		t.Errorf("expected 1 attachment, got %d", len(atts))
// 	}
// }

// ---------------------------------------------------------------------------
// Existing tests (outbox)
// ---------------------------------------------------------------------------

func TestOutbox(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := s.EnqueueOutbox(ctx, `{"test":true}`); err != nil {
		t.Fatalf("EnqueueOutbox error: %v", err)
	}
	items, _ := s.GetPendingOutbox(ctx, 10)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if err := s.MarkOutboxSent(ctx, items[0].ID); err != nil {
		t.Fatalf("MarkOutboxSent error: %v", err)
	}
	items, _ = s.GetPendingOutbox(ctx, 10)
	if len(items) != 0 {
		t.Error("expected 0 pending")
	}
}

func TestMarkOutboxFailed(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := s.EnqueueOutbox(ctx, `{"test":true}`); err != nil {
		t.Fatalf("EnqueueOutbox error: %v", err)
	}
	items, _ := s.GetPendingOutbox(ctx, 1)
	if err := s.MarkOutboxFailed(ctx, items[0].ID, "error"); err != nil {
		t.Fatalf("MarkOutboxFailed error: %v", err)
	}
	items, _ = s.GetPendingOutbox(ctx, 1)
	if len(items) != 0 {
		t.Error("expected 0 pending")
	}
}

func TestPing(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	if err := s.Ping(context.Background()); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestMigrate_ThreadsTable(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	exists, err := s.TableExists(context.Background(), "threads")
	if err != nil {
		t.Fatalf("TableExists error: %v", err)
	}
	if !exists {
		t.Fatal("threads table not created")
	}
}

func TestMigrate_TaskAIColumns(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{
		MessageID: "ai-test",
		Subject:   "Test",
		BodyText:  "B",
		FromEmail: "u@e.com",
		Status:    "new",
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Обновляем ai-поля
	err := s.UpdateTask(ctx, task.ID, map[string]interface{}{
		"thread_id":       "thread-123",
		"source_email_id": "msg-456",
		"ai_verdict":      `{"action":"new"}`,
	})
	if err != nil {
		t.Fatalf("UpdateTask with AI fields error: %v", err)
	}

	// Проверяем что задача создалась и обновилась
	got, err := s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got.Subject != "Test" {
		t.Errorf("Subject = %s", got.Subject)
	}
}

func TestThreadsCRUD(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// CreateThread
	thread := &store.Thread{ThreadID: "thread-001", Summary: ""}
	if err := s.CreateThread(ctx, thread); err != nil {
		t.Fatalf("CreateThread error: %v", err)
	}
	if thread.ID == 0 {
		t.Error("thread ID is 0")
	}

	// GetThread
	got, err := s.GetThread(ctx, "thread-001")
	if err != nil {
		t.Fatalf("GetThread error: %v", err)
	}
	if got == nil {
		t.Fatal("thread not found")
	}
	if got.ThreadID != "thread-001" {
		t.Errorf("ThreadID = %s", got.ThreadID)
	}

	// UpdateThreadSummary
	if err := s.UpdateThreadSummary(ctx, "thread-001", "Обновлённое резюме"); err != nil {
		t.Fatalf("UpdateThreadSummary error: %v", err)
	}

	got, _ = s.GetThread(ctx, "thread-001")
	if got.Summary != "Обновлённое резюме" {
		t.Errorf("Summary = %s, want Обновлённое резюме", got.Summary)
	}
}

func TestGetActiveTasksByThread(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём тред
	if err := s.CreateThread(ctx, &store.Thread{ThreadID: "thread-002"}); err != nil {
		t.Fatalf("CreateThread error: %v", err)
	}
	// Создаём задачи
	if err := s.CreateTask(ctx, &store.Task{
		MessageID: "a1", Subject: "Задача 1", BodyText: "B", FromEmail: "u@e.com",
		Status: "new", ThreadID: "thread-002",
	}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := s.CreateTask(ctx, &store.Task{
		MessageID: "a2", Subject: "Задача 2", BodyText: "B", FromEmail: "u@e.com",
		Status: "in_progress", ThreadID: "thread-002",
	}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := s.CreateTask(ctx, &store.Task{
		MessageID: "a3", Subject: "Задача 3", BodyText: "B", FromEmail: "u@e.com",
		Status: "closed", ThreadID: "thread-002",
	}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Проверяем активные задачи (new + in_progress + resolved, без closed)
	tasks, err := s.GetActiveTasksByThread(ctx, "thread-002")
	if err != nil {
		t.Fatalf("GetActiveTasksByThread error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 active tasks, got %d", len(tasks))
	}
}

func TestMigrate_TaskInboxItemsTable(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	exists, err := s.TableExists(context.Background(), "task_inbox_items")
	if err != nil {
		t.Fatalf("TableExists error: %v", err)
	}
	if !exists {
		t.Fatal("task_inbox_items table not created")
	}
}

// ---------------------------------------------------------------------------
// Inbox Items
// ---------------------------------------------------------------------------

func TestTaskInboxLink(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём задачу
	if err := s.CreateTask(ctx, &store.Task{MessageID: "m-link", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Создаём inbox_item
	if err := s.CreateInboxItem(ctx, &store.InboxItem{Source: "email", SourceID: "msg-link-1", ThreadID: "thread-1", Status: "unread"}); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}

	// Связываем
	if err := s.LinkTaskToInboxItem(ctx, 1, 1, "created_from"); err != nil {
		t.Fatalf("LinkTaskToInboxItem error: %v", err)
	}

	// Проверяем связь
	items, err := s.GetInboxItemsByTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetInboxItemsByTask error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 link, got %d", len(items))
	}
	if items[0].Relation != "created_from" {
		t.Errorf("Relation = %s", items[0].Relation)
	}
}

func TestAIQueue_Enqueue(t *testing.T) {
	st, _ := sqlite.NewStore(":memory:")
	_ = st.Migrate(context.Background())
	defer st.Close()

	queue := ai.NewQueue(st, 10)
	queue.Enqueue(42)

	select {
	case id := <-queue.Channel():
		if id != 42 {
			t.Errorf("id = %d, want 42", id)
		}
	default:
		t.Fatal("expected item in queue")
	}
}

func TestAIQueue_LoadPending(t *testing.T) {
	st, _ := sqlite.NewStore(":memory:")
	_ = st.Migrate(context.Background())
	defer st.Close()

	ctx := context.Background()
	if err := st.CreateInboxItem(ctx, &store.InboxItem{Source: "email", SourceID: "m1", Status: "unread"}); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}
	if err := st.CreateInboxItem(ctx, &store.InboxItem{Source: "email", SourceID: "m2", Status: "unread"}); err != nil {
		t.Fatalf("CreateInboxItem error: %v", err)
	}

	queue := ai.NewQueue(st, 10)
	if err := queue.LoadPending(ctx); err != nil {
		t.Fatalf("LoadPending error: %v", err)
	}

	// Оба должны быть в очереди
	for i := 0; i < 2; i++ {
		select {
		case <-queue.Channel():
		default:
			t.Fatalf("expected item %d in queue", i)
		}
	}
}
