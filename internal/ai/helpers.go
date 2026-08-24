package ai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/audetv/mailbridge/internal/extractor"
)

// determineThreadID определяет ID цепочки по References или Message-ID.
func determineThreadID(email *extractor.ExtractedEmail) string {
	if email.ThreadID != "" {
		return email.ThreadID
	}
	if len(email.References) > 0 {
		return email.References[0]
	}
	return email.MessageID
}

// extractName извлекает имя из адреса вида "Имя Фамилия <email>".
func extractName(from string) string {
	idx := strings.Index(from, "<")
	if idx == -1 {
		return ""
	}
	name := strings.TrimSpace(from[:idx])
	name = strings.Trim(name, `"`)
	return name
}

// verdictToJSON сериализует вердикт для аудита.
func verdictToJSON(v Verdict) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// verdictsToJSON сериализует вердикты для промпта summary.
func verdictsToJSON(response *LLMResponse) string {
	if response == nil {
		return "[]"
	}
	data, _ := json.Marshal(response.Verdicts)
	return string(data)
}

// getThreadSummary возвращает резюме цепочки.
func (o *Orchestrator) getThreadSummary(ctx context.Context, threadID string) string {
	thread, err := o.store.GetThread(ctx, threadID)
	if err != nil || thread == nil {
		return "Нет резюме"
	}
	return thread.Summary
}

// isImageAttachment проверяет является ли вложение изображением.
func isImageAttachment(contentType string) bool {
	imageTypes := []string{"image/png", "image/jpeg", "image/jpg", "image/gif", "image/webp"}
	for _, t := range imageTypes {
		if contentType == t {
			return true
		}
	}
	return false
}
