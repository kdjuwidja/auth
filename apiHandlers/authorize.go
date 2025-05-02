package apiHandlers

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"github.com/kdjuwidja/aishoppercommon/osutil"
	"netherealmstudio.com/m/v2/statestore"
)

type AuthorizeHandler struct {
	srv        *server.Server
	tmpl       *template.Template
	stateStore *statestore.StateStore
}

func InitializeAuthorizeHandler(srv *server.Server, tmpl *template.Template, stateStore *statestore.StateStore) *AuthorizeHandler {
	return &AuthorizeHandler{
		srv:        srv,
		tmpl:       tmpl,
		stateStore: stateStore,
	}
}

func (h *AuthorizeHandler) Handle(c *gin.Context) {
	switch c.Request.Method {
	case "GET":
		clientID := c.Query("client_id")
		redirectURI := c.Query("redirect_uri")
		state := c.Query("state")
		responseType := c.Query("response_type")
		scope := func() string {
			if s := c.Query("scope"); s != "" {
				return s
			}
			return "profile"
		}()

		if clientID == "" || redirectURI == "" || state == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing client_id, redirect_uri, or state"})
			return
		}

		// Store the client's state
		h.stateStore.Add(state, clientID, redirectURI)

		// Get service name from environment variable
		serviceName := osutil.GetEnvString("SERVICE_NAME", "auth")

		// Render template with parameters
		data := struct {
			ClientID     string
			RedirectURI  string
			State        string
			ResponseType string
			Scope        string
			Error        string
			BasePath     string
		}{
			ClientID:     clientID,
			RedirectURI:  redirectURI,
			State:        state,
			ResponseType: responseType,
			Scope:        scope,
			Error:        c.Query("error"),
			BasePath:     "/" + serviceName,
		}

		if err := h.tmpl.Execute(c.Writer, data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Template execution error"})
			logger.Errorf("Template execution error: %v", err)
			return
		}
	case "POST":
		clientID := c.PostForm("client_id")
		redirectURI := c.PostForm("redirect_uri")
		responseType := c.PostForm("response_type")
		scope := c.PostForm("scope")
		state := c.PostForm("state")

		logger.Tracef("/authorize POST clientID: %s, redirectURI: %s, responseType: %s, scope: %s, state: %s", clientID, redirectURI, responseType, scope, state)

		// Validate state with client info
		if !h.stateStore.ValidateWithClientInfo(state, clientID, redirectURI) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state or mismatched client information"})
			return
		}

		if err := h.srv.HandleAuthorizeRequest(c.Writer, c.Request); err != nil {
			logger.Errorf("Authorization error: %v", err)
		}
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}
