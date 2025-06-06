package token

import (
	"context"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/golang-jwt/jwt/v5"
	bizapiclient "netherealmstudio.com/m/v2/biz/apiclient"
	bizscope "netherealmstudio.com/m/v2/biz/scope"
)

// TokenGenerator handles JWT token generation
type AccessTokenGenerator struct {
	*generates.JWTAccessGenerate
	apiClientStore *bizapiclient.APIClientStore
	scopeAuthority *bizscope.ScopeAuthority
}

// NewAccessTokenGenerator creates a new token generator
func NewJWTTokenGenerator(key string, secret []byte, apiClientStore *bizapiclient.APIClientStore, scopeAuthority *bizscope.ScopeAuthority) *AccessTokenGenerator {
	return &AccessTokenGenerator{
		JWTAccessGenerate: generates.NewJWTAccessGenerate(key, secret, jwt.SigningMethodHS256),
		apiClientStore:    apiClientStore,
		scopeAuthority:    scopeAuthority,
	}
}

// Token generates a new JWT token
func (g *AccessTokenGenerator) Token(ctx context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (string, string, error) {
	requestedScope := data.Request.Form.Get("requestedScope")
	err := g.scopeAuthority.AuthorizeScope(ctx, data.Client.GetID(), data.UserID, requestedScope)
	if err != nil {
		return "", "", err
	}

	// Create claims
	claims := jwt.MapClaims{
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"sub":   data.UserID,
		"scope": requestedScope,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token using the secret key from JWTAccessGenerate
	accessToken, err := token.SignedString(g.SignedKey)
	if err != nil {
		return "", "", err
	}

	// Return empty refresh token since we're not implementing refresh token functionality
	return accessToken, "", nil
}
