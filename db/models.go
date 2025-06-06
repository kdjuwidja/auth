package db

import "gorm.io/gorm"

type APIClient struct {
	gorm.Model
	ID          string `json:"id" gorm:"type:varchar(45);primaryKey"`
	Secret      string `json:"secret" gorm:"type:varchar(45);not null"`
	Domain      string `json:"domain" gorm:"type:varchar(255);not null"`
	IsPublic    bool   `json:"is_public" gorm:"type:tinyint(1);not null;default:0"`
	Description string `json:"description" gorm:"type:varchar(255);"`
}

type APIClientScope struct {
	gorm.Model
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	APIClientID string    `json:"api_client_id" gorm:"type:varchar(45);not null;foreignKey:ID;references:APIClient"`
	Scope       string    `json:"scope" gorm:"type:varchar(255);not null"`
	APIClient   APIClient `json:"api_client" gorm:"foreignKey:APIClientID"`
}

type Role struct {
	gorm.Model
	ID          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Description string `json:"description" gorm:"type:varchar(255);not null"`
}

type RoleScope struct {
	gorm.Model
	ID     int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RoleID int    `json:"role_id" gorm:"not null;foreignKey:ID;references:Role"`
	Scope  string `json:"scope" gorm:"type:varchar(255);not null"`
	Role   Role   `json:"role" gorm:"foreignKey:RoleID"`
}

type User struct {
	gorm.Model
	ID       string `json:"id" gorm:"type:varchar(32);primaryKey"`
	Email    string `json:"email" gorm:"type:varchar(255);not null;unique"`
	Password string `json:"password" gorm:"type:varchar(255);not null"`
	IsActive bool   `json:"is_active" gorm:"type:tinyint(1);not null;default:1"`
}

type UserRole struct {
	gorm.Model
	UserID string `json:"user_id" gorm:"type:varchar(32);not null;foreignKey:ID;references:User"`
	RoleID int    `json:"role_id" gorm:"not null;foreignKey:ID;references:Role"`
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	Role   Role   `json:"role" gorm:"foreignKey:RoleID"`
}

type RegistrationCode struct {
	gorm.Model
	Code string `json:"code" gorm:"type:varchar(6);primaryKey"`
}
