package apiHandlers

import (
	"net/http"

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

func (h *TokenHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	code := r.PostFormValue("code")
	state := r.PostFormValue("state")
	redirectURI := r.PostFormValue("redirect_uri")
	clientID := r.PostFormValue("client_id")

	logger.Tracef("/token POST code: %s, state: %s, redirectURI: %s, grant_type: %s, clientID: %s", code, state, redirectURI, r.PostFormValue("grant_type"), clientID)

	if !h.stateStore.ValidateWithClientInfo(state, clientID, redirectURI) {
		http.Error(w, "Invalid state or mismatched redirectURI", http.StatusBadRequest)
		return
	}

	err := h.srv.HandleTokenRequest(w, r)
	if err == nil {
		h.stateStore.DeleteState(state)
	}
}
