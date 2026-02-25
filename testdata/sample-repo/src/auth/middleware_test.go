package auth

import "testing"

func TestCheckAuth(t *testing.T) {
	if !CheckAuth() {
		t.Fatal("expected true")
	}
}
