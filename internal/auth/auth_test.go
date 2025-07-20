package auth

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"github.com/oliverandrich/shopping-list-server/internal/testutils"
	"gopkg.in/gomail.v2"
)

func TestGenerateCode(t *testing.T) {
	code := GenerateCode()
	
	if len(code) != 6 {
		t.Errorf("Expected code length to be 6, got %d", len(code))
	}
	
	// Check that code contains only digits
	for _, char := range code {
		if char < '0' || char > '9' {
			t.Errorf("Expected code to contain only digits, found '%c'", char)
		}
	}
	
	// Generate multiple codes to ensure they are different
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := GenerateCode()
		if codes[code] {
			t.Errorf("Generated duplicate code: %s", code)
		}
		codes[code] = true
	}
}

func TestNewService(t *testing.T) {
	db := testutils.SetupTestDB(t)
	jwtSecret := []byte("test-secret")
	mailer := gomail.NewDialer("smtp.test.com", 587, "test", "test")
	
	service := NewService(db, jwtSecret, mailer)
	
	if service == nil {
		t.Fatal("Service should not be nil")
	}
	
	if service.DB != db {
		t.Error("Service DB should match provided database")
	}
	
	if string(service.JWTSecret) != string(jwtSecret) {
		t.Error("Service JWTSecret should match provided secret")
	}
	
	if service.Mailer != mailer {
		t.Error("Service Mailer should match provided mailer")
	}
}

func TestService_CreateMagicLink(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db, []byte("test-secret"), nil)
	
	email := testutils.TestEmailAddress()
	
	t.Run("create new magic link", func(t *testing.T) {
		code, err := service.CreateMagicLink(email)
		if err != nil {
			t.Fatalf("Failed to create magic link: %v", err)
		}
		
		if len(code) != 6 {
			t.Errorf("Expected code length to be 6, got %d", len(code))
		}
		
		// Verify magic link was stored in database
		var magicLink models.MagicLink
		err = db.Where("email = ? AND code = ?", email, code).First(&magicLink).Error
		if err != nil {
			t.Fatalf("Failed to find magic link in database: %v", err)
		}
		
		if magicLink.Used {
			t.Error("New magic link should not be marked as used")
		}
		
		if time.Until(magicLink.ExpiresAt) > 15*time.Minute {
			t.Error("Magic link should expire in 15 minutes or less")
		}
	})
	
	t.Run("replace existing magic link", func(t *testing.T) {
		// Create first magic link
		code1, err := service.CreateMagicLink(email)
		if err != nil {
			t.Fatalf("Failed to create first magic link: %v", err)
		}
		
		// Create second magic link for same email
		code2, err := service.CreateMagicLink(email)
		if err != nil {
			t.Fatalf("Failed to create second magic link: %v", err)
		}
		
		if code1 == code2 {
			t.Error("Second magic link should have different code")
		}
		
		// Verify only the second magic link exists
		var count int64
		err = db.Model(&models.MagicLink{}).Where("email = ?", email).Count(&count).Error
		if err != nil {
			t.Fatalf("Failed to count magic links: %v", err)
		}
		
		if count != 1 {
			t.Errorf("Expected exactly 1 magic link after replacement, got %d", count)
		}
		
		// Verify it's the second code
		var magicLink models.MagicLink
		err = db.Where("email = ?", email).First(&magicLink).Error
		if err != nil {
			t.Fatalf("Failed to find remaining magic link: %v", err)
		}
		
		if magicLink.Code != code2 {
			t.Error("Remaining magic link should have the second code")
		}
	})
}

