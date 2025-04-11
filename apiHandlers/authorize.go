package apiHandlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/kdjuwidja/aishoppercommon/logger"
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

func (h *AuthorizeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		clientID := r.URL.Query().Get("client_id")
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		responseType := r.URL.Query().Get("response_type")
		scope := func() string {
			if s := r.URL.Query().Get("scope"); s != "" {
				return s
			}
			return "profile"
		}()

		if clientID == "" || redirectURI == "" || state == "" {
			http.Error(w, "Missing client_id, redirect_uri, or state", http.StatusBadRequest)
			return
		}

		// Store the client's state
		h.stateStore.Add(state, clientID, redirectURI)

		// Render template with parameters
		data := struct {
			ClientID     string
			RedirectURI  string
			State        string
			ResponseType string
			Scope        string
			Error        string
		}{
			ClientID:     clientID,
			RedirectURI:  redirectURI,
			State:        state,
			ResponseType: responseType,
			Scope:        scope,
			Error:        r.URL.Query().Get("error"),
		}

		if err := h.tmpl.Execute(w, data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Errorf("Template execution error: %v", err)
			return
		}
	case "POST":
		clientID := r.FormValue("client_id")
		redirectURI := r.FormValue("redirect_uri")
		responseType := r.FormValue("response_type")
		scope := r.FormValue("scope")
		state := r.FormValue("state")

		logger.Tracef("/authorize POST clientID: %s, redirectURI: %s, responseType: %s, scope: %s, state: %s", clientID, redirectURI, responseType, scope, state)

		// Validate state with client info
		if !h.stateStore.ValidateWithClientInfo(state, clientID, redirectURI) {
			http.Error(w, "Invalid state or mismatched client information", http.StatusBadRequest)
			return
		}

		if err := h.srv.HandleAuthorizeRequest(w, r); err != nil {
			log.Printf("Authorization error: %v", err)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
