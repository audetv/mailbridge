// Package processor реализует оркестрацию обработки входящих писем.
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/audetv/mailbridge/internal/adapters"
	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/classifier"
	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/web"
)

// ActionType определяет тип действия над письмом.
type ActionType string

const (
	// ActionCreateIssue — создать новую задачу.
	ActionCreateIssue ActionType = "CREATE_ISSUE"
	// ActionAddComment — добавить комментарий к существующей задаче.
	ActionAddComment ActionType = "ADD_COMMENT"
	// ActionIgnore — проигнорировать письмо (дубликат).
	ActionIgnore ActionType = "IGNORE"
)

// ProcessResult содержит результат обработки письма.
type ProcessResult struct {
	Action    ActionType
	TaskID    int64
	Task      *store.Task
	InboxItem *store.InboxItem
	Extracted *extractor.ExtractedEmail
	Error     error
}

// MessageProcessor оркестрирует обработку входящего письма.
type MessageProcessor struct {
	store        store.Store
	classifier   classifier.Classifier
	extractor    *extractor.Extractor
	parser       *parser.FieldParser
	config       *config.Config
	logger       *slog.Logger
	projectMap   map[string]string
	broker       *web.EventBroker
	orchestrator *ai.Orchestrator
	aiEnabled    bool
	adapter      adapters.Adapter
	aiQueue      *ai.Queue
}

// NewMessageProcessor создаёт новый MessageProcessor.
func NewMessageProcessor(
	st store.Store,
	cl classifier.Classifier,
	ext *extractor.Extractor,
	par *parser.FieldParser,
	cfg *config.Config,
	logger *slog.Logger,
	projectMap map[string]string,
	broker *web.EventBroker,
	orchestrator *ai.Orchestrator,
	aiEnabled bool,
	adapter adapters.Adapter,
	aiQueue *ai.Queue,
) *MessageProcessor {
	return &MessageProcessor{
		store:        st,
		classifier:   cl,
		extractor:    ext,
		parser:       par,
		config:       cfg,
		logger:       logger,
		projectMap:   projectMap,
		broker:       broker,
		orchestrator: orchestrator,
		aiEnabled:    aiEnabled,
		adapter:      adapter,
		aiQueue:      aiQueue,
	}
}

// Process обрабатывает сырое письмо и возвращает результат.
func (p *MessageProcessor) Process(ctx context.Context, rawEmail []byte) (*ProcessResult, error) {
	// Извлекаем данные письма
	email, err := p.extractor.Extract(rawEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to extract email: %w", err)
	}

	p.logger.Debug("processing email",
		"message_id", email.MessageID,
		"from", email.From,
		"subject", email.Subject,
	)

	// Проверяем что задача с таким Message-ID уже существует
	existingTask, err := p.store.GetTaskByMessageID(ctx, email.MessageID)
	if err == nil && existingTask != nil {
		p.logger.Info("duplicate email — task already exists",
			"message_id", email.MessageID,
			"task_id", existingTask.ID,
		)
		return &ProcessResult{
			Action:    ActionIgnore,
			TaskID:    existingTask.ID,
			Task:      existingTask,
			Extracted: email,
		}, nil
	}

	// Парсим через адаптер и сохраняем в ленту (если адаптер задан)
	var inboxItem *store.InboxItem
	var parseResult *adapters.ParseResult
	if p.adapter != nil {
		parseResult, err = p.adapter.Parse(rawEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to parse incoming: %w", err)
		}
		inboxItem = parseResult.InboxItem

		// Проверяем дубликат в ленте
		existingItem, err := p.store.GetInboxItemBySourceID(ctx, inboxItem.Source, inboxItem.SourceID)
		if err == nil && existingItem != nil {
			p.logger.Info("duplicate inbox item, skipping",
				"source", inboxItem.Source,
				"source_id", inboxItem.SourceID,
			)
			return &ProcessResult{
				Action:    ActionIgnore,
				InboxItem: existingItem,
				Extracted: email,
			}, nil
		}

		if err := p.store.CreateInboxItem(ctx, inboxItem); err != nil {
			return nil, fmt.Errorf("failed to save inbox item: %w", err)
		}

		// Связываем вложения ПОСЛЕ создания inbox_item
		for _, att := range parseResult.Attachments {
			if err := p.store.LinkAttachmentToInbox(ctx, inboxItem.ID, att.ID); err != nil {
				p.logger.Warn("failed to link attachment to inbox", "error", err)
			}
		}

		// Публикуем WebSocket-событие о новом входящем
		if p.broker != nil {
			p.broker.Publish(web.WSEvent{
				Type:    "inbox_created",
				Message: fmt.Sprintf("Новое входящее: %s", inboxItem.Subject),
				Data:    inboxItem,
			})
		}

		p.logger.Info("inbox item created",
			"source", inboxItem.Source,
			"source_id", inboxItem.SourceID,
			"thread_id", inboxItem.ThreadID,
		)
	}

	// Если AI включён — ставим в очередь и не обрабатываем синхронно
	if p.aiEnabled && p.aiQueue != nil {
		if inboxItem != nil {
			p.aiQueue.Enqueue(inboxItem.ID)
			p.logger.Info("inbox item queued for AI processing",
				"inbox_item_id", inboxItem.ID,
			)
			return &ProcessResult{
				Action:    ActionCreateIssue,
				InboxItem: inboxItem,
				Extracted: email,
			}, nil
		}
	}

	// Ищем существующую задачу по ID в теме
	existingTaskID := p.findExistingTask(ctx, email)

	if existingTaskID > 0 {
		return p.addCommentToTask(ctx, email, existingTaskID)
	}

	return p.createNewTask(ctx, email)
}

