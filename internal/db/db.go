package db

import (
	"os"
	"path/filepath"

	"github.com/Larguma/stuff/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(path string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.House{},
		&models.HouseMember{},
		&models.Invite{},
		&models.Location{},
		&models.Tag{},
		&models.Item{},
		&models.ItemImage{},
	)
}
