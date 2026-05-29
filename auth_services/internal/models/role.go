package models

import (
	"time"
)

type RolePermission struct {
	RoleID     string `gorm:"type:varchar(36);primaryKey" json:"role_id"`
	Permission string `gorm:"type:varchar(100);primaryKey" json:"permission"`
}

type Role struct {
	ID          string           `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name        string           `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string           `gorm:"type:text" json:"description"`
	Permissions []RolePermission `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"permissions,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
