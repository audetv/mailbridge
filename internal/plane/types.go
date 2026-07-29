// Package plane предоставляет клиент для взаимодействия с Plane REST API.
package plane

import (
	"encoding/json"
	"time"
)

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
// ActorRaw может быть строкой (UUID пользователя) или объектом Actor.
type Comment struct {
	ID         string          `json:"id"`
	Body       string          `json:"comment_html"`
	ExternalID string          `json:"external_id"`
	ActorRaw   json.RawMessage `json:"actor"`
	CreatedAt  time.Time       `json:"created_at"`
}

// Actor представляет пользователя, создавшего комментарий.
type Actor struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

// ActorName возвращает имя пользователя из ActorRaw.
// Если actor — строка, возвращает её. Если объект — DisplayName.
func (c *Comment) ActorName() string {
	if len(c.ActorRaw) == 0 {
		return "Unknown"
	}

	// Пробуем распарсить как строку
	var actorID string
	if err := json.Unmarshal(c.ActorRaw, &actorID); err == nil {
		return actorID
	}

	// Пробуем распарсить как объект
	var actor Actor
	if err := json.Unmarshal(c.ActorRaw, &actor); err == nil {
		if actor.DisplayName != "" {
			return actor.DisplayName
		}
		if actor.FirstName != "" {
			return actor.FirstName + " " + actor.LastName
		}
		return actor.ID
	}

	return "Unknown"
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
