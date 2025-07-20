// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

package models

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTClaims_Structure(t *testing.T) {
	claims := &JWTClaims{
		UserID: "test-user-id",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	if claims.UserID == "" {
		t.Error("JWTClaims UserID should not be empty")
	}

	if claims.Email == "" {
		t.Error("JWTClaims Email should not be empty")
	}

	if claims.ExpiresAt == nil {
		t.Error("JWTClaims ExpiresAt should not be nil")
	}

	if claims.IssuedAt == nil {
		t.Error("JWTClaims IssuedAt should not be nil")
	}
}

func TestUser_Validation(t *testing.T) {
	user := User{
		ID:        "test-id",
		Email:     "test@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	if user.ID == "" {
		t.Error("User ID should not be empty")
	}

	if user.Email == "" {
		t.Error("User Email should not be empty")
	}

	if user.JoinedAt.IsZero() {
		t.Error("User JoinedAt should not be zero")
	}

	if user.CreatedAt.IsZero() {
		t.Error("User CreatedAt should not be zero")
	}
}

func TestShoppingList_Validation(t *testing.T) {
	list := ShoppingList{
		ID:        "test-list-id",
		Name:      "Test List",
		OwnerID:   "test-owner-id",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if list.ID == "" {
		t.Error("ShoppingList ID should not be empty")
	}

	if list.Name == "" {
		t.Error("ShoppingList Name should not be empty")
	}

	if list.OwnerID == "" {
		t.Error("ShoppingList OwnerID should not be empty")
	}
}

func TestShoppingItem_Validation(t *testing.T) {
	item := ShoppingItem{
		ID:        "test-item-id",
		ListID:    "test-list-id",
		Name:      "Test Item",
		Completed: false,
		Tags:      "[]",
		CreatedAt: time.Now(),
	}

	if item.ID == "" {
		t.Error("ShoppingItem ID should not be empty")
	}

	if item.ListID == "" {
		t.Error("ShoppingItem ListID should not be empty")
	}

	if item.Name == "" {
		t.Error("ShoppingItem Name should not be empty")
	}

	if item.Tags == "" {
		t.Error("ShoppingItem Tags should not be empty")
	}
}

func TestMagicLink_Validation(t *testing.T) {
	magicLink := MagicLink{
		Code:      "123456",
		Email:     "test@example.com",
		Used:      false,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if magicLink.Code == "" {
		t.Error("MagicLink Code should not be empty")
	}

	if magicLink.Email == "" {
		t.Error("MagicLink Email should not be empty")
	}

	if magicLink.ExpiresAt.IsZero() {
		t.Error("MagicLink ExpiresAt should not be zero")
	}
}

func TestInvitation_Validation(t *testing.T) {
	invitation := Invitation{
		ID:        "test-invitation-id",
		Code:      "invite123",
		Email:     "test@example.com",
		Type:      "server",
		InvitedBy: "test-inviter-id",
		Used:      false,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if invitation.ID == "" {
		t.Error("Invitation ID should not be empty")
	}

	if invitation.Code == "" {
		t.Error("Invitation Code should not be empty")
	}

	if invitation.Email == "" {
		t.Error("Invitation Email should not be empty")
	}

	if invitation.Type == "" {
		t.Error("Invitation Type should not be empty")
	}

	if invitation.InvitedBy == "" {
		t.Error("Invitation InvitedBy should not be empty")
	}
}

func TestListMember_Validation(t *testing.T) {
	member := ListMember{
		ListID:   "test-list-id",
		UserID:   "test-user-id",
		JoinedAt: time.Now(),
	}

	if member.ListID == "" {
		t.Error("ListMember ListID should not be empty")
	}

	if member.UserID == "" {
		t.Error("ListMember UserID should not be empty")
	}

	if member.JoinedAt.IsZero() {
		t.Error("ListMember JoinedAt should not be zero")
	}
}

func TestSystemSettings_Validation(t *testing.T) {
	settings := SystemSettings{
		ID:           "test-settings-id",
		IsSetup:      true,
		SetupAt:      time.Now(),
		InitialAdmin: "admin@example.com",
	}

	if settings.ID == "" {
		t.Error("SystemSettings ID should not be empty")
	}

	if !settings.IsSetup {
		t.Error("SystemSettings IsSetup should be true in this test")
	}

	if settings.InitialAdmin == "" {
		t.Error("SystemSettings InitialAdmin should not be empty")
	}

	if settings.SetupAt.IsZero() {
		t.Error("SystemSettings SetupAt should not be zero")
	}
}

func TestRequestModels_Validation(t *testing.T) {
	t.Run("LoginRequest", func(t *testing.T) {
		req := LoginRequest{Email: "test@example.com"}
		if req.Email == "" {
			t.Error("LoginRequest Email should not be empty")
		}
	})

	t.Run("VerifyRequest", func(t *testing.T) {
		req := VerifyRequest{
			Email: "test@example.com",
			Code:  "123456",
		}
		if req.Email == "" {
			t.Error("VerifyRequest Email should not be empty")
		}
		if req.Code == "" {
			t.Error("VerifyRequest Code should not be empty")
		}
	})

	t.Run("CreateListRequest", func(t *testing.T) {
		req := CreateListRequest{Name: "Test List"}
		if req.Name == "" {
			t.Error("CreateListRequest Name should not be empty")
		}
	})

	t.Run("UpdateListRequest", func(t *testing.T) {
		req := UpdateListRequest{Name: "Updated List"}
		if req.Name == "" {
			t.Error("UpdateListRequest Name should not be empty")
		}
	})

	t.Run("CreateItemRequest", func(t *testing.T) {
		req := CreateItemRequest{
			Name: "Test Item",
			Tags: "[]",
		}
		if req.Name == "" {
			t.Error("CreateItemRequest Name should not be empty")
		}
	})

	t.Run("CreateInvitationRequest", func(t *testing.T) {
		req := CreateInvitationRequest{
			Email: "test@example.com",
			Type:  "server",
		}
		if req.Email == "" {
			t.Error("CreateInvitationRequest Email should not be empty")
		}
		if req.Type == "" {
			t.Error("CreateInvitationRequest Type should not be empty")
		}
	})
}

func TestResponseModels_Validation(t *testing.T) {
	t.Run("LoginResponse", func(t *testing.T) {
		resp := LoginResponse{
			Token: "test-token",
			User: User{
				ID:    "test-id",
				Email: "test@example.com",
			},
		}
		if resp.Token == "" {
			t.Error("LoginResponse Token should not be empty")
		}
		if resp.User.ID == "" {
			t.Error("LoginResponse User ID should not be empty")
		}
	})
}
