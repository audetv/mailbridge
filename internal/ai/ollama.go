package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient реализует Client для локального Ollama API.
type OllamaClient struct {
	baseURL     string
	model       string
	system      string // Системный промпт (в запрос: "system")
	temperature float64
	httpClient  *http.Client
}

// NewOllamaClient создаёт новый OllamaClient.
func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		baseURL:     baseURL,
		model:       model,
		temperature: 0.1,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// SetSystem устанавливает системный промпт (передаётся в запрос как "system").
func (c *OllamaClient) SetSystem(system string) {
	c.system = system
}

// SetTemperature устанавливает temperature (0.1 — строгий JSON, минимум креатива).
func (c *OllamaClient) SetTemperature(t float64) {
	c.temperature = t
}

// Generate отправляет промпт в Ollama и возвращает ответ.
func (c *OllamaClient) Generate(ctx context.Context, prompt string, images []string) (string, error) {
	reqBody := map[string]interface{}{
		"model":  c.model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": c.temperature,
		},
	}

	if c.system != "" {
		reqBody["system"] = c.system
	}

	if len(images) > 0 {
		reqBody["images"] = images
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Response, nil
}
