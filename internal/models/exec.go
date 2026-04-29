package models

import "database/sql"

type Exec struct {
	ID                int    `json:"id,omitempty" db:"id,omitempty"`
	FirstName         string `json:"firstname,omitempty" db:"first_name,omitempty"`
	LastName          string `json:"lastname,omitempty" db:"last_name,omitempty"`
	Email             string `json:"email,omitempty" db:"email,omitempty"`
	Username          string `json:"username,omitempty" db:"username,omitempty"`
	Password          string `json:"password,omitempty" db:"password,omitempty"`
	PasswordChangedAt sql.NullString `json:"password_changed_at,omitempty" db:"password_changed_at,omitempty"`
	UserCreatedAt     sql.NullString `json:"user_created_at,omitempty" db:"user_created_at,omitempty"`
	PasswordResetCode sql.NullString `json:"password_reset_code,omitempty" db:"password_reset_code,omitempty"`
	PasswordResetCodeExpires sql.NullString `json:"password_reset_code_expires,omitempty" db:"password_reset_code_expires,omitempty"`
	InactiveStatus    bool   `json:"inactive_status,omitempty" db:"inactive_status,omitempty"`
	Role              string `json:"role,omitempty" db:"role,omitempty"`
}

type UpdatePassword struct {
	CurrentPassword string `json:"current_password,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}

type UpdatePasswordResponse struct {
	Token string `json:"token,omitempty"`
	PasswordUpdated bool `json:"password_updated,omitempty"`
}