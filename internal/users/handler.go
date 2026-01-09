package users

import (
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yabeye/addis_verify_backend/internal/media"
	"github.com/yabeye/addis_verify_backend/internal/middlewares"
	"github.com/yabeye/addis_verify_backend/pkg/json"
)

// Handler defines the public interface for user-related requests
type Handler interface {
	GetMe(w http.ResponseWriter, r *http.Request)
	UpdateProfile(w http.ResponseWriter, r *http.Request)
	GetUploadURL(w http.ResponseWriter, r *http.Request)
	HandleBinaryUpload(w http.ResponseWriter, r *http.Request)
}

type handler struct {
	service      Service
	mediaService *media.Service
	logger       *slog.Logger
	validate     *validator.Validate
}

// NewHandler ensures the struct implements the Handler interface
func NewHandler(s Service, m *media.Service, l *slog.Logger) Handler {
	return &handler{
		service:      s,
		mediaService: m,
		logger:       l,
		validate:     validator.New(),
	}
}

// GetMe fetches the combined Account + Profile data
func (h *handler) GetMe(w http.ResponseWriter, r *http.Request) {
	accID := h.getAccountID(r)
	if !accID.Valid {
		json.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fullData, err := h.service.GetFullMe(r.Context(), accID)
	if err != nil {
		h.logger.Error("failed to fetch profile", "error", err)
		json.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	json.Write(w, http.StatusOK, fullData)
}

// GetUploadURL provides the "Ticket" for the frontend to upload a file
func (h *handler) GetUploadURL(w http.ResponseWriter, r *http.Request) {
	accID := h.getAccountID(r)
	if !accID.Valid {
		json.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileType := r.URL.Query().Get("type") // e.g., headshot, gov_id, or passport
	if fileType == "" {
		json.WriteError(w, http.StatusBadRequest, "type query parameter is required")
		return
	}

	resp := h.mediaService.GenerateMockPresignedURL(UUIDToString(accID), fileType)
	json.Write(w, http.StatusOK, resp)
}

// HandleBinaryUpload acts as the "Mock Cloud Storage" endpoint
func (h *handler) HandleBinaryUpload(w http.ResponseWriter, r *http.Request) {
	// Extracting params from the URL: /api/v1/media/upload/{userID}/{fileName}
	userID := chi.URLParam(r, "userID")
	fileName := chi.URLParam(r, "fileName")

	if userID == "" || fileName == "" {
		json.WriteError(w, http.StatusBadRequest, "invalid upload path")
		return
	}

	storageKey := filepath.Join(userID, fileName)

	// Stream the raw request body directly to disk
	err := h.mediaService.SaveMockUpload(storageKey, r.Body)
	if err != nil {
		h.logger.Error("binary upload failed", "error", err)
		json.WriteError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	json.Write(w, http.StatusOK, map[string]string{"status": "success"})
}

// UpdateProfile saves the text data and the final URLs to the database
func (h *handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	accID := h.getAccountID(r)
	if !accID.Valid {
		json.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateProfileRequest
	if err := req.Bind(r); err != nil {
		h.logger.Warn("binding failed", "error", err)
		json.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		json.WriteError(w, http.StatusUnprocessableEntity, fmt.Sprintf("validation failed: %v", err))
		return
	}

	if err := h.service.UpdateFullProfile(r.Context(), accID, req); err != nil {
		h.logger.Error("database update failed", "error", err)
		json.WriteError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	// Return the updated data so the frontend can refresh immediately
	fullData, err := h.service.GetFullMe(r.Context(), accID)
	if err != nil {
		json.Write(w, http.StatusOK, map[string]string{"message": "profile updated"})
		return
	}

	json.Write(w, http.StatusOK, fullData)
}

// Helper to extract account_id from context
func (h *handler) getAccountID(r *http.Request) pgtype.UUID {
	id, ok := r.Context().Value(middlewares.UserIDKey).(pgtype.UUID)
	if !ok {
		h.logger.Error("user_id not found in context")
		return pgtype.UUID{}
	}
	return id
}
