// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

// Package db provides database initialization and connection management using GORM and SQLite.
package db

import (
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Init initializes the database connection and performs auto-migration of all models.
func Init(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.SystemSettings{},
		&models.User{},
		&models.ShoppingList{},
		&models.ListMember{},
		&models.Invitation{},
		&models.MagicLink{},
		&models.ShoppingItem{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
