package bizapiclient

import (
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
		Logger: logger.Default.LogMode(logger.Info), // Enable query logging
	})
	assert.NoError(t, err)

	// Drop existing tables
	err = db.Migrator().DropTable(&dbmodel.APIClientScope{}, &dbmodel.APIClient{})
	assert.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&dbmodel.APIClient{}, &dbmodel.APIClientScope{})
	assert.NoError(t, err)

	return db
}

func TestAPIClientStoreInitialization_IsLocalDevTrue_NoExistingClient(t *testing.T) {
	db := setupTestDB(t)

	// Initialize store with IsLocalDev = true
	store := NewAPIClientStore(db, true)
	assert.NotNil(t, store)

	// Verify default client was created
	var client dbmodel.APIClient
	result := db.First(&client)
	assert.NoError(t, result.Error)
	assert.Equal(t, "82ce1a881b304775ad288e57e41387f3", client.ID)
	assert.Equal(t, "my_secret", client.Secret)
	assert.Equal(t, "http://localhost:3000", client.Domain)
	assert.True(t, client.IsPublic)
	assert.Equal(t, "Default client for ai_shopper_depot", client.Description)

	// Verify default scopes were created
	var scopes []dbmodel.APIClientScope
	result = db.Find(&scopes)
	assert.NoError(t, result.Error)
	assert.Len(t, scopes, 3)

	expectedScopes := map[string]bool{
		"profile":  true,
		"shoplist": true,
		"search":   true,
	}
	for _, scope := range scopes {
		assert.Equal(t, client.ID, scope.APIClientID)
		assert.True(t, expectedScopes[scope.Scope], "Unexpected scope: %s", scope.Scope)
		delete(expectedScopes, scope.Scope)
	}
	assert.Empty(t, expectedScopes, "Some expected scopes were not created")
}

func TestAPIClientStoreInitialization_IsLocalDevTrue_ExistingClient(t *testing.T) {
	db := setupTestDB(t)

	// Create existing client
	existingClient := dbmodel.APIClient{
		ID:          "82ce1a881b304775ad288e57e41387f3",
		Secret:      "my_secret",
		Domain:      "http://localhost:3000",
		IsPublic:    true,
		Description: "Default client for ai_shopper_depot",
	}
	err := db.Create(&existingClient).Error
	assert.NoError(t, err)

	// Initialize store with IsLocalDev = true
	store := NewAPIClientStore(db, true)
	assert.NotNil(t, store)

	// Verify only scopes were created (client should not be recreated)
	var clientCount int64
	db.Model(&dbmodel.APIClient{}).Count(&clientCount)
	assert.Equal(t, int64(1), clientCount)

	// Verify default scopes were created
	var scopes []dbmodel.APIClientScope
	result := db.Find(&scopes)
	assert.NoError(t, result.Error)
	assert.Len(t, scopes, 3)

	expectedScopes := map[string]bool{
		"profile":  true,
		"shoplist": true,
		"search":   true,
	}
	for _, scope := range scopes {
		assert.Equal(t, existingClient.ID, scope.APIClientID)
		assert.True(t, expectedScopes[scope.Scope], "Unexpected scope: %s", scope.Scope)
		delete(expectedScopes, scope.Scope)
	}
	assert.Empty(t, expectedScopes, "Some expected scopes were not created")
}

func TestAPIClientStoreInitialization_IsLocalDevFalse(t *testing.T) {
	db := setupTestDB(t)

	// Initialize store with IsLocalDev = false
	store := NewAPIClientStore(db, false)
	assert.NotNil(t, store)

	// Verify no clients were created
	var clientCount int64
	db.Model(&dbmodel.APIClient{}).Count(&clientCount)
	assert.Equal(t, int64(0), clientCount)

	// Verify no scopes were created
	var scopeCount int64
	db.Model(&dbmodel.APIClientScope{}).Count(&scopeCount)
	assert.Equal(t, int64(0), scopeCount)
}

