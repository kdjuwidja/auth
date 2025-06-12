package apiHandlersauth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/kdjuwidja/aishoppercommon/logger"
)

type TokenHandler struct {
	srv        *server.Server
	tokenStore oauth2.TokenStore
}

func InitializeTokenHandler(srv *server.Server, tokenStore oauth2.TokenStore) *TokenHandler {
	return &TokenHandler{
		srv:        srv,
		tokenStore: tokenStore,
	}
}

func (h *TokenHandler) Handle(c *gin.Context) {
	if c.Request.Method != "POST" {
		logger.Tracef("/token POST Method not allowed")
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		return
	}

	if err := c.Request.ParseForm(); err != nil {
		logger.Tracef("/token POST Failed to parse form: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	code := c.PostForm("code")
	token, err := h.tokenStore.GetByCode(c.Request.Context(), code)
	if err != nil {
		logger.Tracef("/token POST Failed to get token: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code"})
		return
	}

	requestedScope := token.GetScope()

	logger.Tracef("/token POST code: %s, requestedScope: %s, grant_type: %s", code, requestedScope, c.PostForm("grant_type"))

	c.Request.Form.Set("requestedScope", requestedScope)
	err = h.srv.HandleTokenRequest(c.Writer, c.Request)
	if err != nil {
		logger.Tracef("/token POST Failed to handle token request: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to handle token request"})
		return
	}
}
