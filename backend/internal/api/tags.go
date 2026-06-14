package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) HandleGetTrendingTags(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 50 {
			limit = val
		}
	}

	tags, err := h.db.GetTrendingTags(r.Context(), limit)
	if err != nil {
		WriteInternalError(w, "Failed to fetch trending tags")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=300, s-maxage=600")
	WriteSuccess(w, tags)
}

func (h *Handler) HandleGetUserTags(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	if did == "" {
		WriteBadRequest(w, "did is required")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	tags, err := h.db.GetUserTags(r.Context(), did, limit)
	if err != nil {
		WriteInternalError(w, "Failed to fetch user tags")
		return
	}

	WriteSuccess(w, tags)
}
