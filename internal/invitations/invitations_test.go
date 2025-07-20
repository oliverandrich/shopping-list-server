// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

package invitations

import (
	"strings"
	"testing"
	"time"

	"github.com/oliverandrich/shopping-list-server/internal/models"
	"github.com/oliverandrich/shopping-list-server/internal/testutils"
	"gopkg.in/gomail.v2"
)

func TestNewService(t *testing.T) {
	db := testutils.SetupTestDB(t)
	mailer := gomail.NewDialer("smtp.test.com", 587, "test", "test")

	service := NewService(db, mailer)

	if service == nil {
		t.Fatal("Service should not be nil")
	}

	if service.DB != db {
		t.Error("Service DB should match provided database")
	}

	if service.Mailer != mailer {
		t.Error("Service Mailer should match provided mailer")
	}
}

func TestGenerateInvitationCode(t *testing.T) {
	code := GenerateInvitationCode()

	if len(code) != 8 {
		t.Errorf("Expected invitation code length to be 8, got %d", len(code))
	}

	// Check that code contains only hex characters (0-9, A-F)
	for _, char := range code {
		if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F')) {
			t.Errorf("Expected code to contain only hex characters, found '%c'", char)
		}
	}

	// Generate multiple codes to ensure they are different
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := GenerateInvitationCode()
		if codes[code] {
			t.Errorf("Generated duplicate code: %s", code)
		}
		codes[code] = true
	}
}

