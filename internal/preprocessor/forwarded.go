package preprocessor

import "strings"

// MarkForwarded добавляет маркер перед пересланным сообщением.
// Не вырезает текст, а помечает его для LLM.
func (p *Preprocessor) MarkForwarded(body string) string {
	forwardMarkers := []string{
		"---------- Forwarded message ---------",
		"--- Пересланное сообщение ---",
		"--- Пересылаемое сообщение ---",
	}

	for _, marker := range forwardMarkers {
		idx := strings.Index(body, marker)
		if idx != -1 {
			body = body[:idx] +
				"\n\n[ВНИМАНИЕ: Это пересланное сообщение. Контекст оригинальной переписки находится ниже. Проанализируй его на наличие задач.]\n\n" +
				body[idx:]
			break
		}
	}

	return body
}
