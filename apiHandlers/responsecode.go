package apiHandlers

import "net/http"

const (
	ErrInvalidToken         = "GEN_00001"
	ErrInvalidRequestBody   = "GEN_00002"
	ErrMissingRequiredField = "GEN_00003"
	ErrMissingRequiredParam = "GEN_00004"
	ErrInvalidScope         = "GEN_00005"
	ErrInternalServerError  = "GEN_99999"
)

var responseMap = map[string]response{
	ErrInternalServerError:  {ErrInternalServerError, http.StatusInternalServerError, "Internal server error."},
	ErrInvalidToken:         {ErrInvalidToken, http.StatusUnauthorized, "Invalid or missing bearer token."},
	ErrInvalidRequestBody:   {ErrInvalidRequestBody, http.StatusBadRequest, "Invalid request body"},
	ErrMissingRequiredField: {ErrMissingRequiredField, http.StatusBadRequest, "Missing field in body: %s"},
	ErrMissingRequiredParam: {ErrMissingRequiredParam, http.StatusBadRequest, "Missing parameter: %s"},
	ErrInvalidScope:         {ErrInvalidScope, http.StatusForbidden, "Missing scope: %s"},
}
