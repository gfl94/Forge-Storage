package handler

import (
	"net/http"

	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/model"
	"github.com/gfl94/Forge-Storage/copilot_opus46/internal/service"
)

// TransferHandler handles upload and download URL generation endpoints.
type TransferHandler struct {
	uploadService   *service.UploadService
	downloadService *service.DownloadService
}

// NewTransferHandler creates a new TransferHandler.
func NewTransferHandler(uploadService *service.UploadService, downloadService *service.DownloadService) *TransferHandler {
	return &TransferHandler{uploadService: uploadService, downloadService: downloadService}
}

// UploadURL handles POST /api/upload-url.
func (h *TransferHandler) UploadURL(w http.ResponseWriter, r *http.Request) {
	var req model.UploadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body.")
		return
	}

	req.Path = service.NormalizePath(req.Path)
	if err := service.ValidatePath(req.Path); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}

	target, err := h.uploadService.CreateUploadURL(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Failed to create upload URL.")
		return
	}

	writeJSON(w, http.StatusOK, target)
}

// DownloadURL handles POST /api/download-url.
func (h *TransferHandler) DownloadURL(w http.ResponseWriter, r *http.Request) {
	var req model.DownloadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body.")
		return
	}

	req.Path = service.NormalizePath(req.Path)
	if err := service.ValidatePath(req.Path); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}

	target, err := h.downloadService.CreateDownloadURL(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Failed to create download URL.")
		return
	}

	writeJSON(w, http.StatusOK, target)
}

// Preview handles GET /api/preview.
func (h *TransferHandler) Preview(w http.ResponseWriter, r *http.Request) {
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

	target, err := h.downloadService.CreateDownloadURL(r.Context(), model.DownloadRequest{Path: filePath})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider_error", "Failed to create preview URL.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":      filePath,
		"mode":      "redirect",
		"url":       target.URL,
		"expiresAt": target.ExpiresAt,
	})
}
