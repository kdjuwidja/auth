package bizregister

import (
	"context"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	dbmodel "netherealmstudio.com/m/v2/db"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "ai_shopper_dev:password@tcp(localhost:4306)/test_db?parseTime=True"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Clean up and migrate
	db.Migrator().DropTable(&dbmodel.RegistrationCode{}, &dbmodel.User{})
	db.AutoMigrate(&dbmodel.RegistrationCode{}, &dbmodel.User{})

	return db
}

func TestRegistrationManager(t *testing.T) {
	db := setupTestDB(t)
	manager := NewRegistrationManager(db, 3)

	t.Run("GetRegistrationCode", func(t *testing.T) {
		code, err := manager.GetRegistrationCode(context.Background())
		if err != nil {
			t.Fatalf("Failed to get registration code: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("Expected code length 6, got %d", len(code))
		}

		// Verify code exists in DB
		var dbCode dbmodel.RegistrationCode
		if err := db.Where("code = ?", code).First(&dbCode).Error; err != nil {
			t.Errorf("Code not found in database: %v", err)
		}
	})

	t.Run("RegisterUser", func(t *testing.T) {
		// Get a code first
		code, err := manager.GetRegistrationCode(context.Background())
		if err != nil {
			t.Fatalf("Failed to get registration code: %v", err)
		}

		// Register user
		err = manager.RegisterUser(context.Background(), code, "test@example.com", "password123")
		if err != nil {
			t.Fatalf("Failed to register user: %v", err)
		}

		// Verify user was created
		var user dbmodel.User
		if err := db.Where("email = ?", "test@example.com").First(&user).Error; err != nil {
			t.Errorf("User not found in database: %v", err)
		}

		// Verify code was consumed
		var dbCode dbmodel.RegistrationCode
		if err := db.Where("code = ?", code).First(&dbCode).Error; err == nil {
			t.Error("Code still exists in database after registration")
		}
	})

	t.Run("RegisterUserWithInvalidCode", func(t *testing.T) {
		err := manager.RegisterUser(context.Background(), "INVALID", "test2@example.com", "password123")
		if err == nil {
			t.Error("Expected error for invalid code, got nil")
		}
	})
}
