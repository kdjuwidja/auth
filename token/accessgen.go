package token

import (
	"context"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/golang-jwt/jwt/v5"
)

// TokenGenerator handles JWT token generation
type AccessTokenGenerator struct {
	*generates.JWTAccessGenerate
}

// NewAccessTokenGenerator creates a new token generator
func NewJWTTokenGenerator(key string, secret []byte) *AccessTokenGenerator {
	return &AccessTokenGenerator{
		JWTAccessGenerate: generates.NewJWTAccessGenerate(key, secret, jwt.SigningMethodHS256),
	}
}

// Token generates a new JWT token
func (g *AccessTokenGenerator) Token(ctx context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (string, string, error) {
	// Get the scope from the request
	scope := data.Request.FormValue("scope")
	if scope == "" {
		scope = "user profile" // Default scope
	}

	// Create claims
	claims := jwt.MapClaims{
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"sub":   data.UserID,
		"scope": scope,
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
