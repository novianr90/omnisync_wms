package database

import (
	"log"
	"log/slog"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	
	customlogger "wms_dashboard/internal/logger"
)

var DB *gorm.DB

func InitDB() *gorm.DB {
	var err error
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite" // default in development
	}

	slog.Info("Connecting to db_wms", slog.String("driver", dbType))

	gormLogger := customlogger.NewGormLogger()

	if dbType == "postgres" {
		dsn := os.Getenv("WMS_DATABASE_URL")
		if dsn == "" {
			log.Fatal("WMS_DATABASE_URL environment variable is required when DB_TYPE=postgres")
		}
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: gormLogger,
		})
	} else {
		dbPath := "wms.db"
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: gormLogger,
		})
		if err == nil {
			sqlDB, err := DB.DB()
			if err == nil {
				_, _ = sqlDB.Exec("PRAGMA journal_mode = WAL;")
				_, _ = sqlDB.Exec("PRAGMA busy_timeout = 5000;")
				_, _ = sqlDB.Exec("PRAGMA synchronous = NORMAL;")
			}
		}
	}

	if err != nil {
		slog.Error("Failed to connect to db_wms database", slog.Any("error", err))
		log.Fatalf("Failed to connect to db_wms database: %v", err)
	}

	slog.Info("Database db_wms connected successfully")

	// Run Custom SQL Migrations
	RunMigrations(DB, "migrations")

	return DB
}
