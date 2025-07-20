package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string    `gorm:"primarykey" json:"id"`
	Email     string    `gorm:"unique;not null" json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type MagicLink struct {
	Code      string    `gorm:"primarykey" json:"code"`
	Email     string    `gorm:"not null" json:"email"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
}

type ShoppingItem struct {
	ID        string    `gorm:"primarykey" json:"id"`
	UserID    string    `gorm:"not null;index" json:"user_id"`
	Name      string    `json:"name"`
	Completed bool      `json:"completed" gorm:"default:false"`
	Tags      string    `json:"tags" gorm:"default:'[]'"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type VerifyRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

type CreateItemRequest struct {
	Name string `json:"name" validate:"required"`
	Tags string `json:"tags"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}