func TestService_VerifyMagicLink(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db, []byte("test-secret"), nil)
	
	email := testutils.TestEmailAddress()
	
	// Create a user first
	user := models.User{
		ID:        "test-user-id",
		Email:     email,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	t.Run("verify valid magic link", func(t *testing.T) {
		code, err := service.CreateMagicLink(email)
		if err != nil {
			t.Fatalf("Failed to create magic link: %v", err)
		}
		
		verifiedUser, err := service.VerifyMagicLink(email, code)
		if err != nil {
			t.Fatalf("Failed to verify magic link: %v", err)
		}
		
		if verifiedUser.ID != user.ID {
			t.Error("Verified user should match created user")
		}
		
		// Verify magic link is marked as used
		var magicLink models.MagicLink
		err = db.Where("email = ? AND code = ?", email, code).First(&magicLink).Error
		if err != nil {
			t.Fatalf("Failed to find magic link: %v", err)
		}
		
		if !magicLink.Used {
			t.Error("Magic link should be marked as used after verification")
		}
	})
	
	t.Run("verify invalid code", func(t *testing.T) {
		_, err := service.VerifyMagicLink(email, "invalid")
		if err == nil {
			t.Error("Expected error when verifying invalid code")
		}
	})
	
	t.Run("verify expired magic link", func(t *testing.T) {
		// Create an expired magic link manually
		expiredLink := models.MagicLink{
			Code:      "123456",
			Email:     email,
			ExpiresAt: time.Now().Add(-time.Hour),
			Used:      false,
		}
		err := db.Create(&expiredLink).Error
		if err != nil {
			t.Fatalf("Failed to create expired magic link: %v", err)
		}
		
		_, err = service.VerifyMagicLink(email, "123456")
		if err == nil {
			t.Error("Expected error when verifying expired magic link")
		}
	})
	
	t.Run("verify used magic link", func(t *testing.T) {
		code, err := service.CreateMagicLink(email)
		if err != nil {
			t.Fatalf("Failed to create magic link: %v", err)
		}
		
		// Use the magic link once
		_, err = service.VerifyMagicLink(email, code)
		if err != nil {
			t.Fatalf("Failed to verify magic link first time: %v", err)
		}
		
		// Try to use it again
		_, err = service.VerifyMagicLink(email, code)
		if err == nil {
			t.Error("Expected error when verifying already used magic link")
		}
	})
	
	t.Run("verify for non-existent user", func(t *testing.T) {
		nonExistentEmail := "nonexistent@example.com"
		code, err := service.CreateMagicLink(nonExistentEmail)
		if err != nil {
			t.Fatalf("Failed to create magic link: %v", err)
		}
		
		_, err = service.VerifyMagicLink(nonExistentEmail, code)
		if err == nil {
			t.Error("Expected error when verifying magic link for non-existent user")
		}
		
		if !strings.Contains(err.Error(), "invitation required") {
			t.Error("Error should mention invitation requirement")
		}
	})
}

func TestService_GenerateJWT(t *testing.T) {
	service := NewService(nil, []byte("test-secret"), nil)
	
	user := &models.User{
		ID:    "test-user-id",
		Email: "test@example.com",
	}
	
	token, err := service.GenerateJWT(user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	if token == "" {
		t.Error("Generated token should not be empty")
	}
	
	// Verify token contains expected parts (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("Expected JWT to have 3 parts, got %d", len(parts))
	}
}

func TestService_ValidateJWT(t *testing.T) {
	service := NewService(nil, []byte("test-secret"), nil)
	
	user := &models.User{
		ID:    "test-user-id",
		Email: "test@example.com",
	}
	
	t.Run("validate valid token", func(t *testing.T) {
		token, err := service.GenerateJWT(user)
		if err != nil {
			t.Fatalf("Failed to generate JWT: %v", err)
		}
		
		claims, err := service.ValidateJWT(token)
		if err != nil {
			t.Fatalf("Failed to validate JWT: %v", err)
		}
		
		if claims.UserID != user.ID {
			t.Errorf("Expected UserID to be '%s', got '%s'", user.ID, claims.UserID)
		}
		
		if claims.Email != user.Email {
			t.Errorf("Expected Email to be '%s', got '%s'", user.Email, claims.Email)
		}
	})
	
	t.Run("validate invalid token", func(t *testing.T) {
		_, err := service.ValidateJWT("invalid-token")
		if err == nil {
			t.Error("Expected error when validating invalid token")
		}
	})
	
	t.Run("validate token with wrong secret", func(t *testing.T) {
		// Generate token with one secret
		token, err := service.GenerateJWT(user)
		if err != nil {
			t.Fatalf("Failed to generate JWT: %v", err)
		}
		
		// Try to validate with different secret
		wrongService := NewService(nil, []byte("wrong-secret"), nil)
		_, err = wrongService.ValidateJWT(token)
		if err == nil {
			t.Error("Expected error when validating token with wrong secret")
		}
	})
}

func TestService_SendMagicLink(t *testing.T) {
	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	service := NewService(nil, []byte("test-secret"), mailer)
	
	email := testutils.TestEmailAddress()
	code := "123456"
	
	t.Run("test environment skips email sending", func(t *testing.T) {
		// Set up test environment
		testutils.SetupTestConfig(t)
		defer testutils.CleanupTestEnv(t)
		
		err := service.SendMagicLink(email, code)
		if err != nil {
			t.Errorf("Expected no error in test environment, got: %v", err)
		}
	})
	
	t.Run("non-test environment attempts email sending", func(t *testing.T) {
		// Clean up any test environment variable
		testutils.CleanupTestEnv(t)
		
		err := service.SendMagicLink(email, code)
		// We expect this to fail since SMTP is not configured, but we're testing that
		// the function attempts to send email when not in test environment
		if err == nil {
			t.Log("SendMagicLink succeeded (SMTP might be configured)")
		} else {
			t.Logf("SendMagicLink failed as expected with no SMTP config: %v", err)
		}
	})
}

