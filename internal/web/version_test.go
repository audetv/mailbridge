package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// v0.27, шаг 8: GET /api/version — публичный (без JWT), JSON
// {version, commit, built}, значения «как есть» из ldflags.
func TestVersion_Endpoint(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		method  string
		want    int
		wantJWT bool
	}{
		{name: "GET без auth = 200", method: http.MethodGet, want: http.StatusOK},
		{name: "POST = 405", method: http.MethodPost, want: http.StatusMethodNotAllowed},
		{name: "DELETE = 405", method: http.MethodDelete, want: http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			req := httptest.NewRequest(tt.method, "/api/version", nil)
			rec := httptest.NewRecorder()
			Version(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
		})
	}
}

// JSON-контракт: ровно три поля version/commit/built, строки «как есть»
// (dev/none не подавляются — решение владельца 2026-09-18).
func TestVersion_Payload(t *testing.T) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()
	Version(rec, req)

	var body struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Built   string `json:"built"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v (body: %s)", err, rec.Body.String())
	}
	if body.Version == "" || body.Commit == "" || body.Built == "" {
		t.Fatalf("пустые поля: %+v (defaults version.go: dev/none/unknown)", body)
	}
}
