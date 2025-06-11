package apiHandlers

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/kdjuwidja/aishoppercommon/logger"
)

type APIResponse struct {
	Code  string      `json:"code"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type response struct {
	Code     string
	Status   int
	ErrorStr string
}

type ResponseFactory struct {
}

func Initialize() *ResponseFactory {
	return &ResponseFactory{}
}

func (rf *ResponseFactory) CreateErrorResponse(c *gin.Context, err string) {
	response, ok := responseMap[err]
	if !ok {
		logger.Errorf("No status code found for error: %s", err)
		c.JSON(http.StatusInternalServerError, APIResponse{Code: ErrInternalServerError, Error: "Internal server error."})
		return
	}

	c.JSON(response.Status, APIResponse{Code: response.Code, Error: response.ErrorStr})
}

func (rf *ResponseFactory) CreateErrorResponsef(c *gin.Context, err string, args ...interface{}) {
	response, ok := responseMap[err]
	if !ok {
		logger.Errorf("No status code found for error: %s", err)
		c.JSON(http.StatusInternalServerError, APIResponse{Code: ErrInternalServerError, Error: "Internal server error."})
		return
	}

	formattedErrStr := fmt.Sprintf(response.ErrorStr, args...)
	c.JSON(response.Status, APIResponse{Code: response.Code, Error: formattedErrStr})
}

func (rf *ResponseFactory) createSuccessResponse(c *gin.Context, status int, data interface{}) {
	if data == nil || (reflect.ValueOf(data).Kind() == reflect.Slice && reflect.ValueOf(data).Len() == 0) {
		c.JSON(status, gin.H{})
	} else {
		c.JSON(status, data)
	}
}

func (rf *ResponseFactory) CreateOKResponse(c *gin.Context, data interface{}) {
	rf.createSuccessResponse(c, http.StatusOK, data)
}

func (rf *ResponseFactory) CreateCreatedResponse(c *gin.Context, data interface{}) {
	rf.createSuccessResponse(c, http.StatusCreated, data)
}
