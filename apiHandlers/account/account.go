package apiHandlersaccount

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"netherealmstudio.com/m/v2/apiHandlers"
	bizRegister "netherealmstudio.com/m/v2/biz/register"
)

type AccountHandler struct {
	registrationManager *bizRegister.RegistrationManager
	responseFactory     *apiHandlers.ResponseFactory
}

func InitializeAccountHandler(registrationManager *bizRegister.RegistrationManager, responseFactory *apiHandlers.ResponseFactory) *AccountHandler {
	return &AccountHandler{
		registrationManager: registrationManager,
		responseFactory:     responseFactory,
	}
}

func (h *AccountHandler) GetRegistrationCode(c *gin.Context) {
	code, err := h.registrationManager.GetRegistrationCode(c.Request.Context())
	if err != nil {
		h.responseFactory.CreateErrorResponse(c, apiHandlers.ErrInternalServerError)
		return
	}

	h.responseFactory.CreateOKResponse(c, map[string]string{"code": code})
}

func (h *AccountHandler) RegisterAccount(c *gin.Context) {
	var req struct {
		Code     string `json:"code"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.responseFactory.CreateErrorResponse(c, apiHandlers.ErrInvalidRequestBody)
		return
	}

	if req.Code == "" || req.Email == "" || req.Password == "" {
		h.responseFactory.CreateErrorResponsef(c, apiHandlers.ErrMissingRequiredField, "code, email, and password are required")
		return
	}

	err := h.registrationManager.RegisterUser(c.Request.Context(), req.Code, req.Email, req.Password)
	if err != nil {
		logger.Errorf("failed to register user: %v", err)
		h.responseFactory.CreateErrorResponse(c, apiHandlers.ErrInternalServerError)
		return
	}

	h.responseFactory.CreateOKResponse(c, map[string]string{"message": "Account registered successfully"})
}
