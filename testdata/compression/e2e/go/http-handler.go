package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Handler manages HTTP request handling for the API.
type Handler struct {
	logger *slog.Logger
	db     Database
}

// Database defines the interface for data access.
type Database interface {
	GetUser(id string) (*User, error)
	CreateUser(u *User) error
	ListUsers(limit, offset int) ([]*User, error)
}

// User represents a user in the system.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(logger *slog.Logger, db Database) *Handler {
	return &Handler{logger: logger, db: db}
}

// GetUser handles GET /users/:id requests.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUser(id)
	if err != nil {
		h.logger.Error("failed to get user", "id", id, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ListUsers handles GET /users requests.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.db.ListUsers(100, 0)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// CreateUser handles POST /users requests.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.db.CreateUser(&user); err != nil {
		h.logger.Error("failed to create user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