func TestAPIClientStoreInitialization_ExistingDataRetrieval(t *testing.T) {
	db := setupTestDB(t)

	// Create multiple clients with different scopes
	clients := []dbmodel.APIClient{
		{
			ID:          "client1",
			Secret:      "secret1",
			Domain:      "http://client1.com",
			IsPublic:    true,
			Description: "Client 1",
		},
		{
			ID:          "client2",
			Secret:      "secret2",
			Domain:      "http://client2.com",
			IsPublic:    false,
			Description: "Client 2",
		},
	}

	for _, client := range clients {
		err := db.Create(&client).Error
		assert.NoError(t, err)
	}

	// Create scopes for each client
	scopes := []dbmodel.APIClientScope{
		{APIClientID: "client1", Scope: "profile"},
		{APIClientID: "client1", Scope: "shoplist"},
		{APIClientID: "client2", Scope: "search"},
		{APIClientID: "client2", Scope: "profile"},
	}

	for _, scope := range scopes {
		err := db.Create(&scope).Error
		assert.NoError(t, err)
	}

	// Initialize store with IsLocalDev = false to prevent auto-creation
	store := NewAPIClientStore(db, false)
	assert.NotNil(t, store)

	// Get all clients from store
	retrievedClients := store.GetAPIClients()
	assert.Len(t, retrievedClients, 2)

	// Verify first client and its scopes
	client1 := retrievedClients[0]
	assert.Equal(t, "client1", client1.ID)
	assert.Equal(t, "secret1", client1.Secret)
	assert.Equal(t, "http://client1.com", client1.Domain)
	assert.True(t, client1.IsPublic)
	assert.Equal(t, "Client 1", client1.Description)
	assert.Equal(t, "profile shoplist", client1.Scopes)

	// Verify second client and its scopes
	client2 := retrievedClients[1]
	assert.Equal(t, "client2", client2.ID)
	assert.Equal(t, "secret2", client2.Secret)
	assert.Equal(t, "http://client2.com", client2.Domain)
	assert.False(t, client2.IsPublic)
	assert.Equal(t, "Client 2", client2.Description)
	assert.Equal(t, "search profile", client2.Scopes)
}

func TestAPIClientStore_GetScope(t *testing.T) {
	db := setupTestDB(t)

	// Create a client with scopes
	client := dbmodel.APIClient{
		ID:          "test_client",
		Secret:      "test_secret",
		Domain:      "http://test.com",
		IsPublic:    true,
		Description: "Test client",
	}
	err := db.Create(&client).Error
	assert.NoError(t, err)

	// Create scopes for the client
	scopes := []dbmodel.APIClientScope{
		{APIClientID: "test_client", Scope: "profile"},
		{APIClientID: "test_client", Scope: "shoplist"},
	}
	for _, scope := range scopes {
		err := db.Create(&scope).Error
		assert.NoError(t, err)
	}

	// Initialize store
	store := NewAPIClientStore(db, false)
	assert.NotNil(t, store)

	// Test successful scope retrieval
	scope, err := store.GetScope("test_client")
	assert.NoError(t, err)
	assert.Equal(t, "profile shoplist", scope)

	// Test non-existent client
	scope, err = store.GetScope("non_existent_client")
	assert.Error(t, err)
	assert.Empty(t, scope)
}

func TestAPIClientStore_GetClient(t *testing.T) {
	db := setupTestDB(t)

	// Create a client with scopes
	client := dbmodel.APIClient{
		ID:          "test_client",
		Secret:      "test_secret",
		Domain:      "http://test.com",
		IsPublic:    true,
		Description: "Test client",
	}
	err := db.Create(&client).Error
	assert.NoError(t, err)

	// Create scopes for the client
	scopes := []dbmodel.APIClientScope{
		{APIClientID: "test_client", Scope: "profile"},
		{APIClientID: "test_client", Scope: "shoplist"},
	}
	for _, scope := range scopes {
		err := db.Create(&scope).Error
		assert.NoError(t, err)
	}

	// Initialize store
	store := NewAPIClientStore(db, false)
	assert.NotNil(t, store)

	// Test successful client retrieval
	retrievedClient, err := store.GetClient("test_client")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedClient)
	assert.Equal(t, "test_client", retrievedClient.ID)
	assert.Equal(t, "test_secret", retrievedClient.Secret)
	assert.Equal(t, "http://test.com", retrievedClient.Domain)
	assert.True(t, retrievedClient.IsPublic)
	assert.Equal(t, "Test client", retrievedClient.Description)
	assert.Equal(t, "profile shoplist", retrievedClient.Scopes)

	// Test non-existent client
	retrievedClient, err = store.GetClient("non_existent_client")
	assert.Error(t, err)
	assert.Nil(t, retrievedClient)
}
