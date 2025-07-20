package setup

import (
	"testing"
	"time"

	"github.com/oliverandrich/shopping-list-server/internal/models"
	"github.com/oliverandrich/shopping-list-server/internal/testutils"
)

func TestNewService(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)
	
	if service == nil {
		t.Fatal("Service should not be nil")
	}
	
	if service.DB != db {
		t.Error("Service DB should match provided database")
	}
}

func TestService_IsSystemSetup(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)
	
	t.Run("system not setup initially", func(t *testing.T) {
		isSetup, err := service.IsSystemSetup()
		if err != nil {
			t.Fatalf("Failed to check system setup: %v", err)
		}
		
		if isSetup {
			t.Error("System should not be setup initially")
		}
	})
	
	t.Run("system setup after creating settings", func(t *testing.T) {
		// Create system settings manually
		settings := models.SystemSettings{
			ID:           "system",
			IsSetup:      true,
			SetupAt:      time.Now(),
			InitialAdmin: "admin@example.com",
		}
		err := db.Create(&settings).Error
		if err != nil {
			t.Fatalf("Failed to create system settings: %v", err)
		}
		
		isSetup, err := service.IsSystemSetup()
		if err != nil {
			t.Fatalf("Failed to check system setup: %v", err)
		}
		
		if !isSetup {
			t.Error("System should be setup after creating settings")
		}
	})
}

func TestService_SetupSystem(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)
	
	t.Run("setup system with valid email", func(t *testing.T) {
		email := testutils.TestEmailAddress()
		
		user, err := service.SetupSystem(email)
		if err != nil {
			t.Fatalf("Failed to setup system: %v", err)
		}
		
		if user == nil {
			t.Fatal("User should not be nil")
		}
		
		if user.Email != email {
			t.Errorf("Expected user email to be '%s', got '%s'", email, user.Email)
		}
		
		if user.ID == "" {
			t.Error("User ID should not be empty")
		}
		
		if user.JoinedAt.IsZero() {
			t.Error("User JoinedAt should not be zero")
		}
		
		if user.CreatedAt.IsZero() {
			t.Error("User CreatedAt should not be zero")
		}
		
		// Verify user was saved to database
		var dbUser models.User
		err = db.Where("email = ?", email).First(&dbUser).Error
		if err != nil {
			t.Fatal("User should be saved in database")
		}
		
		// Verify system settings were created
		var settings models.SystemSettings
		err = db.Where("id = ?", "system").First(&settings).Error
		if err != nil {
			t.Fatal("System settings should be created")
		}
		
		if !settings.IsSetup {
			t.Error("System settings should be marked as setup")
		}
		
		if settings.InitialAdmin != user.ID {
			t.Errorf("Expected initial admin to be '%s', got '%s'", user.ID, settings.InitialAdmin)
		}
		
		// Verify default list was created
		var lists []models.ShoppingList
		err = db.Where("owner_id = ?", user.ID).Find(&lists).Error
		if err != nil {
			t.Fatal("Failed to find lists for user")
		}
		
		if len(lists) != 1 {
			t.Errorf("Expected 1 default list, got %d", len(lists))
		}
		
		if lists[0].Name != "My Shopping List" {
			t.Errorf("Expected default list name to be 'My Shopping List', got '%s'", lists[0].Name)
		}
		
		// Verify user is owner of the list
		var member models.ListMember
		err = db.Where("list_id = ? AND user_id = ?", lists[0].ID, user.ID).First(&member).Error
		if err != nil {
			t.Fatal("User should be member of default list")
		}
		
		if member.Role != "owner" {
			t.Errorf("Expected user role to be 'owner', got '%s'", member.Role)
		}
	})
	
	t.Run("setup system with empty email", func(t *testing.T) {
		_, err := service.SetupSystem("")
		if err == nil {
			t.Error("Expected error when setting up system with empty email")
		}
	})
	
	t.Run("setup system twice", func(t *testing.T) {
		// Create a fresh database service
		freshDB := testutils.SetupTestDB(t)
		freshService := NewService(freshDB)
		
		email := "second@example.com"
		
		// First setup
		_, err := freshService.SetupSystem(email)
		if err != nil {
			t.Fatalf("Failed to setup system first time: %v", err)
		}
		
		// Second setup should fail
		_, err = freshService.SetupSystem("different@example.com")
		if err == nil {
			t.Error("Expected error when setting up system twice")
		}
	})
	
	t.Run("setup system with invalid email format", func(t *testing.T) {
		_, err := service.SetupSystem("invalid-email")
		if err == nil {
			t.Error("Expected error when setting up system with invalid email")
		}
	})
}

