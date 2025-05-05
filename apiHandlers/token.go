package apiHandlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"netherealmstudio.com/m/v2/statestore"
)

type TokenHandler struct {
	srv        *server.Server
	stateStore *statestore.StateStore
}

func InitializeTokenHandler(srv *server.Server, stateStore *statestore.StateStore) *TokenHandler {
	return &TokenHandler{
		srv:        srv,
		stateStore: stateStore,
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
	state := c.PostForm("state")
	redirectURI := c.PostForm("redirect_uri")
	clientID := c.PostForm("client_id")

	logger.Tracef("/token POST code: %s, state: %s, redirectURI: %s, grant_type: %s, clientID: %s", code, state, redirectURI, c.PostForm("grant_type"), clientID)

	if !h.stateStore.ValidateWithClientInfo(state, clientID, redirectURI) {
		logger.Tracef("/token POST Invalid state or mismatched redirectURI")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state or mismatched redirectURI"})
		return
	}

	err := h.srv.HandleTokenRequest(c.Writer, c.Request)
	if err == nil {
		h.stateStore.DeleteState(state)
	}
}
