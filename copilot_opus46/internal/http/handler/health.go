package handler

import (
	"context"
	"net/http"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/repository/sqlite"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/storage"
)

// HealthHandler handles health and readiness endpoints.
type HealthHandler struct {
	db       *sqlite.DB
	provider storage.Provider
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *sqlite.DB, provider storage.Provider) *HealthHandler {
	return &HealthHandler{db: db, provider: provider}
}

// Health handles GET /health.
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Ready handles GET /ready.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	checks := map[string]string{}

	// Check SQLite
	if err := h.db.Ping(ctx); err != nil {
		checks["sqlite"] = "error: " + err.Error()
	} else {
		checks["sqlite"] = "ok"
	}

	// Check storage provider
	if err := h.provider.Ping(context.Background()); err != nil {
		checks["storageProvider"] = "error: " + err.Error()
	} else {
		checks["storageProvider"] = "ok"
	}

	allOk := true
	for _, v := range checks {
		if v != "ok" {
			allOk = false
			break
		}
	}

	status := http.StatusOK
	if !allOk {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, map[string]any{
		"ok":     allOk,
		"checks": checks,
	})
}
