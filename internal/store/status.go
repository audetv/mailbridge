// Package store содержит доменные типы и константы.
package store

// TaskStatus определяет возможные статусы задачи.
type TaskStatus string

const (
	StatusNew        TaskStatus = "new"
	StatusBacklog    TaskStatus = "backlog"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
	StatusClosed     TaskStatus = "closed"
)

// ValidStatuses возвращает все допустимые статусы.
func ValidStatuses() []TaskStatus {
	return []TaskStatus{
		StatusNew,
		StatusBacklog,
		StatusInProgress,
		StatusCompleted,
		StatusClosed,
	}
}

// IsActive проверяет является ли статус активным (требует внимания).
func (s TaskStatus) IsActive() bool {
	return s == StatusNew || s == StatusInProgress
}

// IsArchived проверяет является ли статус архивным.
func (s TaskStatus) IsArchived() bool {
	return s == StatusCompleted || s == StatusClosed
}
