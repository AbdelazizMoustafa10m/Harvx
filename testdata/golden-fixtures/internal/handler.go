package internal

// Handler processes requests.
type Handler struct {
	Name string
}

// Handle processes a single request.
func (h *Handler) Handle() error {
	return nil
}
