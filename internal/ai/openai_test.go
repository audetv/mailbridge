package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
)

func TestOpenAIClient_Generate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong Authorization header")
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if reqBody["model"] != "test-model" {
			t.Errorf("model = %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "{\"verdicts\":[]}"}},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := ai.NewOpenAIClient(srv.URL, "test-key", "test-model")
	ctx := context.Background()

	resp, err := client.Generate(ctx, "test prompt", nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if resp != `{"verdicts":[]}` {
		t.Errorf("resp = %s", resp)
	}
}

func TestOpenAIClient_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"error":"invalid key"}`)); err != nil {
			t.Fatalf("write error: %v", err)
		}
	}))
	defer srv.Close()

	client := ai.NewOpenAIClient(srv.URL, "bad-key", "test-model")
	ctx := context.Background()

	_, err := client.Generate(ctx, "prompt", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestOpenAIClient_SystemAndTemperature — системный промпт идёт первым сообщением,
// temperature берётся из конфига (решение §7 #16, 2026-08-30).
func TestOpenAIClient_SystemAndTemperature(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		msgs, ok := reqBody["messages"].([]interface{})
		if !ok || len(msgs) != 2 {
			t.Fatalf("messages = %v, want 2 entries (system + user)", reqBody["messages"])
		}
		first := msgs[0].(map[string]interface{})
		if first["role"] != "system" || first["content"] != "СИСТЕМА" {
			t.Errorf("first message = %v, want role=system content=СИСТЕМА", first)
		}
		second := msgs[1].(map[string]interface{})
		if second["role"] != "user" {
			t.Errorf("second message role = %v, want user", second["role"])
		}
		if reqBody["temperature"] != 0.1 {
			t.Errorf("temperature = %v, want 0.1", reqBody["temperature"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "ok"}},
			},
		})
	}))
	defer srv.Close()

	client := ai.NewOpenAIClient(srv.URL, "test-key", "test-model")
	client.SetSystem("СИСТЕМА")
	client.SetTemperature(0.1)

	if _, err := client.Generate(context.Background(), "test", nil); err != nil {
		t.Fatalf("Generate error: %v", err)
	}
}

// png1x1B64 — валидный PNG 1×1 (для MIME-детекта из байтов).
const png1x1B64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// TestOpenAIClient_WithImages — hotfix 29.1 (issue #87): изображения обязаны
// уходить в запрос как мультимодальный content [{type:text},{type:image_url,data:URL}].
func TestOpenAIClient_WithImages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		msgs, ok := reqBody["messages"].([]interface{})
		if !ok || len(msgs) != 1 {
			t.Fatalf("messages = %v, want 1 entry", reqBody["messages"])
		}
		content, ok := msgs[0].(map[string]interface{})["content"].([]interface{})
		if !ok || len(content) != 2 {
			t.Fatalf("expected content array [text,image], got %#v", msgs[0].(map[string]interface{})["content"])
		}
		part0, _ := content[0].(map[string]interface{})
		if part0["type"] != "text" || part0["text"] != "опиши картинку" {
			t.Errorf("bad text part: %#v", part0)
		}
		part1, _ := content[1].(map[string]interface{})
		if part1["type"] != "image_url" {
			t.Fatalf("bad image part type: %#v", part1)
		}
		img, _ := part1["image_url"].(map[string]interface{})
		url, _ := img["url"].(string)
		if !strings.HasPrefix(url, "data:image/png;base64,") {
			t.Errorf("bad data URL prefix: %q", url)
		}
		if !strings.HasSuffix(url, png1x1B64) {
			t.Error("image payload altered")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "ok"}},
			},
		})
	}))
	defer srv.Close()

	client := ai.NewOpenAIClient(srv.URL, "test-key", "test-model")
	if _, err := client.Generate(context.Background(), "опиши картинку", []string{png1x1B64}); err != nil {
		t.Fatalf("Generate error: %v", err)
	}
}

// TestOpenAIClient_WithoutImages — регресс (hotfix 29.1): без картинок content
// остаётся строкой (поведение до фикса не меняется).
func TestOpenAIClient_WithoutImages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		msgs := reqBody["messages"].([]interface{})
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %v", reqBody["messages"])
		}
		if s, ok := msgs[0].(map[string]interface{})["content"].(string); !ok || s != "только текст" {
			t.Errorf("expected string content, got %#v", msgs[0].(map[string]interface{})["content"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "ok"}},
			},
		})
	}))
	defer srv.Close()

	client := ai.NewOpenAIClient(srv.URL, "test-key", "test-model")
	if _, err := client.Generate(context.Background(), "только текст", nil); err != nil {
		t.Fatalf("Generate error: %v", err)
	}
}
