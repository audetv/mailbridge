// Package plane предоставляет клиент для взаимодействия с Plane REST API.
package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client предоставляет методы для работы с Plane API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	workspace  string
}

// NewClient создаёт новый клиент Plane API.
// baseURL может быть:
//   - "http://localhost/gc" — workspace извлекается из URL
//   - "http://localhost" — workspace будет пустым (для совместимости с тестами)
func NewClient(baseURL, apiKey string) *Client {
	workspace, cleanURL := extractWorkspace(baseURL)

	return &Client{
		baseURL:   strings.TrimRight(cleanURL, "/"),
		apiKey:    apiKey,
		workspace: workspace,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// extractWorkspace извлекает slug workspace из URL и возвращает чистый baseURL.
// "http://localhost/gc" → workspace="gc", cleanURL="http://localhost"
// "http://localhost"      → workspace="",    cleanURL="http://localhost"
func extractWorkspace(rawURL string) (workspace string, cleanURL string) {
	rawURL = strings.TrimRight(rawURL, "/")

	// Ищем последний сегмент пути — потенциальный workspace
	// Но только если он не содержит "." (не домен) и не ":" (не порт)
	parts := strings.Split(rawURL, "/")
	if len(parts) >= 4 { // http://host/segment → минимум 4 части
		last := parts[len(parts)-1]
		// Не считаем workspace если это localhost:port
		if !strings.Contains(last, ":") && last != "" {
			workspace = last
			cleanURL = strings.Join(parts[:len(parts)-1], "/")
			return workspace, cleanURL
		}
	}

	return "", rawURL
}

// url строит полный URL к API.
func (c *Client) url(path string) string {
	if c.workspace != "" {
		return fmt.Sprintf("%s/api/v1/workspaces/%s/%s", c.baseURL, c.workspace, path)
	}
	// Для тестов без workspace — старый формат
	return fmt.Sprintf("%s/api/v1/%s", c.baseURL, path)
}

// CreateIssue создаёт новую задачу в указанном проекте.
func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*Issue, error) {
	u := c.url(fmt.Sprintf("projects/%s/issues/", req.ProjectID))

	body, err := json.Marshal(map[string]interface{}{
		"name":             req.Name,
		"description_html": req.Description,
		"priority":         req.Priority,
		"labels":           req.Labels,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var issue Issue
	if err := c.doRequest(ctx, http.MethodPost, u, body, &issue); err != nil {
		return nil, fmt.Errorf("create issue failed: %w", err)
	}

	return &issue, nil
}

// GetIssue возвращает задачу по ID.
func (c *Client) GetIssue(ctx context.Context, issueID string) (*Issue, error) {
	u := c.url(fmt.Sprintf("issues/%s/", issueID))

	var issue Issue
	if err := c.doRequest(ctx, http.MethodGet, u, nil, &issue); err != nil {
		return nil, fmt.Errorf("get issue failed: %w", err)
	}

	return &issue, nil
}

// AddComment добавляет комментарий к задаче.
func (c *Client) AddComment(ctx context.Context, issueID, body string) (*Comment, error) {
	u := c.url(fmt.Sprintf("issues/%s/comments/", issueID))

	reqBody, err := json.Marshal(map[string]string{
		"comment_html": body,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comment: %w", err)
	}

	var comment Comment
	if err := c.doRequest(ctx, http.MethodPost, u, reqBody, &comment); err != nil {
		return nil, fmt.Errorf("add comment failed: %w", err)
	}

	return &comment, nil
}

// GetProjects возвращает список проектов workspace.
func (c *Client) GetProjects(ctx context.Context) ([]Project, error) {
	u := c.url("projects/")

	var response struct {
		Results []Project `json:"results"`
	}
	if err := c.doRequest(ctx, http.MethodGet, u, nil, &response); err != nil {
		return nil, fmt.Errorf("get projects failed: %w", err)
	}

	return response.Results, nil
}

// GetLabels возвращает метки проекта.
func (c *Client) GetLabels(ctx context.Context, projectID string) ([]Label, error) {
	u := c.url(fmt.Sprintf("projects/%s/labels/", projectID))

	var response struct {
		Results []Label `json:"results"`
	}
	if err := c.doRequest(ctx, http.MethodGet, u, nil, &response); err != nil {
		return nil, fmt.Errorf("get labels failed: %w", err)
	}

	return response.Results, nil
}

// doRequest выполняет HTTP-запрос с retry и обработкой ошибок.
func (c *Client) doRequest(ctx context.Context, method, url string, body []byte, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil {
				if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
					return fmt.Errorf("failed to decode response: %w", err)
				}
			}
			return nil
		}

		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("client error %d: %s", resp.StatusCode, string(respBody))
		}

		lastErr = fmt.Errorf("server error %d", resp.StatusCode)

		if body != nil {
			bodyReader = bytes.NewReader(body)
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