func TestService_MigrateExistingData(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)
	
	t.Run("migrate when no existing data", func(t *testing.T) {
		err := service.MigrateExistingData()
		if err != nil {
			t.Fatalf("Failed to migrate existing data: %v", err)
		}
		
		// Should not create system settings when no users exist
		var count int64
		err = db.Model(&models.SystemSettings{}).Count(&count).Error
		if err != nil {
			t.Fatal("Failed to count system settings")
		}
		
		if count != 0 {
			t.Error("Should not create system settings when no users exist")
		}
	})
	
	t.Run("migrate with existing user", func(t *testing.T) {
		// Clear database first to start fresh
		db.Exec("DELETE FROM system_settings")
		db.Exec("DELETE FROM users")
		
		// Create an existing user (simulating old data)
		existingUser := models.User{
			ID:        "existing-user-id",
			Email:     "existing@example.com",
			JoinedAt:  time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
			CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
		}
		err := db.Create(&existingUser).Error
		if err != nil {
			t.Fatalf("Failed to create existing user: %v", err)
		}
		
		// Migrate existing data
		err = service.MigrateExistingData()
		if err != nil {
			t.Fatalf("Failed to migrate existing data: %v", err)
		}
		
		// Verify system settings were created
		var settings models.SystemSettings
		err = db.Where("id = ?", "system").First(&settings).Error
		if err != nil {
			t.Fatal("System settings should be created during migration")
		}
		
		if !settings.IsSetup {
			t.Error("System should be marked as setup after migration")
		}
		
		if settings.InitialAdmin != existingUser.ID {
			t.Error("Initial admin should be set to oldest user's ID")
		}
		
		// Verify system is now considered setup
		isSetup, err := service.IsSystemSetup()
		if err != nil {
			t.Fatal("Failed to check if system is setup")
		}
		
		if !isSetup {
			t.Error("System should be setup after migration")
		}
	})
	
	t.Run("migrate with multiple existing users", func(t *testing.T) {
		// Clear database first
		db.Exec("DELETE FROM system_settings")
		db.Exec("DELETE FROM shopping_lists")
		db.Exec("DELETE FROM users")
		
		// Create multiple users with different creation times
		user1 := models.User{
			ID:        "user1-id",
			Email:     "user1@example.com",
			JoinedAt:  time.Now().Add(-20 * 24 * time.Hour), // 20 days ago
			CreatedAt: time.Now().Add(-20 * 24 * time.Hour),
		}
		user2 := models.User{
			ID:        "user2-id",
			Email:     "user2@example.com",
			JoinedAt:  time.Now().Add(-40 * 24 * time.Hour), // 40 days ago (oldest)
			CreatedAt: time.Now().Add(-40 * 24 * time.Hour),
		}
		user3 := models.User{
			ID:        "user3-id",
			Email:     "user3@example.com",
			JoinedAt:  time.Now().Add(-10 * 24 * time.Hour), // 10 days ago
			CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
		}
		
		err := db.Create(&user1).Error
		if err != nil {
			t.Fatalf("Failed to create user1: %v", err)
		}
		err = db.Create(&user2).Error
		if err != nil {
			t.Fatalf("Failed to create user2: %v", err)
		}
		err = db.Create(&user3).Error
		if err != nil {
			t.Fatalf("Failed to create user3: %v", err)
		}
		
		// Migrate existing data
		err = service.MigrateExistingData()
		if err != nil {
			t.Fatalf("Failed to migrate existing data: %v", err)
		}
		
		// Verify system settings use oldest user as initial admin
		var settings models.SystemSettings
		err = db.Where("id = ?", "system").First(&settings).Error
		if err != nil {
			t.Fatal("System settings should be created during migration")
		}
		
		if settings.InitialAdmin != user2.ID {
			t.Errorf("Expected initial admin to be oldest user (%s), got %s", user2.ID, settings.InitialAdmin)
		}
	})
}


func TestService_Integration(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)
	
	t.Run("complete setup workflow", func(t *testing.T) {
		email := "integration@example.com"
		
		// 1. Check that system is not setup
		isSetup, err := service.IsSystemSetup()
		if err != nil {
			t.Fatalf("Failed to check system setup: %v", err)
		}
		if isSetup {
			t.Error("System should not be setup initially")
		}
		
		// 2. Setup the system
		user, err := service.SetupSystem(email)
		if err != nil {
			t.Fatalf("Failed to setup system: %v", err)
		}
		if user.Email != email {
			t.Error("Setup should create user with correct email")
		}
		
		// 3. Check that system is now setup
		isSetup, err = service.IsSystemSetup()
		if err != nil {
			t.Fatalf("Failed to check system setup after setup: %v", err)
		}
		if !isSetup {
			t.Error("System should be setup after setup")
		}
		
		// 4. Try to setup again (should fail)
		_, err = service.SetupSystem("another@example.com")
		if err == nil {
			t.Error("Should not be able to setup system twice")
		}
		
		// 5. Verify default list exists
		lists := []models.ShoppingList{}
		err = db.Where("owner_id = ?", user.ID).Find(&lists).Error
		if err != nil {
			t.Fatal("Failed to find lists for user")
		}
		if len(lists) != 1 {
			t.Errorf("Expected 1 default list, got %d", len(lists))
		}
	})
}