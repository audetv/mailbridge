// Package extractor извлекает текст и вложения из email-сообщений.
package extractor

import (
	"fmt"
	"mime"
	"strings"
	"time"

	"github.com/jhillyerd/enmime"
)

// ExtractedEmail содержит результат извлечения данных из письма.
type ExtractedEmail struct {
	MessageID   string
	From        string
	To          string
	Cc          string
	Subject     string
	BodyText    string
	BodyHTML    string
	References  []string
	InReplyTo   string
	Attachments []Attachment
	ReceivedAt  time.Time
}

// Extractor извлекает данные из сырого email-сообщения.
type Extractor struct {
	cleaner *Cleaner
	store   *AttachmentStore
}

// NewExtractor создаёт новый Extractor.
func NewExtractor(store *AttachmentStore) *Extractor {
	return &Extractor{
		cleaner: NewCleaner(),
		store:   store,
	}
}

// Extract извлекает данные из сырого email.
func (e *Extractor) Extract(raw []byte) (*ExtractedEmail, error) {
	env, err := enmime.ReadEnvelope(strings.NewReader(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}

	msgID := cleanHeader(env.GetHeader("Message-ID"))
	from := cleanHeader(env.GetHeader("From"))
	to := cleanHeader(env.GetHeader("To"))
	cc := cleanHeader(env.GetHeader("Cc"))
	subject := decodeHeader(env.GetHeader("Subject"))

	// Извлекаем References и In-Reply-To
	refs := parseReferences(env.GetHeader("References"))
	inReplyTo := cleanHeader(env.GetHeader("In-Reply-To"))

	// Текст письма
	bodyText := e.cleaner.CleanBody(env.Text)
	bodyHTML := env.HTML

	// Вложения
	var attachments []Attachment
	for _, att := range env.Attachments {
		a := Attachment{
			Filename:    att.FileName,
			ContentType: att.ContentType,
			Data:        att.Content,
			Size:        int64(len(att.Content)),
		}

		// Сохраняем вложение
		storagePath, err := e.store.Save(&a)
		if err != nil {
			// Логируем ошибку, но не прерываем обработку
			storagePath = fmt.Sprintf("error: %v", err)
		}
		a.StoragePath = storagePath
		attachments = append(attachments, a)
	}

	// Извлекаем вложения из встроенных изображений (inline)
	for _, att := range env.Inlines {
		a := Attachment{
			Filename:    att.FileName,
			ContentType: att.ContentType,
			Data:        att.Content,
			Size:        int64(len(att.Content)),
		}

		storagePath, err := e.store.Save(&a)
		if err != nil {
			storagePath = fmt.Sprintf("error: %v", err)
		}
		a.StoragePath = storagePath
		attachments = append(attachments, a)
	}

	return &ExtractedEmail{
		MessageID:   msgID,
		From:        from,
		To:          to,
		Cc:          cc,
		Subject:     subject,
		BodyText:    bodyText,
		BodyHTML:    bodyHTML,
		References:  refs,
		InReplyTo:   inReplyTo,
		Attachments: attachments,
		ReceivedAt:  time.Now(),
	}, nil
}

// cleanHeader очищает заголовок от MIME-слов и лишних символов.
func cleanHeader(header string) string {
	dec := mime.WordDecoder{}
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		decoded = header
	}
	// Убираем угловые скобки у Message-ID и References
	decoded = strings.Trim(decoded, "<> ")
	return strings.TrimSpace(decoded)
}

// decodeHeader декодирует MIME-заголовок, убирая кодировку.
func decodeHeader(header string) string {
	dec := mime.WordDecoder{}
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		return strings.TrimSpace(header)
	}
	return strings.TrimSpace(decoded)
}

// parseReferences разбирает заголовок References на массив Message-ID.
func parseReferences(refs string) []string {
	if refs == "" {
		return nil
	}

	var result []string
	for _, ref := range strings.Fields(refs) {
		ref = strings.Trim(ref, "<>")
		if ref != "" {
			result = append(result, ref)
		}
	}
	return result
}
