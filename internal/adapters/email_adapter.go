package adapters

import (
	"encoding/json"
	"fmt"

	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
)

// EmailAdapter парсит сырые email-данные в InboxItem.
type EmailAdapter struct {
	extractor *extractor.Extractor
}

// NewEmailAdapter создаёт новый EmailAdapter.
func NewEmailAdapter(ext *extractor.Extractor) *EmailAdapter {
	return &EmailAdapter{extractor: ext}
}

// Source возвращает "email".
func (a *EmailAdapter) Source() string {
	return "email"
}

// Parse преобразует сырое письмо в InboxItem.
func (a *EmailAdapter) Parse(raw []byte) (*store.InboxItem, error) {
	email, err := a.extractor.Extract(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to extract email: %w", err)
	}

	// Определяем thread_id
	threadID := email.MessageID
	if len(email.References) > 0 {
		threadID = email.References[0]
	}

	// Формируем meta JSON
	meta := map[string]interface{}{
		"message_id":  email.MessageID,
		"to":          email.To,
		"cc":          email.Cc,
		"references":  email.References,
		"in_reply_to": email.InReplyTo,
	}
	metaJSON, _ := json.Marshal(meta)

	cleaner := extractor.NewCleaner()
	bodyHTML := cleaner.SanitizeHTML(email.BodyHTML)

	return &store.InboxItem{
		Source:      "email",
		SourceID:    email.MessageID,
		ThreadID:    threadID,
		FromContact: email.From,
		FromName:    extractNameFromEmail(email.From),
		Subject:     email.Subject,
		BodyText:    email.BodyText,
		BodyHTML:    bodyHTML,
		Meta:        string(metaJSON),
		Status:      "unread",
	}, nil
}

// extractNameFromEmail извлекает имя из адреса "Имя <email>".
func extractNameFromEmail(from string) string {
	for i := 0; i < len(from); i++ {
		if from[i] == '<' {
			name := from[:i]
			name = trimQuotes(name)
			return name
		}
	}
	return ""
}

func trimQuotes(s string) string {
	for len(s) > 0 && (s[0] == '"' || s[0] == ' ') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == '"' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}
