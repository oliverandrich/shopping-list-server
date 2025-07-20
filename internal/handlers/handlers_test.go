package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"github.com/oliverandrich/shopping-list-server/internal/testutils"
	"gopkg.in/gomail.v2"
)

func setupTestServer(t *testing.T) (*Server, *fiber.App) {
	t.Helper()
	
	// Set up test environment
	testutils.SetupTestConfig(t)
	
	db := testutils.SetupTestDB(t)
	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	server := NewServer(db, []byte("test-secret"), mailer)
	
	app := fiber.New()
	
	// Add routes
	app.Get("/api/v1/health", server.Health)
	app.Post("/api/v1/auth/login", server.RequestLogin)
	app.Post("/api/v1/auth/verify", server.VerifyLogin)
	
	// Protected routes
	protected := app.Group("/api/v1", server.Auth.JWTMiddleware())
	protected.Get("/lists", server.GetLists)
	protected.Post("/lists", server.CreateList)
	protected.Get("/lists/:id", server.GetList)
	protected.Put("/lists/:id", server.UpdateList)
	protected.Delete("/lists/:id", server.DeleteList)
	protected.Get("/lists/:id/members", server.GetListMembers)
	protected.Delete("/lists/:id/members/:userId", server.RemoveListMember)
	protected.Get("/lists/:id/items", server.GetListItems)
	protected.Post("/lists/:id/items", server.CreateListItem)
	protected.Put("/lists/:id/items/:itemId", server.UpdateListItem)
	protected.Post("/lists/:id/items/:itemId/toggle", server.ToggleListItem)
	protected.Delete("/lists/:id/items/:itemId", server.DeleteListItem)
	protected.Post("/invitations", server.CreateInvitation)
	protected.Get("/invitations", server.GetInvitations)
	protected.Delete("/invitations/:id", server.RevokeInvitation)
	
	return server, app
}

func TestNewServer(t *testing.T) {
	db := testutils.SetupTestDB(t)
	mailer := gomail.NewDialer("localhost", 587, "test", "test")
	
	server := NewServer(db, []byte("test-secret"), mailer)
	
	if server == nil {
		t.Fatal("Server should not be nil")
	}
	
	if server.DB != db {
		t.Error("Server DB should match provided database")
	}
	
	if server.Auth == nil {
		t.Error("Auth service should be initialized")
	}
	
	if server.Lists == nil {
		t.Error("Lists service should be initialized")
	}
	
	if server.Invitations == nil {
		t.Error("Invitations service should be initialized")
	}
}

func TestServer_Health(t *testing.T) {
	_, app := setupTestServer(t)
	
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
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
	
	if response["status"] != "healthy" {
		t.Error("Expected status to be 'healthy'")
	}
	
	if response["service"] != "shopping-list-api" {
		t.Error("Expected service to be 'shopping-list-api'")
	}
}

