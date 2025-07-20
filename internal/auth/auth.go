// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

// Package auth provides authentication services including magic link generation, JWT token management,
// and user verification for the shopping list application.
package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

// Service provides authentication services including magic link generation and JWT management.
type Service struct {
	DB        *gorm.DB
	JWTSecret []byte
	Mailer    *gomail.Dialer
}

// NewService creates a new authentication service with database, JWT secret, and email mailer.
func NewService(db *gorm.DB, jwtSecret []byte, mailer *gomail.Dialer) *Service {
	return &Service{
		DB:        db,
		JWTSecret: jwtSecret,
		Mailer:    mailer,
	}
}

// GenerateCode generates a secure 6-digit numeric code for magic link authentication.
func GenerateCode() string {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based random if crypto/rand fails
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", int(bytes[0])<<16|int(bytes[1])<<8|int(bytes[2]))[:6]
}

// SendMagicLink sends a magic link code to the specified email address.
func (s *Service) SendMagicLink(email, code string) error {
	// Skip email sending in test environment
	if os.Getenv("GO_ENV") == "test" {
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("SMTP_FROM"))
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Your Shopping List Login Code")

	body := fmt.Sprintf(`
Your login code is: %s

This code will expire in 15 minutes.

If you didn't request this, please ignore this email.
	`, code)

	m.SetBody("text/plain", body)

	return s.Mailer.DialAndSend(m)
}

// CreateMagicLink creates a new magic link for the given email and returns the code.
func (s *Service) CreateMagicLink(email string) (string, error) {
	code := GenerateCode()
	expiresAt := time.Now().Add(15 * time.Minute)

	// Clean up old codes for this email
	s.DB.Where("email = ?", email).Delete(&models.MagicLink{})

	// Store magic link
	magicLink := models.MagicLink{
		Code:      code,
		Email:     email,
		ExpiresAt: expiresAt,
	}

	if err := s.DB.Create(&magicLink).Error; err != nil {
		return "", err
	}

	return code, nil
}

// VerifyMagicLink verifies a magic link code and returns the associated user.
func (s *Service) VerifyMagicLink(email, code string) (*models.User, error) {
	var magicLink models.MagicLink
	result := s.DB.Where("code = ? AND email = ? AND used = false AND expires_at > ?",
		code, email, time.Now()).First(&magicLink)

	if result.Error != nil {
		return nil, result.Error
	}

	// Mark code as used
	magicLink.Used = true
	s.DB.Save(&magicLink)

	// Find or create user
	var user models.User
	result = s.DB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, errors.New("user not found - invitation required for new users")
	}

	return &user, nil
}

// VerifyMagicLinkWithInvitation verifies a magic link and processes any pending invitations.
func (s *Service) VerifyMagicLinkWithInvitation(email, code string) (*models.User, *models.Invitation, error) {
	var magicLink models.MagicLink
	result := s.DB.Where("code = ? AND email = ? AND used = false AND expires_at > ?",
		code, email, time.Now()).First(&magicLink)

	if result.Error != nil {
		return nil, nil, result.Error
	}

	// Mark code as used
	magicLink.Used = true
	s.DB.Save(&magicLink)

	// Find existing user
	var user models.User
	result = s.DB.Where("email = ?", email).First(&user)
	if result.Error == nil {
		// User exists, check for pending list invitation
		var invitation models.Invitation
		err := s.DB.Where("email = ? AND used = false AND expires_at > ? AND type = ?",
			email, time.Now(), "list").First(&invitation).Error
		if err == nil {
			return &user, &invitation, nil
		}
		return &user, nil, nil
	}

	// User doesn't exist, check for invitation
	var invitation models.Invitation
	err := s.DB.Where("email = ? AND used = false AND expires_at > ?",
		email, time.Now()).First(&invitation).Error
	if err != nil {
		return nil, nil, errors.New("invitation required for new users")
	}

	// Create new user
	user = models.User{
		ID:        uuid.New().String(),
		Email:     email,
		InvitedBy: &invitation.InvitedBy,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	if err := s.DB.Create(&user).Error; err != nil {
		return nil, nil, err
	}

	return &user, &invitation, nil
}

// GenerateJWT creates a new JWT token for the given user with 30-day expiry.
func (s *Service) GenerateJWT(user *models.User) (string, error) {
	claims := &models.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)), // 30 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.JWTSecret)
}

// ValidateJWT validates a JWT token and returns the claims if valid.
func (s *Service) ValidateJWT(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(_ *jwt.Token) (interface{}, error) {
		return s.JWTSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// JWTMiddleware returns a Fiber middleware that validates JWT tokens in requests.
func (s *Service) JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization format",
			})
		}

		claims, err := s.ValidateJWT(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Add user info to context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)

		return c.Next()
	}
}
