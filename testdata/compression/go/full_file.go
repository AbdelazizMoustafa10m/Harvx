// Package handler provides HTTP request handling.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ErrInvalidInput indicates the request payload is malformed.
var ErrInvalidInput = fmt.Errorf("invalid input")

// DefaultTimeout is the default request timeout.
const DefaultTimeout = 30 * time.Second

// ContentType constants for response headers.
const (
	ContentTypeJSON = "application/json"
	ContentTypeText = "text/plain"
)

// Handler processes HTTP requests.
type Handler struct {
	logger *slog.Logger
	store  Store
}

// Store defines the data access interface.
type Store interface {
	Get(ctx context.Context, id string) (*Item, error)
	List(ctx context.Context, limit int) ([]*Item, error)
	Create(ctx context.Context, item *Item) error
}

// Item represents a stored item.
type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     int       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(logger *slog.Logger, store Store) *Handler {
	return &Handler{
		logger: logger,
		store:  store,
	}
}

// HandleGet retrieves a single item by ID.
func (h *Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
	defer cancel()

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	item, err := h.store.Get(ctx, id)
	if err != nil {
		h.logger.Error("failed to get item", "id", id, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ContentTypeJSON)
	json.NewEncoder(w).Encode(item)
}

// HandleList retrieves a list of items.
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
	defer cancel()

	items, err := h.store.List(ctx, 100)
	if err != nil {
		h.logger.Error("failed to list items", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ContentTypeJSON)
	json.NewEncoder(w).Encode(items)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}