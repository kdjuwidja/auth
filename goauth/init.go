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

	//token memory store
	goAuth.manager.MustTokenStorage(InitializeJWTTokenStore(redisClient, "./lua/create.lua"))

	// Configure JWT token generation with custom claims
	jwtSecret := osutil.GetEnvString("JWT_SECRET", "your-secret-key")
	accessGen := token.NewJWTTokenGenerator("jwt-key", []byte(jwtSecret))
	goAuth.manager.MapAccessGenerate(accessGen)
	goAuth.manager.SetAuthorizeCodeExp(time.Duration(codeTTL) * time.Second)
	goAuth.manager.SetAuthorizeCodeTokenCfg(&manage.Config{
		AccessTokenExp:    time.Duration(accessTTL) * time.Second,
		RefreshTokenExp:   time.Duration(refreshTTL) * time.Second,
		IsGenerateRefresh: true,
	})

	// Initialize API client store
	clientStore, err := initializeAPIClientStore(dbConn, isLocalDev)
	if err != nil {
		return nil, err
	}
	goAuth.manager.MapClientStorage(clientStore)

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

func initializeAPIClientStore(dbConn *gorm.DB, isLocalDev bool) (*store.ClientStore, error) {
	clientStore := store.NewClientStore()

	result := dbConn.Find(&dbmodel.APIClient{})
	if result.Error != nil {
		return nil, fmt.Errorf("error loading clients: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		if isLocalDev {
			//create default local dev client
			defaultClientId := osutil.GetEnvString("DEFAULT_CLIENT_ID", "82ce1a881b304775ad288e57e41387f3")
			defaultClientSecret := osutil.GetEnvString("DEFAULT_CLIENT_SECRET", "my_secret")
			defaultClientDomain := osutil.GetEnvString("DEFAULT_CLIENT_DOMAIN", "http://localhost:3000")
			defaultIsPublic := osutil.GetEnvString("DEFAULT_IS_PUBLIC", "1")
			defaultDescription := osutil.GetEnvString("DEFAULT_DESCRIPTION", "Default client for ai_shopper_depot")

			client := dbmodel.APIClient{
				ID:          defaultClientId,
				Secret:      defaultClientSecret,
				Domain:      defaultClientDomain,
				IsPublic:    defaultIsPublic == "1",
				Description: defaultDescription,
			}
			dbConn.Create(&client)

			clientStore.Set(client.ID, &oauthmodels.Client{
				ID:     client.ID,
				Secret: client.Secret,
				Domain: client.Domain,
			})
		} else {
			return nil, fmt.Errorf("no clients found")
		}
	} else {
		// Retrieve all clients from db and set to client store
		var clients []dbmodel.APIClient
		result.Scan(&clients)
		for _, client := range clients {
			clientStore.Set(client.ID, &oauthmodels.Client{
				ID:     client.ID,
				Secret: client.Secret,
				Domain: client.Domain,
			})

			logger.Tracef("Client: %s, %s, %s", client.ID, client.Secret, client.Domain)
		}
	}

	return clientStore, nil
}
