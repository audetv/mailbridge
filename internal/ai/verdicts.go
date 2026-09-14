package ai

import (
	"context"
	"fmt"
	"log"

	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
)

// ApplyVerdicts применяет решения LLM к БД.
func (o *Orchestrator) ApplyVerdicts(ctx context.Context, email *extractor.ExtractedEmail, response *LLMResponse, inboxItemID int64) error {
	if response == nil || len(response.Verdicts) == 0 {
		return nil
	}

	for i, verdict := range response.Verdicts {
		switch verdict.Action {
		case "new":
			if verdict.Task != nil {
				if err := o.createTaskFromVerdict(ctx, email, verdict, inboxItemID, i); err != nil {
					return fmt.Errorf("failed to create task: %w", err)
				}
			}

		case "update":
			if verdict.TaskID != nil && verdict.Updates != nil {
				if err := o.updateTaskFromVerdict(ctx, *verdict.TaskID, verdict, inboxItemID); err != nil {
					return fmt.Errorf("failed to update task: %w", err)
				}
			}

		case "completed":
			if verdict.TaskID != nil {
				if err := o.completeTaskFromVerdict(ctx, *verdict.TaskID, verdict, inboxItemID); err != nil {
					return fmt.Errorf("failed to complete task: %w", err)
				}
			} else {
				if verdict.Comment != "" {
					if err := o.createCompletedTaskFromVerdict(ctx, email, verdict, inboxItemID, i); err != nil {
						return fmt.Errorf("failed to create resolved task: %w", err)
					}
				}
			}

		case "none":
			// Ничего не делаем
		}
	}

	return nil
}

// copyInboxAttachmentsToTask — общая функция: переносит вложения входящего
// в задачу (идемпотентно, дубли по hash/filename невозможны: связь — по ID
// вложения, INSERT OR IGNORE).
func (o *Orchestrator) copyInboxAttachmentsToTask(ctx context.Context, taskID, inboxItemID int64) {
	if inboxItemID <= 0 {
		return
	}
	if n, err := o.store.CopyInboxAttachmentsToTask(ctx, taskID, inboxItemID); err != nil {
		log.Printf("[AI] failed to copy inbox attachments to task %d: %v", taskID, err)
	} else if n > 0 {
		log.Printf("[AI] copied %d attachment(s) from inbox item %d to task %d", n, inboxItemID, taskID)
	}
}

// createTaskFromVerdict создаёт новую задачу.
func (o *Orchestrator) createTaskFromVerdict(ctx context.Context, email *extractor.ExtractedEmail, verdict Verdict, inboxItemID int64, verdictIndex int) error {
	// Уникальный MessageID для каждой задачи из одного письма
	uniqueMessageID := fmt.Sprintf("%s-task-%d", email.MessageID, verdictIndex)

	task := &store.Task{
		MessageID:     uniqueMessageID,
		Subject:       verdict.Task.Title,
		BodyText:      verdict.Task.Description,
		FromEmail:     email.From,
		FromName:      extractName(email.From),
		Project:       o.ResolveVerdictProject(ctx, verdict.Task.Project),
		Type:          verdict.Task.Type,
		Priority:      verdict.Task.Priority,
		Status:        "new",
		ThreadID:      determineThreadID(email),
		SourceEmailID: email.MessageID,
		AIVerdict:     verdictToJSON(verdict),
	}

	imageNote := verdict.ImageNote
	if imageNote == "" && verdict.Task != nil {
		imageNote = verdict.Task.ImageNote
	}
	if imageNote != "" {
		task.BodyText = verdict.Task.Description + "\n\n[Изображение]: " + imageNote
	}

	if err := o.store.CreateTask(ctx, task); err != nil {
		return err
	}

	// Вложения входящего достают в задачу (общий хелпер, идемпотентно)
	o.copyInboxAttachmentsToTask(ctx, task.ID, inboxItemID)

	if inboxItemID > 0 {
		if err := o.store.LinkTaskToInboxItem(ctx, task.ID, inboxItemID, "created_from"); err != nil {
			log.Printf("[AI] failed to link task to inbox: %v", err)
		}
	}

	return nil
}

