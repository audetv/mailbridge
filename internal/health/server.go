package health

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// Server предоставляет HTTP-обработчики для health check'ов.
type Server struct {
	checks []Checker
	mu     sync.RWMutex
}

// NewServer создаёт новый health-сервер.
func NewServer(_ string) *Server {
	return &Server{
		checks: make([]Checker, 0),
	}
}

// Register добавляет проверку.
func (s *Server) Register(check Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks = append(s.checks, check)
}

// Handler возвращает http.Handler с маршрутами /health, /ready, /metrics.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/metrics", s.handleMetrics)
	return mux
}

// handleHealth отвечает на liveness-проверку.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("health: encode error: %v", err)
	}
}

// handleReady отвечает на readiness-проверку.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	checks := make([]Checker, len(s.checks))
	copy(checks, s.checks)
	s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	results := make(map[string]string)
	allOk := true

	for _, check := range checks {
		if err := check.Check(ctx); err != nil {
			results[check.Name()] = err.Error()
			allOk = false
		} else {
			results[check.Name()] = "ok"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if allOk {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": allOk,
		"checks": results,
	}); err != nil {
		log.Printf("health: encode error: %v", err)
	}
}

// handleMetrics отдаёт метрики в Prometheus-формате.
func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	// Заглушка — будет расширено позже
	_, _ = w.Write([]byte("# HELP mailbridge_up Whether the service is up\n"))
	_, _ = w.Write([]byte("# TYPE mailbridge_up gauge\n"))
	_, _ = w.Write([]byte("mailbridge_up 1\n"))
}
