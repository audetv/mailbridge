// Package web предоставляет HTTP-обработчики для веб-интерфейса Mailbridge.
package web

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// AuthHandler обрабатывает запросы аутентификации.
// Поддерживает двух пользователей: admin (MAILBRIDGE_AUTH_USER/PASS) и
// агента (MAILBRIDGE_AGENT_USER, по умолчанию hermes; активен, если задан
// MAILBRIDGE_AGENT_PASS).
type AuthHandler struct {
	users map[string]string // активные пользователи: username -> password
}

// NewAuthHandler создаёт новый AuthHandler.
func NewAuthHandler() *AuthHandler {
	users := map[string]string{
		getEnv("MAILBRIDGE_AUTH_USER", "admin"): getEnv("MAILBRIDGE_AUTH_PASS", "admin"),
	}
	// Агент-юзер (ФАЗА 4): существует только если задан пароль.
	agentUser := getEnv("MAILBRIDGE_AGENT_USER", "hermes")
	if pass, ok := os.LookupEnv("MAILBRIDGE_AGENT_PASS"); ok && pass != "" {
		users[agentUser] = pass
	}
	return &AuthHandler{users: users}
}

// Login обрабатывает POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	expectedPass, userExists := h.users[req.Username]
	authorized := userExists && subtle.ConstantTimeCompare([]byte(req.Password), []byte(expectedPass)) == 1

	if !authorized {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "неверный логин или пароль"}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	token := generateToken(req.Username)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user": map[string]string{
			"username": req.Username,
		},
	}); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// Me обрабатывает GET /api/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	username := extractUserFromToken(r)
	if username == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"username": username,
	}); err != nil {
		log.Printf("encode error: %v", err)
	}
}

func generateToken(username string) string {
	return "token-" + username + "-" + time.Now().Format("20060102")
}

func extractUserFromToken(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if len(token) < 7 || token[:7] != "Bearer " {
		return ""
	}
	token = token[7:]

	if strings.HasPrefix(token, "token-") {
		parts := strings.SplitN(token, "-", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}
