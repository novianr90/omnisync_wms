package database

import (
	"log"
	"os"
	"gorm.io/driver/postgres"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() *gorm.DB {
	var err error
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite" // default in development
	}

	log.Printf("Connecting to db_wms using driver: %s", dbType)

	if dbType == "postgres" {
		dsn := os.Getenv("WMS_DATABASE_URL")
		if dsn == "" {
			log.Fatal("WMS_DATABASE_URL environment variable is required when DB_TYPE=postgres")
		}
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	} else {
		dbPath := "wms.db"
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	}

	if err != nil {
		log.Fatalf("Failed to connect to db_wms database: %v", err)
	}

	log.Println("Database db_wms connected successfully.")

	// Run Custom SQL Migrations
	RunMigrations(DB, "migrations")

	return DB
}
