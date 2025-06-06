package goauth

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/manage"
	oauthmodels "github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"github.com/kdjuwidja/aishoppercommon/osutil"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	bizapiclient "netherealmstudio.com/m/v2/biz/apiclient"
	dbmodel "netherealmstudio.com/m/v2/db"
	"netherealmstudio.com/m/v2/statestore"
	"netherealmstudio.com/m/v2/token"
)

type GoAuth struct {
	srv        *server.Server
	statestore *statestore.StateStore
	manager    *manage.Manager
}

func (g *GoAuth) GetSrv() *server.Server {
	return g.srv
}

func (g *GoAuth) GetStateStore() *statestore.StateStore {
	return g.statestore
}

func InitializeGoAuth(dbConn *gorm.DB, isLocalDev bool) (*GoAuth, error) {
	goAuth := &GoAuth{}

	// Initialize state store
	goAuth.statestore = statestore.NewStateStore()
	goAuth.manager = manage.NewDefaultManager()

	codeTTL := osutil.GetEnvInt("CODE_TTL", 300)
	accessTTL := osutil.GetEnvInt("ACCESS_TTL", 3600)
	refreshTTL := osutil.GetEnvInt("REFRESH_TTL", 86400)

	// Initialize API client store
	goAuthClientStore, apiClientStore, err := initializeAPIClientStore(dbConn, isLocalDev)
	if err != nil {
		return nil, err
	}
	goAuth.manager.MapClientStorage(goAuthClientStore)

	//token memory store

	var jwtTokenStore oauth2.TokenStore
	hasKeyLimit := osutil.GetEnvBool("RESTRICT_NUM_KEYS", false)
	if hasKeyLimit {
		redisHost := osutil.GetEnvString("REDIS_HOST", "localhost")
		redisPort := osutil.GetEnvString("REDIS_PORT", "6379")
		redisUser := osutil.GetEnvString("REDIS_USER", "default")
		redisPassword := osutil.GetEnvString("REDIS_PASSWORD", "password")

		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
			Password: redisPassword,
			Username: redisUser,
		})
		jwtTokenStore, err = InitializeJWTTokenStoreWithKeyLimit(redisClient, "./lua/create.lua", osutil.GetEnvInt("MAX_NUM_KEYS", 5))
	} else {
		jwtTokenStore, err = InitializeJWTTokenStore()
	}
	goAuth.manager.MustTokenStorage(jwtTokenStore, err)

	// Configure JWT token generation with custom claims
	jwtSecret := osutil.GetEnvString("JWT_SECRET", "your-secret-key")
	accessGen := token.NewJWTTokenGenerator("jwt-key", []byte(jwtSecret), apiClientStore)
	goAuth.manager.MapAccessGenerate(accessGen)
	goAuth.manager.SetAuthorizeCodeExp(time.Duration(codeTTL) * time.Second)
	goAuth.manager.SetAuthorizeCodeTokenCfg(&manage.Config{
		AccessTokenExp:    time.Duration(accessTTL) * time.Second,
		RefreshTokenExp:   time.Duration(refreshTTL) * time.Second,
		IsGenerateRefresh: true,
	})

	goAuth.srv = server.NewDefaultServer(goAuth.manager)
	goAuth.srv.SetAllowGetAccessRequest(true)
	goAuth.srv.SetClientInfoHandler(server.ClientFormHandler)

	//create default local dev user
	if isLocalDev {
		if err := createLocalDevUser(dbConn); err != nil {
			logger.Fatalf("Failed to create local dev users: %v", err)
		}
	}

	goAuthHandler := &GoAuthHandler{
		dbConn: dbConn,
	}

	goAuth.srv.SetUserAuthorizationHandler(goAuthHandler.userAuthorizationHandler)
	goAuth.srv.SetInternalErrorHandler(goAuthHandler.setInternalErrorHandler)
	goAuth.srv.SetResponseErrorHandler(goAuthHandler.setResponseErrorHandler)

	return goAuth, nil
}

func createLocalDevUser(dbConn *gorm.DB) error {
	var count int64
	result := dbConn.Find(&dbmodel.User{}).Count(&count)
	if result.Error != nil {
		return fmt.Errorf("failed to access user table: %v", result.Error)
	}

	if count == 0 {
		userIds := osutil.GetEnvString("DEFAULT_USER_IDS", "")
		userEmails := osutil.GetEnvString("DEFAULT_USER_EMAILS", "")
		passwords := osutil.GetEnvString("DEFAULT_PASSWORDS", "")

		userIdsList := strings.Split(userIds, ",")
		userEmailsList := strings.Split(userEmails, ",")
		passwordsList := strings.Split(passwords, ",")

		for i, userId := range userIdsList {
			userEmail := userEmailsList[i]
			password := passwordsList[i]

			user := dbmodel.User{
				ID:       userId,
				Email:    userEmail,
				Password: password,
				IsActive: true,
			}

			if err := dbConn.Create(&user).Error; err != nil {
				return fmt.Errorf("failed to create user: %v", err)
			}
		}
	}
	return nil
}

func initializeAPIClientStore(dbConn *gorm.DB, isLocalDev bool) (*store.ClientStore, *bizapiclient.APIClientStore, error) {
	apiClientStore := bizapiclient.NewAPIClientStore(dbConn, isLocalDev)

	goauthClientStore := store.NewClientStore()
	goauthClients := apiClientStore.GetAPIClients()
	for _, client := range goauthClients {
		goauthClientStore.Set(client.ID, &oauthmodels.Client{
			ID:     client.ID,
			Secret: client.Secret,
			Domain: client.Domain,
		})
	}

	return goauthClientStore, apiClientStore, nil
}
