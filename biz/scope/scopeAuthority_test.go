package bizscope

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	dbmodel "netherealmstudio.com/m/v2/db"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "ai_shopper_dev:password@tcp(localhost:4306)/test_db?parseTime=True"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	// Drop existing tables
	err = db.Migrator().DropTable(&dbmodel.UserRole{}, &dbmodel.RoleScope{}, &dbmodel.Role{}, &dbmodel.User{}, &dbmodel.APIClientScope{}, &dbmodel.APIClient{})
	assert.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&dbmodel.User{}, &dbmodel.Role{}, &dbmodel.RoleScope{}, &dbmodel.UserRole{}, &dbmodel.APIClient{}, &dbmodel.APIClientScope{})
	assert.NoError(t, err)

	return db
}

func createTestData(db *gorm.DB) error {
	// Create test users
	users := []dbmodel.User{
		{
			ID:       "test_user_1",
			Email:    "test1@example.com",
			Password: "hashed_password_1",
			IsActive: true,
		},
		{
			ID:       "test_user_2",
			Email:    "test2@example.com",
			Password: "hashed_password_2",
			IsActive: true,
		},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}

	// Create test roles
	roles := []dbmodel.Role{
		{
			ID:          1,
			Description: "Administrator role",
		},
		{
			ID:          2,
			Description: "Regular user role",
		},
	}
	for _, role := range roles {
		if err := db.Create(&role).Error; err != nil {
			return err
		}
	}

	// Create role scopes
	roleScopes := []dbmodel.RoleScope{
		{
			ID:     1,
			RoleID: 1,
			Scope:  "profile",
		},
		{
			ID:     2,
			RoleID: 1,
			Scope:  "shoplist",
		},
		{
			ID:     3,
			RoleID: 1,
			Scope:  "search",
		},
		{
			ID:     4,
			RoleID: 2,
			Scope:  "profile",
		},
		{
			ID:     5,
			RoleID: 2,
			Scope:  "shoplist",
		},
	}
	for _, scope := range roleScopes {
		if err := db.Create(&scope).Error; err != nil {
			return err
		}
	}

	// Assign roles to users
	userRoles := []dbmodel.UserRole{
		{
			UserID: "test_user_1",
			RoleID: 1,
		},
		{
			UserID: "test_user_1",
			RoleID: 2,
		},
		{
			UserID: "test_user_2",
			RoleID: 2,
		},
	}
	for _, userRole := range userRoles {
		if err := db.Create(&userRole).Error; err != nil {
			return err
		}
	}

	// Create test API clients
	apiClients := []dbmodel.APIClient{
		{
			ID:          "test_client_1",
			Secret:      "test_secret_1",
			Domain:      "http://test1.example.com",
			IsPublic:    true,
			Description: "Test client 1",
		},
		{
			ID:          "test_client_2",
			Secret:      "test_secret_2",
			Domain:      "http://test2.example.com",
			IsPublic:    false,
			Description: "Test client 2",
		},
	}
	for _, client := range apiClients {
		if err := db.Create(&client).Error; err != nil {
			return err
		}
	}

	// Create API client scopes
	apiClientScopes := []dbmodel.APIClientScope{
		{
			APIClientID: "test_client_1",
			Scope:       "profile",
		},
		{
			APIClientID: "test_client_1",
			Scope:       "shoplist",
		},
		{
			APIClientID: "test_client_1",
			Scope:       "search",
		},
		{
			APIClientID: "test_client_2",
			Scope:       "profile",
		},
		{
			APIClientID: "test_client_2",
			Scope:       "shoplist",
		},
	}
	for _, scope := range apiClientScopes {
		if err := db.Create(&scope).Error; err != nil {
			return err
		}
	}

	return nil
}

func TestScopeAuthority(t *testing.T) {
	db := setupTestDB(t)
	err := createTestData(db)
	assert.NoError(t, err)

	scopeAuth := NewScopeAuthority(db)

	tests := []struct {
		name           string
		apiClientID    string
		userID         string
		requestedScope string
		wantErr        bool
		description    string
	}{
		{
			name:           "Valid scopes for admin user with client 1",
			apiClientID:    "test_client_1",
			userID:         "test_user_1",
			requestedScope: "profile shoplist search",
			wantErr:        false,
			description:    "Admin user with all scopes requesting all available scopes from client 1",
		},
		{
			name:           "Valid scopes for regular user with client 2",
			apiClientID:    "test_client_2",
			userID:         "test_user_2",
			requestedScope: "profile shoplist",
			wantErr:        false,
			description:    "Regular user requesting available scopes from client 2",
		},
		{
			name:           "Invalid scope for user",
			apiClientID:    "test_client_1",
			userID:         "test_user_2",
			requestedScope: "search",
			wantErr:        true,
			description:    "Regular user requesting admin-only scope",
		},
		{
			name:           "Invalid scope for client",
			apiClientID:    "test_client_2",
			userID:         "test_user_1",
			requestedScope: "search",
			wantErr:        true,
			description:    "Client 2 requesting scope it doesn't have",
		},
		{
			name:           "Empty scope string",
			apiClientID:    "test_client_1",
			userID:         "test_user_1",
			requestedScope: "",
			wantErr:        false,
			description:    "Empty scope string is valid - no special permissions requested",
		},
		{
			name:           "Non-existent user",
			apiClientID:    "test_client_1",
			userID:         "non_existent_user",
			requestedScope: "profile",
			wantErr:        true,
			description:    "Non-existent user should fail",
		},
		{
			name:           "Non-existent client",
			apiClientID:    "non_existent_client",
			userID:         "test_user_1",
			requestedScope: "profile",
			wantErr:        true,
			description:    "Non-existent client should fail",
		},
		{
			name:           "Malformed scope with special characters",
			apiClientID:    "test_client_1",
			userID:         "test_user_1",
			requestedScope: "profile@shoplist",
			wantErr:        true,
			description:    "Scope with special characters should be rejected",
		},
		{
			name:           "Malformed scope with uppercase",
			apiClientID:    "test_client_1",
			userID:         "test_user_1",
			requestedScope: "PROFILE shoplist",
			wantErr:        true,
			description:    "Scope with uppercase should be rejected",
		},
		{
			name:           "Malformed scope with numbers",
			apiClientID:    "test_client_1",
			userID:         "test_user_1",
			requestedScope: "profile123 shoplist",
			wantErr:        true,
			description:    "Scope with numbers should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scopeAuth.AuthorizeScope(context.Background(), tt.apiClientID, tt.userID, tt.requestedScope)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
