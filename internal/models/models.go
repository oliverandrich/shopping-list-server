package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type SystemSettings struct {
	ID           string    `gorm:"primarykey" json:"id"`
	IsSetup      bool      `gorm:"default:false" json:"is_setup"`
	SetupAt      time.Time `json:"setup_at"`
	InitialAdmin string    `json:"initial_admin"`
}

type User struct {
	ID        string    `gorm:"primarykey" json:"id"`
	Email     string    `gorm:"unique;not null" json:"email"`
	InvitedBy *string   `json:"invited_by"`
	JoinedAt  time.Time `json:"joined_at"`
	CreatedAt time.Time `json:"created_at"`
}

type ShoppingList struct {
	ID        string    `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	OwnerID   string    `gorm:"not null;index" json:"owner_id"`
	Owner     User      `gorm:"foreignKey:OwnerID" json:"owner"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListMember struct {
	ListID   string    `gorm:"primarykey" json:"list_id"`
	UserID   string    `gorm:"primarykey" json:"user_id"`
	Role     string    `gorm:"default:'member'" json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

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

type MagicLink struct {
	Code      string    `gorm:"primarykey" json:"code"`
	Email     string    `gorm:"not null" json:"email"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
}

type ShoppingItem struct {
	ID        string       `gorm:"primarykey" json:"id"`
	ListID    string       `gorm:"not null;index" json:"list_id"`
	List      ShoppingList `gorm:"foreignKey:ListID" json:"list,omitempty"`
	Name      string       `json:"name"`
	Completed bool         `json:"completed" gorm:"default:false"`
	Tags      string       `json:"tags" gorm:"default:'[]'"`
	CreatedAt time.Time    `json:"created_at"`
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

type CreateListRequest struct {
	Name string `json:"name" validate:"required"`
}

type UpdateListRequest struct {
	Name string `json:"name" validate:"required"`
}

type CreateInvitationRequest struct {
	Email  string  `json:"email" validate:"required,email"`
	Type   string  `json:"type" validate:"required"`
	ListID *string `json:"list_id"`
}

type AcceptInvitationRequest struct {
	Code string `json:"code" validate:"required"`
}

type SetupRequest struct {
	Email string `json:"email" validate:"required,email"`
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