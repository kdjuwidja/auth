package goauth

import (
	"context"
	"testing"
	"time"

	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// lua script is created after the first running of the test. In order to test the create and reuse of the script, we may need to directly shell into redis and flush the script.
func TestJWTTokenStore(t *testing.T) {
	logger.SetLevel("trace")
	logger.SetServiceName("test")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:7379",
		Password: "testpassword",
		Username: "default",
	})

	codeTTL := 300
	accessTTL := 3600
	refreshTTL := 86400

	ctx := context.Background()
	store, err := InitializeJWTTokenStore(redisClient, "../lua/create.lua")
	assert.NoError(t, err)

	// Clean up Redis after tests
	defer func() {
		jwtStore := store.(*JWTTokenStore)
		err := jwtStore.redisClient.FlushAll(ctx).Err()
		if err != nil {
			t.Logf("Failed to flush Redis: %v", err)
		}
	}()

	t.Run("Create and Get Token", func(t *testing.T) {
		// Create test token
		token := &models.Token{
			ClientID:         "test_client",
			UserID:           "test_user",
			Access:           "test_access_token",
			Refresh:          "test_refresh_token",
			Code:             "test_code",
			AccessExpiresIn:  time.Duration(accessTTL) * time.Second,
			RefreshExpiresIn: time.Duration(refreshTTL) * time.Second,
			CodeExpiresIn:    time.Duration(codeTTL) * time.Second,
			AccessCreateAt:   time.Now().Add(time.Duration(accessTTL) * time.Second),
			RefreshCreateAt:  time.Now().Add(time.Duration(refreshTTL) * time.Second),
		}

		// Create token
		err := store.Create(ctx, token)
		assert.NoError(t, err)

		// Test GetByCode
		foundToken, err := store.GetByCode(ctx, token.Code)
		assert.NoError(t, err)
		assert.Equal(t, token.UserID, foundToken.GetUserID())
		assert.Equal(t, token.Access, foundToken.GetAccess())

		// Test GetByAccess
		foundToken, err = store.GetByAccess(ctx, token.Access)
		assert.NoError(t, err)
		assert.Equal(t, token.UserID, foundToken.GetUserID())

		// Test GetByRefresh
		foundToken, err = store.GetByRefresh(ctx, token.Refresh)
		assert.NoError(t, err)
		assert.Equal(t, token.UserID, foundToken.GetUserID())

		// Test RemoveByCode
		err = store.RemoveByCode(ctx, token.Code)
		assert.NoError(t, err)
		_, err = store.GetByCode(ctx, token.Code)
		assert.Error(t, err)

		// Test RemoveByAccess
		err = store.RemoveByAccess(ctx, token.Access)
		assert.NoError(t, err)
		_, err = store.GetByAccess(ctx, token.Access)
		assert.Error(t, err)

		// Test RemoveByRefresh
		err = store.RemoveByRefresh(ctx, token.Refresh)
		assert.NoError(t, err)
		_, err = store.GetByRefresh(ctx, token.Refresh)
		assert.Error(t, err)
	})

	t.Run("Non-existent Token", func(t *testing.T) {
		// Test getting non-existent tokens
		_, err := store.GetByCode(ctx, "non_existent_code")
		assert.Error(t, err)

		_, err = store.GetByAccess(ctx, "non_existent_access")
		assert.Error(t, err)

		_, err = store.GetByRefresh(ctx, "non_existent_refresh")
		assert.Error(t, err)

		// Test removing non-existent tokens
		err = store.RemoveByCode(ctx, "non_existent_code")
		assert.Error(t, err)

		err = store.RemoveByAccess(ctx, "non_existent_access")
		assert.Error(t, err)

		err = store.RemoveByRefresh(ctx, "non_existent_refresh")
		assert.Error(t, err)
	})
}
