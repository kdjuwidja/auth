package apiHandlers

import (
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kdjuwidja/aishoppercommon/logger"
)

type TokenVerifier struct {
	responseFactory ResponseFactory
}

func InitializeTokenVerifier(responseFactory ResponseFactory) *TokenVerifier {
	return &TokenVerifier{
		responseFactory: responseFactory,
	}
}

func (v *TokenVerifier) VerifyToken(scopes []string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" || len(token) < 7 || token[:7] != "Bearer " {
			v.responseFactory.CreateErrorResponse(c, ErrInvalidToken)
			c.Abort()
			return
		}

		token = token[7:]

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			logger.Warn("JWT secret not configured. Using default secret.")
			secret = "my-secret-key"
		}

		tokenObj, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != "HS256" {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !tokenObj.Valid {
			v.responseFactory.CreateErrorResponse(c, ErrInvalidToken)
			c.Abort()
			return
		}

		mapClaims, ok := tokenObj.Claims.(jwt.MapClaims)
		if !ok {
			v.responseFactory.CreateErrorResponse(c, ErrInvalidToken)
			c.Abort()
			return
		}

		if mapClaims["scope"] == nil {
			v.responseFactory.CreateErrorResponse(c, ErrInvalidToken)
			c.Abort()
			return
		}

		jwtScopes := strings.Split(mapClaims["scope"].(string), " ")
		for _, scope := range scopes {
			if !slices.Contains(jwtScopes, scope) {
				v.responseFactory.CreateErrorResponsef(c, ErrInvalidScope, scope)
				c.Abort()
				return
			}
		}

		// Extract and set user ID
		if userID, exists := mapClaims["sub"].(string); exists {
			c.Set("userID", userID)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user ID in token"})
			c.Abort()
			return
		}

		next(c)
	}
}
