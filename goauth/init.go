package goauth

import (
	"fmt"
	"time"

	"github.com/go-oauth2/oauth2/v4/manage"
	oauthmodels "github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"github.com/kdjuwidja/aishoppercommon/osutil"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"netherealmstudio.com/m/v2/biz"
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

	redisHost := osutil.GetEnvString("REDIS_HOST", "localhost")
	redisPort := osutil.GetEnvString("REDIS_PORT", "6379")
	redisUser := osutil.GetEnvString("REDIS_USER", "default")
	redisPassword := osutil.GetEnvString("REDIS_PASSWORD", "password")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		Username: redisUser,
	})

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
	goAuth.manager.MustTokenStorage(InitializeJWTTokenStore(redisClient, "./lua/create.lua"))

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
		user1 := dbmodel.User{
			ID:       "eb5dc850f1fb40a8b9b2bffd89c6a32d",
			Email:    "kdjuwidja@netherrealmstudio.com",
			Password: "$2a$10$vZU8LUTitjbU.FrFHIVkkuF7Gb6SrF3Zz0Eqq5coet/MuYEzRQ2Qm",
			IsActive: true,
		}
		user2 := dbmodel.User{
			ID:       "73064f370eda46a48a86e1fd8118be4c",
			Email:    "timmyk@netherrealmstudio.com",
			Password: "$2a$10$s5jg5gL1/2tWDOX5JcgWk.wqZR8kTxb46X3thb8JIsaD6HVYtpKGG",
			IsActive: true,
		}
		if err := dbConn.Create(&user1).Error; err != nil {
			return fmt.Errorf("failed to create user1: %v", err)
		}
		if err := dbConn.Create(&user2).Error; err != nil {
			return fmt.Errorf("failed to create user2: %v", err)
		}
	}
	return nil
}

func initializeAPIClientStore(dbConn *gorm.DB, isLocalDev bool) (*store.ClientStore, *biz.APIClientStore, error) {
	apiClientStore := biz.NewAPIClientStore(dbConn, isLocalDev)

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
