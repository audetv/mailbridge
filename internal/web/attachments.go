package web

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

// GetAttachment обрабатывает GET /api/attachments/{path...}
func (h *TaskHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Защита от path traversal
	cleanPath := filepath.Clean(path)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Путь к файлу вложений
	fullPath := filepath.Join("data", "attachments", cleanPath)

	// Проверяем что файл существует
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "not found", http.StatusNotFound)
		return
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
