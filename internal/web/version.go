package web

import (
	"encoding/json"
	"net/http"

	"github.com/audetv/mailbridge/internal/version"
)

// VersionInfo — ответ GET /api/version (v0.27, шаг 8): вшитые значения как есть.
type VersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

// Version обрабатывает GET /api/version — публичный эндпоинт (без JWT,
// решение владельца 2026-09-18): {version, commit, built} из ldflags,
// значения показываем как есть (dev/none допустимы).
func Version(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(VersionInfo{
		Version: version.Version,
		Commit:  version.Commit,
		Built:   version.BuildTime,
	})
}
