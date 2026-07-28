// Package plane предоставляет клиент для взаимодействия с Plane REST API.
package plane

import "time"

// Issue представляет задачу в Plane.
type Issue struct {
	ID          string    `json:"id"`
	SequenceID  string    `json:"sequence_id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description_html"`
	State       string    `json:"state"`
	Priority    string    `json:"priority"`
	Labels      []string  `json:"labels"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateIssueRequest содержит данные для создания задачи.
type CreateIssueRequest struct {
	ProjectID   string   `json:"project_id"`
	Name        string   `json:"name"`
	Description string   `json:"description_html"`
	Priority    string   `json:"priority"`
	Labels      []string `json:"labels"`
}

// Comment представляет комментарий к задаче.
type Comment struct {
	ID        string    `json:"id"`
	IssueID   string    `json:"issue"`
	Body      string    `json:"comment_html"`
	ActorName string    `json:"actor_detail"`
	CreatedAt time.Time `json:"created_at"`
}

// Project представляет проект в Plane.
type Project struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}

// Label представляет метку в Plane.
type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}
