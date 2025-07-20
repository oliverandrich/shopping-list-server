package testutils

import (
	"os"
	"testing"

	"github.com/oliverandrich/shopping-list-server/internal/config"
	"github.com/oliverandrich/shopping-list-server/internal/db"
	"gorm.io/gorm"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	
	return database
}

// SetupTestConfig sets up test environment variables and returns a config
func SetupTestConfig(t *testing.T) *config.Config {
	t.Helper()
	
	// Set test environment variables
	os.Setenv("GO_ENV", "test")
	os.Setenv("SMTP_HOST", "test.smtp.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USER", "test@example.com")
	os.Setenv("SMTP_PASS", "testpass")
	os.Setenv("SMTP_FROM", "test@example.com")
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
	os.Setenv("PORT", ":8080")
	os.Setenv("DB_PATH", ":memory:")
	
	return config.Load()
}

// CleanupTestEnv cleans up test environment variables
func CleanupTestEnv(t *testing.T) {
	t.Helper()
	
	envVars := []string{
		"GO_ENV", "SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS", "SMTP_FROM",
		"JWT_SECRET", "PORT", "DB_PATH",
	}
	
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

// TestEmailAddress returns a consistent test email address
func TestEmailAddress() string {
	return "test@example.com"
}

// TestListName returns a consistent test list name
func TestListName() string {
	return "Test Shopping List"
}

// TestItemName returns a consistent test item name
func TestItemName() string {
	return "Test Item"
}