package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
)

// EmailAdapter парсит сырые email-данные в InboxItem.
type EmailAdapter struct {
	extractor *extractor.Extractor
	store     store.Store
	attachDir string
}

// NewEmailAdapter создаёт новый EmailAdapter.
func NewEmailAdapter(ext *extractor.Extractor, st store.Store, attachDir string) *EmailAdapter {
	return &EmailAdapter{
		extractor: ext,
		store:     st,
		attachDir: attachDir,
	}
}

// Source возвращает "email".
func (a *EmailAdapter) Source() string {
	return "email"
}

// Parse преобразует сырое письмо в InboxItem и сохраняет вложения.
func (a *EmailAdapter) Parse(raw []byte) (*ParseResult, error) {
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

	item := &store.InboxItem{
		Source:      "email",
		SourceID:    email.MessageID,
		ThreadID:    threadID,
		FromContact: email.From,
		FromName:    extractNameFromEmail(email.From),
		Subject:     email.Subject,
		BodyText:    email.BodyText,
		BodyHTML:    email.BodyHTML,
		Meta:        string(metaJSON),
		Status:      "unread",
	}

	// Сохраняем вложения через hash-дедупликацию
	var attachments []*store.Attachment
	for _, att := range email.Attachments {
		// Hash уже закодирован в пути: {hash[0:2]}/{hash[2:4]}/{hash}
		hash := filepath.Base(att.StoragePath)

		// Проверяем существует ли уже
		existing, err := a.store.GetAttachmentByHash(context.Background(), hash)
		if err == nil && existing != nil {
			attachments = append(attachments, existing)
			continue
		}

		// Создаём новую запись
		newAtt := &store.Attachment{
			Hash:        hash,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			StoragePath: att.StoragePath,
		}
		if err := a.store.CreateAttachment(context.Background(), newAtt); err != nil {
			continue
		}
		attachments = append(attachments, newAtt)
	}

	return &ParseResult{
		InboxItem:   item,
		Attachments: attachments,
	}, nil
}

// extractNameFromEmail извлекает имя из адреса "Имя <email>".
func extractNameFromEmail(from string) string {
	for i := 0; i < len(from); i++ {
		if from[i] == '<' {
			return trimQuotes(from[:i])
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
