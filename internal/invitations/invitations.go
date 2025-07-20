// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

package invitations

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type Service struct {
	DB     *gorm.DB
	Mailer *gomail.Dialer
}

func NewService(db *gorm.DB, mailer *gomail.Dialer) *Service {
	return &Service{
		DB:     db,
		Mailer: mailer,
	}
}

func GenerateInvitationCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based random if crypto/rand fails
		return fmt.Sprintf("%X", time.Now().UnixNano())[:8]
	}
	return fmt.Sprintf("%X", bytes)
}

func (s *Service) CreateInvitation(inviterID, email, invType string, listID *string) (*models.Invitation, error) {
	// Validate invitation type
	if invType != "server" && invType != "list" {
		return nil, errors.New("invalid invitation type")
	}

	// For list invitations, ensure list exists and inviter has permission
	if invType == "list" {
		if listID == nil {
			return nil, errors.New("list_id required for list invitations")
		}

		var member models.ListMember
		err := s.DB.Where("list_id = ? AND user_id = ? AND role = ?", *listID, inviterID, "owner").First(&member).Error
		if err != nil {
			return nil, errors.New("user is not the owner of this list")
		}
	}

	// Check if user is already registered
	var existingUser models.User
	err := s.DB.Where("email = ?", email).First(&existingUser).Error
	if err == nil {
		// User exists, check if they're already a member of the list (for list invitations)
		if invType == "list" {
			var existingMember models.ListMember
			err = s.DB.Where("list_id = ? AND user_id = ?", *listID, existingUser.ID).First(&existingMember).Error
			if err == nil {
				return nil, errors.New("user is already a member of this list")
			}
		} else {
			return nil, errors.New("user already exists")
		}
	}

	// Check for existing unused invitations for this email and type
	var existingInvitation models.Invitation
	err = s.DB.Where("email = ? AND used = false AND type = ?", email, invType).First(&existingInvitation).Error
	if err == nil {
		return nil, errors.New("user is already invited")
	}

	// Delete any existing unused invitations for this email (of any type)
	s.DB.Where("email = ? AND used = false", email).Delete(&models.Invitation{})

	// Create new invitation
	invitation := models.Invitation{
		ID:        uuid.New().String(),
		Code:      GenerateInvitationCode(),
		Email:     email,
		Type:      invType,
		ListID:    listID,
		InvitedBy: inviterID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		Used:      false,
		CreatedAt: time.Now(),
	}

	if err := s.DB.Create(&invitation).Error; err != nil {
		return nil, err
	}

	// Send invitation email (skip in test environment)
	if os.Getenv("GO_ENV") != "test" {
		if err := s.SendInvitationEmail(&invitation); err != nil {
			// Log error but don't fail invitation creation
			fmt.Printf("Warning: Failed to send invitation email: %v\n", err)
		}
	}

	return &invitation, nil
}

func (s *Service) SendInvitationEmail(invitation *models.Invitation) error {
	var inviterEmail string
	s.DB.Model(&models.User{}).Select("email").Where("id = ?", invitation.InvitedBy).Scan(&inviterEmail)

	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("SMTP_FROM"))
	m.SetHeader("To", invitation.Email)

	var subject, body string
	if invitation.Type == "server" {
		subject = "Invitation to Shopping List Server"
		body = fmt.Sprintf(`
You've been invited to join the Shopping List Server by %s.

Your invitation code is: %s

This invitation will expire in 7 days.

To accept this invitation, use the code when logging in for the first time.
`, inviterEmail, invitation.Code)
	} else {
		var listName string
		s.DB.Model(&models.ShoppingList{}).Select("name").Where("id = ?", invitation.ListID).Scan(&listName)

		subject = fmt.Sprintf("Invitation to shopping list: %s", listName)
		body = fmt.Sprintf(`
You've been invited to join the shopping list "%s" by %s.

Your invitation code is: %s

This invitation will expire in 7 days.

To accept this invitation, use the code when logging in.
`, listName, inviterEmail, invitation.Code)
	}

	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	return s.Mailer.DialAndSend(m)
}

func (s *Service) AcceptInvitation(email, code string) (*models.Invitation, error) {
	var invitation models.Invitation
	err := s.DB.Where("email = ? AND code = ? AND used = false AND expires_at > ?",
		email, strings.ToUpper(code), time.Now()).First(&invitation).Error
	if err != nil {
		return nil, errors.New("invalid or expired invitation")
	}

	// Mark invitation as used
	invitation.Used = true
	s.DB.Save(&invitation)

	return &invitation, nil
}

func (s *Service) GetUserInvitations(userID string) ([]models.Invitation, error) {
	var invitations []models.Invitation
	err := s.DB.Where("invited_by = ?", userID).Order("created_at DESC").Find(&invitations).Error
	return invitations, err
}

func (s *Service) RevokeInvitation(invitationID, userID string) error {
	result := s.DB.Where("id = ? AND invited_by = ? AND used = false", invitationID, userID).Delete(&models.Invitation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("invitation not found or already used")
	}
	return nil
}
