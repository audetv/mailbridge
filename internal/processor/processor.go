// Package processor реализует оркестрацию обработки входящих писем.
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/audetv/mailbridge/internal/classifier"
	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/plane"
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
	Action        ActionType
	IssueID       string
	IssueSequence string
	Mapping       *store.EmailMapping
	Extracted     *extractor.ExtractedEmail
	Error         error
}

// MessageProcessor оркестрирует обработку входящего письма.
type MessageProcessor struct {
	store       store.Store
	classifier  classifier.Classifier
	extractor   *extractor.Extractor
	parser      *parser.FieldParser
	planeClient *plane.Client
	config      *config.Config
	logger      *slog.Logger
}

// NewMessageProcessor создаёт новый MessageProcessor.
func NewMessageProcessor(
	st store.Store,
	cl classifier.Classifier,
	ext *extractor.Extractor,
	par *parser.FieldParser,
	pc *plane.Client,
	cfg *config.Config,
	logger *slog.Logger,
) *MessageProcessor {
	return &MessageProcessor{
		store:       st,
		classifier:  cl,
		extractor:   ext,
		parser:      par,
		planeClient: pc,
		config:      cfg,
		logger:      logger,
	}
}

// Process обрабатывает сырое письмо и возвращает результат.
func (p *MessageProcessor) Process(ctx context.Context, rawEmail []byte) (*ProcessResult, error) {
	// Шаг 1: извлекаем данные из письма
	email, err := p.extractor.Extract(rawEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to extract email: %w", err)
	}

	p.logger.Debug("processing email",
		"message_id", email.MessageID,
		"from", email.From,
		"subject", email.Subject,
	)

	// Шаг 2: проверяем дубликат по Message-ID
	exists, err := p.store.MessageExists(ctx, email.MessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to check message existence: %w", err)
	}
	if exists {
		p.logger.Info("duplicate email ignored", "message_id", email.MessageID)
		return &ProcessResult{
			Action:    ActionIgnore,
			Extracted: email,
		}, nil
	}

	// Шаг 3: ищем существующую задачу по ID в теме или References
	existingIssueID := p.findExistingIssue(ctx, email)

	if existingIssueID != "" {
		// Это ответ на существующую задачу — добавляем комментарий
		return p.addCommentToIssue(ctx, email, existingIssueID)
	}

	// Шаг 4: создаём новую задачу
	return p.createNewIssue(ctx, email)
}

// findExistingIssue ищет ID существующей задачи.
func (p *MessageProcessor) findExistingIssue(ctx context.Context, email *extractor.ExtractedEmail) string {
	// Проверяем тему на наличие [PLANE-XXX] или [WEB-XXX]
	issueID := extractIssueIDFromSubject(email.Subject)
	if issueID != "" {
		return issueID
	}

	// Проверяем References / In-Reply-To
	refs := email.References
	if email.InReplyTo != "" {
		refs = append(refs, email.InReplyTo)
	}
	if len(refs) > 0 {
		mapping, err := p.store.FindMappingByReferences(ctx, refs)
		if err == nil && mapping != nil {
			return mapping.PlaneIssueID
		}
	}

	return ""
}

