package models

import (
	"time"
)

type User struct {
	ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	FirstName    string    `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName     string    `gorm:"type:varchar(100);not null" json:"last_name"`
	Role         string    `gorm:"type:varchar(50);default:'operator'" json:"role"` // admin, manager, operator
	IsActive     bool      `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
