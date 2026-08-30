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

// OpenAIClient реализует Client для OpenAI-совместимых API (Cloud.ru, OpenAI).
type OpenAIClient struct {
	baseURL     string
	apiKey      string
	model       string
	system      string
	temperature float64
	httpClient  *http.Client
}

// NewOpenAIClient создаёт новый OpenAIClient.
func NewOpenAIClient(baseURL, apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		baseURL:     baseURL,
		apiKey:      apiKey,
		model:       model,
		temperature: 0.1,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// SetSystem устанавливает системный промпт (в запрос — messages[0] role=system).
func (c *OpenAIClient) SetSystem(system string) {
	c.system = system
}

// SetTemperature устанавливает temperature (0.1 — строгий JSON, минимум креатива).
func (c *OpenAIClient) SetTemperature(t float64) {
	c.temperature = t
}

// Generate отправляет промпт в OpenAI-совместимый API и возвращает ответ.
func (c *OpenAIClient) Generate(ctx context.Context, prompt string, _ []string) (string, error) {
	messages := []map[string]string{}
	if c.system != "" {
		messages = append(messages, map[string]string{"role": "system", "content": c.system})
	}
	messages = append(messages, map[string]string{"role": "user", "content": prompt})

	reqBody := map[string]interface{}{
		"model":       c.model,
		"messages":    messages,
		"temperature": c.temperature,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return "", fmt.Errorf("openai error %d: %s", resp.StatusCode, string(respBody))
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return openaiResp.Choices[0].Message.Content, nil
}
