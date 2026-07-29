// Package plane предоставляет клиент для взаимодействия с Plane REST API.
package plane

import "time"

// Project представляет проект в Plane.
type Project struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Emoji      string `json:"emoji"`
}

// Label представляет метку проекта.
type Label struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	Project     string `json:"project"`
}

// WorkItem представляет задачу (work-item) в Plane.
type WorkItem struct {
	ID          string    `json:"id"`
	SequenceID  int       `json:"sequence_id"`
	ProjectID   string    `json:"project"`
	Name        string    `json:"name"`
	Description string    `json:"description_html"`
	State       string    `json:"state"`
	Priority    string    `json:"priority"`
	Labels      []string  `json:"labels"`
	CreatedAt   time.Time `json:"created_at"`
	ExternalID  string    `json:"external_id"`
}

// CreateWorkItemRequest содержит данные для создания задачи.
type CreateWorkItemRequest struct {
	ProjectID      string   `json:"-"`
	Name           string   `json:"name"`
	Description    string   `json:"description_html"`
	Priority       string   `json:"priority"`
	Labels         []string `json:"labels"`
	ExternalID     string   `json:"external_id,omitempty"`
	ExternalSource string   `json:"external_source,omitempty"`
}

// Comment представляет комментарий к задаче.
type Comment struct {
	ID         string    `json:"id"`
	Body       string    `json:"comment_html"`
	ExternalID string    `json:"external_id"`
	Actor      *Actor    `json:"actor,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Actor struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// CreateLabelRequest содержит данные для создания метки.
type CreateLabelRequest struct {
	Name           string `json:"name"`
	Color          string `json:"color"`
	Description    string `json:"description"`
	ExternalID     string `json:"external_id,omitempty"`
	ExternalSource string `json:"external_source,omitempty"`
}

// LabelConflictError возвращается при попытке создать существующую метку.
type LabelConflictError struct {
	Error string `json:"error"`
	ID    string `json:"id"`
}
