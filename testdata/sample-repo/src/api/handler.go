package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler holds the dependencies for HTTP request handling.
type Handler struct {
	version string
}

// NewHandler creates a new Handler with the given version string.
func NewHandler(version string) *Handler {
	return &Handler{version: version}
}

// HealthCheck returns a 200 OK with version information.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":  "ok",
		"version": h.version,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ListUsers returns a JSON array of user summaries.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"users":[],"total":0}`)
}