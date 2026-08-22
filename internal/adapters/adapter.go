// Package adapters определяет адаптеры источников входящих данных.
package adapters

import (
	"github.com/audetv/mailbridge/internal/store"
)

// Adapter определяет интерфейс для парсинга входящих данных из разных источников.
type Adapter interface {
	// Source возвращает имя источника.
	Source() string

	// Parse преобразует сырые данные в InboxItem.
	Parse(raw []byte) (*store.InboxItem, error)
}
