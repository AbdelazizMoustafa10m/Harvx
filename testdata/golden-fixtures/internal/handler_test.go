package internal

import "testing"

func TestHandler_Handle(t *testing.T) {
	h := &Handler{Name: "test"}
	if err := h.Handle(); err != nil {
		t.Fatal(err)
	}
}
