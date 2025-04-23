package biz

import (
	"fmt"
	"strings"

	"github.com/kdjuwidja/aishoppercommon/osutil"
	"gorm.io/gorm"
	dbmodel "netherealmstudio.com/m/v2/db"
)

type APIClient struct {
	ID          string `json:"id"`
	Secret      string `json:"secret"`
	Domain      string `json:"domain"`
	IsPublic    bool   `json:"is_public"`
	Description string `json:"description"`
	Scopes      string `json:"scopes"`
}

type APIClientStore struct {
	apiClients map[string]*APIClient
	dbConn     *gorm.DB
	isLocalDev bool
}

func NewAPIClientStore(dbConn *gorm.DB, isLocalDev bool) *APIClientStore {
	store := &APIClientStore{
		apiClients: make(map[string]*APIClient),
		dbConn:     dbConn,
		isLocalDev: isLocalDev,
	}
	store.initializeAPIClientStore()
	return store
}

func (s *APIClientStore) GetAPIClients() []APIClient {
	apiClients := make([]APIClient, 0)
	for _, apiClient := range s.apiClients {
		apiClients = append(apiClients, *apiClient)
	}
	return apiClients
}

func (s *APIClientStore) GetScope(clientId string) (string, error) {
	if scope, ok := s.apiClients[clientId]; !ok {
		return "", fmt.Errorf("client not found")
	} else {
		return scope.Scopes, nil
	}
}

func (s *APIClientStore) GetClient(clientId string) (*APIClient, error) {
	if client, ok := s.apiClients[clientId]; !ok {
		return nil, fmt.Errorf("client not found")
	} else {
		return client, nil
	}
}

func createDefaultAPIClient(dbConn *gorm.DB) error {
	apiClientResult := dbConn.Find(&dbmodel.APIClient{})
	if apiClientResult.Error != nil {
		return fmt.Errorf("error loading clients: %v", apiClientResult.Error)
	}

	if apiClientResult.RowsAffected == 0 {
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
		return dbConn.Create(&client).Error
	}

	return nil
}

func loadAPIClient(dbConn *gorm.DB) (map[string]*APIClient, error) {
	var dbClients []dbmodel.APIClient
	apiClientResult := dbConn.Find(&dbClients)
	if apiClientResult.Error != nil {
		return nil, fmt.Errorf("error loading clients: %v", apiClientResult.Error)
	}

	if len(dbClients) == 0 {
		return nil, fmt.Errorf("no clients found")
	}

	apiClients := make(map[string]*APIClient)
	for _, client := range dbClients {
		apiClient := &APIClient{
			ID:          client.ID,
			Secret:      client.Secret,
			Domain:      client.Domain,
			IsPublic:    client.IsPublic,
			Description: client.Description,
		}
		apiClients[client.ID] = apiClient
	}

	return apiClients, nil
}

// loadOrCreateDefaultAPIClientScope loads the default API client scope
func createDefaultAPIClientScope(dbConn *gorm.DB) error {
	apiClientScopeResult := dbConn.Find(&dbmodel.APIClientScope{})
	if apiClientScopeResult.Error != nil {
		return fmt.Errorf("error loading client scopes: %v", apiClientScopeResult.Error)
	}

	if apiClientScopeResult.RowsAffected == 0 {
		defaultClientId := osutil.GetEnvString("DEFAULT_CLIENT_ID", "82ce1a881b304775ad288e57e41387f3")
		scopes := osutil.GetEnvString("DEFAULT_CLIENT_SCOPE_SCOPE", "profile shoplist search")
		scopeList := strings.Split(scopes, " ")
		for _, scope := range scopeList {
			err := dbConn.Create(&dbmodel.APIClientScope{
				APIClientID: defaultClientId,
				Scope:       scope,
			}).Error
			if err != nil {
				return err
			}
		}

		return nil
	}

	return nil
}

func loadAPIClientScope(dbConn *gorm.DB, apiClients map[string]*APIClient) error {
	var dbScopes []dbmodel.APIClientScope
	apiClientScopeResult := dbConn.Find(&dbScopes)
	if apiClientScopeResult.Error != nil {
		return fmt.Errorf("error loading client scopes: %v", apiClientScopeResult.Error)
	}

	if len(dbScopes) == 0 {
		return fmt.Errorf("no client scopes found")
	}

	apiClientScopes := make(map[string][]string)
	for _, scope := range dbScopes {
		var scopeList []string
		if _, ok := apiClientScopes[scope.APIClientID]; !ok {
			scopeList = make([]string, 0)
		} else {
			scopeList = apiClientScopes[scope.APIClientID]
		}
		scopeList = append(scopeList, scope.Scope)
		apiClientScopes[scope.APIClientID] = scopeList
	}

	for apiClientId, scopeList := range apiClientScopes {
		apiClient := apiClients[apiClientId]
		apiClient.Scopes = strings.Join(scopeList, " ")
		apiClients[apiClientId] = apiClient
	}

	return nil
}

func (c *APIClientStore) initializeAPIClientStore() error {
	if c.isLocalDev {
		err := createDefaultAPIClient(c.dbConn)
		if err != nil {
			return err
		}
		err = createDefaultAPIClientScope(c.dbConn)
		if err != nil {
			return err
		}
	}

	apiClients, err := loadAPIClient(c.dbConn)
	if err != nil {
		return err
	}
	c.apiClients = apiClients
	return loadAPIClientScope(c.dbConn, c.apiClients)
}
