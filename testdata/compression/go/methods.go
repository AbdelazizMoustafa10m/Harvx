package server

import (
	"net/http"
	"sync"
)

// Server handles HTTP requests.
type Server struct {
	addr    string
	handler http.Handler
	mu      sync.Mutex
}

// NewServer creates a new Server.
func NewServer(addr string, handler http.Handler) *Server {
	return &Server{
		addr:    addr,
		handler: handler,
	}
}

// Start begins listening on the configured address.
func (s *Server) Start() error {
	return http.ListenAndServe(s.addr, s.handler)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// cleanup
}

// String returns a string representation.
func (s Server) String() string {
	return s.addr
}

// unexported method with pointer receiver
func (s *Server) reset() {
	s.addr = ""
	s.handler = nil
}