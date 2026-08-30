package sender

import "fmt"

// AcknowledgementData содержит данные для письма-подтверждения.
type AcknowledgementData struct {
	To                 string
	Subject            string
	InReplyToMessageID string
	IssueSequence      string
	ProjectName        string
	TypeName           string
	Priority           string
}

// CommentReplyData содержит данные для письма с комментарием.
type CommentReplyData struct {
	To                 string
	Subject            string
	InReplyToMessageID string
	References         []string
	IssueSequence      string
	CommentText        string
	CommentAuthor      string
}

// StatusChangeData содержит данные для уведомления о смене статуса.
type StatusChangeData struct {
	To                 string
	Subject            string
	InReplyToMessageID string
	IssueSequence      string
	OldStatus          string
	NewStatus          string
}

// FormatAcknowledgement формирует тело письма-подтверждения.
func FormatAcknowledgement(data *AcknowledgementData) (string, string) {
	subject := fmt.Sprintf("Re: [%s] %s", data.IssueSequence, data.Subject)
	body := fmt.Sprintf(`[MAILBRIDGE-INTERNAL]

Ваше обращение зарегистрировано.

Создана задача: %s
Проект: %s
Тип: %s
Приоритет: %s

Ответьте на это письмо, чтобы добавить комментарий к задаче.
`, data.IssueSequence, data.ProjectName, data.TypeName, data.Priority)

	return subject, body
}

// FormatCommentReply формирует письмо с комментарием к задаче.
func FormatCommentReply(data *CommentReplyData) (string, string) {
	subject := fmt.Sprintf("Re: [%s] %s", data.IssueSequence, data.Subject)
	body := fmt.Sprintf(`[MAILBRIDGE-INTERNAL]

%s добавил(а) комментарий к задаче %s:
---
%s
---

Ответьте на это письмо, чтобы добавить комментарий к задаче.
`, data.CommentAuthor, data.IssueSequence, data.CommentText)

	return subject, body
}

// FormatStatusChange формирует уведомление о смене статуса.
func FormatStatusChange(data *StatusChangeData) (string, string) {
	subject := fmt.Sprintf("Re: [%s] %s", data.IssueSequence, data.Subject)
	body := fmt.Sprintf(`[MAILBRIDGE-INTERNAL]

Статус задачи %s изменён: %s → %s
`, data.IssueSequence, data.OldStatus, data.NewStatus)

	return subject, body
}
