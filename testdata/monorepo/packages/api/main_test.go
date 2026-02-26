package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleItems(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	rec := httptest.NewRecorder()

	handleItems(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}