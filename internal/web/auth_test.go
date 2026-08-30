package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/web"
)

func TestAuthHandler_Login_Success(t *testing.T) {
	handler := web.NewAuthHandler()

	body := `{"username":"admin","password":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
		User  struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Token == "" {
		t.Error("token is empty")
	}
	if resp.User.Username != "admin" {
		t.Errorf("username = %s", resp.User.Username)
	}
}

func TestAuthHandler_Login_Failure(t *testing.T) {
	handler := web.NewAuthHandler()

	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_Me_Authenticated(t *testing.T) {
	handler := web.NewAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer token-admin-20260101")
	w := httptest.NewRecorder()

	handler.Me(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp["username"] != "admin" {
		t.Errorf("username = %s", resp["username"])
	}
}

func TestAuthHandler_Me_Unauthenticated(t *testing.T) {
	handler := web.NewAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()

	handler.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// TestAuthHandler_AgentLogin — агент-юзер hermes (ФАЗА 4, шаг 18.2).
func TestAuthHandler_AgentLogin(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		body     string
		wantCode int
		wantUser string
	}{
		{
			name:     "agent enabled: success",
			env:      map[string]string{"MAILBRIDGE_AGENT_PASS": "agent-secret"},
			body:     `{"username":"hermes","password":"agent-secret"}`,
			wantCode: http.StatusOK,
			wantUser: "hermes",
		},
		{
			name:     "agent enabled: wrong password",
			env:      map[string]string{"MAILBRIDGE_AGENT_PASS": "agent-secret"},
			body:     `{"username":"hermes","password":"wrong"}`,
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "agent disabled (no pass): login rejected",
			env:      map[string]string{"MAILBRIDGE_AGENT_PASS": ""},
			body:     `{"username":"hermes","password":"whatever"}`,
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "agent enabled: admin still works",
			env:      map[string]string{"MAILBRIDGE_AGENT_PASS": "agent-secret"},
			body:     `{"username":"admin","password":"admin"}`,
			wantCode: http.StatusOK,
			wantUser: "admin",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				if v == "" {
					t.Setenv(k, "")
				} else {
					t.Setenv(k, v)
				}
			}
			handler := web.NewAuthHandler()
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.Login(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d: %s", tc.wantCode, w.Code, w.Body.String())
			}
			if tc.wantCode == http.StatusOK {
				var resp struct {
					Token string `json:"token"`
					User  struct {
						Username string `json:"username"`
					} `json:"user"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp.User.Username != tc.wantUser {
					t.Errorf("username = %s, want %s", resp.User.Username, tc.wantUser)
				}
				// Токен содержит имя пользователя (Me работает)
				reqMe := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
				reqMe.Header.Set("Authorization", "Bearer "+resp.Token)
				wMe := httptest.NewRecorder()
				handler.Me(wMe, reqMe)
				if wMe.Code != http.StatusOK {
					t.Errorf("Me with agent token: expected 200, got %d", wMe.Code)
				}
			}
		})
	}
}
