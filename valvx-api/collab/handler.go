// Package collab implements the BCF (BIM Collaboration Format) API endpoints.
//
// Provides CRUD operations for BCF topics, comments, and viewpoints,
// plus BCF 2.1 export/import functionality.
package collab

import (
	"encoding/json"
	"net/http"

	"github.com/nsssthlm/valvx-api/internal/auth"
)

// Handler holds the BCF HTTP handler dependencies.
type Handler struct {
	Service      *Service
	SessionStore *auth.SessionStore
}

// NewHandler creates a new BCF handler.
func NewHandler(svc *Service, sessionStore *auth.SessionStore) *Handler {
	return &Handler{Service: svc, SessionStore: sessionStore}
}

// getProfileID extracts the account from context and resolves the iam_profile
// for the given project. Returns empty string if unauthenticated.
func (h *Handler) getProfileID(r *http.Request) string {
	accountID := auth.AccountIDFromContext(r.Context())
	if accountID == "" {
		return ""
	}
	projectID := r.PathValue("projectId")
	if projectID == "" || h.SessionStore == nil {
		return accountID
	}
	profileID, err := h.SessionStore.GetProfileForProject(r.Context(), accountID, projectID)
	if err != nil {
		return accountID
	}
	return profileID
}

// RegisterRoutes registers BCF API routes on the given mux.
// All routes are under /api/projects/{projectId}/bcf/
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects/{projectId}/bcf/topics", h.ListTopics)
	mux.HandleFunc("POST /api/projects/{projectId}/bcf/topics", h.CreateTopic)
	mux.HandleFunc("GET /api/projects/{projectId}/bcf/topics/{topicId}", h.GetTopic)
	mux.HandleFunc("PUT /api/projects/{projectId}/bcf/topics/{topicId}", h.UpdateTopic)
	mux.HandleFunc("DELETE /api/projects/{projectId}/bcf/topics/{topicId}", h.DeleteTopic)

	mux.HandleFunc("GET /api/projects/{projectId}/bcf/topics/{topicId}/comments", h.ListComments)
	mux.HandleFunc("POST /api/projects/{projectId}/bcf/topics/{topicId}/comments", h.CreateComment)
	mux.HandleFunc("DELETE /api/projects/{projectId}/bcf/topics/{topicId}/comments/{commentId}", h.DeleteComment)

	mux.HandleFunc("POST /api/projects/{projectId}/bcf/topics/{topicId}/viewpoints", h.CreateViewpoint)
	mux.HandleFunc("GET /api/projects/{projectId}/bcf/topics/{topicId}/viewpoints/{vpId}/snapshot", h.GetSnapshot)

	mux.HandleFunc("GET /api/projects/{projectId}/bcf/export", h.ExportBCF)
	mux.HandleFunc("POST /api/projects/{projectId}/bcf/import", h.ImportBCF)
}

// ListTopics returns all BCF topics for a project.
func (h *Handler) ListTopics(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	if projectID == "" {
		http.Error(w, "missing projectId", http.StatusBadRequest)
		return
	}

	filters := TopicFilters{
		Status:     r.URL.Query().Get("status"),
		Priority:   r.URL.Query().Get("priority"),
		AssignedTo: r.URL.Query().Get("assigned_to"),
	}

	topics, err := h.Service.ListTopics(r.Context(), projectID, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, topics)
}

// CreateTopic creates a new BCF topic with optional viewpoint.
func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")

	var req CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	creatorID := h.getProfileID(r)
	if creatorID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	topic, err := h.Service.CreateTopic(r.Context(), projectID, creatorID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, topic)
}

// GetTopic returns a single topic with comments and viewpoints.
func (h *Handler) GetTopic(w http.ResponseWriter, r *http.Request) {
	topicID := r.PathValue("topicId")

	topic, err := h.Service.GetTopic(r.Context(), topicID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, topic)
}

// UpdateTopic updates a topic's fields.
func (h *Handler) UpdateTopic(w http.ResponseWriter, r *http.Request) {
	topicID := r.PathValue("topicId")

	var req CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	topic, err := h.Service.UpdateTopic(r.Context(), topicID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, topic)
}

// DeleteTopic deletes a topic and all its viewpoints/comments.
func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	topicID := r.PathValue("topicId")

	if err := h.Service.DeleteTopic(r.Context(), topicID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListComments returns all comments for a topic.
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	topicID := r.PathValue("topicId")

	comments, err := h.Service.ListComments(r.Context(), topicID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, comments)
}

// CreateComment adds a comment to a topic.
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	topicID := r.PathValue("topicId")

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	authorID := h.getProfileID(r)
	if authorID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	comment, err := h.Service.CreateComment(r.Context(), topicID, authorID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, comment)
}

// DeleteComment removes a comment.
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("commentId")

	if err := h.Service.DeleteComment(r.Context(), commentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateViewpoint adds a viewpoint to a topic.
func (h *Handler) CreateViewpoint(w http.ResponseWriter, r *http.Request) {
	topicID := r.PathValue("topicId")

	var req CreateViewpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	viewpoint, err := h.Service.CreateViewpoint(r.Context(), topicID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, viewpoint)
}

// GetSnapshot returns the PNG snapshot for a viewpoint.
func (h *Handler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	vpID := r.PathValue("vpId")

	data, contentType, err := h.Service.GetSnapshot(r.Context(), vpID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Write(data)
}

// ExportBCF generates a BCF 2.1 ZIP file for all topics in the project.
func (h *Handler) ExportBCF(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")

	data, err := h.Service.ExportBCF(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="bcf-export.bcf"`)
	w.Write(data)
}

// ImportBCF imports topics from a BCF 2.1 ZIP file.
func (h *Handler) ImportBCF(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "file too large or invalid form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	importerID := h.getProfileID(r)
	if importerID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	count, err := h.Service.ImportBCF(r.Context(), projectID, importerID, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"imported_topics": count,
	})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