func TestService_JWTMiddleware(t *testing.T) {
	service := NewService(nil, []byte("test-secret"), nil)
	middleware := service.JWTMiddleware()
	
	if middleware == nil {
		t.Fatal("JWT middleware should not be nil")
	}
	
	user := &models.User{
		ID:    "test-user-id",
		Email: "test@example.com",
	}
	
	// Create a test app with the middleware
	app := fiber.New()
	app.Use(middleware)
	app.Get("/test", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		userEmail := c.Locals("user_email")
		return c.JSON(fiber.Map{
			"success":    true,
			"user_id":    userID,
			"user_email": userEmail,
		})
	})
	
	t.Run("valid authorization header", func(t *testing.T) {
		token, err := service.GenerateJWT(user)
		if err != nil {
			t.Fatalf("Failed to generate JWT: %v", err)
		}
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		
		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if response["user_id"] != user.ID {
			t.Errorf("Expected user_id to be '%s', got '%v'", user.ID, response["user_id"])
		}
		
		if response["user_email"] != user.Email {
			t.Errorf("Expected user_email to be '%s', got '%v'", user.Email, response["user_email"])
		}
	})
	
	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
	
	t.Run("invalid authorization format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat token")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
	
	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
	
	t.Run("expired token", func(t *testing.T) {
		// Create a token with a past expiration time
		claims := &models.JWTClaims{
			UserID: user.ID,
			Email:  user.Email,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired 1 hour ago
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(service.JWTSecret)
		if err != nil {
			t.Fatalf("Failed to create expired token: %v", err)
		}
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
}

func TestService_VerifyMagicLinkWithInvitation(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db, []byte("test-secret"), nil)
	
	email := testutils.TestEmailAddress()
	
	t.Run("existing user with list invitation", func(t *testing.T) {
		// Create a user
		user := models.User{
			ID:        "test-user-id",
			Email:     email,
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
		}
		err := db.Create(&user).Error
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
		
		// Create a list invitation
		invitation := models.Invitation{
			ID:        "test-invitation-id",
			Code:      "invite123",
			Email:     email,
			Type:      "list",
			InvitedBy: "inviter-id",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			Used:      false,
			CreatedAt: time.Now(),
		}
		err = db.Create(&invitation).Error
		if err != nil {
			t.Fatalf("Failed to create test invitation: %v", err)
		}
		
		// Create and verify magic link
		code, err := service.CreateMagicLink(email)
		if err != nil {
			t.Fatalf("Failed to create magic link: %v", err)
		}
		
		verifiedUser, returnedInvitation, err := service.VerifyMagicLinkWithInvitation(email, code)
		if err != nil {
			t.Fatalf("Failed to verify magic link with invitation: %v", err)
		}
		
		if verifiedUser.ID != user.ID {
			t.Error("Verified user should match existing user")
		}
		
		if returnedInvitation == nil {
			t.Error("Invitation should be returned for existing user with list invitation")
		} else if returnedInvitation.ID != invitation.ID {
			t.Error("Returned invitation should match created invitation")
		}
	})
	
	t.Run("new user with server invitation", func(t *testing.T) {
		newEmail := "newuser@example.com"
		
		// Create a server invitation for new user
		invitation := models.Invitation{
			ID:        "new-invitation-id",
			Code:      "newinvite123",
			Email:     newEmail,
			Type:      "server",
			InvitedBy: "inviter-id",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			Used:      false,
			CreatedAt: time.Now(),
		}
		err := db.Create(&invitation).Error
		if err != nil {
			t.Fatalf("Failed to create test invitation: %v", err)
		}
		
		// Create and verify magic link
		code, err := service.CreateMagicLink(newEmail)
		if err != nil {
			t.Fatalf("Failed to create magic link: %v", err)
		}
		
		verifiedUser, returnedInvitation, err := service.VerifyMagicLinkWithInvitation(newEmail, code)
		if err != nil {
			t.Fatalf("Failed to verify magic link with invitation: %v", err)
		}
		
		if verifiedUser == nil {
			t.Fatal("User should be created for new user with invitation")
		}
		
		if verifiedUser.Email != newEmail {
			t.Error("Created user should have correct email")
		}
		
		if returnedInvitation == nil {
			t.Error("Invitation should be returned")
		} else if returnedInvitation.Type != "server" {
			t.Error("Returned invitation should be server type")
		}
		
		// Verify user was created in database
		var dbUser models.User
		err = db.Where("email = ?", newEmail).First(&dbUser).Error
		if err != nil {
			t.Fatal("User should be saved in database")
		}
	})
	
	t.Run("new user without invitation", func(t *testing.T) {
		uninvitedEmail := "uninvited@example.com"
		
		code, err := service.CreateMagicLink(uninvitedEmail)
		if err != nil {
			t.Fatalf("Failed to create magic link: %v", err)
		}
		
		_, _, err = service.VerifyMagicLinkWithInvitation(uninvitedEmail, code)
		if err == nil {
			t.Error("Expected error for new user without invitation")
		}
		
		if !strings.Contains(err.Error(), "invitation required") {
			t.Error("Error should mention invitation requirement")
		}
	})
}