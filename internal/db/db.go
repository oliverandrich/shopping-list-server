package db

import (
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Init(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&models.User{}, &models.MagicLink{}, &models.ShoppingItem{})
	if err != nil {
		return nil, err
	}

	return db, nil
}