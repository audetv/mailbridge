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
	// projectMap: имя проекта → Project (содержит ID и Identifier)
	projectMap map[string]*plane.Project
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
	projectMap map[string]*plane.Project,
) *MessageProcessor {
	return &MessageProcessor{
		store:       st,
		classifier:  cl,
		extractor:   ext,
		parser:      par,
		planeClient: pc,
		config:      cfg,
		logger:      logger,
		projectMap:  projectMap,
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

	exists, err := p.store.MessageExists(ctx, email.MessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to check message existence: %w", err)
	}
	if exists {
		p.logger.Info("duplicate email ignored", "message_id", email.MessageID)
		return &ProcessResult{Action: ActionIgnore, Extracted: email}, nil
	}

	existingIssueID, existingProjectID := p.findExistingIssue(ctx, email)

	if existingIssueID != "" && existingProjectID != "" {
		return p.addCommentToIssue(ctx, email, existingProjectID, existingIssueID)
	}

	return p.createNewIssue(ctx, email)
}

// findExistingIssue ищет ID существующей задачи и проекта.
func (p *MessageProcessor) findExistingIssue(ctx context.Context, email *extractor.ExtractedEmail) (issueID, projectID string) {
	// Проверяем тему на наличие [IDENTIFIER-XXX]
	identifier, seq := extractIssueIDFromSubject(email.Subject)
	if identifier != "" && seq > 0 {
		workItem, err := p.planeClient.GetWorkItemByIdentifier(ctx, identifier, seq)
		if err == nil && workItem != nil {
			return workItem.ID, workItem.ProjectID
		}
		p.logger.Debug("work item not found by identifier",
			"identifier", identifier,
			"seq", seq,
		)
	}

	// Проверяем References / In-Reply-To
	refs := email.References
	if email.InReplyTo != "" {
		refs = append(refs, email.InReplyTo)
	}
	if len(refs) > 0 {
		mapping, err := p.store.FindMappingByReferences(ctx, refs)
		if err == nil && mapping != nil {
			return mapping.PlaneIssueID, mapping.PlaneProjectID
		}
	}

	return "", ""
}

// createNewIssue создаёт новую задачу в Plane.
func (p *MessageProcessor) createNewIssue(ctx context.Context, email *extractor.ExtractedEmail) (*ProcessResult, error) {
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

	// Определяем проект
	targetProject := p.resolveProject(classification.Project)

	// Разрешаем метки
	labelIDs := p.resolveLabels(ctx, targetProject.ID, classification)

	// Создаём задачу
	description := formatIssueDescription(email)

	workItem, err := p.planeClient.CreateWorkItem(ctx, &plane.CreateWorkItemRequest{
		ProjectID:      targetProject.ID,
		Name:           email.Subject,
		Description:    description,
		Priority:       classification.Priority,
		Labels:         labelIDs,
		ExternalID:     email.MessageID,
		ExternalSource: "mailbridge",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create work item: %w", err)
	}

	issueSeq := fmt.Sprintf("%s-%d", targetProject.Identifier, workItem.SequenceID)

	p.logger.Info("work item created",
		"work_item_id", workItem.ID,
		"project_id", targetProject.ID,
		"issue_seq", issueSeq,
	)

	mapping := &store.EmailMapping{
		MessageID:        email.MessageID,
		PlaneIssueID:     workItem.ID,
		PlaneProjectID:   targetProject.ID,
		PlaneIssueSeq:    issueSeq,
		OriginalFrom:     email.From,
		OriginalSubject:  email.Subject,
		ThreadReferences: append(email.References, email.MessageID),
		ActionType:       string(ActionCreateIssue),
	}

	if err := p.store.SaveMapping(ctx, mapping); err != nil {
		p.logger.Error("failed to save mapping", "error", err)
	}

	return &ProcessResult{
		Action:        ActionCreateIssue,
		IssueID:       workItem.ID,
		IssueSequence: issueSeq,
		Mapping:       mapping,
		Extracted:     email,
	}, nil
}

// addCommentToIssue добавляет комментарий к существующей задаче.
func (p *MessageProcessor) addCommentToIssue(ctx context.Context, email *extractor.ExtractedEmail, projectID, issueID string) (*ProcessResult, error) {
	commentText := formatCommentText(email)

	_, err := p.planeClient.AddComment(ctx, projectID, issueID, commentText, email.MessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	p.logger.Info("comment added",
		"issue_id", issueID,
		"message_id", email.MessageID,
	)

	mapping := &store.EmailMapping{
		MessageID:        email.MessageID,
		PlaneIssueID:     issueID,
		PlaneProjectID:   projectID,
		OriginalFrom:     email.From,
		OriginalSubject:  email.Subject,
		ThreadReferences: append(email.References, email.MessageID),
		ActionType:       string(ActionAddComment),
	}

	if err := p.store.SaveMapping(ctx, mapping); err != nil {
		p.logger.Error("failed to save mapping", "error", err)
	}

	issueSeq := ""
	workItem, err := p.planeClient.GetWorkItemByIdentifier(ctx, projectID, 0)
	if err == nil && workItem != nil {
		// Не можем получить seq без identifier, но он есть в mapping ответа
		_ = workItem
	}
	_ = issueSeq

	return &ProcessResult{
		Action:    ActionAddComment,
		IssueID:   issueID,
		Mapping:   mapping,
		Extracted: email,
	}, nil
}

// resolveProject находит проект по имени или возвращает проект по умолчанию.
func (p *MessageProcessor) resolveProject(name string) *plane.Project {
	if name != "" {
		if proj, ok := p.projectMap[name]; ok {
			return proj
		}
	}
	// Проект по умолчанию
	for _, proj := range p.projectMap {
		if proj.Name == p.config.Plane.DefaultProject {
			return proj
		}
	}
	// Возвращаем первый попавшийся
	for _, proj := range p.projectMap {
		return proj
	}
	return &plane.Project{ID: "", Identifier: "INBOX"}
}

// resolveLabels разрешает имена меток в UUID'ы, создавая отсутствующие.
func (p *MessageProcessor) resolveLabels(ctx context.Context, projectID string, classification *classifier.Classification) []string {
	if projectID == "" {
		return nil
	}

	// Загружаем существующие метки
	existingLabels, err := p.planeClient.GetLabels(ctx, projectID)
	if err != nil {
		p.logger.Warn("failed to get labels", "error", err)
	}

	labelMap := make(map[string]string) // имя → UUID
	for _, l := range existingLabels {
		labelMap[strings.ToLower(l.Name)] = l.ID
	}

	var labelIDs []string
	labelsToCreate := []string{"from:email"}

	if classification.Type != "" {
		labelsToCreate = append(labelsToCreate, classification.Type)
	}
	if classification.NeedsTriage {
		labelsToCreate = append(labelsToCreate, "needs_triage")
	}

	for _, name := range labelsToCreate {
		key := strings.ToLower(name)
		if id, ok := labelMap[key]; ok {
			labelIDs = append(labelIDs, id)
			continue
		}

		// Создаём метку
		label, err := p.planeClient.CreateLabel(ctx, projectID, &plane.CreateLabelRequest{
			Name:           name,
			Color:          labelColor(name),
			Description:    name,
			ExternalSource: "mailbridge",
		})
		if err != nil {
			p.logger.Warn("failed to create label", "name", name, "error", err)
			continue
		}
		labelMap[key] = label.ID
		labelIDs = append(labelIDs, label.ID)
	}

	return labelIDs
}

// labelColor возвращает цвет для метки по имени.
func labelColor(name string) string {
	colors := map[string]string{
		"bug":          "#ff0000",
		"feature":      "#00ff00",
		"support":      "#0000ff",
		"access":       "#ffaa00",
		"seo":          "#aa00ff",
		"content":      "#00aaaa",
		"urgent":       "#ff0000",
		"high":         "#ff6600",
		"medium":       "#ffaa00",
		"low":          "#00aa00",
		"needs_triage": "#888888",
		"from:email":   "#666666",
	}
	if c, ok := colors[strings.ToLower(name)]; ok {
		return c
	}
	return "#666666"
}

// extractIssueIDFromSubject извлекает идентификатор проекта и номер задачи из темы.
// Поддерживает форматы: [INBOX-123], [TRK-5], #INBOX-123
func extractIssueIDFromSubject(subject string) (identifier string, seq int) {
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

	// Ищем [XXX-NNN] или #XXX-NNN
	re := regexp.MustCompile(`[\[#]([A-Za-z0-9]+)-(\d+)[\]\s]`)
	matches := re.FindStringSubmatch(subject)
	if len(matches) == 3 {
		identifier = strings.ToUpper(matches[1])
		seq, _ = strconv.Atoi(matches[2])
		return
	}

	return "", 0
}

// formatIssueDescription форматирует описание задачи.
func formatIssueDescription(email *extractor.ExtractedEmail) string {
	author := email.From
	if idx := strings.LastIndex(email.From, "<"); idx != -1 {
		author = strings.TrimSpace(email.From[:idx])
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("<p><strong>От:</strong> %s (%s)</p>", author, email.From))
	parts = append(parts, fmt.Sprintf("<p><strong>Тема:</strong> %s</p>", email.Subject))

	if email.BodyText != "" {
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
	author := email.From
	if idx := strings.LastIndex(email.From, "<"); idx != -1 {
		author = strings.TrimSpace(email.From[:idx])
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("<p><strong>Ответ от:</strong> %s (%s)</p>", author, email.From))

	if email.BodyText != "" {
		escaped := strings.ReplaceAll(email.BodyText, "&", "&amp;")
		escaped = strings.ReplaceAll(escaped, "<", "&lt;")
		escaped = strings.ReplaceAll(escaped, ">", "&gt;")
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		parts = append(parts, fmt.Sprintf("<p>%s</p>", escaped))
	}

	if len(email.Attachments) > 0 {
		parts = append(parts, "<p><em>Вложения:</em></p><ul>")
		for _, att := range email.Attachments {
			parts = append(parts, fmt.Sprintf("<li>%s (%s)</li>", att.Filename, att.StoragePath))
		}
		parts = append(parts, "</ul>")
	}

	return strings.Join(parts, "\n")
}
