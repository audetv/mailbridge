// Package adapters определяет адаптеры источников входящих данных.
package adapters

import (
	"github.com/audetv/mailbridge/internal/store"
)

// ParseResult содержит результат парсинга входящего.
type ParseResult struct {
	InboxItem   *store.InboxItem
	Attachments []*store.Attachment
}

// Adapter определяет интерфейс для парсинга входящих данных из разных источников.
type Adapter interface {
	// Source возвращает имя источника.
	Source() string

	// Parse преобразует сырые данные в InboxItem с вложениями.
	Parse(raw []byte) (*ParseResult, error)
}
