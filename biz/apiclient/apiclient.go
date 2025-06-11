package bizapiclient

import (
	"fmt"
	"strings"

	"github.com/kdjuwidja/aishoppercommon/logger"
	"gorm.io/gorm"
	dbmodel "netherealmstudio.com/m/v2/db"
	"netherealmstudio.com/m/v2/defaults"
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
	for _, client := range defaults.DEFAULT_API_CLIENTS {
		err := createDBRecords(dbConn,
			client["id"].(string),
			client["secret"].(string),
			client["domain"].(string),
			client["is_public"].(bool),
			client["description"].(string),
			client["scopes"].(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func createDBRecords(dbConn *gorm.DB, clientId string, clientSecret string, clientDomain string, clientIsPublic bool, clientDescription string, clientScopes string) error {
	var client dbmodel.APIClient
	result := dbConn.Where("id = ?", clientId).First(&client)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("error loading user client: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		client = dbmodel.APIClient{
			ID:          clientId,
			Secret:      clientSecret,
			Domain:      clientDomain,
			IsPublic:    clientIsPublic,
			Description: clientDescription,
		}
		err := dbConn.Create(&client).Error
		if err != nil {
			return fmt.Errorf("error creating api client: %v", err)
		}
	}

	scopes := strings.Split(clientScopes, " ")
	for _, scope := range scopes {
		var apiClientScope dbmodel.APIClientScope
		result := dbConn.Where("api_client_id = ? AND scope = ?", clientId, scope).First(&apiClientScope)
		if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			return fmt.Errorf("error loading api client scope: %v", result.Error)
		}

		if result.RowsAffected == 0 {
			apiClientScope = dbmodel.APIClientScope{
				APIClientID: clientId,
				Scope:       scope,
			}
			err := dbConn.Create(&apiClientScope).Error
			if err != nil {
				return fmt.Errorf("error creating api client scope: %v", err)
			}
		}
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
		logger.Info("Creating default API clients...")
		err := createDefaultAPIClient(c.dbConn)
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
