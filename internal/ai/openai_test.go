package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
