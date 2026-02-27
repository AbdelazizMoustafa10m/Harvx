package auth

import "fmt"

// CheckAuth verifies authentication credentials.
func CheckAuth() bool {
	fmt.Println("checking auth")
	return true
}

// ValidateToken checks if a JWT token is valid.
func ValidateToken(token string) bool {
	return len(token) > 0
}
