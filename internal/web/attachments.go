package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

// GetAttachment обрабатывает GET /api/attachments/{path...}
// path может быть: {hash[0:2]}/{hash[2:4]}/{hash}/{filename}
func (h *TaskHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	cleanPath := filepath.Clean(path)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Разбиваем на части: {hash[0:2]}/{hash[2:4]}/{hash}/{filename}
	parts := strings.Split(cleanPath, string(filepath.Separator))

	var hashPath string
	var hash string
	if len(parts) >= 3 {
		// Первые три части — hash-путь, остальное — filename
		hashPath = filepath.Join(parts[0], parts[1], parts[2])
		hash = parts[2]
	} else {
		// Без filename
		hashPath = cleanPath
		hash = filepath.Base(cleanPath)
	}

	// Путь к файлу — только hash-путь
	fullPath := filepath.Join("data", "attachments", hashPath)

	// Проверяем что файл существует
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Ищем вложение в БД
	if hash != "" {
		if att, err := h.store.GetAttachmentByHash(r.Context(), hash); err == nil && att != nil {
			disposition := fmt.Sprintf(`inline; filename*=UTF-8''%s`, url.PathEscape(att.Filename))
			w.Header().Set("Content-Disposition", disposition)
			w.Header().Set("Content-Type", att.ContentType)
		}
	}

	http.ServeFile(w, r, fullPath)
}

// GetInboxAttachments обрабатывает GET /api/inbox/{id}/attachments
func (h *TaskHandler) GetInboxAttachments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	atts, err := h.store.GetAttachmentsByInbox(r.Context(), id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	if atts == nil {
		atts = []*store.Attachment{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(atts); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// GetTaskAttachments обрабатывает GET /api/tasks/{id}/attachments
func (h *TaskHandler) GetTaskAttachments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	atts, err := h.store.GetAttachmentsByTask(r.Context(), id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	if atts == nil {
		atts = []*store.Attachment{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(atts); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// UnlinkTaskAttachment обрабатывает DELETE /api/tasks/{id}/attachments/{attId}
func (h *TaskHandler) UnlinkTaskAttachment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	attStr := r.PathValue("attId")
	attID, _ := strconv.ParseInt(attStr, 10, 64)

	if err := h.store.UnlinkAttachmentFromTask(r.Context(), id, attID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("encode error: %v", err)
	}
}
