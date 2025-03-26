package models

import "gorm.io/gorm"

type APIClient struct {
	gorm.Model
	ID          string `json:"id" gorm:"type:varchar(45);primaryKey"`
	Secret      string `json:"secret" gorm:"type:varchar(45);not null"`
	Domain      string `json:"domain" gorm:"type:varchar(255);not null"`
	IsPublic    bool   `json:"is_public" gorm:"type:tinyint(1);not null;default:0"`
	Description string `json:"description" gorm:"type:varchar(255);"`
}

type User struct {
	gorm.Model
	ID       string `json:"id" gorm:"type:varchar(32);primaryKey"`
	Email    string `json:"email" gorm:"type:varchar(255);not null;unique"`
	Password string `json:"password" gorm:"type:varchar(255);not null"`
	IsActive bool   `json:"is_active" gorm:"type:tinyint(1);not null;default:1"`
}
