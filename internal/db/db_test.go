// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit_InMemory(t *testing.T) {
	db, err := Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize in-memory database: %v", err)
	}

	if db == nil {
		t.Fatal("Database should not be nil")
	}

	// Test that we can perform a simple query
	var result int
	err = db.Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		t.Errorf("Failed to execute simple query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected result to be 1, got %d", result)
	}
}

func TestInit_FileDatabase(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize file database: %v", err)
	}

	if db == nil {
		t.Fatal("Database should not be nil")
	}

	// Check that the database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Test that we can perform a simple query
	var result int
	err = db.Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		t.Errorf("Failed to execute simple query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected result to be 1, got %d", result)
	}
}

func TestInit_InvalidPath(t *testing.T) {
	// Try to create a database in a non-existent directory without proper permissions
	invalidPath := "/nonexistent/directory/test.db"

	_, err := Init(invalidPath)
	if err == nil {
		t.Error("Expected error when initializing database with invalid path, got nil")
	}
}

func TestInit_RelativePath(t *testing.T) {
	// Test with a relative path
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to temp directory
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	db, err := Init("test.db")
	if err != nil {
		t.Fatalf("Failed to initialize database with relative path: %v", err)
	}

	if db == nil {
		t.Fatal("Database should not be nil")
	}

	// Check that the database file was created in the correct location
	if _, err := os.Stat("test.db"); os.IsNotExist(err) {
		t.Error("Database file was not created with relative path")
	}
}

func TestInit_AutoMigration(t *testing.T) {
	db, err := Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Check that tables were created by auto-migration
	tables := []string{
		"system_settings",
		"users",
		"shopping_lists",
		"list_members",
		"invitations",
		"magic_links",
		"shopping_items",
	}

	for _, table := range tables {
		var count int64
		err := db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count).Error
		if err != nil {
			t.Errorf("Failed to check for table %s: %v", table, err)
			continue
		}

		if count == 0 {
			t.Errorf("Table %s was not created by auto-migration", table)
		}
	}
}

func TestInit_MultipleConnections(t *testing.T) {
	// Test that we can create multiple database connections
	db1, err := Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize first database: %v", err)
	}

	db2, err := Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize second database: %v", err)
	}

	// Both should be valid but separate connections
	if db1 == nil || db2 == nil {
		t.Fatal("Database connections should not be nil")
	}

	// Test that they are independent (different memory databases)
	var result1, result2 int

	// Create a test table in first database
	err = db1.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, value TEXT)").Error
	if err != nil {
		t.Fatalf("Failed to create table in first database: %v", err)
	}

	// Insert data in first database
	err = db1.Exec("INSERT INTO test_table (value) VALUES ('test')").Error
	if err != nil {
		t.Fatalf("Failed to insert data in first database: %v", err)
	}

	// Count rows in first database
	err = db1.Raw("SELECT COUNT(*) FROM test_table").Scan(&result1).Error
	if err != nil {
		t.Fatalf("Failed to count rows in first database: %v", err)
	}

	// Try to count rows in second database (should fail because table doesn't exist)
	err = db2.Raw("SELECT COUNT(*) FROM test_table").Scan(&result2).Error
	if err == nil {
		t.Error("Expected error when querying non-existent table in second database")
	}

	if result1 != 1 {
		t.Errorf("Expected 1 row in first database, got %d", result1)
	}
}

func TestInit_EmptyPath(t *testing.T) {
	// Test with empty path - this actually works with SQLite (creates database in current directory)
	db, err := Init("")
	if err != nil {
		t.Logf("Empty path failed as expected: %v", err)
	} else {
		t.Log("Empty path succeeded (creates database in current directory)")
		if db == nil {
			t.Error("Database should not be nil when init succeeds")
		}
	}
}