func TestService_CreateInvitation_ServerType(t *testing.T) {
	db := testutils.SetupTestDB(t)
	// Set up test environment
	testutils.SetupTestConfig(t)
	defer testutils.CleanupTestEnv(t)

	// Use a test mailer to avoid nil pointer issues
	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	service := NewService(db, mailer)

	// Create a test user who will send the invitation
	inviter := models.User{
		ID:        "inviter-id",
		Email:     "inviter@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&inviter).Error
	if err != nil {
		t.Fatalf("Failed to create inviter: %v", err)
	}

	t.Run("create server invitation", func(t *testing.T) {
		email := "newuser@example.com"

		// Use a test mailer to avoid nil pointer
		testMailer := gomail.NewDialer("localhost", 587, "test", "test")
		testService := NewService(db, testMailer)

		invitation, err := testService.CreateInvitation(inviter.ID, email, "server", nil)
		if err != nil {
			t.Fatalf("Failed to create server invitation: %v", err)
		}

		if invitation.Email != email {
			t.Errorf("Expected invitation email to be '%s', got '%s'", email, invitation.Email)
		}

		if invitation.Type != "server" {
			t.Errorf("Expected invitation type to be 'server', got '%s'", invitation.Type)
		}

		if invitation.InvitedBy != inviter.ID {
			t.Errorf("Expected invited by to be '%s', got '%s'", inviter.ID, invitation.InvitedBy)
		}

		if invitation.ListID != nil {
			t.Error("Server invitation should not have a list ID")
		}

		if invitation.Used {
			t.Error("New invitation should not be marked as used")
		}

		if len(invitation.Code) != 8 {
			t.Errorf("Expected invitation code length to be 8, got %d", len(invitation.Code))
		}

		// Check expiration (should be 7 days from now)
		expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
		if invitation.ExpiresAt.Before(expectedExpiry.Add(-time.Minute)) || invitation.ExpiresAt.After(expectedExpiry.Add(time.Minute)) {
			t.Error("Invitation should expire in approximately 7 days")
		}

		// Verify invitation was saved to database
		var dbInvitation models.Invitation
		err = db.Where("id = ?", invitation.ID).First(&dbInvitation).Error
		if err != nil {
			t.Fatal("Invitation should be saved in database")
		}
	})

	t.Run("create invitation for existing user", func(t *testing.T) {
		// Create an existing user
		existingUser := models.User{
			ID:        "existing-user-id",
			Email:     "existing@example.com",
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
		}
		err := db.Create(&existingUser).Error
		if err != nil {
			t.Fatalf("Failed to create existing user: %v", err)
		}

		_, err = service.CreateInvitation(inviter.ID, existingUser.Email, "server", nil)
		if err == nil {
			t.Error("Expected error when creating server invitation for existing user")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Error("Error should mention that user already exists")
		}
	})

	t.Run("create duplicate invitation", func(t *testing.T) {
		email := "duplicate@example.com"

		// Create first invitation
		_, err := service.CreateInvitation(inviter.ID, email, "server", nil)
		if err != nil {
			t.Fatalf("Failed to create first invitation: %v", err)
		}

		// Try to create duplicate
		_, err = service.CreateInvitation(inviter.ID, email, "server", nil)
		if err == nil {
			t.Error("Expected error when creating duplicate invitation")
		}

		if !strings.Contains(err.Error(), "already invited") {
			t.Error("Error should mention that user is already invited")
		}
	})
}

func TestService_CreateInvitation_ListType(t *testing.T) {
	db := testutils.SetupTestDB(t)
	// Set up test environment
	testutils.SetupTestConfig(t)
	defer testutils.CleanupTestEnv(t)

	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	service := NewService(db, mailer)

	// Create test users
	owner := models.User{
		ID:        "owner-id",
		Email:     "owner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	invitee := models.User{
		ID:        "invitee-id",
		Email:     "invitee@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := db.Create(&owner).Error
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}
	err = db.Create(&invitee).Error
	if err != nil {
		t.Fatalf("Failed to create invitee: %v", err)
	}

	// Create a test list
	list := models.ShoppingList{
		ID:        "test-list-id",
		Name:      "Test List",
		OwnerID:   owner.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = db.Create(&list).Error
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	// Add owner as list member
	member := models.ListMember{
		ListID:   list.ID,
		UserID:   owner.ID,
		Role:     "owner",
		JoinedAt: time.Now(),
	}
	err = db.Create(&member).Error
	if err != nil {
		t.Fatalf("Failed to create list member: %v", err)
	}

	t.Run("create list invitation as owner", func(t *testing.T) {
		listID := list.ID

		invitation, err := service.CreateInvitation(owner.ID, invitee.Email, "list", &listID)
		if err != nil {
			t.Fatalf("Failed to create list invitation: %v", err)
		}

		if invitation.Type != "list" {
			t.Errorf("Expected invitation type to be 'list', got '%s'", invitation.Type)
		}

		if invitation.ListID == nil || *invitation.ListID != listID {
			t.Error("List invitation should have correct list ID")
		}

		if invitation.InvitedBy != owner.ID {
			t.Error("Invitation should be created by the owner")
		}
	})

	t.Run("create list invitation as non-owner", func(t *testing.T) {
		// Create another user who is not the owner
		nonOwner := models.User{
			ID:        "non-owner-id",
			Email:     "nonowner@example.com",
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
		}
		err := db.Create(&nonOwner).Error
		if err != nil {
			t.Fatalf("Failed to create non-owner: %v", err)
		}

		listID := list.ID
		_, err = service.CreateInvitation(nonOwner.ID, "someone@example.com", "list", &listID)
		if err == nil {
			t.Error("Expected error when creating list invitation as non-owner")
		}

		if !strings.Contains(err.Error(), "not the owner") {
			t.Error("Error should mention that user is not the owner")
		}
	})

	t.Run("create list invitation without list ID", func(t *testing.T) {
		_, err := service.CreateInvitation(owner.ID, "someone@example.com", "list", nil)
		if err == nil {
			t.Error("Expected error when creating list invitation without list ID")
		}
	})

	t.Run("create list invitation for non-existent list", func(t *testing.T) {
		nonExistentListID := "non-existent-list"
		_, err := service.CreateInvitation(owner.ID, "someone@example.com", "list", &nonExistentListID)
		if err == nil {
			t.Error("Expected error when creating invitation for non-existent list")
		}
	})
}

func TestService_GetUserInvitations(t *testing.T) {
	db := testutils.SetupTestDB(t)
	// Set up test environment
	testutils.SetupTestConfig(t)
	defer testutils.CleanupTestEnv(t)

	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	service := NewService(db, mailer)

	// Create test users
	user1 := models.User{
		ID:        "user1-id",
		Email:     "user1@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	user2 := models.User{
		ID:        "user2-id",
		Email:     "user2@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := db.Create(&user1).Error
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}
	err = db.Create(&user2).Error
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create invitations for user1
	inv1, err := service.CreateInvitation(user1.ID, "invitee1@example.com", "server", nil)
	if err != nil {
		t.Fatalf("Failed to create invitation 1: %v", err)
	}

	inv2, err := service.CreateInvitation(user1.ID, "invitee2@example.com", "server", nil)
	if err != nil {
		t.Fatalf("Failed to create invitation 2: %v", err)
	}

	// Create invitation for user2
	_, err = service.CreateInvitation(user2.ID, "invitee3@example.com", "server", nil)
	if err != nil {
		t.Fatalf("Failed to create invitation for user2: %v", err)
	}

	t.Run("get invitations for user1", func(t *testing.T) {
		invitations, err := service.GetUserInvitations(user1.ID)
		if err != nil {
			t.Fatalf("Failed to get user invitations: %v", err)
		}

		if len(invitations) != 2 {
			t.Errorf("Expected 2 invitations for user1, got %d", len(invitations))
		}

		// Verify invitation IDs
		foundInv1, foundInv2 := false, false
		for _, inv := range invitations {
			if inv.ID == inv1.ID {
				foundInv1 = true
			} else if inv.ID == inv2.ID {
				foundInv2 = true
			}
		}

		if !foundInv1 || !foundInv2 {
			t.Error("Should find both invitations created by user1")
		}
	})

	t.Run("get invitations for user2", func(t *testing.T) {
		invitations, err := service.GetUserInvitations(user2.ID)
		if err != nil {
			t.Fatalf("Failed to get user invitations: %v", err)
		}

		if len(invitations) != 1 {
			t.Errorf("Expected 1 invitation for user2, got %d", len(invitations))
		}
	})

	t.Run("get invitations for user with no invitations", func(t *testing.T) {
		// Create a user with no invitations
		userWithNoInvitations := models.User{
			ID:        "no-invitations-user",
			Email:     "noinvitations@example.com",
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
		}
		err := db.Create(&userWithNoInvitations).Error
		if err != nil {
			t.Fatalf("Failed to create user with no invitations: %v", err)
		}

		invitations, err := service.GetUserInvitations(userWithNoInvitations.ID)
		if err != nil {
			t.Fatalf("Failed to get user invitations: %v", err)
		}

		if len(invitations) != 0 {
			t.Errorf("Expected 0 invitations for user with no invitations, got %d", len(invitations))
		}
	})
}

func TestService_AcceptInvitation(t *testing.T) {
	db := testutils.SetupTestDB(t)
	// Set up test environment
	testutils.SetupTestConfig(t)
	defer testutils.CleanupTestEnv(t)

	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	service := NewService(db, mailer)

	// Create test user
	inviter := models.User{
		ID:        "inviter-id",
		Email:     "inviter@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&inviter).Error
	if err != nil {
		t.Fatalf("Failed to create inviter: %v", err)
	}

	t.Run("accept valid invitation", func(t *testing.T) {
		email := "newuser@example.com"

		// Create invitation
		invitation, err := service.CreateInvitation(inviter.ID, email, "server", nil)
		if err != nil {
			t.Fatalf("Failed to create invitation: %v", err)
		}

		// Accept invitation
		acceptedInvitation, err := service.AcceptInvitation(email, invitation.Code)
		if err != nil {
			t.Fatalf("Failed to accept invitation: %v", err)
		}

		if acceptedInvitation.ID != invitation.ID {
			t.Error("Accepted invitation should match created invitation")
		}

		if !acceptedInvitation.Used {
			t.Error("Accepted invitation should be marked as used")
		}

		// Verify invitation is marked as used in database
		var dbInvitation models.Invitation
		err = db.Where("id = ?", invitation.ID).First(&dbInvitation).Error
		if err != nil {
			t.Fatal("Failed to retrieve invitation from database")
		}

		if !dbInvitation.Used {
			t.Error("Invitation should be marked as used in database")
		}
	})

	t.Run("accept invitation with wrong email", func(t *testing.T) {
		// Create invitation
		invitation, err := service.CreateInvitation(inviter.ID, "correct@example.com", "server", nil)
		if err != nil {
			t.Fatalf("Failed to create invitation: %v", err)
		}

		// Try to accept with wrong email
		_, err = service.AcceptInvitation("wrong@example.com", invitation.Code)
		if err == nil {
			t.Error("Expected error when accepting invitation with wrong email")
		}
	})

	t.Run("accept invitation with invalid code", func(t *testing.T) {
		_, err := service.AcceptInvitation("test@example.com", "invalid-code")
		if err == nil {
			t.Error("Expected error when accepting invitation with invalid code")
		}
	})

	t.Run("accept expired invitation", func(t *testing.T) {
		// Create an expired invitation manually
		expiredInvitation := models.Invitation{
			ID:        "expired-invitation-id",
			Code:      "expired123",
			Email:     "expired@example.com",
			Type:      "server",
			InvitedBy: inviter.ID,
			ExpiresAt: time.Now().Add(-time.Hour),
			Used:      false,
			CreatedAt: time.Now().Add(-8 * 24 * time.Hour),
		}
		err := db.Create(&expiredInvitation).Error
		if err != nil {
			t.Fatalf("Failed to create expired invitation: %v", err)
		}

		_, err = service.AcceptInvitation(expiredInvitation.Email, expiredInvitation.Code)
		if err == nil {
			t.Error("Expected error when accepting expired invitation")
		}
	})

	t.Run("accept already used invitation", func(t *testing.T) {
		email := "alreadyused@example.com"

		// Create and accept invitation
		invitation, err := service.CreateInvitation(inviter.ID, email, "server", nil)
		if err != nil {
			t.Fatalf("Failed to create invitation: %v", err)
		}

		_, err = service.AcceptInvitation(email, invitation.Code)
		if err != nil {
			t.Fatalf("Failed to accept invitation first time: %v", err)
		}

		// Try to accept again
		_, err = service.AcceptInvitation(email, invitation.Code)
		if err == nil {
			t.Error("Expected error when accepting already used invitation")
		}
	})
}

func TestService_RevokeInvitation(t *testing.T) {
	db := testutils.SetupTestDB(t)
	// Set up test environment
	testutils.SetupTestConfig(t)
	defer testutils.CleanupTestEnv(t)

	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	service := NewService(db, mailer)

	// Create test user
	inviter := models.User{
		ID:        "inviter-id",
		Email:     "inviter@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&inviter).Error
	if err != nil {
		t.Fatalf("Failed to create inviter: %v", err)
	}

	t.Run("revoke own invitation", func(t *testing.T) {
		// Create invitation
		invitation, err := service.CreateInvitation(inviter.ID, "revoke@example.com", "server", nil)
		if err != nil {
			t.Fatalf("Failed to create invitation: %v", err)
		}

		// Revoke invitation
		err = service.RevokeInvitation(invitation.ID, inviter.ID)
		if err != nil {
			t.Fatalf("Failed to revoke invitation: %v", err)
		}

		// Verify invitation was deleted from database
		var count int64
		err = db.Model(&models.Invitation{}).Where("id = ?", invitation.ID).Count(&count).Error
		if err != nil {
			t.Fatal("Failed to count invitations after revocation")
		}

		if count != 0 {
			t.Error("Invitation should be deleted after revocation")
		}
	})

	t.Run("revoke someone else's invitation", func(t *testing.T) {
		// Create another user
		otherUser := models.User{
			ID:        "other-user-id",
			Email:     "other@example.com",
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
		}
		err := db.Create(&otherUser).Error
		if err != nil {
			t.Fatalf("Failed to create other user: %v", err)
		}

		// Create invitation
		invitation, err := service.CreateInvitation(inviter.ID, "someone@example.com", "server", nil)
		if err != nil {
			t.Fatalf("Failed to create invitation: %v", err)
		}

		// Try to revoke as other user
		err = service.RevokeInvitation(invitation.ID, otherUser.ID)
		if err == nil {
			t.Error("Expected error when revoking someone else's invitation")
		}
	})

	t.Run("revoke non-existent invitation", func(t *testing.T) {
		err := service.RevokeInvitation("non-existent-invitation", inviter.ID)
		if err == nil {
			t.Error("Expected error when revoking non-existent invitation")
		}
	})
}

func TestService_SendInvitationEmail(t *testing.T) {
	// Set up test environment
	testutils.SetupTestConfig(t)
	defer testutils.CleanupTestEnv(t)

	db := testutils.SetupTestDB(t)
	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	service := NewService(db, mailer)

	invitation := &models.Invitation{
		ID:        "test-invitation-id",
		Code:      "TEST1234",
		Email:     "test@example.com",
		Type:      "server",
		InvitedBy: "inviter-id",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}

	// Note: This test doesn't actually send email, but tests the function structure
	err := service.SendInvitationEmail(invitation)
	// We expect this to fail in test environment since SMTP is not configured
	// but we're testing that the function doesn't panic and handles the error gracefully
	if err == nil {
		t.Log("SendInvitationEmail succeeded (test SMTP might be configured)")
	} else {
		t.Logf("SendInvitationEmail failed as expected in test environment: %v", err)
	}
}
