package handlers

import (
	"net/http"

	"github.com/fisher/backend/internal/services/deezer"
)

type PreviewHandler struct{}

func NewPreviewHandler() *PreviewHandler {
	return &PreviewHandler{}
}

func (h *PreviewHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tracks []deezer.TrackRequest `json:"tracks"`
	}
	if !decode(w, r, &req) {
		return
	}
	if len(req.Tracks) == 0 {
		respondError(w, http.StatusBadRequest, "tracks array is required")
		return
	}
	if len(req.Tracks) > 50 {
		respondError(w, http.StatusBadRequest, "maximum 50 tracks per request")
		return
	}

	results := deezer.ResolvePreviews(req.Tracks)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"previews": results,
	})
}
