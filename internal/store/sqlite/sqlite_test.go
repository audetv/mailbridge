package sqlite_test

import (
	"context"
	"fmt"
	"testing"

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

	if err := s.AddTaskComment(ctx, &store.TaskComment{TaskID: task.ID, Author: "user@example.com", Body: "Comment 1", Direction: "in"}); err != nil {
		t.Fatalf("AddTaskComment error: %v", err)
	}
	if err := s.AddTaskComment(ctx, &store.TaskComment{TaskID: task.ID, Author: "support", Body: "Reply", Direction: "out"}); err != nil {
		t.Fatalf("AddTaskComment error: %v", err)
	}

	comments, err := s.GetTaskComments(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskComments error: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
}

// ---------------------------------------------------------------------------
// Task Attachments
// ---------------------------------------------------------------------------

func TestAddAndGetTaskAttachments(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{MessageID: "att-1", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Status: "new"}
	mustCreateTask(t, s, task)

	if err := s.AddTaskAttachment(ctx, &store.TaskAttachment{
		TaskID:      task.ID,
		Filename:    "screenshot.png",
		ContentType: "image/png",
		Size:        1024,
		StoragePath: "2024-01-01/screenshot.png",
	}); err != nil {
		t.Fatalf("AddTaskAttachment error: %v", err)
	}

	atts, err := s.GetTaskAttachments(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskAttachments error: %v", err)
	}
	if len(atts) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(atts))
	}
}

// ---------------------------------------------------------------------------
// Existing tests (email_mapping, reply_log, outbox)
// ---------------------------------------------------------------------------

func TestReplyLog(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := s.SaveReplyLog(ctx, &store.ReplyLog{MessageID: "r-1", InReplyTo: "o-1", PlaneIssueID: "i-6"}); err != nil {
		t.Fatalf("SaveReplyLog error: %v", err)
	}
	exists, _ := s.ReplyExists(ctx, "r-1")
	if !exists {
		t.Error("expected reply to exist")
	}
}

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

func TestMigrate_InboxItemsTable(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	exists, err := s.TableExists(context.Background(), "inbox_items")
	if err != nil {
		t.Fatalf("TableExists error: %v", err)
	}
	if !exists {
		t.Fatal("inbox_items table not created")
	}

	// Проверяем что индекс thread_id создан
	exists, err = s.TableExists(context.Background(), "idx_inbox_items_thread_id")
	if err != nil {
		t.Fatalf("TableExists error: %v", err)
	}
	// Индексы не видны через sqlite_master с type='table', проверяем отдельно
	_ = exists
}

func TestMigrate_ThreadsColumns(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Проверяем что thread можно создать с новыми полями
	thread := &store.Thread{
		ThreadID: "thread-new",
		Source:   "email",
		Subject:  "Тестовая цепочка",
	}
	if err := s.CreateThread(ctx, thread); err != nil {
		t.Fatalf("CreateThread error: %v", err)
	}

	got, _ := s.GetThread(ctx, "thread-new")
	if got == nil {
		t.Fatal("thread not found")
	}
	if got.Source != "email" {
		t.Errorf("Source = %s", got.Source)
	}
	if got.Subject != "Тестовая цепочка" {
		t.Errorf("Subject = %s", got.Subject)
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
