package models

import "time"

// UserProfile represents additional user information.
type UserProfile struct {
	FirstName    string            `json:"first_name"`
	LastName     string            `json:"last_name"`
	PhoneNumber  string            `json:"phone_number"`
	AvatarURL    string            `json:"avatar_url"`
	Bio          string            `json:"bio"`
	Timezone     string            `json:"timezone"`
	Language     string            `json:"language"`
	CustomFields map[string]string `json:"custom_fields"`
}

// User represents a user in the system.
type User struct {
	ID        int32        `json:"id"`
	Username  string       `json:"username"`
	Email     string       `json:"email"`
	Password  string       `json:"-"`
	Roles     []string     `json:"roles"`
	Profile   *UserProfile `json:"profile"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}
