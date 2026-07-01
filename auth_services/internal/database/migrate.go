package database

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type SchemaMigration struct {
	Version   string    `gorm:"primaryKey"`
	AppliedAt time.Time
}

func (SchemaMigration) TableName() string {
	return "auth_schema_migrations"
}

func RunMigrations(db *gorm.DB, migrationsDir string) {
	// Create schema_migrations table to track applied migrations
	if err := db.AutoMigrate(&SchemaMigration{}); err != nil {
		log.Fatalf("Failed to create schema_migrations table: %v", err)
	}

	// Read migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Migrations directory '%s' does not exist. Skipping migrations.", migrationsDir)
			return
		}
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}

	// Sort migrations alphabetically
	sort.Strings(sqlFiles)

	for _, fileName := range sqlFiles {
		var count int64
		db.Model(&SchemaMigration{}).Where("version = ?", fileName).Count(&count)

		if count == 0 {
			log.Printf("Applying migration: %s", fileName)

			content, err := os.ReadFile(filepath.Join(migrationsDir, fileName))
			if err != nil {
				log.Fatalf("Failed to read migration file %s: %v", fileName, err)
			}

			// Execute the SQL within a transaction
			err = db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec(string(content)).Error; err != nil {
					return err
				}
				
				migration := SchemaMigration{
					Version:   fileName,
					AppliedAt: time.Now(),
				}
				if err := tx.Create(&migration).Error; err != nil {
					return err
				}
				return nil
			})

			if err != nil {
				log.Fatalf("Migration %s failed: %v", fileName, err)
			}
			log.Printf("Migration %s applied successfully", fileName)
		}
	}
	
	log.Println("All database migrations are up to date.")
}