// createNewIssue создаёт новую задачу в Plane.
func (p *MessageProcessor) createNewIssue(ctx context.Context, email *extractor.ExtractedEmail) (*ProcessResult, error) {
	// Классифицируем текст
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
		"needs_triage", classification.NeedsTriage,
	)

	// Если требуется ручной разбор — создаём в проекте по умолчанию
	projectID := p.config.Plane.DefaultProject
	if classification.Project != "" {
		// TODO: маппинг названия проекта на UUID через GetProjects
		projectID = classification.Project
	}

	// Формируем описание
	description := formatIssueDescription(email)

	// Создаём задачу в Plane
	issue, err := p.planeClient.CreateIssue(ctx, &plane.CreateIssueRequest{
		ProjectID:   projectID,
		Name:        email.Subject,
		Description: description,
		Priority:    classification.Priority,
		Labels:      buildLabels(classification, email),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	p.logger.Info("issue created",
		"issue_id", issue.ID,
		"sequence_id", issue.SequenceID,
		"project", projectID,
	)

	// Сохраняем маппинг
	mapping := &store.EmailMapping{
		MessageID:        email.MessageID,
		PlaneIssueID:     issue.ID,
		PlaneIssueSeq:    issue.SequenceID,
		OriginalFrom:     email.From,
		OriginalSubject:  email.Subject,
		ThreadReferences: append(email.References, email.MessageID),
		ActionType:       string(ActionCreateIssue),
	}

	if err := p.store.SaveMapping(ctx, mapping); err != nil {
		p.logger.Error("failed to save mapping", "error", err)
		// Не фатально — задача уже создана
	}

	return &ProcessResult{
		Action:        ActionCreateIssue,
		IssueID:       issue.ID,
		IssueSequence: issue.SequenceID,
		Mapping:       mapping,
		Extracted:     email,
	}, nil
}

// addCommentToIssue добавляет комментарий к существующей задаче.
func (p *MessageProcessor) addCommentToIssue(ctx context.Context, email *extractor.ExtractedEmail, issueID string) (*ProcessResult, error) {
	commentText := formatCommentText(email)

	_, err := p.planeClient.AddComment(ctx, issueID, commentText)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	p.logger.Info("comment added",
		"issue_id", issueID,
		"message_id", email.MessageID,
	)

	// Сохраняем маппинг
	mapping := &store.EmailMapping{
		MessageID:        email.MessageID,
		PlaneIssueID:     issueID,
		OriginalFrom:     email.From,
		OriginalSubject:  email.Subject,
		ThreadReferences: append(email.References, email.MessageID),
		ActionType:       string(ActionAddComment),
	}

	if err := p.store.SaveMapping(ctx, mapping); err != nil {
		p.logger.Error("failed to save mapping", "error", err)
	}

	// Получаем sequence_id задачи
	issue, err := p.planeClient.GetIssue(ctx, issueID)
	if err == nil {
		mapping.PlaneIssueSeq = issue.SequenceID
	}

	return &ProcessResult{
		Action:        ActionAddComment,
		IssueID:       issueID,
		IssueSequence: mapping.PlaneIssueSeq,
		Mapping:       mapping,
		Extracted:     email,
	}, nil
}

// extractIssueIDFromSubject извлекает ID задачи из темы письма.
// Поддерживает форматы: [WEB-123], [PLANE-123], #WEB-123
func extractIssueIDFromSubject(subject string) string {
	// Убираем Re:, Fwd: префиксы
	subject = strings.TrimSpace(subject)
	for {
		lower := strings.ToLower(subject)
		if strings.HasPrefix(lower, "re:") {
			subject = strings.TrimSpace(subject[3:])
		} else if strings.HasPrefix(lower, "fwd:") || strings.HasPrefix(lower, "fw:") {
			subject = strings.TrimSpace(subject[4:])
			if strings.HasPrefix(strings.ToLower(subject), "d:") {
				subject = strings.TrimSpace(subject[2:])
			}
		} else {
			break
		}
	}

	// Ищем [WEB-XXX] или [PLANE-XXX]
	patterns := []string{"[web-", "[plane-", "#web-", "#plane-"}
	lower := strings.ToLower(subject)

	for _, pattern := range patterns {
		idx := strings.Index(lower, pattern)
		if idx == -1 {
			continue
		}

		start := idx + len(pattern)
		end := start
		for end < len(subject) {
			c := subject[end]
			if c >= '0' && c <= '9' {
				end++
			} else if c == ']' && pattern[0] == '[' {
				break
			} else {
				break
			}
		}

		if end > start {
			return subject[idx : end+1] // возвращаем вместе с [] или #
		}
	}

	return ""
}

// formatIssueDescription форматирует описание задачи для Plane.
func formatIssueDescription(email *extractor.ExtractedEmail) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("<p><strong>От:</strong> %s</p>", email.From))
	parts = append(parts, fmt.Sprintf("<p><strong>Тема:</strong> %s</p>", email.Subject))

	if email.BodyText != "" {
		// Экранируем HTML
		escaped := strings.ReplaceAll(email.BodyText, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		parts = append(parts, fmt.Sprintf("<p>%s</p>", escaped))
	}

	if len(email.Attachments) > 0 {
		parts = append(parts, "<p><strong>Вложения:</strong></p><ul>")
		for _, att := range email.Attachments {
			parts = append(parts, fmt.Sprintf("<li>%s (%s, %d байт)</li>",
				att.Filename, att.ContentType, att.Size))
		}
		parts = append(parts, "</ul>")
	}

	return strings.Join(parts, "\n")
}

// formatCommentText форматирует текст комментария.
func formatCommentText(email *extractor.ExtractedEmail) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("<p><strong>Ответ от:</strong> %s</p>", email.From))

	if email.BodyText != "" {
		escaped := strings.ReplaceAll(email.BodyText, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		parts = append(parts, fmt.Sprintf("<p>%s</p>", escaped))
	}

	// Информация о вложениях
	if len(email.Attachments) > 0 {
		parts = append(parts, "<p><em>Вложения:</em></p><ul>")
		for _, att := range email.Attachments {
			parts = append(parts, fmt.Sprintf("<li>%s (%s)</li>", att.Filename, att.StoragePath))
		}
		parts = append(parts, "</ul>")
	}

	return strings.Join(parts, "\n")
}

// buildLabels формирует список меток для задачи.
func buildLabels(classification *classifier.Classification, email *extractor.ExtractedEmail) []string {
	var labels []string

	if classification.Type != "" {
		labels = append(labels, "type:"+classification.Type)
	}
	if classification.Priority != "" {
		labels = append(labels, "priority:"+classification.Priority)
	}
	if classification.NeedsTriage {
		labels = append(labels, "needs_triage")
	}

	// Метка источника
	labels = append(labels, "from:email")

	// Если есть вложения — добавляем метку
	if len(email.Attachments) > 0 {
		labels = append(labels, "has_attachments")
	}

	return labels
}
