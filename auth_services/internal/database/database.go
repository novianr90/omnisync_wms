package database

import (
	"log"
	"os"
	"path/filepath"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() *gorm.DB {
	// In development we write the database file locally in the working directory
	dbPath := "auth.db"

	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, os.ModePerm)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to db_auth database: %v", err)
	}

	log.Println("Database db_auth connected successfully.")

	// Run Custom SQL Migrations
	RunMigrations(DB, "migrations")

	return DB
}


