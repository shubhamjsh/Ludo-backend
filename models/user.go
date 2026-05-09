package models

import (
	"time"
)

type User struct {
	ID          int64      `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Phone       *string    `json:"phone,omitempty" db:"phone"`
	Email       *string    `json:"email,omitempty" db:"email"`
	Password    *string    `json:"-" db:"password"` // Only for regular users, not guests
	AvatarURL   string     `json:"avatar_url" db:"avatar_url"`
	Coins       int        `json:"coins" db:"coins"`
	IsGuest     bool       `json:"is_guest" db:"is_guest"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

type SignupRequest struct {
	Name     string  `json:"name" validate:"required,min=2,max=50"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Phone    *string `json:"phone,omitempty" validate:"omitempty,min=10,max=15"`
	Password string  `json:"password" validate:"required,min=6,max=100"`
}

type LoginRequest struct {
	EmailOrPhone string `json:"email_or_phone" validate:"required"`
	Password     string `json:"password" validate:"required"`
}

type GuestLoginRequest struct {
	DeviceID string `json:"device_id" validate:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UpdateProfileRequest struct {
	Name      *string `json:"name,omitempty" validate:"omitempty,min=2,max=50"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