// updateTaskFromVerdict обновляет существующую задачу.
func (o *Orchestrator) updateTaskFromVerdict(ctx context.Context, taskID int, verdict Verdict, inboxItemID int64) error {
	updates := make(map[string]interface{})

	if verdict.Updates.Priority != "" {
		updates["priority"] = verdict.Updates.Priority
	}
	if verdict.Updates.ChangeStatus != "" {
		updates["status"] = verdict.Updates.ChangeStatus
	}

	if len(updates) > 0 {
		// status — отдельным вызовом SetTaskStatus: пишет и строку истории
		// (v0.23, шаг 2; единый путь смены статуса во всех слоях).
		if status, ok := updates["status"].(string); ok {
			delete(updates, "status")
			if err := o.store.SetTaskStatus(ctx, int64(taskID), status, "ai"); err != nil {
				return err
			}
		}
		if len(updates) > 0 {
			if err := o.store.UpdateTask(ctx, int64(taskID), updates); err != nil {
				return err
			}
		}
	}

	var userComment *store.TaskComment
	var aiComment *store.TaskComment

	// Пользовательский комментарий
	if verdict.Updates.AddComment != "" {
		userComment = &store.TaskComment{
			TaskID:      int64(taskID),
			Author:      "user",
			Body:        verdict.Updates.AddComment,
			Direction:   "in",
			Kind:        "user_comment",
			InboxItemID: int64Ptr(inboxItemID),
		}
		if err := o.store.AddTaskComment(ctx, userComment); err != nil {
			return err
		}
	}

	// AI-вердикт как комментарий
	verdictJSON := verdictToJSON(verdict)
	aiComment = &store.TaskComment{
		TaskID:      int64(taskID),
		Author:      "ai",
		Body:        fmt.Sprintf("Задача обновлена: %s", verdict.Updates.AddComment),
		Direction:   "in",
		Kind:        "ai_verdict",
		InboxItemID: int64Ptr(inboxItemID),
		VerdictJSON: verdictJSON,
	}
	if err := o.store.AddTaskComment(ctx, aiComment); err != nil {
		return err
	}

	// Привязываем вложения входящего к комментариям
	if inboxItemID > 0 {
		inboxAtts, err := o.store.GetAttachmentsByInbox(ctx, inboxItemID)
		if err == nil {
			for _, att := range inboxAtts {
				if userComment != nil {
					if err := o.store.LinkAttachmentToComment(ctx, userComment.ID, att.ID); err != nil {
						log.Printf("[AI] failed to link attachment to user comment: %v", err)
					}
				}
				if err := o.store.LinkAttachmentToComment(ctx, aiComment.ID, att.ID); err != nil {
					log.Printf("[AI] failed to link attachment to ai comment: %v", err)
				}
			}
		}
	}

	// Вложения входящего достают в задаu (update-путь)
	o.copyInboxAttachmentsToTask(ctx, int64(taskID), inboxItemID)

	// Связь с входящим
	if inboxItemID > 0 {
		if err := o.store.LinkTaskToInboxItem(ctx, int64(taskID), inboxItemID, "updated_by"); err != nil {
			log.Printf("[AI] failed to link task to inbox: %v", err)
		}
	}

	return nil
}

func int64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

