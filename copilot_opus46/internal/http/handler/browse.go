package handler

import (
	"net/http"
	"strconv"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/service"
)

// BrowseHandler handles directory listing and file metadata endpoints.
type BrowseHandler struct {
	browseService *service.BrowseService
	fileService   *service.FileService
}

// NewBrowseHandler creates a new BrowseHandler.
func NewBrowseHandler(browseService *service.BrowseService, fileService *service.FileService) *BrowseHandler {
	return &BrowseHandler{browseService: browseService, fileService: fileService}
}

// List handles GET /api/list.
func (h *BrowseHandler) List(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/"
	}

	dirPath = service.NormalizePath(dirPath)
	if err := service.ValidatePath(dirPath); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	result, err := h.browseService.List(r.Context(), dirPath, model.ListOptions{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Failed to list directory.")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Meta handles GET /api/meta.
func (h *BrowseHandler) Meta(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Path parameter is required.")
		return
	}

	filePath = service.NormalizePath(filePath)
	if err := service.ValidatePath(filePath); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}

	info, err := h.fileService.Meta(r.Context(), filePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Failed to retrieve metadata.")
		return
	}

	writeJSON(w, http.StatusOK, info)
}
