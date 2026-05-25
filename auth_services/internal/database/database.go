package database

import (
	"log"
	"os"
	"path/filepath"

	"auth_services/internal/models"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
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

	// Run Automigrations
	err = DB.AutoMigrate(&models.Role{}, &models.User{})
	if err != nil {
		log.Fatalf("Failed to run db_auth automigrations: %v", err)
	}
	log.Println("Database db_auth automigrations completed.")

	seedRoles(DB)

	return DB
}

func seedRoles(db *gorm.DB) {
	roles := []string{"System Admin", "Admin WMS", "Procurement", "POS"}
	for _, roleName := range roles {
		var role models.Role
		if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				newRole := models.Role{
					ID:          uuid.New().String(),
					Name:        roleName,
					Description: "Auto-seeded role: " + roleName,
				}
				if err := db.Create(&newRole).Error; err != nil {
					log.Printf("Failed to seed role %s: %v", roleName, err)
				} else {
					log.Printf("Successfully seeded role: %s", roleName)
				}
			} else {
				log.Printf("Error checking for role %s: %v", roleName, err)
			}
		}
	}
}
