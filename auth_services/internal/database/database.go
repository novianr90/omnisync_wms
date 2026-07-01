package database

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() *gorm.DB {
	var err error

	if os.Getenv("DB_TYPE") == "postgres" {
		dsn := os.Getenv("AUTH_DATABASE_URL")
		if dsn == "" {
			log.Fatal("AUTH_DATABASE_URL is required when DB_TYPE=postgres")
		}
		for i := 0; i < 10; i++ {
			DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Info),
			})
			if err == nil {
				break
			}
			log.Printf("DB connection attempt %d/10 failed, retrying in 3s: %v", i+1, err)
			time.Sleep(3 * time.Second)
		}
	} else {
		dbPath := "auth.db"
		dir := filepath.Dir(dbPath)
		if dir != "." && dir != "" {
			_ = os.MkdirAll(dir, os.ModePerm)
		}
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err == nil {
			sqlDB, dbErr := DB.DB()
			if dbErr == nil {
				_, _ = sqlDB.Exec("PRAGMA journal_mode = WAL;")
				_, _ = sqlDB.Exec("PRAGMA busy_timeout = 5000;")
				_, _ = sqlDB.Exec("PRAGMA synchronous = NORMAL;")
			}
		}
	}

	if err != nil {
		log.Fatalf("Failed to connect to db_auth database: %v", err)
	}

	log.Println("Database db_auth connected successfully.")

	// Run Custom SQL Migrations
	RunMigrations(DB, "migrations")

	return DB
}


