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
	workspace  string // slug рабочего пространства
}

// NewClient создаёт новый клиент Plane API.
func NewClient(baseURL, apiKey string) *Client {
	// Извлекаем workspace из URL: https://plane.example.com/workspace-slug
	workspace := extractWorkspace(baseURL)

	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		workspace: workspace,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// extractWorkspace извлекает slug workspace из URL.
func extractWorkspace(baseURL string) string {
	parts := strings.Split(strings.TrimRight(baseURL, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// CreateIssue создаёт новую задачу в указанном проекте.
func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*Issue, error) {
	url := fmt.Sprintf("%s/api/workspaces/%s/projects/%s/issues/",
		c.baseURL, c.workspace, req.ProjectID)

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
	if err := c.doRequest(ctx, http.MethodPost, url, body, &issue); err != nil {
		return nil, fmt.Errorf("create issue failed: %w", err)
	}

	return &issue, nil
}

// GetIssue возвращает задачу по ID.
func (c *Client) GetIssue(ctx context.Context, issueID string) (*Issue, error) {
	url := fmt.Sprintf("%s/api/workspaces/%s/issues/%s/",
		c.baseURL, c.workspace, issueID)

	var issue Issue
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &issue); err != nil {
		return nil, fmt.Errorf("get issue failed: %w", err)
	}

	return &issue, nil
}

// AddComment добавляет комментарий к задаче.
func (c *Client) AddComment(ctx context.Context, issueID, body string) (*Comment, error) {
	url := fmt.Sprintf("%s/api/workspaces/%s/issues/%s/comments/",
		c.baseURL, c.workspace, issueID)

	reqBody, err := json.Marshal(map[string]string{
		"comment_html": body,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comment: %w", err)
	}

	var comment Comment
	if err := c.doRequest(ctx, http.MethodPost, url, reqBody, &comment); err != nil {
		return nil, fmt.Errorf("add comment failed: %w", err)
	}

	return &comment, nil
}

// GetProjects возвращает список проектов workspace.
func (c *Client) GetProjects(ctx context.Context) ([]Project, error) {
	url := fmt.Sprintf("%s/api/workspaces/%s/projects/",
		c.baseURL, c.workspace)

	var response struct {
		Results []Project `json:"results"`
	}
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &response); err != nil {
		return nil, fmt.Errorf("get projects failed: %w", err)
	}

	return response.Results, nil
}

// GetLabels возвращает метки проекта.
func (c *Client) GetLabels(ctx context.Context, projectID string) ([]Label, error) {
	url := fmt.Sprintf("%s/api/workspaces/%s/projects/%s/labels/",
		c.baseURL, c.workspace, projectID)

	var response struct {
		Results []Label `json:"results"`
	}
	if err := c.doRequest(ctx, http.MethodGet, url, nil, &response); err != nil {
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
			// Exponential backoff: 1s, 2s, 4s
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

		// Успешные коды
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil {
				if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
					return fmt.Errorf("failed to decode response: %w", err)
				}
			}
			return nil
		}

		// Не повторяем для 4xx (кроме 429)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("client error %d: %s", resp.StatusCode, string(respBody))
		}

		lastErr = fmt.Errorf("server error %d", resp.StatusCode)

		// Для retry нужно пересоздать reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
