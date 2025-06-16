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
	db.Migrator().DropTable(&dbmodel.RegistrationCode{}, &dbmodel.User{}, &dbmodel.UserRole{}, &dbmodel.Role{}, &dbmodel.RoleScope{})
	db.AutoMigrate(&dbmodel.RegistrationCode{}, &dbmodel.User{}, &dbmodel.UserRole{}, &dbmodel.Role{}, &dbmodel.RoleScope{})

	// Create test role
	testRole := dbmodel.Role{
		Description: "Test role for registration",
	}
	if err := db.Create(&testRole).Error; err != nil {
		t.Fatalf("Failed to create test role: %v", err)
	}

	// Create test role scope
	testRoleScope := dbmodel.RoleScope{
		RoleID: testRole.ID,
		Scope:  "test_scope",
	}
	if err := db.Create(&testRoleScope).Error; err != nil {
		t.Fatalf("Failed to create test role scope: %v", err)
	}

	return db
}

func TestRegistrationManager(t *testing.T) {
	db := setupTestDB(t)

	// Get the test role ID
	var testRole dbmodel.Role
	if err := db.Where("description = ?", "Test role for registration").First(&testRole).Error; err != nil {
		t.Fatalf("Failed to get test role: %v", err)
	}

	manager := NewRegistrationManager(db, 3, testRole.ID)

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

		// Verify user role was created
		var userRole dbmodel.UserRole
		if err := db.Where("user_id = ?", user.ID).First(&userRole).Error; err != nil {
			t.Errorf("User role not found in database: %v", err)
		}

		// Verify role exists
		var role dbmodel.Role
		if err := db.Where("id = ?", userRole.RoleID).First(&role).Error; err != nil {
			t.Errorf("Role not found in database: %v", err)
		}

		// Verify role scope exists
		var roleScope dbmodel.RoleScope
		if err := db.Where("role_id = ?", role.ID).First(&roleScope).Error; err != nil {
			t.Errorf("Role scope not found in database: %v", err)
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
