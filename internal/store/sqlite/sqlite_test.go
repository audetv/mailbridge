package sqlite_test

import (
	"context"
	"fmt"
	"testing"
	"time"

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

// ---------------------------------------------------------------------------
// v0.28.0, шаг 28a — план задачи (scheduled_date)
// ---------------------------------------------------------------------------

func TestScheduledDate_MigrationColumn(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Колонка появляется миграцией идемпотентно (схема свежей БД = ALTER).
	has, err := s.ColumnExistsForTest(ctx, "tasks", "scheduled_date")
	if err != nil {
		t.Fatalf("ColumnExistsForTest: %v", err)
	}
	if !has {
		t.Fatal("tasks.scheduled_date отсутствует после миграции")
	}

	// Повторная миграция (идемпотентность) не ломает и не дублирует.
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestScheduledDate_CreateGet(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	t1 := &store.Task{MessageID: "sched-1", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new",
		ScheduledDate: strp("2026-10-01")}
	mustCreateTask(t, s, t1)

	got, err := s.GetTask(ctx, t1.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.ScheduledDate == nil || *got.ScheduledDate != "2026-10-01" {
		t.Errorf("scheduled_date = %v, want 2026-10-01", got.ScheduledDate)
	}
	if got.DueDate != nil {
		t.Errorf("due_date = %v, want nil (не задана)", got.DueDate)
	}

	// Без плана — null (поле просто отсутствует).
	t2 := &store.Task{MessageID: "sched-2", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
	mustCreateTask(t, s, t2)
	g2, err := s.GetTask(ctx, t2.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if g2.ScheduledDate != nil {
		t.Errorf("scheduled_date = %v, want nil", g2.ScheduledDate)
	}
}

func TestScheduledDate_UpdateAndClear(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	tk := &store.Task{MessageID: "sched-u", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
	mustCreateTask(t, s, tk)

	// Установка плана (как делает API: scheduled_date как есть).
	if err := s.UpdateTask(ctx, tk.ID, map[string]interface{}{"scheduled_date": "2026-12-31"}); err != nil {
		t.Fatalf("UpdateTask set: %v", err)
	}
	g, _ := s.GetTask(ctx, tk.ID)
	if g.ScheduledDate == nil || *g.ScheduledDate != "2026-12-31" {
		t.Errorf("after set, scheduled_date = %v, want 2026-12-31", g.ScheduledDate)
	}

	// Снятие плана (NULL).
	if err := s.UpdateTask(ctx, tk.ID, map[string]interface{}{"scheduled_date": nil}); err != nil {
		t.Fatalf("UpdateTask clear: %v", err)
	}
	g2, _ := s.GetTask(ctx, tk.ID)
	if g2.ScheduledDate != nil {
		t.Errorf("after clear, scheduled_date = %v, want nil", g2.ScheduledDate)
	}

	// due_date при этом не трогается (две независимые даты).
	if g2.DueDate != nil {
		t.Errorf("due_date = %v, want nil (scheduled не влияет на due)", g2.DueDate)
	}
}

func TestScheduledDate_ListReturnsField(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	mustCreateTask(t, s, &store.Task{
		MessageID: "sched-list", Subject: "T", BodyText: "B", FromEmail: "u@e.com",
		Project: "ТРК", Status: "new", ScheduledDate: strp("2026-11-11"),
	})

	res, err := s.ListTasks(ctx, &store.TaskFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(res.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(res.Tasks))
	}
	got := res.Tasks[0].ScheduledDate
	if got == nil || *got != "2026-11-11" {
		t.Errorf("ListTasks scheduled_date = %v, want 2026-11-11", got)
	}
}

func TestListTasks_ListSortCreatedDefault(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// v0.29.0 (issue #81): дефолтная сортировка — «По дате создания» (новые сверху).
	// Те же данные, что в старом дефолт-тесте (due): создаём 3 с разными
	// сроками + 1 без срока. С v0.29.0 дефолт — created: самая новая (dNone)
	// первой, самая старая (dOverdue) последней.
	d := func(id, date string) *store.Task {
		tk := &store.Task{MessageID: id, Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
		if date != "" {
			tk.DueDate = strp(date)
			tk.DueSource = strp("manual")
		}
		return tk
	}
	dOverdue := d("due-over", "2026-01-05") // просроченный — срок больше НЕ приоритетен
	dMid := d("due-mid", "2026-10-15")
	dFar := d("due-far", "2026-12-01")
	dNone := d("due-none", "")
	mustCreateTask(t, s, dOverdue)
	mustCreateTask(t, s, dMid)
	mustCreateTask(t, s, dFar)
	mustCreateTask(t, s, dNone)

	// Дефолтная сортировка ("created"): новые сверху — dNone → dFar → dMid → dOverdue.
	res, err := s.ListTasks(ctx, &store.TaskFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	var ids []string
	for _, tk := range res.Tasks {
		ids = append(ids, tk.MessageID)
	}
	want := []string{"due-none", "due-far", "due-mid", "due-over"}
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Errorf("default order = %v, want %v (newest first)", ids, want)
	}
}

// В явном режиме sort=due (v0.27.1 хотфикс): в группе «без срока» — новые выше
// (created_at DESC), а не старые (id ASC). Создаём 3 без срока последовательно —
// created_at монотонно растут, порядок «последние сверху» отличим от старого.
func TestListTasks_ListSortDue_NoDueByCreatedDesc(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	d := func(id, date string) *store.Task {
		tk := &store.Task{MessageID: id, Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
		if date != "" {
			tk.DueDate = strp(date)
			tk.DueSource = strp("manual")
		}
		return tk
	}
	mustCreateTask(t, s, d("due-task", "2026-10-01"))
	n1 := d("n1", "") // созданная первой — должна уйти ВНИЗ группы без срока
	n2 := d("n2", "")
	n3 := d("n3", "") // созданная последней — должна быть ПЕРВОЙ в группе без срока
	mustCreateTask(t, s, n1)
	mustCreateTask(t, s, n2)
	mustCreateTask(t, s, n3)

	res, err := s.ListTasks(ctx, &store.TaskFilter{Sort: "due", Page: 1, PerPage: 10})
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
	// 1) задачи со сроком — сверху (приоритет по сроку сохранён).
	if ids[0] != "due-task" {
		t.Errorf("first = %v, want due-task (срок сохраняется приоритетом)", ids[0])
	}
	// 2) группа без срока — новые выше: n3 → n2 → n1 (а не n1 → n2 → n3).
	if ids[1] != "n3" || ids[2] != "n2" || ids[3] != "n1" {
		t.Errorf("no-due segment = %v, want [n3 n2 n1] (сначала последние созданные)", ids[1:])
	}
}

// Явный sort=created (v0.29.0): те же 4 задачи — порядок строго по созданию,
// срок ни на что не влияет.
func TestListTasks_ListSortCreated(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	mk := func(id string) {
		tk := &store.Task{MessageID: id, Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
		mustCreateTask(t, s, tk)
	}
	mk("cr-old") // первая — уходит последней
	mk("cr-mid")
	mk("cr-new") // последняя — наверху

	res, err := s.ListTasks(ctx, &store.TaskFilter{Sort: "created", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(res.Tasks) != 3 {
		t.Fatalf("expected 3, got %d", len(res.Tasks))
	}
	if res.Tasks[0].MessageID != "cr-new" || res.Tasks[2].MessageID != "cr-old" {
		t.Errorf("sort=created: first/last = %v/%v, want cr-new/cr-old", res.Tasks[0].MessageID, res.Tasks[2].MessageID)
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

// v0.26, шаг 7d: фильтры по срокам (due=overdue|today|tomorrow|7d|30d|none|due_pending).
func TestDueDate_FilterByDueBuckets(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	in7 := now.AddDate(0, 0, 5).Format("2006-01-02")
	in30 := now.AddDate(0, 0, 25).Format("2006-01-02")    // в пределах 30-дневного, вне 7-дневного
	after30 := now.AddDate(0, 0, 33).Format("2006-01-02") // вне даже 30-дневного окна
	overdue := now.AddDate(0, 0, -3).Format("2006-01-02")

	mk := func(id, date string) *store.Task {
		tk := &store.Task{MessageID: id, Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
		if date != "" {
			tk.DueDate = strp(date)
			tk.DueSource = strp("manual")
		}
		mustCreateTask(t, s, tk)
		return tk
	}
	pending := mk("due-pend", after30)
	_ = s.SetTaskDueAIPending(ctx, pending.ID, strp(after30), true)
	mk("due-none", "")
	mk("due-overdue", overdue)
	mk("due-today", today)
	mk("due-tomorrow", tomorrow)
	mk("due-in7", in7)
	mk("due-in30", in30)
	mk("due-far", after30)

	want := map[string][]string{
		"overdue":     {"due-overdue"},
		"today":       {"due-today"},
		"tomorrow":    {"due-tomorrow"},
		"7d":          {"due-today", "due-tomorrow", "due-in7"},
		"30d":         {"due-today", "due-tomorrow", "due-in7", "due-in30"},
		"none":        {"due-none"},
		"due_pending": {"due-pend"},
	}
	for _, d := range []string{"overdue", "today", "tomorrow", "7d", "30d", "none", "due_pending"} {
		res, err := s.ListTasks(ctx, &store.TaskFilter{Due: d, Page: 1, PerPage: 20})
		if err != nil {
			t.Fatalf("ListTasks due=%s: %v", d, err)
		}
		got := map[string]bool{}
		for _, tk := range res.Tasks {
			got[tk.MessageID] = true
		}
		if len(res.Tasks) != len(want[d]) {
			t.Errorf("due=%s: got %d tasks (%v), want %v", d, len(res.Tasks), got, want[d])
			continue
		}
		for _, id := range want[d] {
			if !got[id] {
				t.Errorf("due=%s: missing %s in %v", d, id, got)
			}
		}
	}

	// Неизвестное значение = фильтр не применяется (store его игнорирует;
	// API-слой отвечает 400 — здесь проверяем только устойчивость store).
	res, err := s.ListTasks(ctx, &store.TaskFilter{Due: "bogus", Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("ListTasks due=bogus: %v", err)
	}
	if len(res.Tasks) != 8 {
		t.Errorf("due=bogus: expected no filter applied (8 tasks), got %d", len(res.Tasks))
	}
}

// v0.28, шаг 28b: срез «План» (plan=today|tomorrow|week).
// Семантика (решение владельца 2026-09-19): окно scheduled (today = сегодня,
// tomorrow = завтра, week = 7 дней включая сегодня) + ВСЕГДА просроченные due
// (due_date < сегодня, активные статусы new/in_progress/backlog) —
// «совсем не видеть просроченные» отклонено. completed/closed — вне плана.
func TestPlan_FilterByPlanBuckets(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now()
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	inWeek := now.AddDate(0, 0, 5).Format("2006-01-02")  // в пределах 7-дневного окна
	outWeek := now.AddDate(0, 0, 8).Format("2006-01-02") // вне окна
	overdue := now.AddDate(0, 0, -2).Format("2006-01-02")

	mk := func(mid, status, due, sched string) {
		tk := &store.Task{MessageID: mid, Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: status}
		if due != "" {
			tk.DueDate = strp(due)
			tk.DueSource = strp("manual")
		}
		if sched != "" {
			tk.ScheduledDate = strp(sched)
		}
		mustCreateTask(t, s, tk)
	}
	mk("p-sched-today", "new", "", today)
	mk("p-sched-tomorrow", "new", "", tomorrow)
	mk("p-sched-week", "new", "", inWeek)
	mk("p-sched-out", "new", "", outWeek)
	mk("p-overdue-new", "new", overdue, "")
	mk("p-overdue-prog", "in_progress", overdue, "")
	mk("p-overdue-backlog", "backlog", overdue, "")
	// Просроченные due, НО закрытые статусы — вне плана (решение 28b).
	mk("p-overdue-done", "completed", overdue, "")
	mk("p-overdue-closed", "closed", overdue, "")
	// Без обеих дат — не в плане.
	mk("p-bare", "new", "", "")
	// due = сегодня — это НЕ просрочено (просрочка = due < сегодня).
	mk("p-due-today", "new", today, "")

	want := map[string][]string{
		// scheduled сегодня + все просроченные due (активные статусы).
		"today": {"p-sched-today", "p-overdue-new", "p-overdue-prog", "p-overdue-backlog"},
		// scheduled завтра + все просроченные due (просрочка — всегда).
		"tomorrow": {"p-sched-tomorrow", "p-overdue-new", "p-overdue-prog", "p-overdue-backlog"},
		// scheduled в 7-дневном окне (сегодня..+6; +8 — вне) + просроченные.
		"week": {"p-sched-today", "p-sched-tomorrow", "p-sched-week", "p-overdue-new", "p-overdue-prog", "p-overdue-backlog"},
		// 28e: «Без плана» = scheduled_date IS NULL (срез по дате НЕ добавляется
		// — это фильтр по полю scheduled; просроченные due — не в окне):
		// у p-sched-today/tomorrow/week/out scheduled_date ЕСТЬ — их нет.
		"none": {"p-overdue-new", "p-overdue-prog", "p-overdue-backlog", "p-overdue-done", "p-overdue-closed", "p-bare", "p-due-today"},
	}
	for _, p := range []string{"today", "tomorrow", "week", "none"} {
		res, err := s.ListTasks(ctx, &store.TaskFilter{Plan: p, Page: 1, PerPage: 50})
		if err != nil {
			t.Fatalf("ListTasks plan=%s: %v", p, err)
		}
		got := map[string]bool{}
		for _, tk := range res.Tasks {
			got[tk.MessageID] = true
		}
		if len(res.Tasks) != len(want[p]) {
			t.Errorf("plan=%s: got %d tasks (%v), want %v", p, len(res.Tasks), gotKeys(got), want[p])
			continue
		}
		for _, id := range want[p] {
			if !got[id] {
				t.Errorf("plan=%s: missing %s in %v", p, id, gotKeys(got))
			}
		}
	}

	// Неизвестное значение — фильтр игнорируется (400 отвечает API-слой).
	res, err := s.ListTasks(ctx, &store.TaskFilter{Plan: "bogus", Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("ListTasks plan=bogus: %v", err)
	}
	if len(res.Tasks) != 11 {
		t.Errorf("plan=bogus: expected no filter applied (11 tasks), got %d", len(res.Tasks))
	}
}

func gotKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// v0.26, шаг 7d: вклад-активность — новый комментарий поднимает
// tasks.updated_at (сортировка «по активности» учитывает ответы, не только
// правки метаданных).
func TestAddTaskComment_BumpsUpdatedAt(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	task := &store.Task{MessageID: "touch-1", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
	mustCreateTask(t, s, task)
	before, err := s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	// Даем обновлению заметную просадку (updated_at имеет секундную точность).
	time.Sleep(1100 * time.Millisecond)
	if err := s.AddTaskComment(ctx, &store.TaskComment{TaskID: task.ID, Body: "reply"}); err != nil {
		t.Fatalf("AddTaskComment: %v", err)
	}
	after, err := s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("expected updated_at to bump after comment: before=%v after=%v", before.UpdatedAt, after.UpdatedAt)
	}
}

// v0.26, шаг 7d: sort=updated после ответа — отвеченная задача поднимается.
func TestListTasks_SortUpdatedAfterReply(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	quiet := &store.Task{MessageID: "act-quiet", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
	busy := &store.Task{MessageID: "act-busy", Subject: "T", BodyText: "B", FromEmail: "u@e.com", Project: "ТРК", Status: "new"}
	mustCreateTask(t, s, quiet)
	mustCreateTask(t, s, busy)

	time.Sleep(50 * time.Millisecond)
	if err := s.AddTaskComment(ctx, &store.TaskComment{TaskID: busy.ID, Body: "hot thread"}); err != nil {
		t.Fatalf("AddTaskComment: %v", err)
	}

	res, err := s.ListTasks(ctx, &store.TaskFilter{Sort: "updated", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(res.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(res.Tasks))
	}
	if res.Tasks[0].ID != busy.ID {
		t.Errorf("first with sort=updated after reply should be %d, got %d", busy.ID, res.Tasks[0].ID)
	}
}
