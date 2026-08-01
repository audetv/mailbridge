// Package processor реализует оркестрацию обработки входящих писем.
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/audetv/mailbridge/internal/classifier"
	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/store"
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
	Extracted *extractor.ExtractedEmail
	Error     error
}

// MessageProcessor оркестрирует обработку входящего письма.
type MessageProcessor struct {
	store      store.Store
	classifier classifier.Classifier
	extractor  *extractor.Extractor
	parser     *parser.FieldParser
	config     *config.Config
	logger     *slog.Logger
	// projectMap: имя проекта → проект (для маппинга имени на UUID, опционально)
	projectMap  map[string]string
	onTaskEvent func(eventType string, taskID int64)
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
	onTaskEvent func(eventType string, taskID int64),
) *MessageProcessor {
	return &MessageProcessor{
		store:       st,
		classifier:  cl,
		extractor:   ext,
		parser:      par,
		config:      cfg,
		logger:      logger,
		projectMap:  projectMap,
		onTaskEvent: onTaskEvent,
	}
}

// Process обрабатывает сырое письмо и возвращает результат.
func (p *MessageProcessor) Process(ctx context.Context, rawEmail []byte) (*ProcessResult, error) {
	email, err := p.extractor.Extract(rawEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to extract email: %w", err)
	}

	p.logger.Debug("processing email",
		"message_id", email.MessageID,
		"from", email.From,
		"subject", email.Subject,
	)

	// Проверяем дубликат по Message-ID через таблицу задач
	existingTask, err := p.store.GetTaskByMessageID(ctx, email.MessageID)
	if err != nil {
		p.logger.Debug("message_id not found, checking email_mapping", "error", err)
	}

	if existingTask != nil {
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

	// Проверяем старую таблицу email_mapping
	exists, err := p.store.MessageExists(ctx, email.MessageID)
	if err == nil && exists {
		p.logger.Info("duplicate email ignored (mapping)", "message_id", email.MessageID)
		return &ProcessResult{Action: ActionIgnore, Extracted: email}, nil
	}

	// Ищем существующую задачу по ID в теме или References
	existingTaskID := p.findExistingTask(ctx, email)

	if existingTaskID > 0 {
		return p.addCommentToTask(ctx, email, existingTaskID)
	}

	return p.createNewTask(ctx, email)
}

// findExistingTask ищет ID существующей задачи.
func (p *MessageProcessor) findExistingTask(ctx context.Context, email *extractor.ExtractedEmail) int64 {
	// Проверяем тему на наличие [TASK-XXX]
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

	// Проверяем References / In-Reply-To через email_mapping
	refs := email.References
	if email.InReplyTo != "" {
		refs = append(refs, email.InReplyTo)
	}
	if len(refs) > 0 {
		mapping, err := p.store.FindMappingByReferences(ctx, refs)
		if err == nil && mapping != nil {
			// Ищем задачу по message_id из mapping
			task, err := p.store.GetTaskByMessageID(ctx, mapping.MessageID)
			if err == nil && task != nil {
				return task.ID
			}
		}
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
		Status:    "new",
	}

	if err := p.store.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	if p.onTaskEvent != nil {
		p.onTaskEvent("task_created", task.ID)
	}

	p.logger.Info("task created",
		"task_id", task.ID,
		"project", task.Project,
		"type", task.Type,
	)

	// Сохраняем вложения
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

	// Сохраняем маппинг для обратной совместимости
	mapping := &store.EmailMapping{
		MessageID:        email.MessageID,
		PlaneIssueID:     fmt.Sprintf("task-%d", task.ID),
		PlaneIssueSeq:    fmt.Sprintf("TASK-%d", task.ID),
		OriginalFrom:     email.From,
		OriginalSubject:  email.Subject,
		ThreadReferences: append(email.References, email.MessageID),
		ActionType:       string(ActionCreateIssue),
	}
	if err := p.store.SaveMapping(ctx, mapping); err != nil {
		p.logger.Error("failed to save mapping", "error", err)
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

	if p.onTaskEvent != nil {
		p.onTaskEvent("task_comment", taskID)
	}

	p.logger.Info("comment added to task",
		"task_id", taskID,
		"message_id", email.MessageID,
	)

	// Сохраняем вложения к существующей задаче
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
// Поддерживает форматы: [TASK-123]
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

// extractName извлекает имя из адреса вида "Имя Фамилия <email>" или просто email.
func extractName(from string) string {
	if idx := strings.LastIndex(from, "<"); idx != -1 {
		name := strings.TrimSpace(from[:idx])
		name = strings.Trim(name, `"`)
		return name
	}
	return ""
}
