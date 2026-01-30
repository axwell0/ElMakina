package replay

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Handler struct {
	Store Recorder
}

func NewHandler(store Recorder) *Handler {
	return &Handler{Store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		http.Error(w, "replay store not configured", http.StatusServiceUnavailable)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/replay/")
	if path == "" || path == r.URL.Path {
		http.NotFound(w, r)
		return
	}
	matchID := strings.Split(path, "/")[0]
	viewerID := r.URL.Query().Get("viewer_id")
	payload, err := h.Store.LoadReplay(r.Context(), matchID, viewerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