// completeTaskFromVerdict завершает задачу.
func (o *Orchestrator) completeTaskFromVerdict(ctx context.Context, taskID int, verdict Verdict, inboxItemID int64) error {
	// status → SetTaskStatus: строка в task_status_history (by=ai) + статус.
	if err := o.store.SetTaskStatus(ctx, int64(taskID), "completed", "ai"); err != nil {
		return err
	}

	var userComment *store.TaskComment
	var aiComment *store.TaskComment

	// Пользовательский комментарий
	if verdict.Comment != "" {
		userComment = &store.TaskComment{
			TaskID:      int64(taskID),
			Author:      "user",
			Body:        verdict.Comment,
			Direction:   "in",
			Kind:        "user_comment",
			InboxItemID: int64Ptr(inboxItemID),
		}
		if err := o.store.AddTaskComment(ctx, userComment); err != nil {
			return err
		}
	}

	// AI-вердикт
	verdictJSON := verdictToJSON(verdict)
	aiComment = &store.TaskComment{
		TaskID:      int64(taskID),
		Author:      "ai",
		Body:        "Задача завершена",
		Direction:   "in",
		Kind:        "ai_verdict",
		InboxItemID: int64Ptr(inboxItemID),
		VerdictJSON: verdictJSON,
	}
	if err := o.store.AddTaskComment(ctx, aiComment); err != nil {
		return err
	}

	// Привязываем вложения входящего к комментариям
	if inboxItemID > 0 {
		inboxAtts, err := o.store.GetAttachmentsByInbox(ctx, inboxItemID)
		if err == nil {
			for _, att := range inboxAtts {
				if userComment != nil {
					if err := o.store.LinkAttachmentToComment(ctx, userComment.ID, att.ID); err != nil {
						log.Printf("[AI] failed to link attachment to user comment: %v", err)
					}
				}
				if err := o.store.LinkAttachmentToComment(ctx, aiComment.ID, att.ID); err != nil {
					log.Printf("[AI] failed to link attachment to ai comment: %v", err)
				}
			}
		}
	}

	// Вложения входящего достают в задаu (complete-путь)
	o.copyInboxAttachmentsToTask(ctx, int64(taskID), inboxItemID)

	// Связь с входящим
	if inboxItemID > 0 {
		if err := o.store.LinkTaskToInboxItem(ctx, int64(taskID), inboxItemID, "completed_by"); err != nil {
			log.Printf("[AI] failed to link task to inbox: %v", err)
		}
	}

	return nil
}

// createCompletedTaskFromVerdict создаёт задачу со статусом completed (из пересланного письма).
func (o *Orchestrator) createCompletedTaskFromVerdict(ctx context.Context, email *extractor.ExtractedEmail, verdict Verdict, inboxItemID int64, verdictIndex int) error {
	uniqueMessageID := fmt.Sprintf("%s-task-%d", email.MessageID, verdictIndex)
	title := "Выполнено: " + email.Subject
	desc := verdict.Comment
	proj := "Входящие"
	tType := "support"
	prio := "medium"
	sourceID := email.MessageID

	if verdict.Task != nil {
		if verdict.Task.Title != "" {
			title = verdict.Task.Title
		}
		if verdict.Task.Description != "" {
			desc = verdict.Task.Description
		}
		if verdict.Task.Project != "" {
			proj = verdict.Task.Project
		}
		if verdict.Task.Type != "" {
			tType = verdict.Task.Type
		}
		if verdict.Task.Priority != "" {
			prio = verdict.Task.Priority
		}
		if verdict.Task.SourceEmailID != "" {
			sourceID = verdict.Task.SourceEmailID
		}
	}

	if verdict.Comment != "" && desc != verdict.Comment {
		desc = desc + "\n\nКомментарий: " + verdict.Comment
	}

	task := &store.Task{
		MessageID:     uniqueMessageID,
		Subject:       title,
		BodyText:      desc,
		FromEmail:     email.From,
		FromName:      extractName(email.From),
		Project:       proj,
		Type:          tType,
		Priority:      prio,
		Status:        string(store.StatusCompleted),
		ThreadID:      determineThreadID(email),
		SourceEmailID: sourceID,
		AIVerdict:     verdictToJSON(verdict),
	}

	if err := o.store.CreateTask(ctx, task); err != nil {
		return err
	}

	// Вложения входящего достают в задачу (общий хелпер, идемпотентно)
	o.copyInboxAttachmentsToTask(ctx, task.ID, inboxItemID)

	if inboxItemID > 0 {
		if err := o.store.LinkTaskToInboxItem(ctx, task.ID, inboxItemID, "created_from"); err != nil {
			log.Printf("[AI] failed to link task to inbox: %v", err)
		}
	}

	return nil
}
