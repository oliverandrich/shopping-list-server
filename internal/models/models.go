// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

// Package models defines all data structures and DTOs used throughout the shopping list application,
// including database models, request/response types, and JWT claims.
package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SystemSettings represents the global configuration and setup status of the application.
type SystemSettings struct {
	ID           string    `gorm:"primarykey" json:"id"`
	IsSetup      bool      `gorm:"default:false" json:"is_setup"`
	SetupAt      time.Time `json:"setup_at"`
	InitialAdmin string    `json:"initial_admin"`
}

// User represents a user account in the shopping list system.
type User struct {
	ID        string    `gorm:"primarykey" json:"id"`
	Email     string    `gorm:"unique;not null" json:"email"`
	InvitedBy *string   `json:"invited_by"`
	JoinedAt  time.Time `json:"joined_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ShoppingList represents a shopping list that can be shared among users.
type ShoppingList struct {
	ID        string    `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	OwnerID   string    `gorm:"not null;index" json:"owner_id"`
	Owner     User      `gorm:"foreignKey:OwnerID" json:"owner"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListMember represents a user's membership in a shopping list with their role.
type ListMember struct {
	ListID   string    `gorm:"primarykey" json:"list_id"`
	UserID   string    `gorm:"primarykey" json:"user_id"`
	Role     string    `gorm:"default:'member'" json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// Invitation represents an invitation for a user to join the system or a specific list.
type Invitation struct {
	ID        string    `gorm:"primarykey" json:"id"`
	Code      string    `gorm:"unique;not null" json:"code"`
	Email     string    `gorm:"not null" json:"email"`
	Type      string    `gorm:"not null" json:"type"`
	ListID    *string   `json:"list_id"`
	InvitedBy string    `gorm:"not null" json:"invited_by"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

// MagicLink represents a temporary authentication code sent via email.
type MagicLink struct {
	Code      string    `gorm:"primarykey" json:"code"`
	Email     string    `gorm:"not null" json:"email"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
}

// ShoppingItem represents an item in a shopping list.
type ShoppingItem struct {
	ID        string       `gorm:"primarykey" json:"id"`
	ListID    string       `gorm:"not null;index" json:"list_id"`
	List      ShoppingList `gorm:"foreignKey:ListID" json:"list,omitempty"`
	Name      string       `json:"name"`
	Completed bool         `json:"completed" gorm:"default:false"`
	Tags      string       `json:"tags" gorm:"default:'[]'"`
	CreatedAt time.Time    `json:"created_at"`
}

// LoginRequest represents a request to initiate login via magic link.
type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// VerifyRequest represents a request to verify a magic link code.
type VerifyRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

// CreateItemRequest represents a request to create a new shopping item.
type CreateItemRequest struct {
	Name string `json:"name" validate:"required"`
	Tags string `json:"tags"`
}

// CreateListRequest represents a request to create a new shopping list.
type CreateListRequest struct {
	Name string `json:"name" validate:"required"`
}

// UpdateListRequest represents a request to update a shopping list.
type UpdateListRequest struct {
	Name string `json:"name" validate:"required"`
}

// CreateInvitationRequest represents a request to create an invitation.
type CreateInvitationRequest struct {
	Email  string  `json:"email" validate:"required,email"`
	Type   string  `json:"type" validate:"required"`
	ListID *string `json:"list_id"`
}

// AcceptInvitationRequest represents a request to accept an invitation.
type AcceptInvitationRequest struct {
	Code string `json:"code" validate:"required"`
}

// SetupRequest represents a request to set up the system with an admin user.
type SetupRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// LoginResponse represents the response after successful authentication.
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// JWTClaims represents the custom claims included in JWT tokens.
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}