// findExistingTask ищет ID существующей задачи по ID в теме письма.
func (p *MessageProcessor) findExistingTask(ctx context.Context, email *extractor.ExtractedEmail) int64 {
	taskID := extractTaskIDFromSubject(email.Subject)
	if taskID > 0 {
		task, err := p.store.GetTask(ctx, taskID)
		if err == nil && task != nil {
			return task.ID
		}
		p.logger.Debug("task not found by ID from subject",
			"task_id", taskID,
		)
	}
	return 0
}

// createNewTask создаёт новую задачу в БД.
func (p *MessageProcessor) createNewTask(ctx context.Context, email *extractor.ExtractedEmail) (*ProcessResult, error) {
	text := email.BodyText
	if text == "" {
		text = email.Subject
	}

	classification, err := p.classifier.Classify(ctx, text, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("classification failed: %w", err)
	}

	p.logger.Info("classification result",
		"project", classification.Project,
		"type", classification.Type,
		"priority", classification.Priority,
		"confidence", classification.Confidence,
	)

	project := classification.Project
	if project == "" {
		project = p.config.Plane.DefaultProject
	}

	task := &store.Task{
		MessageID: email.MessageID,
		Subject:   email.Subject,
		BodyText:  email.BodyText,
		BodyHTML:  email.BodyHTML,
		FromEmail: email.From,
		FromName:  extractName(email.From),
		Project:   project,
		Type:      classification.Type,
		Priority:  classification.Priority,
		Status:    string(store.StatusNew),
	}

	if err := p.store.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	p.logger.Info("task created",
		"task_id", task.ID,
		"project", task.Project,
		"type", task.Type,
	)

	if p.broker != nil {
		p.broker.Publish(web.WSEvent{
			Type:     "task_created",
			TaskID:   task.ID,
			Username: task.Assignee,
			Message:  fmt.Sprintf("Новая задача #%d: %s", task.ID, task.Subject),
			Data:     task,
		})
	}

	for _, att := range email.Attachments {
		taskAtt := &store.TaskAttachment{
			TaskID:      task.ID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			StoragePath: att.StoragePath,
		}
		if err := p.store.AddTaskAttachment(ctx, taskAtt); err != nil {
			p.logger.Error("failed to save task attachment", "error", err)
		}
	}

	return &ProcessResult{
		Action:    ActionCreateIssue,
		TaskID:    task.ID,
		Task:      task,
		Extracted: email,
	}, nil
}

// addCommentToTask добавляет комментарий к существующей задаче.
func (p *MessageProcessor) addCommentToTask(ctx context.Context, email *extractor.ExtractedEmail, taskID int64) (*ProcessResult, error) {
	task, err := p.store.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return nil, fmt.Errorf("task %d not found: %w", taskID, err)
	}

	comment := &store.TaskComment{
		TaskID:    taskID,
		Author:    email.From,
		Body:      email.BodyText,
		Direction: "in",
	}

	if err := p.store.AddTaskComment(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	p.logger.Info("comment added to task",
		"task_id", taskID,
		"message_id", email.MessageID,
	)

	if err := p.store.ResetTaskReads(ctx, taskID); err != nil {
		p.logger.Error("failed to reset task reads", "error", err)
	}

	if p.broker != nil {
		p.broker.Publish(web.WSEvent{
			Type:     "task_updated",
			TaskID:   taskID,
			Username: task.Assignee,
			Message:  fmt.Sprintf("Новый комментарий в задаче #%d от %s", taskID, email.From),
			Data:     map[string]interface{}{"task_id": taskID, "from": email.From},
		})
	}

	for _, att := range email.Attachments {
		taskAtt := &store.TaskAttachment{
			TaskID:      taskID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			StoragePath: att.StoragePath,
		}
		if err := p.store.AddTaskAttachment(ctx, taskAtt); err != nil {
			p.logger.Error("failed to save attachment", "error", err)
		}
	}

	return &ProcessResult{
		Action:    ActionAddComment,
		TaskID:    taskID,
		Task:      task,
		Extracted: email,
	}, nil
}

// extractTaskIDFromSubject извлекает ID задачи из темы письма.
func extractTaskIDFromSubject(subject string) int64 {
	subject = strings.TrimSpace(subject)
	for {
		lower := strings.ToLower(subject)
		if strings.HasPrefix(lower, "re:") {
			subject = strings.TrimSpace(subject[3:])
		} else if strings.HasPrefix(lower, "fwd:") || strings.HasPrefix(lower, "fw:") {
			subject = strings.TrimSpace(subject[4:])
		} else {
			break
		}
	}

	re := regexp.MustCompile(`\[TASK-(\d+)\]`)
	matches := re.FindStringSubmatch(subject)
	if len(matches) == 2 {
		id, _ := strconv.ParseInt(matches[1], 10, 64)
		return id
	}

	return 0
}

// extractName извлекает имя из адреса вида "Имя Фамилия <email>".
func extractName(from string) string {
	if idx := strings.LastIndex(from, "<"); idx != -1 {
		name := strings.TrimSpace(from[:idx])
		name = strings.Trim(name, `"`)
		return name
	}
	return ""
}