func TestServer_RequestLogin(t *testing.T) {
	_, app := setupTestServer(t)
	
	t.Run("valid login request", func(t *testing.T) {
		loginReq := models.LoginRequest{
			Email: testutils.TestEmailAddress(),
		}
		
		reqBody, err := json.Marshal(loginReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
	
	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestServer_VerifyLogin(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create a test user first
	user := models.User{
		ID:        "test-user-id",
		Email:     testutils.TestEmailAddress(),
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	t.Run("valid verification with existing user", func(t *testing.T) {
		// Create magic link first
		code, err := server.Auth.CreateMagicLink(user.Email)
		if err != nil {
			t.Fatalf("Failed to create magic link: %v", err)
		}
		
		verifyReq := models.VerifyRequest{
			Email: user.Email,
			Code:  code,
		}
		
		reqBody, err := json.Marshal(verifyReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("POST", "/api/v1/auth/verify", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		
		var response models.LoginResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if response.Token == "" {
			t.Error("Expected token to be present")
		}
		
		if response.User.ID != user.ID {
			t.Error("Expected user ID to match")
		}
	})
	
	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/verify", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
	
	t.Run("invalid code", func(t *testing.T) {
		verifyReq := models.VerifyRequest{
			Email: user.Email,
			Code:  "invalid",
		}
		
		reqBody, err := json.Marshal(verifyReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("POST", "/api/v1/auth/verify", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
}

func createAuthenticatedRequest(t *testing.T, server *Server, method, url string, body io.Reader) *http.Request {
	t.Helper()
	
	// Create a test user with unique ID and email
	randomID := fmt.Sprintf("%d", rand.Intn(1000000))
	userID := "auth-user-" + randomID + "-" + strings.ReplaceAll(t.Name(), "/", "-")
	email := "auth-" + randomID + "@example.com"
	user := models.User{
		ID:        userID,
		Email:     email,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	return req
}

func TestServer_GetLists(t *testing.T) {
	server, app := setupTestServer(t)
	
	req := createAuthenticatedRequest(t, server, "GET", "/api/v1/lists", nil)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestServer_CreateList(t *testing.T) {
	server, app := setupTestServer(t)
	
	t.Run("valid list creation", func(t *testing.T) {
		createReq := models.CreateListRequest{
			Name: testutils.TestListName(),
		}
		
		reqBody, err := json.Marshal(createReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := createAuthenticatedRequest(t, server, "POST", "/api/v1/lists", bytes.NewReader(reqBody))
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
	
	t.Run("invalid request body", func(t *testing.T) {
		req := createAuthenticatedRequest(t, server, "POST", "/api/v1/lists", strings.NewReader("invalid json"))
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestServer_GetList(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create a test user and list
	user := models.User{
		ID:        "list-owner-id",
		Email:     "listowner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	list, err := server.Lists.CreateList(user.ID, "Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Generate JWT token for the user
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest("GET", "/api/v1/lists/"+list.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestServer_UpdateList(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create a test user and list
	user := models.User{
		ID:        "update-owner-id",
		Email:     "updateowner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	list, err := server.Lists.CreateList(user.ID, "Original Name")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Generate JWT token for the user
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	t.Run("valid list update", func(t *testing.T) {
		updateReq := models.UpdateListRequest{
			Name: "Updated Name",
		}
		
		reqBody, err := json.Marshal(updateReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("PUT", "/api/v1/lists/"+list.ID, bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
	
	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/lists/"+list.ID, strings.NewReader("invalid json"))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestServer_DeleteList(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create a test user and list
	user := models.User{
		ID:        "delete-owner-id",
		Email:     "deleteowner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	list, err := server.Lists.CreateList(user.ID, "List to Delete")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Generate JWT token for the user
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest("DELETE", "/api/v1/lists/"+list.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 204, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestServer_Unauthorized(t *testing.T) {
	_, app := setupTestServer(t)
	
	// Test protected endpoint without authentication
	req := httptest.NewRequest("GET", "/api/v1/lists", nil)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestServer_GetListMembers(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test users
	owner := models.User{
		ID:        "members-owner-id",
		Email:     "membersowner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	member := models.User{
		ID:        "list-member-id",
		Email:     "listmember@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	
	err := server.DB.Create(&owner).Error
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}
	err = server.DB.Create(&member).Error
	if err != nil {
		t.Fatalf("Failed to create member: %v", err)
	}
	
	// Create a list and add member
	list, err := server.Lists.CreateList(owner.ID, "Members Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	err = server.Lists.AddMemberToList(list.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to add member to list: %v", err)
	}
	
	// Generate JWT token for the owner
	token, err := server.Auth.GenerateJWT(&owner)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest("GET", "/api/v1/lists/"+list.ID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	var members []models.User
	if err := json.Unmarshal(body, &members); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestServer_RemoveListMember(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test users
	owner := models.User{
		ID:        "remove-owner-id",
		Email:     "removeowner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	member := models.User{
		ID:        "remove-member-id",
		Email:     "removemember@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	
	err := server.DB.Create(&owner).Error
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}
	err = server.DB.Create(&member).Error
	if err != nil {
		t.Fatalf("Failed to create member: %v", err)
	}
	
	// Create a list and add member
	list, err := server.Lists.CreateList(owner.ID, "Remove Member Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	err = server.Lists.AddMemberToList(list.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to add member to list: %v", err)
	}
	
	// Generate JWT token for the owner
	token, err := server.Auth.GenerateJWT(&owner)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest("DELETE", "/api/v1/lists/"+list.ID+"/members/"+member.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 204, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestServer_GetListItems(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test user and list
	user := models.User{
		ID:        "items-user-id",
		Email:     "itemsuser@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	list, err := server.Lists.CreateList(user.ID, "Items Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Create some test items
	item1 := models.ShoppingItem{
		ID:        "item1-id",
		ListID:    list.ID,
		Name:      "Test Item 1",
		Completed: false,
		Tags:      "[]",
	}
	item2 := models.ShoppingItem{
		ID:        "item2-id",
		ListID:    list.ID,
		Name:      "Test Item 2",
		Completed: true,
		Tags:      "[\"important\"]",
	}
	
	err = server.DB.Create(&item1).Error
	if err != nil {
		t.Fatalf("Failed to create item1: %v", err)
	}
	err = server.DB.Create(&item2).Error
	if err != nil {
		t.Fatalf("Failed to create item2: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest("GET", "/api/v1/lists/"+list.ID+"/items", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	var items []models.ShoppingItem
	if err := json.Unmarshal(body, &items); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

func TestServer_CreateListItem(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test user and list
	user := models.User{
		ID:        "create-item-user-id",
		Email:     "createitemuser@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	list, err := server.Lists.CreateList(user.ID, "Create Item Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	t.Run("create item with valid data", func(t *testing.T) {
		createReq := models.CreateItemRequest{
			Name: "New Shopping Item",
			Tags: "[\"grocery\", \"urgent\"]",
		}
		
		reqBody, err := json.Marshal(createReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("POST", "/api/v1/lists/"+list.ID+"/items", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		
		var item models.ShoppingItem
		if err := json.Unmarshal(body, &item); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if item.Name != createReq.Name {
			t.Errorf("Expected item name to be '%s', got '%s'", createReq.Name, item.Name)
		}
		
		if item.Completed {
			t.Error("New item should not be completed")
		}
	})
	
	t.Run("create item without tags", func(t *testing.T) {
		createReq := models.CreateItemRequest{
			Name: "Item Without Tags",
		}
		
		reqBody, err := json.Marshal(createReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("POST", "/api/v1/lists/"+list.ID+"/items", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		
		var item models.ShoppingItem
		if err := json.Unmarshal(body, &item); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if item.Tags != "[]" {
			t.Errorf("Expected tags to be '[]', got '%s'", item.Tags)
		}
	})
	
	t.Run("create item with invalid body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/lists/"+list.ID+"/items", strings.NewReader("invalid json"))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestServer_UpdateListItem(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test user and list
	user := models.User{
		ID:        "update-item-user-id",
		Email:     "updateitemuser@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	list, err := server.Lists.CreateList(user.ID, "Update Item Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Create a test item
	item := models.ShoppingItem{
		ID:        "update-item-id",
		ListID:    list.ID,
		Name:      "Original Item Name",
		Completed: false,
		Tags:      "[]",
	}
	err = server.DB.Create(&item).Error
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	t.Run("update item successfully", func(t *testing.T) {
		updateReq := models.CreateItemRequest{
			Name: "Updated Item Name",
			Tags: "[\"updated\"]",
		}
		
		reqBody, err := json.Marshal(updateReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("PUT", "/api/v1/lists/"+list.ID+"/items/"+item.ID, bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		
		var updatedItem models.ShoppingItem
		if err := json.Unmarshal(body, &updatedItem); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		
		if updatedItem.Name != updateReq.Name {
			t.Errorf("Expected item name to be '%s', got '%s'", updateReq.Name, updatedItem.Name)
		}
		
		if updatedItem.Tags != updateReq.Tags {
			t.Errorf("Expected tags to be '%s', got '%s'", updateReq.Tags, updatedItem.Tags)
		}
	})
	
	t.Run("update non-existent item", func(t *testing.T) {
		updateReq := models.CreateItemRequest{
			Name: "Updated Name",
		}
		
		reqBody, err := json.Marshal(updateReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("PUT", "/api/v1/lists/"+list.ID+"/items/non-existent", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
}

func TestServer_ToggleListItem(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test user and list
	user := models.User{
		ID:        "toggle-item-user-id",
		Email:     "toggleitemuser@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	list, err := server.Lists.CreateList(user.ID, "Toggle Item Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Create a test item
	item := models.ShoppingItem{
		ID:        "toggle-item-id",
		ListID:    list.ID,
		Name:      "Item to Toggle",
		Completed: false,
		Tags:      "[]",
	}
	err = server.DB.Create(&item).Error
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest("POST", "/api/v1/lists/"+list.ID+"/items/"+item.ID+"/toggle", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	var toggledItem models.ShoppingItem
	if err := json.Unmarshal(body, &toggledItem); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	if !toggledItem.Completed {
		t.Error("Item should be completed after toggle")
	}
}

func TestServer_DeleteListItem(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test user and list
	user := models.User{
		ID:        "delete-item-user-id",
		Email:     "deleteitemuser@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	list, err := server.Lists.CreateList(user.ID, "Delete Item Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Create a test item
	item := models.ShoppingItem{
		ID:        "delete-item-id",
		ListID:    list.ID,
		Name:      "Item to Delete",
		Completed: false,
		Tags:      "[]",
	}
	err = server.DB.Create(&item).Error
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	t.Run("delete existing item", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/lists/"+list.ID+"/items/"+item.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 204, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
	
	t.Run("delete non-existent item", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/lists/"+list.ID+"/items/non-existent", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
}

func TestServer_CreateInvitation(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test user
	user := models.User{
		ID:        "invitation-user-id",
		Email:     "invitationuser@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	t.Run("create server invitation", func(t *testing.T) {
		createReq := models.CreateInvitationRequest{
			Email: "newinvitee@example.com",
			Type:  "server",
		}
		
		reqBody, err := json.Marshal(createReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("POST", "/api/v1/invitations", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
	
	t.Run("create list invitation", func(t *testing.T) {
		// Create a list first
		list, err := server.Lists.CreateList(user.ID, "Invitation Test List")
		if err != nil {
			t.Fatalf("Failed to create test list: %v", err)
		}
		
		listID := list.ID
		createReq := models.CreateInvitationRequest{
			Email:  "listinvitee@example.com",
			Type:   "list",
			ListID: &listID,
		}
		
		reqBody, err := json.Marshal(createReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("POST", "/api/v1/invitations", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
	
	t.Run("create invitation with invalid body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/invitations", strings.NewReader("invalid json"))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestServer_GetInvitations(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test user
	user := models.User{
		ID:        "get-invitations-user-id",
		Email:     "getinvitationsuser@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	// Create some invitations
	invitation1 := models.Invitation{
		ID:        "inv1-id",
		Code:      "INV12345",
		Email:     "invitee1@example.com",
		Type:      "server",
		InvitedBy: user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}
	invitation2 := models.Invitation{
		ID:        "inv2-id",
		Code:      "INV67890",
		Email:     "invitee2@example.com",
		Type:      "server",
		InvitedBy: user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}
	
	err = server.DB.Create(&invitation1).Error
	if err != nil {
		t.Fatalf("Failed to create invitation1: %v", err)
	}
	err = server.DB.Create(&invitation2).Error
	if err != nil {
		t.Fatalf("Failed to create invitation2: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest("GET", "/api/v1/invitations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	var invitations []models.Invitation
	if err := json.Unmarshal(body, &invitations); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	if len(invitations) != 2 {
		t.Errorf("Expected 2 invitations, got %d", len(invitations))
	}
}

func TestServer_RevokeInvitation(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create test user
	user := models.User{
		ID:        "revoke-invitation-user-id",
		Email:     "revokeinvitationuser@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := server.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	// Create an invitation
	invitation := models.Invitation{
		ID:        "revoke-inv-id",
		Code:      "REV12345",
		Email:     "revokee@example.com",
		Type:      "server",
		InvitedBy: user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}
	
	err = server.DB.Create(&invitation).Error
	if err != nil {
		t.Fatalf("Failed to create invitation: %v", err)
	}
	
	// Generate JWT token
	token, err := server.Auth.GenerateJWT(&user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}
	
	req := httptest.NewRequest("DELETE", "/api/v1/invitations/"+invitation.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	
	if resp.StatusCode != fiber.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 204, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	// Verify invitation was deleted
	var count int64
	err = server.DB.Model(&models.Invitation{}).Where("id = ?", invitation.ID).Count(&count).Error
	if err != nil {
		t.Fatal("Failed to count invitations")
	}
	
	if count != 0 {
		t.Error("Invitation should be deleted")
	}
}

func TestServer_ItemAccessControl(t *testing.T) {
	server, app := setupTestServer(t)
	
	// Create two users
	user1 := models.User{
		ID:        "user1-access-id",
		Email:     "user1access@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	user2 := models.User{
		ID:        "user2-access-id",
		Email:     "user2access@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	
	err := server.DB.Create(&user1).Error
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}
	err = server.DB.Create(&user2).Error
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}
	
	// Create a list for user1
	list, err := server.Lists.CreateList(user1.ID, "User1 Private List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}
	
	// Generate JWT tokens
	token1, err := server.Auth.GenerateJWT(&user1)
	if err != nil {
		t.Fatalf("Failed to generate JWT for user1: %v", err)
	}
	token2, err := server.Auth.GenerateJWT(&user2)
	if err != nil {
		t.Fatalf("Failed to generate JWT for user2: %v", err)
	}
	
	t.Run("user2 cannot access user1's list items", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/lists/"+list.ID+"/items", nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusForbidden {
			t.Errorf("Expected status 403, got %d", resp.StatusCode)
		}
	})
	
	t.Run("user2 cannot create items in user1's list", func(t *testing.T) {
		createReq := models.CreateItemRequest{
			Name: "Unauthorized Item",
		}
		
		reqBody, err := json.Marshal(createReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		req := httptest.NewRequest("POST", "/api/v1/lists/"+list.ID+"/items", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token2)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusForbidden {
			t.Errorf("Expected status 403, got %d", resp.StatusCode)
		}
	})
	
	t.Run("user1 can access their own list items", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/lists/"+list.ID+"/items", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}