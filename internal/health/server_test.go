package health_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/audetv/mailbridge/internal/health"
)

func TestHealthEndpoint(t *testing.T) {
	srv := health.NewServer(":0")
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestReadyEndpoint_NoChecks(t *testing.T) {
	srv := health.NewServer(":0")
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with no checks, got %d", w.Code)
	}
}

func TestReadyEndpoint_WithFailingCheck(t *testing.T) {
	srv := health.NewServer(":0")
	srv.Register(health.NewNamedCheck("db", func(_ context.Context) error {
		return fmt.Errorf("connection refused")
	}))
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["status"] != false {
		t.Error("expected status false")
	}
}

func TestReadyEndpoint_AllOk(t *testing.T) {
	srv := health.NewServer(":0")
	srv.Register(health.NewNamedCheck("db", func(_ context.Context) error {
		return nil
	}))
	srv.Register(health.NewNamedCheck("plane", func(_ context.Context) error {
		return nil
	}))
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	srv := health.NewServer(":0")
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
