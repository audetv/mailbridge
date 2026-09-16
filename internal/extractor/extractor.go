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
//
// From — legacy-совместимый человек-читаемый текст «имя <email>»
// (с закрывающей скобкой; шаг 6-F) для inbox_items.from_contact.
// Разобранные части — FromName / FromEmail (источник для persons).
type ExtractedEmail struct {
	MessageID   string
	ThreadID    string
	From        string
	FromName    string
	FromEmail   string
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
	fromRaw := env.GetHeader("From")
	fromEmail, fromName := ParseFromHeader(fromRaw)
	// legacy From — «имя <email>» с закрывающей скобкой (шаг 6-F), либо
	// чистый адрес, если имени нет (в обоих случаях скобки на месте).
	from := fromEmail
	if fromName != "" {
		from = fromName + " <" + fromEmail + ">"
	}
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
		FromName:    fromName,
		FromEmail:   fromEmail,
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

// validDomainChar: допустимый символ домена RFC 5322 (атомные имена).
func validDomainChar(c rune) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return true
	case c >= '0' && c <= '9':
		return true
	}
	return c == '.' || c == '_' || c == '-'
}

// ParseFromHeader разбирает RFC 5322 From-заголовок («Имя <a@b>», «Имя <a@
// b>», голый email, «Имя <bad> , a@b»). Экспортируется (6-F п.4/п.5):
// backfill персон и inbox→task-путь получают имя+email из legacy-колонки.
// Шаг 6-F: чистый адрес — только из <...> (или из голого токена), если по
// RFC 5322 (local-atom@domain, один символ до «@», локаль/домен 1-63;
// «user..x@bad» — НЕ адрес: пустой атом внутри); имя — display-name;
// «имя <bad> , a@b» — имя от «a@b», адрес «a@b» (комма вне угловых скобок).
func ParseFromHeader(raw string) (email string, name string) {
	s := decodeFrom(raw)
	if s == "" {
		return "", ""
	}
	// Кавычки внутри display-name снимаем для хранения имени.
	stripQuotes := func(v string) string {
		v = strings.TrimSpace(v)
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			v = v[1 : len(v)-1]
		}
		return strings.TrimSpace(v)
	}
	valid := func(addr string) bool {
		if len(addr) < 3 || len(addr) > 254 {
			return false
		}
		at := strings.LastIndexByte(addr, '@')
		if at <= 0 || at == len(addr)-1 {
			return false
		}
		local := addr[:at]
		domain := addr[at+1:]
		if len(local) > 64 || len(domain) < 4 {
			return false
		}
		if local[0] == '.' || local[len(local)-1] == '.' {
			return false
		}
		if strings.Contains(local, "..") {
			return false
		}
		if domain[0] == '.' || domain[len(domain)-1] == '.' || domain[0] == '-' {
			return false
		}
		if strings.Contains(domain, "..") || strings.Contains(domain, "-.") ||
			strings.Contains(domain, ".-") {
			return false
		}
		ok := domain[0] == '_' || (domain[0] >= 'a' && domain[0] <= 'z') ||
			(domain[0] >= 'A' && domain[0] <= 'Z')
		if ok {
			for _, c := range domain {
				if validDomainChar(c) {
					continue
				}
				ok = false
				break
			}
		}
		return ok
	}

	open := strings.IndexByte(s, '<')
	closeIdx := strings.LastIndexByte(s, '>')
	if open >= 0 {
		name = stripQuotes(s[:open])
		var inner, rest string
		if closeIdx > open {
			inner = s[open+1 : closeIdx]
			rest = s[closeIdx+1:]
		} else {
			// Legacy shape без закрывающей скобки «Имя <a@b» —
			// адрес = всё после «<».
			inner = s[open+1:]
			rest = ""
		}
		for _, part := range strings.Split(inner, ",") {
			p := strings.TrimSpace(part)
			if p != "" && valid(p) {
				return p, name
			}
		}
		for _, part := range strings.Split(rest, ",") {
			p := strings.TrimSpace(part)
			if p != "" && valid(p) {
				return p, name
			}
		}
	} else if strings.IndexByte(s, '@') > 0 {
		p := strings.TrimSpace(s)
		if valid(p) {
			return p, ""
		}
	}
	return "", name
}

// cleanHeader — утилита для очистки заголовков от лишнего.
func cleanHeader(header string) string {
	dec := mime.WordDecoder{}
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		decoded = header
	}
	decoded = strings.Trim(decoded, "<> ")
	return strings.TrimSpace(decoded)
}

// decodeFrom декодирует RFC 2047 words (=?charset?B?...?=) в From-заголовке,
// сохраняя угловые скобки: имя из «Имя <а@b>» остаётся на месте (у
// WordDecoder.DecodeHeader срезает их — имя теряется вместе со скобкой при
// парсинге legacy-формы).
func decodeFrom(header string) string {
	dec := mime.WordDecoder{}
	var b strings.Builder
	for i := 0; i < len(header); {
		rest := header[i:]
		if strings.HasPrefix(rest, "=?") {
			if j := strings.Index(rest, "?="); j > 4 {
				word := rest[:j+2]
				if decoded, err := dec.Decode(word); err == nil {
					b.WriteString(decoded)
					i += j + 2
					// разделитель « ?» между RFC2047-словами
					if i < len(header) && header[i] == ' ' {
						i++
						if i < len(header) && header[i] == '?' {
							i++
						}
					}
					continue
				}
			}
		}
		b.WriteByte(header[i])
		i++
	}
	res := b.String()
	// Нормализуем двойные пробелы, но НЕ трогаем «И»/«I».
	return strings.TrimSpace(strings.Join(strings.Fields(res), " "))
}

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
