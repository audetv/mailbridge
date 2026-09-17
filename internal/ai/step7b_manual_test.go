package ai_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/store"
)

// Критерий 7b: «у задачи с manual-сроком — due_date и due_source не трогается».
func TestApplyDueDate_ManualPriority(t *testing.T) {
	st := newStoreFixture(t)
	o := ai.NewOrchestrator(nil, st)
	ctx := context.Background()

	base := &store.Task{
		MessageID: "due-man-base", Subject: "М", BodyText: "м",
		Project: "Входящие", Type: "task", Priority: "medium",
		Status: "in_progress", ThreadID: "due-man",
	}
	if err := st.CreateTask(ctx, base); err != nil {
		t.Fatal(err)
	}
	manual := "2026-09-10"
	base.DueDate = &manual
	base.DueSource = ptr("manual")
	if err := st.UpdateTask(ctx, base.ID, map[string]interface{}{
		"due_date": "2026-09-10", "due_source": "manual",
	}); err != nil {
		t.Fatal(err)
	}

	email := newEmail()
	resp := &ai.LLMResponse{
		Verdicts: []ai.Verdict{{
			Action: "update",
			TaskID: ptr(int(base.ID)),
			Updates: &ai.TaskUpdates{
				AddComment: "уточнили срок", Quote: "делайте до 1.10",
				DueDate: ptr("2026-10-01"),
			},
		}},
	}
	if err := o.ApplyVerdicts(ctx, email, resp, 0); err != nil {
		t.Fatal(err)
	}

	task, _ := st.GetTask(ctx, base.ID)
	if task.DueDate == nil || *task.DueDate != "2026-09-10" {
		t.Errorf("DueDate = %v, want 2026-09-10 (manual сохранён)", task.DueDate)
	}
	if task.DueSource == nil || *task.DueSource != "manual" {
		t.Errorf("DueSource = %v, want manual (AI не перезаписал)", task.DueSource)
	}
	if task.AIDueDate == nil || *task.AIDueDate != "2026-10-01" {
		t.Errorf("AIDueDate = %v, want 2026-10-01 (предложение хранится)", task.AIDueDate)
	}
}
