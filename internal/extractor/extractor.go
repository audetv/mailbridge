// Package extractor извлекает текст и вложения из email-сообщений.
package extractor

import (
	"fmt"
	"mime"
	"strings"
	"time"

	"github.com/jhillyerd/enmime"

	"github.com/audetv/mailbridge/internal/preprocessor"
)

// ExtractedEmail содержит результат извлечения данных из письма.
type ExtractedEmail struct {
	MessageID   string
	ThreadID    string
	From        string
	To          string
	Cc          string
	Subject     string
	BodyText    string
	BodyHTML    string
	Calendar    string
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

	refs := parseReferences(env.GetHeader("References"))
	inReplyTo := cleanHeader(env.GetHeader("In-Reply-To"))

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

		storagePath, err := e.store.Save(&a)
		if err != nil {
			storagePath = fmt.Sprintf("error: %v", err)
		}
		a.StoragePath = storagePath
		attachments = append(attachments, a)
	}

	// Inline-изображения: сохраняем и заменяем cid: ссылки в HTML
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

		// Заменяем cid: ссылки в HTML на URL сохранённого файла
		if att.ContentID != "" && bodyHTML != "" {
			cid := "cid:" + att.ContentID
			fileURL := "/api/attachments/" + storagePath
			bodyHTML = strings.ReplaceAll(bodyHTML, cid, fileURL)
		}
	}

	// Санитайзим HTML после замены cid
	bodyHTML = e.cleaner.SanitizeHTML(bodyHTML)

	return &ExtractedEmail{
		MessageID:   msgID,
		From:        from,
		To:          to,
		Cc:          cc,
		Subject:     subject,
		BodyText:    bodyText,
		BodyHTML:    bodyHTML,
		Calendar:    extractCalendarParts(env),
		References:  refs,
		InReplyTo:   inReplyTo,
		Attachments: attachments,
		ReceivedAt:  time.Now(),
	}, nil
}

// extractCalendarParts извлекает iCalendar-части (text/calendar) из OtherParts
// envelope и превращает их в читаемый текст секций событий.
// Календарные приглашения Exchange (multipart/alternative → text/calendar)
// enmime кладёт в OtherParts, а не в Attachments — без этой функции такие
// письма приходят в AI с пустым телом (issue #1).
func extractCalendarParts(env *enmime.Envelope) string {
	var calendars []string
	for _, p := range env.OtherParts {
		if p == nil || !strings.HasPrefix(p.ContentType, "text/calendar") {
			continue
		}
		if text := preprocessor.ExtractICal(p.Content); text != "" {
			calendars = append(calendars, text)
		}
	}

	return strings.Join(calendars, "")
}

// cleanHeader очищает заголовок от MIME-слов и лишних символов.
func cleanHeader(header string) string {
	dec := mime.WordDecoder{}
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		decoded = header
	}
	decoded = strings.Trim(decoded, "<> ")
	return strings.TrimSpace(decoded)
}

// decodeHeader декодирует MIME-заголовок.
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
