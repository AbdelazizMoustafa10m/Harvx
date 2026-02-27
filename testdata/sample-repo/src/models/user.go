package models

// User represents an authenticated user in the system.
type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

// NewUser creates a new User with the given email and name.
func NewUser(email, name string) *User {
	return &User{
		Email:    email,
		Name:     name,
		Role:     "user",
		IsActive: true,
	}
}

// IsAdmin reports whether the user has admin privileges.
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}