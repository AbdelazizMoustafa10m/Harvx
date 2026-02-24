package models

import "time"

// User represents a registered user.
type User struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AdminUser embeds User and adds admin-specific fields.
type AdminUser struct {
	User
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// empty struct
type Empty struct{}

type (
	// Point represents a 2D point.
	Point struct {
		X float64
		Y float64
	}

	// Size represents dimensions.
	Size struct {
		Width  int
		Height int
	}
)