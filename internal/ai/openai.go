package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
// hotfix 29.1 (issue #87): images (base64, без префикса data:) передаются как
// мультимодальный content [{type:text},{type:image_url,...}] — формат принимают
// llama.cpp/llama-server и остальные OpenAI-совместимые vision-эндпоинты.
func (c *OpenAIClient) Generate(ctx context.Context, prompt string, images []string) (string, error) {
	messages := []map[string]interface{}{}
	if c.system != "" {
		messages = append(messages, map[string]interface{}{"role": "system", "content": c.system})
	}
	if len(images) > 0 {
		parts := []map[string]interface{}{{"type": "text", "text": prompt}}
		for _, img := range images {
			parts = append(parts, map[string]interface{}{
				"type":      "image_url",
				"image_url": map[string]string{"url": "data:" + imageMIME(img) + ";base64," + img},
			})
		}
		messages = append(messages, map[string]interface{}{"role": "user", "content": parts})
	} else {
		messages = append(messages, map[string]interface{}{"role": "user", "content": prompt})
	}

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

// imageMIME определяет тип изображения из base64-байтов (оркестратор отдаёт
// «голый» base64 без MIME). Неизвестное — image/png (безопасный дефолт для
// растеров из preprocessor).
func imageMIME(b64 string) string {
	if data, err := base64.StdEncoding.DecodeString(b64); err == nil {
		if ct := http.DetectContentType(data); strings.HasPrefix(ct, "image/") {
			return ct
		}
	}
	return "image/png"
}
