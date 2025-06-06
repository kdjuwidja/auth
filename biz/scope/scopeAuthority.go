package bizscope

import (
	"context"
	"fmt"
	"strings"

	"github.com/kdjuwidja/aishoppercommon/logger"
	"gorm.io/gorm"
)

type ScopeAuthority struct {
	dbConn *gorm.DB
}

func NewScopeAuthority(dbConn *gorm.DB) *ScopeAuthority {
	return &ScopeAuthority{
		dbConn: dbConn,
	}
}

func (s *ScopeAuthority) AuthorizeScope(ctx context.Context, apiClientID string, userID string, requestedScope string) error {
	if requestedScope == "" {
		return nil
	}

	apiClientScopes := []string{}
	err := s.dbConn.WithContext(ctx).Raw("SELECT DISTINCT(scope) FROM api_clients INNER JOIN api_client_scopes ON api_clients.id = api_client_scopes.api_client_id WHERE api_client_id = ?", apiClientID).Scan(&apiClientScopes).Error
	if err != nil {
		return err
	}
	if len(apiClientScopes) == 0 {
		logger.Errorf("api client does not have any scopes, apiClientID: %s", apiClientID)
		return fmt.Errorf("the requested scope is invalid, unknown, or malformed")
	}

	userScopes := []string{}
	err = s.dbConn.WithContext(ctx).Raw("SELECT DISTINCT(scope) FROM role_scopes INNER JOIN (SELECT user_id, role_id FROM user_roles WHERE user_id = ?) as tbl1 ON role_scopes.role_id = tbl1.role_id", userID).Scan(&userScopes).Error
	if err != nil {
		return err
	}
	if len(userScopes) == 0 {
		logger.Errorf("user does not have any scopes, userID: %s", userID)
		return fmt.Errorf("the requested scope is invalid, unknown, or malformed")
	}

	rs := strings.Split(requestedScope, " ")

	// Check if requestedScopes is a subset of userScopes
	if !isSubset(rs, userScopes) {
		logger.Errorf("user does not have all requested scopes, userID: %s, requestedScope: %s, userScopes: %v", userID, requestedScope, userScopes)
		return fmt.Errorf("the requested scope is invalid, unknown, or malformed")
	}

	// Check if requestedScopes is a subset of apiClientScopes
	if !isSubset(rs, apiClientScopes) {
		logger.Errorf("api client does not have all requested scopes, apiClientID: %s, requestedScope: %s, apiClientScopes: %v", apiClientID, requestedScope, apiClientScopes)
		return fmt.Errorf("the requested scope is invalid, unknown, or malformed")
	}

	return nil
}

// isSubset checks if all elements in subset are present in superset
func isSubset(subset, superset []string) bool {
	// Create a map for O(1) lookup
	supersetMap := make(map[string]bool)
	for _, s := range superset {
		supersetMap[s] = true
	}

	// Check if all elements in subset exist in superset
	for _, s := range subset {
		if !supersetMap[s] {
			return false
		}
	}
	return true
}
