package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
)

func TestOllamaClient_Generate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if reqBody["model"] != "test-model" {
			t.Errorf("model = %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"response": `{"verdicts":[]}`}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := ai.NewOllamaClient(srv.URL, "test-model")
	ctx := context.Background()

	resp, err := client.Generate(ctx, "test prompt", nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if resp != `{"verdicts":[]}` {
		t.Errorf("resp = %s", resp)
	}
}

// TestOllamaClient_SystemAndTemperature — системный промпт передаётся как "system",
// temperature не хардкодится, а берётся из конфига (решение §7 #16, 2026-08-30).
func TestOllamaClient_SystemAndTemperature(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if reqBody["system"] != "СТРОГИЙ СИСТЕМНЫЙ ПРОМПТ" {
			t.Errorf("system = %v, want СТРОГИЙ СИСТЕМНЫЙ ПРОМПТ", reqBody["system"])
		}

		opts, ok := reqBody["options"].(map[string]interface{})
		if !ok {
			t.Fatal("options missing")
		}
		if opts["temperature"] != 0.1 {
			t.Errorf("temperature = %v, want 0.1", opts["temperature"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"response": "{...}"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := ai.NewOllamaClient(srv.URL, "test-model")
	client.SetSystem("СТРОГИЙ СИСТЕМНЫЙ ПРОМПТ")
	client.SetTemperature(0.1)

	if _, err := client.Generate(context.Background(), "test", nil); err != nil {
		t.Fatalf("Generate error: %v", err)
	}
}

func TestOllamaClient_WithImages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		images, ok := reqBody["images"].([]interface{})
		if !ok || len(images) != 1 {
			t.Errorf("expected 1 image, got %v", reqBody["images"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"response": "{}"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := ai.NewOllamaClient(srv.URL, "test-model")
	ctx := context.Background()

	_, err := client.Generate(ctx, "prompt", []string{"base64data"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
}

func TestOllamaClient_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("internal error")); err != nil {
			t.Fatalf("write error: %v", err)
		}
	}))
	defer srv.Close()

	client := ai.NewOllamaClient(srv.URL, "test-model")
	ctx := context.Background()

	_, err := client.Generate(ctx, "prompt", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
