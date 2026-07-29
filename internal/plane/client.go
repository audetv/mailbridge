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

// Client предоставляет методы для работы с Plane API v1.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	workspace  string
}

// NewClient создаёт новый клиент Plane API.
// baseURL может быть "http://localhost/gc" (workspace извлекается из URL)
// или "http://localhost" (workspace будет пустым).
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
func extractWorkspace(rawURL string) (workspace string, cleanURL string) {
	rawURL = strings.TrimRight(rawURL, "/")
	parts := strings.Split(rawURL, "/")
	if len(parts) >= 4 {
		last := parts[len(parts)-1]
		if !strings.Contains(last, ":") && last != "" {
			return last, strings.Join(parts[:len(parts)-1], "/")
		}
	}
	return "", rawURL
}

// apiURL строит полный URL к API.
func (c *Client) apiURL(path string) string {
	if c.workspace != "" {
		return fmt.Sprintf("%s/api/v1/workspaces/%s/%s", c.baseURL, c.workspace, path)
	}
	return fmt.Sprintf("%s/api/v1/%s", c.baseURL, path)
}

// GetProjects возвращает список проектов workspace.
func (c *Client) GetProjects(ctx context.Context) ([]Project, error) {
	var response struct {
		Results []Project `json:"results"`
	}
	if err := c.doRequest(ctx, http.MethodGet, c.apiURL("projects/"), nil, &response); err != nil {
		return nil, fmt.Errorf("get projects failed: %w", err)
	}
	return response.Results, nil
}

// GetLabels возвращает список меток проекта.
func (c *Client) GetLabels(ctx context.Context, projectID string) ([]Label, error) {
	var response struct {
		Results []Label `json:"results"`
	}
	path := fmt.Sprintf("projects/%s/labels/", projectID)
	if err := c.doRequest(ctx, http.MethodGet, c.apiURL(path), nil, &response); err != nil {
		return nil, fmt.Errorf("get labels failed: %w", err)
	}
	return response.Results, nil
}

// CreateLabel создаёт новую метку в проекте.
// Если метка с таким именем уже существует, возвращает её ID (ошибка 409 обрабатывается).
func (c *Client) CreateLabel(ctx context.Context, projectID string, req *CreateLabelRequest) (*Label, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal label request: %w", err)
	}

	path := fmt.Sprintf("projects/%s/labels/", projectID)

	resp, err := c.doRawRequest(ctx, http.MethodPost, c.apiURL(path), body)
	if err != nil {
		return nil, fmt.Errorf("create label failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 409 Conflict — метка уже существует, парсим id из ответа
	if resp.StatusCode == http.StatusConflict {
		var conflict LabelConflictError
		if err := json.Unmarshal(respBody, &conflict); err != nil {
			return nil, fmt.Errorf("failed to parse conflict response: %w", err)
		}
		return &Label{ID: conflict.ID, Name: req.Name, Color: req.Color}, nil
	}

	if resp.StatusCode == http.StatusCreated {
		var label Label
		if err := json.Unmarshal(respBody, &label); err != nil {
			return nil, fmt.Errorf("failed to decode label: %w", err)
		}
		return &label, nil
	}

	return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
}

// CreateWorkItem создаёт новую задачу в проекте.
func (c *Client) CreateWorkItem(ctx context.Context, req *CreateWorkItemRequest) (*WorkItem, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal work item request: %w", err)
	}

	path := fmt.Sprintf("projects/%s/work-items/", req.ProjectID)
	var workItem WorkItem
	if err := c.doRequest(ctx, http.MethodPost, c.apiURL(path), body, &workItem); err != nil {
		return nil, fmt.Errorf("create work item failed: %w", err)
	}
	return &workItem, nil
}

// GetWorkItemByIdentifier получает задачу по идентификатору проекта и номеру.
// Например: GetWorkItemByIdentifier("INBOX", 1) → GET .../work-items/INBOX-1/
func (c *Client) GetWorkItemByIdentifier(ctx context.Context, projectIdentifier string, sequenceID int) (*WorkItem, error) {
	path := fmt.Sprintf("work-items/%s-%d/", projectIdentifier, sequenceID)
	var workItem WorkItem
	if err := c.doRequest(ctx, http.MethodGet, c.apiURL(path), nil, &workItem); err != nil {
		return nil, fmt.Errorf("get work item by identifier failed: %w", err)
	}
	return &workItem, nil
}

// AddComment добавляет комментарий к задаче.
func (c *Client) AddComment(ctx context.Context, projectID, workItemID, body, externalID string) (*Comment, error) {
	reqBody, err := json.Marshal(map[string]string{
		"comment_html":    body,
		"external_id":     externalID,
		"external_source": "mailbridge",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comment: %w", err)
	}

	path := fmt.Sprintf("projects/%s/work-items/%s/comments/", projectID, workItemID)
	var comment Comment
	if err := c.doRequest(ctx, http.MethodPost, c.apiURL(path), reqBody, &comment); err != nil {
		return nil, fmt.Errorf("add comment failed: %w", err)
	}
	return &comment, nil
}

// doRawRequest выполняет HTTP-запрос и возвращает сырой ответ.
func (c *Client) doRawRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	return c.httpClient.Do(req)
}

// doRequest выполняет HTTP-запрос с retry и декодированием ответа.
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
