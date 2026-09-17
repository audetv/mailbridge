package ai_test

// Шаг 7b v0.25 (Due date, AI-предложение срока):
// вердикт несёт due_date → ai_due_date + due_ai_pending=1 (дефолт, решение
// человека — 7c) ИЛИ авто-принятие (флаг SetAutoApplyDue → due_date +
// due_source='ai', pending=0). Ручной срок (due_date/due_source) AI НЕ трогает,
// срок «скоро»/мусор не предлагается, срок null — ничего (не выдумываем).
// Тесты без LLM-модели (реальный sqlite :memory:, как шаг 5).

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

func ptr[T any](v T) *T { return &v }

func newStoreFixture(t *testing.T) *sqlite.Store {
	t.Helper()
	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return st
}

func newEmail() *extractor.ExtractedEmail {
	return &extractor.ExtractedEmail{
		MessageID: "due-x",
		From:      "vika@example.com",
		Subject:   "Сроки",
		BodyText:  "Нужно сделать до 20.09",
	}
}

// newTask — new-вердикт с заданным DueDate; возвращает задачу из цепочки.
func newTask(t *testing.T, o *ai.Orchestrator, st *sqlite.Store, due *string) *store.Task {
	t.Helper()
	email := newEmail()
	resp := &ai.LLMResponse{
		Verdicts: []ai.Verdict{{
			Action: "new",
			Task: &ai.NewTaskData{
				Title:    "Задача 7b",
				Priority: "medium",
				Project:  "Входящие",
				Type:     "task",
				DueDate:  due,
			},
		}},
	}
	if err := o.ApplyVerdicts(context.Background(), email, resp, 0); err != nil {
		t.Fatalf("ApplyVerdicts(new): %v", err)
	}
	tasks, err := st.GetActiveTasksByThread(context.Background(), email.MessageID)
	if err != nil {
		t.Fatalf("GetActiveTasksByThread: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	return tasks[0]
}

func TestApplyDueDate_New_ProposalDefault(t *testing.T) {
	st := newStoreFixture(t)
	o := ai.NewOrchestrator(nil, st) // дефолт: авто-принятие ВЫКЛЕНО

	task := newTask(t, o, st, ptr("2026-09-20"))

	if task.DueDate != nil {
		t.Errorf("DueDate = %q, want nil (дефолт: предложение, не принятие)", *task.DueDate)
	}
	if task.DueSource != nil {
		t.Errorf("DueSource = %q, want nil (ручное решение — 7c)", *task.DueSource)
	}
	if task.DueAIPending == nil || !*task.DueAIPending {
		t.Errorf("DueAIPending = %v, want true (предложение ждёт решения)", task.DueAIPending)
	}
	if task.AIDueDate == nil || *task.AIDueDate != "2026-09-20" {
		t.Errorf("AIDueDate = %v, want 2026-09-20 (база ошибок AI — ВСЕГДА)", task.AIDueDate)
	}
}

func TestApplyDueDate_New_AutoApply(t *testing.T) {
	st := newStoreFixture(t)
	o := ai.NewOrchestrator(nil, st)
	o.SetAutoApplyDue(true) // MAILBRIDGE_AI_AUTO_APPLY_DUE=true

	task := newTask(t, o, st, ptr("2026-10-05"))

	if task.DueDate == nil || *task.DueDate != "2026-10-05" {
		t.Errorf("DueDate = %v, want 2026-10-05 (авто-принято)", task.DueDate)
	}
	if task.DueSource == nil || *task.DueSource != "ai" {
		t.Errorf("DueSource = %v, want ai", task.DueSource)
	}
	if task.DueAIPending != nil && *task.DueAIPending {
		t.Errorf("DueAIPending = %v, want false/null (уже принято)", *task.DueAIPending)
	}
	if task.AIDueDate == nil || *task.AIDueDate != "2026-10-05" {
		t.Errorf("AIDueDate = %v, want 2026-10-05 (хранится ВСЕГДА)", task.AIDueDate)
	}
}

func TestApplyDueDate_New_NoDue(t *testing.T) {
	st := newStoreFixture(t)
	o := ai.NewOrchestrator(nil, st)

	task := newTask(t, o, st, nil) // срок не извлечён — null

	if task.AIDueDate != nil {
		t.Errorf("AIDueDate = %q, want nil (срок не извлечён)", *task.AIDueDate)
	}
	if task.DueAIPending != nil && *task.DueAIPending {
		t.Errorf("DueAIPending = true, want false/null (не предложено)")
	}
	if task.DueDate != nil || task.DueSource != nil {
		t.Errorf("DueDate=%v DueSource=%v, want nil/nil", task.DueDate, task.DueSource)
	}
}

func TestApplyDueDate_New_JunkRejected(t *testing.T) {
	// «скоро»/мусор из письма → срок НЕ предлагается (никогда не выдумывать),
	// ai_due_date не записывается предложением.
	st := newStoreFixture(t)
	o := ai.NewOrchestrator(nil, st)

	task := newTask(t, o, st, ptr("скоро"))

	if task.AIDueDate != nil {
		t.Errorf("AIDueDate = %q, want nil (мусор не записывается)", *task.AIDueDate)
	}
	if task.DueAIPending != nil && *task.DueAIPending {
		t.Errorf("DueAIPending = true, want false/null")
	}
	if task.DueDate != nil {
		t.Errorf("DueDate = %q, want nil", *task.DueDate)
	}
}

func TestApplyDueDate_New_TimeStripped(t *testing.T) {
	// Срок с временем нормализуется в YYYY-MM-DD (онтология v0.5.2 §7.5).
	st := newStoreFixture(t)
	o := ai.NewOrchestrator(nil, st)

	task := newTask(t, o, st, ptr("2026-10-01 14:00:00"))

	if task.AIDueDate == nil || *task.AIDueDate != "2026-10-01" {
		t.Errorf("AIDueDate = %v, want 2026-10-01 (время срублено)", task.AIDueDate)
	}
}

func TestApplyDueDate_Update_Path(t *testing.T) {
	st := newStoreFixture(t)
	o := ai.NewOrchestrator(nil, st)
	ctx := context.Background()

	// Активная задача в цепочке письма — для update-вердикта.
	base := &store.Task{
		MessageID: "due-x-base",
		Subject:   "Базовая задача",
		BodyText:  "текст",
		Project:   "Входящие",
		Type:      "task",
		Priority:  "medium",
		Status:    "in_progress",
		ThreadID:  "due-x",
	}
	if err := st.CreateTask(ctx, base); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	email := newEmail()
	resp := &ai.LLMResponse{
		Verdicts: []ai.Verdict{{
			Action: "update",
			TaskID: ptr(int(base.ID)),
			Updates: &ai.TaskUpdates{
				AddComment: "Уточнить срок",
				Quote:      "делать до 25.09",
				DueDate:    ptr("2026-09-25"),
			},
		}},
	}
	if err := o.ApplyVerdicts(ctx, email, resp, 0); err != nil {
		t.Fatalf("ApplyVerdicts(update): %v", err)
	}

	task, err := st.GetTask(ctx, base.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.AIDueDate == nil || *task.AIDueDate != "2026-09-25" {
		t.Errorf("AIDueDate = %v, want 2026-09-25", task.AIDueDate)
	}
	if task.DueAIPending == nil || !*task.DueAIPending {
		t.Errorf("DueAIPending = %v, want true (дефолт: предложение)", task.DueAIPending)
	}
	if task.DueDate != nil || task.DueSource != nil {
		t.Errorf("DueDate=%v DueSource=%v — AI не должно трогать ручной срок", task.DueDate, task.DueSource)
	}
}
